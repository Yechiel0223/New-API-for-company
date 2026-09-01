package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	kitdto "github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModelUsageRelayTest(t *testing.T, requestID string, baseURL string, modelMapping *string, retries int) *gin.Context {
	t.Helper()
	originalDB := model.DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalRedisEnabled := common.RedisEnabled
	originalRetryTimes := common.RetryTimes
	originalCountToken := constant.CountToken
	originalErrorLogEnabled := constant.ErrorLogEnabled
	t.Cleanup(func() {
		model.DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		if originalMemoryCacheEnabled && originalDB != nil {
			model.InitChannelCache()
		}
		common.RedisEnabled = originalRedisEnabled
		common.RetryTimes = originalRetryTimes
		constant.CountToken = originalCountToken
		constant.ErrorLogEnabled = originalErrorLogEnabled
	})

	common.MemoryCacheEnabled = true
	common.RedisEnabled = false
	common.RetryTimes = retries
	constant.CountToken = false
	constant.ErrorLogEnabled = false

	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.User{},
		&model.Channel{},
		&model.Ability{},
		&model.ModelUsageEvent{},
	))
	model.DB = database
	require.NoError(t, model.DB.Create(&model.User{
		Id: 7, Username: "alice", Password: "test-password", Quota: 100_000_000,
		Status: common.UserStatusEnabled,
	}).Error)

	priority := int64(0)
	weight := uint(1)
	autoBan := 0
	channel := &model.Channel{
		Id: 23, Type: constant.ChannelTypeOpenAI, Key: "test-key",
		Status: common.ChannelStatusEnabled, Name: "local-test",
		Weight: &weight, BaseURL: &baseURL, Models: "gpt-3.5-turbo", Group: "default",
		ModelMapping: modelMapping, Priority: &priority, AutoBan: &autoBan,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group: "default", Model: "gpt-3.5-turbo", ChannelId: channel.Id,
		Enabled: true, Priority: &priority, Weight: weight,
	}).Error)
	model.InitChannelCache()

	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(
		`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"hello"}]}`,
	))
	request.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = request
	c.Set(common.RequestIdKey, requestID)
	c.Set("token_quota", 100_000_000)
	c.Set("token_name", "test-token")
	common.SetContextKey(c, constant.ContextKeyUserId, 7)
	common.SetContextKey(c, constant.ContextKeyUserName, "alice")
	common.SetContextKey(c, constant.ContextKeyUserQuota, 100_000_000)
	common.SetContextKey(c, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(c, constant.ContextKeyUsingGroup, "default")
	common.SetContextKey(c, constant.ContextKeyTokenGroup, "default")
	common.SetContextKey(c, constant.ContextKeyTokenId, 11)
	common.SetContextKey(c, constant.ContextKeyTokenKey, "sk-test-key")
	common.SetContextKey(c, constant.ContextKeyTokenUnlimited, true)
	common.SetContextKey(c, constant.ContextKeyUserSetting, kitdto.UserSetting{
		BillingPreference:     "wallet_only",
		AcceptUnsetRatioModel: true,
	})
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now().Add(-time.Second))
	require.Nil(t, middleware.SetupContextForSelectedChannel(c, channel, "gpt-3.5-turbo"))
	return c
}

func TestRelayPreUpstreamFailureDoesNotRecordModelUsage(t *testing.T) {
	invalidMapping := "{"
	c := setupModelUsageRelayTest(t, "req-local-validation", "http://unused.invalid", &invalidMapping, 0)

	Relay(c, types.RelayFormatOpenAI)

	var count int64
	require.NoError(t, model.DB.Model(&model.ModelUsageEvent{}).Count(&count).Error)
	assert.Zero(t, count)
}

func TestRelayRetryFailureRecordsOneRunningThenFailedModelUsage(t *testing.T) {
	type usageSnapshot struct {
		Count  int64
		Status model.ModelUsageStatus
		Err    error
	}
	var attempts atomic.Int32
	snapshots := make(chan usageSnapshot, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		var rows []model.ModelUsageEvent
		err := model.DB.Find(&rows).Error
		snapshot := usageSnapshot{Count: int64(len(rows)), Err: err}
		if len(rows) == 1 {
			snapshot.Status = rows[0].Status
		}
		snapshots <- snapshot
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"retryable upstream failure","type":"server_error"}}`))
	}))
	defer server.Close()

	c := setupModelUsageRelayTest(t, "req-retry-failure", server.URL, nil, 1)
	Relay(c, types.RelayFormatOpenAI)

	require.Equal(t, int32(2), attempts.Load())
	for range 2 {
		snapshot := <-snapshots
		require.NoError(t, snapshot.Err)
		assert.Equal(t, int64(1), snapshot.Count)
		assert.Equal(t, model.ModelUsageStatusRunning, snapshot.Status)
	}
	var rows []model.ModelUsageEvent
	require.NoError(t, model.DB.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, "request:req-retry-failure", rows[0].EventKey)
	assert.Equal(t, model.ModelUsageStatusFailure, rows[0].Status)
}
