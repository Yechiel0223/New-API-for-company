package model

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestModelUsageEventMigrationSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	testModelUsageEventMigration(t, db)
}

func TestModelUsageEventMigrationConfiguredDatabase(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("MODEL_USAGE_MIGRATION_DSN"))
	dialect := strings.TrimSpace(os.Getenv("MODEL_USAGE_MIGRATION_DIALECT"))
	if dsn == "" || dialect == "" {
		t.Skip("MODEL_USAGE_MIGRATION_DSN and MODEL_USAGE_MIGRATION_DIALECT are not configured")
	}

	var dialector gorm.Dialector
	switch dialect {
	case "mysql":
		dialector = mysql.Open(dsn)
	case "postgres":
		dialector = postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true})
	default:
		t.Fatalf("unsupported MODEL_USAGE_MIGRATION_DIALECT %q", dialect)
	}
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	testModelUsageEventMigration(t, db)
}

func testModelUsageEventMigration(t *testing.T, db *gorm.DB) {
	t.Helper()
	previousDB := DB
	DB = db
	t.Cleanup(func() { DB = previousDB })

	require.NoError(t, db.AutoMigrate(&Task{}, &QuotaData{}))
	require.NoError(t, db.Create(&Task{TaskID: "preserved-task", Status: TaskStatusSuccess, SubmitTime: 1}).Error)
	require.NoError(t, db.Create(&QuotaData{Username: "alice", ModelName: "seedance", Count: 1, Quota: 10, CreatedAt: 1}).Error)

	require.NoError(t, db.AutoMigrate(&ModelUsageEvent{}, &ModelUsageAttempt{}))
	require.NoError(t, db.AutoMigrate(&ModelUsageEvent{}, &ModelUsageAttempt{}))

	event := &ModelUsageEvent{
		EventKey: "migration:event:1", Kind: ModelUsageKindSync, ModelName: "seedance",
		Status: ModelUsageStatusRunning, SubmittedAt: 1, Source: "live",
	}
	require.NoError(t, CreateModelUsageEvent(event))
	require.NoError(t, CreateModelUsageEvent(event))
	require.NoError(t, FinalizeModelUsageEvent("migration:event:1", ModelUsageFinal{
		Status: modelUsageStatusSuccessForMigration(), CompletedAt: 2, LastProgressAt: 2,
		TotalTokens: 123, OutputCount: 1, OutputUnit: "video", FinalQuota: 456, DurationMs: 1000,
	}))

	attempt := &ModelUsageAttempt{EventKey: "migration:event:1", ChannelID: 1, ModelName: "seedance", Status: ModelUsageStatusSuccess, CompletedAt: 2}
	require.NoError(t, UpsertModelUsageAttempt(attempt))
	require.NoError(t, UpsertModelUsageAttempt(attempt))

	var eventCount int64
	require.NoError(t, db.Model(&ModelUsageEvent{}).Where("event_key = ?", "migration:event:1").Count(&eventCount).Error)
	assert.Equal(t, int64(1), eventCount)
	var attemptCount int64
	require.NoError(t, db.Model(&ModelUsageAttempt{}).Where("event_key = ?", "migration:event:1").Count(&attemptCount).Error)
	assert.Equal(t, int64(1), attemptCount)

	var taskCount int64
	require.NoError(t, db.WithContext(context.Background()).Model(&Task{}).Where("task_id = ?", "preserved-task").Count(&taskCount).Error)
	assert.Equal(t, int64(1), taskCount)
	var quotaCount int64
	require.NoError(t, db.Model(&QuotaData{}).Where("username = ?", "alice").Count(&quotaCount).Error)
	assert.Equal(t, int64(1), quotaCount)
}

func modelUsageStatusSuccessForMigration() ModelUsageStatus {
	return ModelUsageStatusSuccess
}
