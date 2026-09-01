package service

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUsageRecorderDB(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	t.Cleanup(func() { model.DB = originalDB })

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.ModelUsageEvent{}, &model.User{}, &model.Channel{}))
	model.DB = database
}

type usageRecorderBillingSettler struct{}

func (usageRecorderBillingSettler) Settle(int) error         { return nil }
func (usageRecorderBillingSettler) Refund(*gin.Context)      {}
func (usageRecorderBillingSettler) NeedsRefund() bool        { return false }
func (usageRecorderBillingSettler) GetPreConsumedQuota() int { return 0 }
func (usageRecorderBillingSettler) Reserve(int) error        { return nil }

func usageRecorderContext(requestID string) (*gin.Context, *relaycommon.RelayInfo) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeyUserName, "alice")
	common.SetContextKey(c, constant.ContextKeyChannelId, 23)
	return c, &relaycommon.RelayInfo{
		RequestId:       requestID,
		UserId:          7,
		TokenId:         11,
		OriginModelName: "doubao-seedream-5-0-pro-260628",
		UsingGroup:      "vip",
		UserQuota:       1_000_000,
		StartTime:       time.Now().Add(-1500 * time.Millisecond),
	}
}

func TestRecordSyncModelUsageUpsertsOneTerminalRequest(t *testing.T) {
	setupUsageRecorderDB(t)
	c, info := usageRecorderContext("req-sync-1")

	RecordSyncModelUsageStarted(c, info)

	var running model.ModelUsageEvent
	require.NoError(t, model.DB.Where("event_key = ?", "request:req-sync-1").First(&running).Error)
	assert.Equal(t, model.ModelUsageStatusRunning, running.Status)
	assert.Equal(t, "req-sync-1", running.RequestID)
	assert.Equal(t, 7, running.UserID)
	assert.Equal(t, "alice", running.Username)
	assert.Equal(t, 11, running.TokenID)
	assert.Equal(t, 23, running.ChannelID)
	assert.Equal(t, "doubao-seedream-5-0-pro-260628", running.ModelName)
	assert.Equal(t, "vip", running.UseGroup)
	assert.Equal(t, info.StartTime.Unix(), running.SubmittedAt)
	assert.Equal(t, info.StartTime.Unix(), running.LastProgressAt)
	assert.Equal(t, "live", running.Source)

	RecordSyncModelUsage(c, info, SyncModelUsageResult{
		Success: true, TotalTokens: 7, OutputCount: 2, OutputUnit: "image", FinalQuota: 123,
	})
	RecordSyncModelUsage(c, info, SyncModelUsageResult{
		Success: true, TotalTokens: 7, OutputCount: 2, OutputUnit: "image", FinalQuota: 123,
	})

	var rows []model.ModelUsageEvent
	require.NoError(t, model.DB.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, "request:req-sync-1", rows[0].EventKey)
	assert.Equal(t, model.ModelUsageStatusSuccess, rows[0].Status)
	assert.Equal(t, int64(7), rows[0].TotalTokens)
	assert.Equal(t, 2, rows[0].OutputCount)
	assert.Equal(t, "image", rows[0].OutputUnit)
	assert.Equal(t, 123, rows[0].FinalQuota)
	assert.GreaterOrEqual(t, rows[0].DurationMs, int64(1_500))
	assert.GreaterOrEqual(t, rows[0].CompletedAt, rows[0].SubmittedAt)
	assert.Empty(t, rows[0].FailureReason)
}

func TestRecordSyncModelUsageFinalRetryFailureKeepsOneMaskedFact(t *testing.T) {
	setupUsageRecorderDB(t)
	c, info := usageRecorderContext("req-sync-failure")

	RecordSyncModelUsageStarted(c, info)
	common.SetContextKey(c, constant.ContextKeyChannelId, 24)
	maskedReason := strings.Repeat("错", 511) + "终止"
	RecordSyncModelUsage(c, info, SyncModelUsageResult{
		Success: false, FailureReason: maskedReason,
	})

	var rows []model.ModelUsageEvent
	require.NoError(t, model.DB.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, model.ModelUsageStatusFailure, rows[0].Status)
	assert.Zero(t, rows[0].TotalTokens)
	assert.Zero(t, rows[0].FinalQuota)
	assert.Zero(t, rows[0].OutputCount)
	assert.Empty(t, rows[0].OutputUnit)
	assert.Equal(t, 23, rows[0].ChannelID, "retry channel changes must not create or replace the request fact")
	assert.Equal(t, 512, utf8.RuneCountInString(rows[0].FailureReason))
	assert.Equal(t, strings.Repeat("错", 511)+"终", rows[0].FailureReason)
	assert.GreaterOrEqual(t, rows[0].DurationMs, int64(1_500))
}

func TestRecordSyncModelUsageAfterGeneralQuotaSettlement(t *testing.T) {
	setupUsageRecorderDB(t)
	originalLogConsumeEnabled := common.LogConsumeEnabled
	originalQuotaPerUnit := common.QuotaPerUnit
	common.LogConsumeEnabled = false
	common.QuotaPerUnit = 1_000
	t.Cleanup(func() {
		common.LogConsumeEnabled = originalLogConsumeEnabled
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	c, info := usageRecorderContext("req-sync-image")
	info.Billing = usageRecorderBillingSettler{}
	info.PriceData = hosttypes.PriceData{
		ModelRatio:     1,
		GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
	}
	RecordSyncModelUsageStarted(c, info)
	info.ChannelMeta = &relaycommon.ChannelMeta{ChannelId: 23}

	PostAudioConsumeQuota(c, info, &dto.Usage{
		PromptTokens: 5, CompletionTokens: 2, TotalTokens: 7, GeneratedImages: 2,
		PromptTokensDetails: dto.InputTokenDetails{TextTokens: 123},
	}, "")

	var event model.ModelUsageEvent
	require.NoError(t, model.DB.Where("event_key = ?", "request:req-sync-image").First(&event).Error)
	assert.Equal(t, model.ModelUsageStatusSuccess, event.Status)
	assert.Equal(t, int64(7), event.TotalTokens)
	assert.Equal(t, 2, event.OutputCount)
	assert.Equal(t, "image", event.OutputUnit)
	assert.Equal(t, 123, event.FinalQuota)
}

func TestRecordSyncModelUsageAfterTextQuotaSettlement(t *testing.T) {
	setupUsageRecorderDB(t)
	originalLogConsumeEnabled := common.LogConsumeEnabled
	originalQuotaPerUnit := common.QuotaPerUnit
	common.LogConsumeEnabled = false
	common.QuotaPerUnit = 1_000
	t.Cleanup(func() {
		common.LogConsumeEnabled = originalLogConsumeEnabled
		common.QuotaPerUnit = originalQuotaPerUnit
	})

	c, info := usageRecorderContext("req-sync-text")
	info.Billing = usageRecorderBillingSettler{}
	info.ChannelMeta = &relaycommon.ChannelMeta{ChannelId: 23}
	info.PriceData = hosttypes.PriceData{
		UsePrice: true, ModelPrice: 0.456, ModelRatio: 1,
		GroupRatioInfo: hosttypes.GroupRatioInfo{GroupRatio: 1},
	}
	RecordSyncModelUsageStarted(c, info)

	PostTextConsumeQuota(c, info, &dto.Usage{
		PromptTokens: 4, CompletionTokens: 3, TotalTokens: 7,
	}, nil)

	var event model.ModelUsageEvent
	require.NoError(t, model.DB.Where("event_key = ?", "request:req-sync-text").First(&event).Error)
	assert.Equal(t, model.ModelUsageStatusSuccess, event.Status)
	assert.Equal(t, int64(7), event.TotalTokens)
	assert.Zero(t, event.OutputCount)
	assert.Empty(t, event.OutputUnit)
	assert.Equal(t, 456, event.FinalQuota)
}
