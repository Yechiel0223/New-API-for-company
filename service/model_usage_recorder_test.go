package service

import (
	"fmt"
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
	originalLogDB := model.LOG_DB
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
	})

	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, database.AutoMigrate(
		&model.ModelUsageEvent{},
		&model.ModelUsageAttempt{},
		&model.Task{},
		&model.User{},
		&model.Token{},
		&model.Channel{},
		&model.Log{},
	))
	model.DB = database
	model.LOG_DB = database
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

func taskUsageFixture(taskID string, status model.TaskStatus, quota int) *model.Task {
	return &model.Task{
		TaskID:     taskID,
		UserId:     7,
		Group:      "vip",
		ChannelId:  23,
		Quota:      quota,
		Status:     status,
		Progress:   "10%",
		SubmitTime: 1_700_000_000,
		Properties: model.Properties{OriginModelName: "doubao-seedance-1-0-pro-250528"},
		PrivateData: model.TaskPrivateData{
			TokenId: 11,
			BillingContext: &model.TaskBillingContext{
				OriginModelName: "doubao-seedance-1-0-pro-250528",
			},
		},
	}
}

func requireSingleTaskUsageEvent(t *testing.T, eventKey string) model.ModelUsageEvent {
	t.Helper()
	var events []model.ModelUsageEvent
	require.NoError(t, model.DB.Where("event_key = ?", eventKey).Find(&events).Error)
	require.Len(t, events, 1)
	return events[0]
}

func TestTaskUsageLifecycleKeepsOneEventAndActualTokens(t *testing.T) {
	setupUsageRecorderDB(t)
	c, info := usageRecorderContext("req-task-1")
	task := taskUsageFixture("task-1", model.TaskStatusSubmitted, 1_000)

	RecordTaskUsageSubmitted(c, info, task)
	TouchTaskUsageProgress(task, model.TaskStatusInProgress, 1_700_000_010)
	task.Status = model.TaskStatusSuccess
	task.FinishTime = 1_700_000_050
	task.Quota = 900
	FinalizeTaskUsage(t.Context(), task, &relaycommon.TaskInfo{TotalTokens: 87_300})
	FinalizeTaskUsage(t.Context(), task, &relaycommon.TaskInfo{TotalTokens: 87_300})

	event := requireSingleTaskUsageEvent(t, "task:task-1")
	assert.Equal(t, model.ModelUsageKindTask, event.Kind)
	assert.Equal(t, "req-task-1", event.RequestID)
	assert.Equal(t, "task-1", event.TaskID)
	assert.Equal(t, int64(87_300), event.TotalTokens)
	assert.Equal(t, 1, event.OutputCount)
	assert.Equal(t, "video", event.OutputUnit)
	assert.Equal(t, 900, event.FinalQuota)
	assert.Equal(t, int64(50_000), event.DurationMs)
	assert.Equal(t, model.ModelUsageStatusSuccess, event.Status)
}

func TestTaskUsageLifecycleUsesActualTokenFallbackOrder(t *testing.T) {
	testCases := []struct {
		name       string
		result     *relaycommon.TaskInfo
		wantTokens int64
	}{
		{name: "completion tokens", result: &relaycommon.TaskInfo{CompletionTokens: 321, UsageFacts: map[string]any{"tokens": float64(654)}}, wantTokens: 321},
		{name: "numeric usage fact", result: &relaycommon.TaskInfo{UsageFacts: map[string]any{"tokens": float64(654)}}, wantTokens: 654},
		{name: "non-positive usage fact", result: &relaycommon.TaskInfo{UsageFacts: map[string]any{"tokens": float64(0)}}, wantTokens: 0},
	}
	for index, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			setupUsageRecorderDB(t)
			c, info := usageRecorderContext("req-token-fallback")
			task := taskUsageFixture(fmt.Sprintf("token-fallback-%d", index), model.TaskStatusSuccess, 100)
			task.FinishTime = task.SubmitTime + 1
			RecordTaskUsageSubmitted(c, info, task)

			FinalizeTaskUsage(t.Context(), task, testCase.result)

			event := requireSingleTaskUsageEvent(t, "task:"+task.TaskID)
			assert.Equal(t, testCase.wantTokens, event.TotalTokens)
		})
	}
}

func TestTaskUsageFailureRecordsRefundedQuotaAndReason(t *testing.T) {
	setupUsageRecorderDB(t)
	c, info := usageRecorderContext("req-task-failure")
	task := taskUsageFixture("task-failure", model.TaskStatusSubmitted, 1_000)
	RecordTaskUsageSubmitted(c, info, task)

	task.Status = model.TaskStatusFailure
	task.FinishTime = 1_700_000_025
	task.FailReason = "upstream rejected the video"
	task.Quota = 0
	FinalizeTaskUsage(t.Context(), task, &relaycommon.TaskInfo{TotalTokens: 999})

	event := requireSingleTaskUsageEvent(t, "task:task-failure")
	assert.Equal(t, model.ModelUsageStatusFailure, event.Status)
	assert.Zero(t, event.TotalTokens)
	assert.Zero(t, event.OutputCount)
	assert.Empty(t, event.OutputUnit)
	assert.Zero(t, event.FinalQuota)
	assert.Equal(t, "upstream rejected the video", event.FailureReason)
}

func TestTaskUsageFailureTimeoutSweeperFinalizesAfterRefund(t *testing.T) {
	setupUsageRecorderDB(t)
	const userID = 7
	require.NoError(t, model.DB.Create(&model.User{Id: userID, Username: "alice", Quota: 10_000, Status: common.UserStatusEnabled}).Error)

	c, info := usageRecorderContext("req-task-timeout")
	task := taskUsageFixture("task-timeout", model.TaskStatusInProgress, 1_200)
	task.PrivateData.TokenId = 0
	task.SubmitTime = time.Now().Add(-20 * time.Minute).Unix()
	task.CreatedAt = task.SubmitTime
	task.UpdatedAt = task.SubmitTime
	require.NoError(t, model.DB.Create(task).Error)
	RecordTaskUsageSubmitted(c, info, task)

	previousTimeout := constant.TaskTimeoutMinutes
	constant.TaskTimeoutMinutes = 15
	t.Cleanup(func() { constant.TaskTimeoutMinutes = previousTimeout })
	sweepTimedOutTasks(t.Context())

	var persisted model.Task
	require.NoError(t, model.DB.First(&persisted, task.ID).Error)
	assert.Equal(t, model.TaskStatus(model.TaskStatusFailure), persisted.Status)
	assert.Zero(t, persisted.Quota)
	event := requireSingleTaskUsageEvent(t, "task:task-timeout")
	assert.Equal(t, model.ModelUsageStatusFailure, event.Status)
	assert.Zero(t, event.FinalQuota)
	assert.Contains(t, event.FailureReason, "任务超时")
}

func TestTaskUsageLifecycleBatchNoOpDoesNotRefreshProgress(t *testing.T) {
	setupUsageRecorderDB(t)
	const channelID = 231
	require.NoError(t, model.DB.Create(&model.Channel{
		Id: channelID, Type: constant.ChannelTypeKling, Name: "usage-no-op", Key: "sk-test", Status: common.ChannelStatusEnabled,
	}).Error)

	c, info := usageRecorderContext("req-task-no-op")
	task := taskUsageFixture("task-no-op", model.TaskStatusInProgress, 1_000)
	task.ChannelId = channelID
	task.PrivateData.UpstreamTaskID = "upstream-no-op"
	require.NoError(t, model.DB.Create(task).Error)
	RecordTaskUsageSubmitted(c, info, task)

	adaptor := &batchPollingAdaptor{results: map[string]*BatchTaskResult{
		task.GetUpstreamTaskID(): {TaskInfo: relaycommon.TaskInfo{
			TaskID: task.GetUpstreamTaskID(), Status: model.TaskStatusInProgress, Progress: task.Progress,
		}},
	}}
	require.NoError(t, updateBatchTasks(t.Context(), adaptor, channelID, []string{task.GetUpstreamTaskID()}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	}))

	event := requireSingleTaskUsageEvent(t, "task:task-no-op")
	assert.Equal(t, task.SubmitTime, event.LastProgressAt)
}

func TestTaskUsageFailureBatchPollingFinalizesAfterRefund(t *testing.T) {
	setupUsageRecorderDB(t)
	const channelID = 232
	require.NoError(t, model.DB.Create(&model.User{Id: 7, Username: "alice", Quota: 10_000, Status: common.UserStatusEnabled}).Error)
	require.NoError(t, model.DB.Create(&model.Channel{
		Id: channelID, Type: constant.ChannelTypeKling, Name: "usage-failure", Key: "sk-test", Status: common.ChannelStatusEnabled,
	}).Error)

	c, info := usageRecorderContext("req-task-batch-failure")
	task := taskUsageFixture("task-batch-failure", model.TaskStatusInProgress, 1_000)
	task.ChannelId = channelID
	task.PrivateData.TokenId = 0
	task.PrivateData.UpstreamTaskID = "upstream-batch-failure"
	task.PrivateData.BillingSource = BillingSourceWallet
	require.NoError(t, model.DB.Create(task).Error)
	RecordTaskUsageSubmitted(c, info, task)

	adaptor := &batchPollingAdaptor{results: map[string]*BatchTaskResult{
		task.GetUpstreamTaskID(): {TaskInfo: relaycommon.TaskInfo{
			TaskID: task.GetUpstreamTaskID(), Status: model.TaskStatusFailure, Reason: "provider failed", TotalTokens: 999,
		}},
	}}
	require.NoError(t, updateBatchTasks(t.Context(), adaptor, channelID, []string{task.GetUpstreamTaskID()}, map[string]*model.Task{
		task.GetUpstreamTaskID(): task,
	}))

	event := requireSingleTaskUsageEvent(t, "task:task-batch-failure")
	assert.Equal(t, model.ModelUsageStatusFailure, event.Status)
	assert.Zero(t, event.FinalQuota)
	assert.Zero(t, event.TotalTokens)
	assert.Equal(t, "provider failed", event.FailureReason)
	assert.Greater(t, event.CompletedAt, task.SubmitTime)
	assert.Equal(t, (event.CompletedAt-task.SubmitTime)*1_000, event.DurationMs)
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

func TestRecordSyncModelUsageAttemptPersistsRetryChannelsWithoutDuplicatingFact(t *testing.T) {
	setupUsageRecorderDB(t)
	c, info := usageRecorderContext("req-sync-attempts")
	RecordSyncModelUsageStarted(c, info)

	RecordSyncModelUsageAttempt(c, info, 23, false)
	RecordSyncModelUsageAttempt(c, info, 24, false)
	RecordSyncModelUsageAttempt(c, info, 24, true)

	var facts []model.ModelUsageEvent
	require.NoError(t, model.DB.Find(&facts).Error)
	require.Len(t, facts, 1)
	var attempts []model.ModelUsageAttempt
	require.NoError(t, model.DB.Order("channel_id ASC").Find(&attempts).Error)
	require.Len(t, attempts, 2)
	assert.Equal(t, 23, attempts[0].ChannelID)
	assert.Equal(t, model.ModelUsageStatusFailure, attempts[0].Status)
	assert.Equal(t, 24, attempts[1].ChannelID)
	assert.Equal(t, model.ModelUsageStatusSuccess, attempts[1].Status)
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
