package model

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm/clause"
)

type ModelUsageStatus string
type ModelUsageKind string

const (
	ModelUsageKindSync ModelUsageKind = "sync"
	ModelUsageKindTask ModelUsageKind = "task"

	ModelUsageStatusRunning ModelUsageStatus = "running"
	ModelUsageStatusSuccess ModelUsageStatus = "success"
	ModelUsageStatusFailure ModelUsageStatus = "failure"
)

type ModelUsageEvent struct {
	ID             int64            `json:"id" gorm:"primaryKey"`
	EventKey       string           `json:"event_key" gorm:"size:191;not null;uniqueIndex:uk_model_usage_event_key"`
	Kind           ModelUsageKind   `json:"kind" gorm:"size:16;not null;index"`
	RequestID      string           `json:"request_id" gorm:"size:64;index"`
	TaskID         string           `json:"task_id" gorm:"size:191;index"`
	UserID         int              `json:"user_id" gorm:"index"`
	Username       string           `json:"username" gorm:"size:64;index"`
	TokenID        int              `json:"token_id" gorm:"index"`
	ChannelID      int              `json:"channel_id" gorm:"index"`
	ModelName      string           `json:"model_name" gorm:"size:128;not null;index:idx_model_usage_model_submitted,priority:1"`
	UseGroup       string           `json:"use_group" gorm:"column:use_group;size:64;index"`
	Status         ModelUsageStatus `json:"status" gorm:"size:16;not null;index"`
	SubmittedAt    int64            `json:"submitted_at" gorm:"index:idx_model_usage_model_submitted,priority:2;index"`
	CompletedAt    int64            `json:"completed_at" gorm:"index"`
	LastProgressAt int64            `json:"last_progress_at" gorm:"index"`
	TotalTokens    int64            `json:"total_tokens"`
	OutputCount    int              `json:"output_count"`
	OutputUnit     string           `json:"output_unit" gorm:"size:16"`
	FinalQuota     int              `json:"final_quota"`
	DurationMs     int64            `json:"duration_ms"`
	FailureReason  string           `json:"failure_reason" gorm:"type:text"`
	Source         string           `json:"source" gorm:"size:24;not null;index"`
	BackfillBatch  string           `json:"backfill_batch" gorm:"size:64;index"`
	RecordedAt     int64            `json:"recorded_at"`
	UpdatedAt      int64            `json:"updated_at"`
}

type ModelUsageFinal struct {
	Status         ModelUsageStatus
	CompletedAt    int64
	LastProgressAt int64
	TotalTokens    int64
	OutputCount    int
	OutputUnit     string
	FinalQuota     int
	DurationMs     int64
	FailureReason  string
}

type ModelUsageQuery struct {
	StartTimestamp int64
	EndTimestamp   int64
	Username       string
	Models         []string
}

func validateModelUsageValues(totalTokens int64, outputCount int, outputUnit string, finalQuota int, durationMs int64) error {
	if totalTokens < 0 || outputCount < 0 || finalQuota < 0 || durationMs < 0 {
		return fmt.Errorf("model usage values cannot be negative")
	}
	if outputUnit != "" && outputUnit != "image" && outputUnit != "video" {
		return fmt.Errorf("invalid model usage output unit")
	}
	if outputCount == 0 && outputUnit != "" {
		return fmt.Errorf("model usage output unit requires output count")
	}
	if outputCount > 0 && outputUnit == "" {
		return fmt.Errorf("model usage output count requires output unit")
	}
	return nil
}

func CreateModelUsageEvent(event *ModelUsageEvent) error {
	if event == nil {
		return fmt.Errorf("model usage event is required")
	}
	if err := validateModelUsageValues(event.TotalTokens, event.OutputCount, event.OutputUnit, event.FinalQuota, event.DurationMs); err != nil {
		return err
	}
	now := common.GetTimestamp()
	if event.RecordedAt == 0 {
		event.RecordedAt = now
	}
	if event.UpdatedAt == 0 {
		event.UpdatedAt = now
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).Create(event).Error
}

func FinalizeModelUsageEvent(eventKey string, final ModelUsageFinal) error {
	if err := validateModelUsageValues(final.TotalTokens, final.OutputCount, final.OutputUnit, final.FinalQuota, final.DurationMs); err != nil {
		return err
	}
	return DB.Model(&ModelUsageEvent{}).Where("event_key = ?", eventKey).Updates(map[string]any{
		"status":           final.Status,
		"completed_at":     final.CompletedAt,
		"last_progress_at": final.LastProgressAt,
		"total_tokens":     final.TotalTokens,
		"output_count":     final.OutputCount,
		"output_unit":      final.OutputUnit,
		"final_quota":      final.FinalQuota,
		"duration_ms":      final.DurationMs,
		"failure_reason":   final.FailureReason,
		"updated_at":       common.GetTimestamp(),
	}).Error
}

func TouchModelUsageEvent(eventKey string, status ModelUsageStatus, progressAt int64) error {
	return DB.Model(&ModelUsageEvent{}).
		Where("event_key = ? AND status = ?", eventKey, ModelUsageStatusRunning).
		Updates(map[string]any{
			"status":           status,
			"last_progress_at": progressAt,
			"updated_at":       common.GetTimestamp(),
		}).Error
}

func ListModelUsageEvents(query ModelUsageQuery) ([]ModelUsageEvent, error) {
	events := make([]ModelUsageEvent, 0)
	dbQuery := DB.Model(&ModelUsageEvent{})
	if query.StartTimestamp > 0 {
		dbQuery = dbQuery.Where("submitted_at >= ?", query.StartTimestamp)
	}
	if query.EndTimestamp > 0 {
		dbQuery = dbQuery.Where("submitted_at <= ?", query.EndTimestamp)
	}
	if query.Username != "" {
		dbQuery = dbQuery.Where("username = ?", query.Username)
	}
	if query.Models != nil {
		if len(query.Models) == 0 {
			return events, nil
		}
		dbQuery = dbQuery.Where("model_name IN ?", query.Models)
	}
	err := dbQuery.Order("submitted_at ASC").Find(&events).Error
	return events, err
}

func DeleteModelUsageBackfillBatch(batch string) (int64, error) {
	result := DB.Where("backfill_batch = ?", batch).Delete(&ModelUsageEvent{})
	return result.RowsAffected, result.Error
}
