package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBackfillDatabases(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
	})

	mainDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "main.db")), &gorm.Config{})
	require.NoError(t, err)
	logDB, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "logs.db")), &gorm.Config{})
	require.NoError(t, err)
	mainSQLDB, err := mainDB.DB()
	require.NoError(t, err)
	logSQLDB, err := logDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, mainSQLDB.Close())
		require.NoError(t, logSQLDB.Close())
	})
	require.NoError(t, mainDB.AutoMigrate(
		&model.Task{}, &model.User{}, &model.PerfMetric{},
		&model.ModelUsageEvent{}, &model.ModelUsageAttempt{},
	))
	require.NoError(t, logDB.AutoMigrate(&model.Log{}))
	model.DB = mainDB
	model.LOG_DB = logDB
	return mainDB, logDB
}

func seedBackfillTaskAndLogs(t *testing.T, mainDB *gorm.DB, logDB *gorm.DB) {
	t.Helper()
	require.NoError(t, mainDB.Create(&model.User{Id: 7, Username: "alice"}).Error)
	require.NoError(t, mainDB.Create(&model.Task{
		TaskID: "task-history-1", UserId: 7, Group: "vip", ChannelId: 23,
		Quota: 900, Status: model.TaskStatusSuccess,
		SubmitTime: 1_700_000_010, FinishTime: 1_700_000_050,
		Properties: model.Properties{OriginModelName: "video-model"},
		PrivateData: model.TaskPrivateData{
			TokenId:   11,
			Execution: &model.TaskExecutionSnapshot{RequestID: "task-request-1"},
		},
		Data: json.RawMessage(`{"usage":{"total_tokens":321}}`),
	}).Error)
	require.NoError(t, logDB.Create(&model.Log{
		UserId: 7, Username: "alice", CreatedAt: 1_700_000_100,
		Type: model.LogTypeConsume, ModelName: "image-model", Quota: 120,
		PromptTokens: 4, CompletionTokens: 3, UseTime: 2,
		ChannelId: 24, TokenId: 12, Group: "vip", RequestId: "request-history-1",
		Other: `{"generated_images":2}`,
	}).Error)
}

func countUsageEvents(t *testing.T, db *gorm.DB) int {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&model.ModelUsageEvent{}).Count(&count).Error)
	return int(count)
}

func TestBackfillModelUsageEventsDryRunAndApplyAreIdempotent(t *testing.T) {
	mainDB, logDB := setupBackfillDatabases(t)
	seedBackfillTaskAndLogs(t, mainDB, logDB)
	opts := BackfillOptions{DryRun: true, Cutoff: 1_800_000_000, Batch: "baseline"}

	report, err := BackfillModelUsageEvents(t.Context(), opts)
	require.NoError(t, err)
	assert.Equal(t, 2, report.WouldInsert)
	assert.Zero(t, report.Inserted)
	assert.Equal(t, int64(2), report.TotalCalls)
	assert.Equal(t, int64(1_020), report.TotalQuota)
	assert.Equal(t, int64(328), report.TotalTokens)
	assert.Zero(t, countUsageEvents(t, mainDB))

	opts.DryRun = false
	first, err := BackfillModelUsageEvents(t.Context(), opts)
	require.NoError(t, err)
	second, err := BackfillModelUsageEvents(t.Context(), opts)
	require.NoError(t, err)
	assert.Equal(t, 2, first.Inserted)
	assert.Zero(t, second.Inserted)
	assert.Equal(t, 2, second.Skipped)
	assert.Equal(t, 2, countUsageEvents(t, mainDB))

	var attempts []model.ModelUsageAttempt
	require.NoError(t, mainDB.Find(&attempts).Error)
	require.Len(t, attempts, 1)
	assert.Equal(t, "request:request-history-1", attempts[0].EventKey)
	assert.Equal(t, model.ModelUsageStatusSuccess, attempts[0].Status)
}

func TestRollbackModelUsageBatchDeletesOnlyExactBatchAndItsAttempts(t *testing.T) {
	mainDB, logDB := setupBackfillDatabases(t)
	seedBackfillTaskAndLogs(t, mainDB, logDB)
	require.NoError(t, mainDB.Create(&model.ModelUsageEvent{
		EventKey: "request:live", Kind: model.ModelUsageKindSync, ModelName: "live-model",
		Status: model.ModelUsageStatusSuccess, Source: "live", SubmittedAt: 1_700_000_000,
	}).Error)

	_, err := BackfillModelUsageEvents(t.Context(), BackfillOptions{
		DryRun: false, Cutoff: 1_800_000_000, Batch: "baseline",
	})
	require.NoError(t, err)
	deleted, err := RollbackModelUsageBatch(t.Context(), "baseline")
	require.NoError(t, err)
	assert.Equal(t, int64(2), deleted)

	var events []model.ModelUsageEvent
	require.NoError(t, mainDB.Find(&events).Error)
	require.Len(t, events, 1)
	assert.Equal(t, "request:live", events[0].EventKey)
	var attemptCount int64
	require.NoError(t, mainDB.Model(&model.ModelUsageAttempt{}).Count(&attemptCount).Error)
	assert.Zero(t, attemptCount)
}

func TestBackfillAndRollbackNeverModifyLogDatabase(t *testing.T) {
	mainDB, logDB := setupBackfillDatabases(t)
	seedBackfillTaskAndLogs(t, mainDB, logDB)
	var before []model.Log
	require.NoError(t, logDB.Order("id ASC").Find(&before).Error)

	_, err := BackfillModelUsageEvents(t.Context(), BackfillOptions{
		DryRun: true, Cutoff: 1_800_000_000, Batch: "baseline",
	})
	require.NoError(t, err)
	_, err = BackfillModelUsageEvents(t.Context(), BackfillOptions{
		DryRun: false, Cutoff: 1_800_000_000, Batch: "baseline",
	})
	require.NoError(t, err)
	_, err = RollbackModelUsageBatch(t.Context(), "baseline")
	require.NoError(t, err)

	var after []model.Log
	require.NoError(t, logDB.Order("id ASC").Find(&after).Error)
	assert.Equal(t, before, after)
}

func TestBackfillReconstructsRefundsFailuresAndLegacyPerfGaps(t *testing.T) {
	mainDB, logDB := setupBackfillDatabases(t)
	require.NoError(t, logDB.Create(&[]model.Log{
		{CreatedAt: 7_205, Type: model.LogTypeError, ModelName: "chat-model", ChannelId: 31, UseTime: 3, RequestId: "retry-request", Content: "first channel failed"},
		{CreatedAt: 7_210, Type: model.LogTypeConsume, ModelName: "chat-model", ChannelId: 32, Quota: 100, PromptTokens: 10, CompletionTokens: 5, UseTime: 4, RequestId: "retry-request"},
		{CreatedAt: 7_211, Type: model.LogTypeRefund, ModelName: "chat-model", ChannelId: 32, Quota: 30, RequestId: "retry-request"},
		{CreatedAt: 7_220, Type: model.LogTypeError, ModelName: "chat-model", ChannelId: 33, UseTime: 2, RequestId: "failed-request", Content: "all channels failed"},
	}).Error)
	require.NoError(t, mainDB.Create(&model.PerfMetric{
		ModelName: "chat-model", Group: "default", BucketTs: 7_200,
		RequestCount: 3, SuccessCount: 2,
	}).Error)

	report, err := BackfillModelUsageEvents(t.Context(), BackfillOptions{
		DryRun: false, Cutoff: 8_000, Batch: "legacy",
	})
	require.NoError(t, err)
	assert.Equal(t, 3, report.Inserted)
	assert.Equal(t, int64(3), report.TotalCalls)
	assert.Equal(t, int64(70), report.TotalQuota)
	assert.Equal(t, int64(15), report.TotalTokens)

	var events []model.ModelUsageEvent
	require.NoError(t, mainDB.Order("event_key ASC").Find(&events).Error)
	require.Len(t, events, 3)
	assert.Equal(t, "legacy-perf:chat-model:7200:success:0", events[0].EventKey)
	assert.Zero(t, events[0].DurationMs)
	assert.Equal(t, "backfill_perf", events[0].Source)

	var success model.ModelUsageEvent
	require.NoError(t, mainDB.Where("event_key = ?", "request:retry-request").First(&success).Error)
	assert.Equal(t, int64(7_202), success.SubmittedAt)
	assert.Equal(t, int64(4_000), success.DurationMs)
	assert.Equal(t, 70, success.FinalQuota)
	assert.Equal(t, int64(15), success.TotalTokens)

	var attempts []model.ModelUsageAttempt
	require.NoError(t, mainDB.Order("event_key ASC, channel_id ASC").Find(&attempts).Error)
	require.Len(t, attempts, 3)
	assert.Equal(t, model.ModelUsageStatusFailure, attempts[1].Status)
	assert.Equal(t, model.ModelUsageStatusSuccess, attempts[2].Status)
}

func TestBackfillRejectsNegativeLegacyPerformanceGapBeforeApply(t *testing.T) {
	mainDB, logDB := setupBackfillDatabases(t)
	require.NoError(t, logDB.Create(&[]model.Log{
		{CreatedAt: 7_210, Type: model.LogTypeConsume, ModelName: "chat-model", RequestId: "one"},
		{CreatedAt: 7_220, Type: model.LogTypeConsume, ModelName: "chat-model", RequestId: "two"},
	}).Error)
	require.NoError(t, mainDB.Create(&model.PerfMetric{
		ModelName: "chat-model", Group: "default", BucketTs: 7_200,
		RequestCount: 1, SuccessCount: 1,
	}).Error)

	_, err := BackfillModelUsageEvents(t.Context(), BackfillOptions{
		DryRun: false, Cutoff: 8_000, Batch: "invalid",
	})
	require.ErrorContains(t, err, "negative legacy performance gap")
	assert.Zero(t, countUsageEvents(t, mainDB))
}
