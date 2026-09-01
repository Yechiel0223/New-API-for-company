package relay

import (
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type usageDoRequestAdaptor struct {
	channel.Adaptor
	calls int
}

func (a *usageDoRequestAdaptor) DoRequest(*gin.Context, *relaycommon.RelayInfo, io.Reader) (any, error) {
	a.calls++
	return nil, nil
}

func TestSyncModelUsageAdaptorSkipsChannelTest(t *testing.T) {
	originalDB := model.DB
	t.Cleanup(func() { model.DB = originalDB })
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(&model.ModelUsageEvent{}))
	model.DB = database

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(c, constant.ContextKeyUserName, "admin")
	common.SetContextKey(c, constant.ContextKeyChannelId, 23)
	info := &relaycommon.RelayInfo{
		RequestId: "channel-test-request", IsChannelTest: true,
		OriginModelName: "gpt-4o-mini", StartTime: time.Now(),
	}
	delegate := &usageDoRequestAdaptor{}
	adaptor := &syncModelUsageAdaptor{Adaptor: delegate}

	_, err = adaptor.DoRequest(c, info, nil)

	require.NoError(t, err)
	assert.Equal(t, 1, delegate.calls)
	assert.False(t, info.SyncModelUsageStarted)
	var count int64
	require.NoError(t, model.DB.Model(&model.ModelUsageEvent{}).Count(&count).Error)
	assert.Zero(t, count)
}
