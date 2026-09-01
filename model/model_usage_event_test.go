package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModelUsageEventTestDB(t *testing.T) {
	t.Helper()
	originalDB := DB
	t.Cleanup(func() { DB = originalDB })
	var err error
	DB, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, DB.AutoMigrate(&ModelUsageEvent{}, &ModelUsageAttempt{}))
}

func TestModelUsageEventCreateAndFinalizeIsIdempotent(t *testing.T) {
	setupModelUsageEventTestDB(t)
	event := &ModelUsageEvent{
		EventKey: "request:req-1", Kind: ModelUsageKindSync,
		RequestID: "req-1", ModelName: "doubao-seedream-5-0-pro-260628",
		Status: ModelUsageStatusRunning, SubmittedAt: 1_700_000_000,
		LastProgressAt: 1_700_000_000,
	}
	require.NoError(t, CreateModelUsageEvent(event))
	require.NoError(t, CreateModelUsageEvent(event))
	require.NoError(t, FinalizeModelUsageEvent("request:req-1", ModelUsageFinal{
		Status: ModelUsageStatusSuccess, CompletedAt: 1_700_000_050,
		LastProgressAt: 1_700_000_050, TotalTokens: 99_712,
		OutputCount: 1, OutputUnit: "image",
		FinalQuota: 975_000, DurationMs: 50_000,
	}))

	var rows []ModelUsageEvent
	require.NoError(t, DB.Find(&rows).Error)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(99_712), rows[0].TotalTokens)
	assert.Equal(t, 1, rows[0].OutputCount)
	assert.Equal(t, "image", rows[0].OutputUnit)
	assert.Equal(t, 975_000, rows[0].FinalQuota)
	assert.Equal(t, ModelUsageStatusSuccess, rows[0].Status)
}

func TestModelUsageEventRejectsInvalidUsageValues(t *testing.T) {
	setupModelUsageEventTestDB(t)

	err := CreateModelUsageEvent(&ModelUsageEvent{
		EventKey: "request:invalid", ModelName: "model", TotalTokens: -1,
	})
	require.Error(t, err)

	err = FinalizeModelUsageEvent("request:invalid", ModelUsageFinal{
		Status: ModelUsageStatusSuccess, OutputCount: 1,
	})
	require.Error(t, err)
}

func TestModelUsageEventTouchListAndDeleteBackfillBatch(t *testing.T) {
	setupModelUsageEventTestDB(t)
	events := []*ModelUsageEvent{
		{EventKey: "request:running", Kind: ModelUsageKindSync, RequestID: "running", ModelName: "model-a", Username: "alice", Status: ModelUsageStatusRunning, SubmittedAt: 100, BackfillBatch: "batch-1"},
		{EventKey: "request:complete", Kind: ModelUsageKindSync, RequestID: "complete", ModelName: "model-b", Username: "alice", Status: ModelUsageStatusSuccess, SubmittedAt: 200, BackfillBatch: "batch-2"},
	}
	for _, event := range events {
		require.NoError(t, CreateModelUsageEvent(event))
	}

	require.NoError(t, TouchModelUsageEvent("request:running", ModelUsageStatusRunning, 150))
	require.NoError(t, TouchModelUsageEvent("request:complete", ModelUsageStatusRunning, 250))

	var completed ModelUsageEvent
	require.NoError(t, DB.Where("event_key = ?", "request:complete").First(&completed).Error)
	assert.Equal(t, ModelUsageStatusSuccess, completed.Status)
	assert.Zero(t, completed.LastProgressAt)

	rows, err := ListModelUsageEvents(ModelUsageQuery{StartTimestamp: 100, EndTimestamp: 150, Username: "alice", Models: []string{"model-a"}})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "request:running", rows[0].EventKey)

	deleted, err := DeleteModelUsageBackfillBatch("batch-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)
}

func TestDeleteModelUsageBackfillBatchRejectsEmptyBatch(t *testing.T) {
	setupModelUsageEventTestDB(t)
	require.NoError(t, CreateModelUsageEvent(&ModelUsageEvent{
		EventKey: "request:native", Kind: ModelUsageKindSync, ModelName: "model",
		Status: ModelUsageStatusSuccess,
	}))

	_, err := DeleteModelUsageBackfillBatch("")
	require.Error(t, err)

	var count int64
	require.NoError(t, DB.Model(&ModelUsageEvent{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestListModelUsageEventsForHealthIncludesRecentActivityAndRunningCalls(t *testing.T) {
	setupModelUsageEventTestDB(t)
	events := []*ModelUsageEvent{
		{EventKey: "request:submitted", Kind: ModelUsageKindSync, ModelName: "model-a", Status: ModelUsageStatusSuccess, SubmittedAt: 950, CompletedAt: 960},
		{EventKey: "task:completed", Kind: ModelUsageKindTask, ModelName: "model-b", Status: ModelUsageStatusSuccess, SubmittedAt: 100, CompletedAt: 975},
		{EventKey: "task:running", Kind: ModelUsageKindTask, ModelName: "model-c", Status: ModelUsageStatusRunning, SubmittedAt: 100, LastProgressAt: 200},
		{EventKey: "request:old", Kind: ModelUsageKindSync, ModelName: "model-d", Status: ModelUsageStatusFailure, SubmittedAt: 100, CompletedAt: 200},
		{EventKey: "request:future", Kind: ModelUsageKindSync, ModelName: "model-e", Status: ModelUsageStatusRunning, SubmittedAt: 1_001},
	}
	for _, event := range events {
		require.NoError(t, CreateModelUsageEvent(event))
	}

	rows, err := ListModelUsageEventsForHealth(context.Background(), 900, 1_000)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.ElementsMatch(t, []string{"task:running", "request:submitted", "task:completed"}, []string{
		rows[0].EventKey,
		rows[1].EventKey,
		rows[2].EventKey,
	})
}

func TestListModelUsageEventsForAnalyticsIncludesSubmittedOrCompletedRange(t *testing.T) {
	setupModelUsageEventTestDB(t)
	events := []*ModelUsageEvent{
		{EventKey: "request:submitted", Kind: ModelUsageKindSync, ModelName: "model-a", Username: "alice", Status: ModelUsageStatusSuccess, SubmittedAt: 950, CompletedAt: 960},
		{EventKey: "task:completed", Kind: ModelUsageKindTask, ModelName: "model-b", Username: "alice", Status: ModelUsageStatusSuccess, SubmittedAt: 100, CompletedAt: 975},
		{EventKey: "request:old", Kind: ModelUsageKindSync, ModelName: "model-c", Username: "alice", Status: ModelUsageStatusFailure, SubmittedAt: 100, CompletedAt: 200},
		{EventKey: "request:bob", Kind: ModelUsageKindSync, ModelName: "model-a", Username: "bob", Status: ModelUsageStatusSuccess, SubmittedAt: 960, CompletedAt: 970},
	}
	for _, event := range events {
		require.NoError(t, CreateModelUsageEvent(event))
	}

	rows, err := ListModelUsageEventsForAnalytics(context.Background(), ModelUsageQuery{
		StartTimestamp: 900, EndTimestamp: 1_000, Username: "alice",
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.ElementsMatch(t, []string{"request:submitted", "task:completed"}, []string{rows[0].EventKey, rows[1].EventKey})
}

func TestModelUsageAttemptUpsertKeepsOneResultPerEventAndChannel(t *testing.T) {
	setupModelUsageEventTestDB(t)
	require.NoError(t, UpsertModelUsageAttempt(&ModelUsageAttempt{
		EventKey: "request:a", ModelName: "model-a", ChannelID: 1,
		Status: ModelUsageStatusFailure, CompletedAt: 950,
	}))
	require.NoError(t, UpsertModelUsageAttempt(&ModelUsageAttempt{
		EventKey: "request:a", ModelName: "model-a", ChannelID: 1,
		Status: ModelUsageStatusSuccess, CompletedAt: 975,
	}))
	require.NoError(t, UpsertModelUsageAttempt(&ModelUsageAttempt{
		EventKey: "request:a", ModelName: "model-a", ChannelID: 2,
		Status: ModelUsageStatusFailure, CompletedAt: 980,
	}))

	rows, err := ListModelUsageAttemptsForHealth(context.Background(), 900, 1_000)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	byChannel := map[int]ModelUsageAttempt{rows[0].ChannelID: rows[0], rows[1].ChannelID: rows[1]}
	assert.Equal(t, ModelUsageStatusSuccess, byChannel[1].Status)
	assert.Equal(t, int64(975), byChannel[1].CompletedAt)
	assert.Equal(t, ModelUsageStatusFailure, byChannel[2].Status)
	require.NoError(t, DB.AutoMigrate(&ModelUsageAttempt{}))
	var count int64
	require.NoError(t, DB.Model(&ModelUsageAttempt{}).Count(&count).Error)
	assert.Equal(t, int64(2), count)
}
