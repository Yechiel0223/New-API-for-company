package model

import (
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
	require.NoError(t, DB.AutoMigrate(&ModelUsageEvent{}))
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
