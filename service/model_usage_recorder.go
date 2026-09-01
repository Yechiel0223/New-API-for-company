package service

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

type SyncModelUsageResult struct {
	Success       bool
	TotalTokens   int64
	OutputCount   int
	OutputUnit    string
	FinalQuota    int
	FailureReason string
}

func RecordTaskUsageSubmitted(c *gin.Context, info *relaycommon.RelayInfo, task *model.Task) {
	if c == nil || info == nil || task == nil || task.TaskID == "" {
		return
	}
	requestID := info.RequestId
	if requestID == "" {
		requestID = c.GetString(common.RequestIdKey)
	}
	modelName := taskModelName(task)
	if modelName == "" {
		modelName = info.OriginModelName
	}
	event := &model.ModelUsageEvent{
		EventKey:       "task:" + task.TaskID,
		Kind:           model.ModelUsageKindTask,
		RequestID:      requestID,
		TaskID:         task.TaskID,
		UserID:         task.UserId,
		Username:       common.GetContextKeyString(c, constant.ContextKeyUserName),
		TokenID:        task.PrivateData.TokenId,
		ChannelID:      task.ChannelId,
		ModelName:      modelName,
		UseGroup:       task.Group,
		Status:         model.ModelUsageStatusRunning,
		SubmittedAt:    task.SubmitTime,
		LastProgressAt: task.SubmitTime,
		Source:         "live",
	}
	if err := model.CreateModelUsageEvent(event); err != nil {
		logger.LogError(c, fmt.Sprintf("failed to record task usage submission, task_id=%s: %s", task.TaskID, err.Error()))
	}
}

func TouchTaskUsageProgress(task *model.Task, status model.TaskStatus, progressAt int64) {
	if task == nil || task.TaskID == "" || progressAt <= 0 {
		return
	}
	if status == model.TaskStatusSuccess || status == model.TaskStatusFailure {
		return
	}
	if err := model.TouchModelUsageEvent("task:"+task.TaskID, model.ModelUsageStatusRunning, progressAt); err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("failed to touch task usage progress, task_id=%s: %s", task.TaskID, err.Error()))
	}
}

func FinalizeTaskUsage(ctx context.Context, task *model.Task, result *relaycommon.TaskInfo) {
	if task == nil || task.TaskID == "" {
		return
	}

	completedAt := task.FinishTime
	if completedAt == 0 {
		completedAt = common.GetTimestamp()
	}
	durationMs := int64(0)
	if task.FinishTime > task.SubmitTime {
		durationMs = (task.FinishTime - task.SubmitTime) * 1_000
	}

	final := model.ModelUsageFinal{
		Status:         model.ModelUsageStatusFailure,
		CompletedAt:    completedAt,
		LastProgressAt: completedAt,
		FinalQuota:     task.Quota,
		DurationMs:     durationMs,
	}
	if task.Status == model.TaskStatusSuccess {
		final.Status = model.ModelUsageStatusSuccess
		final.TotalTokens = taskUsageTotalTokens(result)
		final.OutputCount = 1
		final.OutputUnit = "video"
	} else {
		failureReason := task.FailReason
		if failureReason == "" && result != nil {
			failureReason = result.Reason
		}
		runes := []rune(failureReason)
		if len(runes) > 512 {
			runes = runes[:512]
		}
		final.FailureReason = string(runes)
	}

	if err := model.FinalizeModelUsageEvent("task:"+task.TaskID, final); err != nil {
		logger.LogError(ctx, fmt.Sprintf("failed to finalize task usage, task_id=%s: %s", task.TaskID, err.Error()))
	}
}

func taskUsageTotalTokens(result *relaycommon.TaskInfo) int64 {
	if result == nil {
		return 0
	}
	if result.TotalTokens > 0 {
		return int64(result.TotalTokens)
	}
	if result.CompletionTokens > 0 {
		return int64(result.CompletionTokens)
	}
	value, ok := result.UsageFacts["tokens"]
	if !ok {
		return 0
	}
	numeric := reflect.ValueOf(value)
	switch numeric.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		tokens := numeric.Int()
		if tokens > 0 {
			return tokens
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		tokens := numeric.Uint()
		if tokens <= math.MaxInt64 && tokens > 0 {
			return int64(tokens)
		}
	case reflect.Float32, reflect.Float64:
		tokens := numeric.Float()
		if tokens > 0 && tokens < float64(math.MaxInt64) {
			return int64(tokens)
		}
	}
	return 0
}

func RecordSyncModelUsageStarted(c *gin.Context, info *relaycommon.RelayInfo) {
	if c == nil || info == nil {
		return
	}
	event := syncModelUsageEvent(c, info)
	if err := model.CreateModelUsageEvent(event); err != nil {
		logger.LogError(c, fmt.Sprintf("failed to record model usage start, request_id=%s: %s", event.RequestID, err.Error()))
	}
}

func RecordSyncModelUsage(c *gin.Context, info *relaycommon.RelayInfo, result SyncModelUsageResult) {
	if c == nil || info == nil {
		return
	}

	event := syncModelUsageEvent(c, info)
	if err := model.CreateModelUsageEvent(event); err != nil {
		logger.LogError(c, fmt.Sprintf("failed to ensure model usage event, request_id=%s: %s", event.RequestID, err.Error()))
	}

	completedAt := common.GetTimestamp()
	durationMs := time.Since(info.StartTime).Milliseconds()
	if durationMs < 0 {
		durationMs = 0
	}
	status := model.ModelUsageStatusFailure
	if result.Success {
		status = model.ModelUsageStatusSuccess
	}
	failureReason := []rune(result.FailureReason)
	if len(failureReason) > 512 {
		failureReason = failureReason[:512]
	}
	if err := model.FinalizeModelUsageEvent(event.EventKey, model.ModelUsageFinal{
		Status:         status,
		CompletedAt:    completedAt,
		LastProgressAt: completedAt,
		TotalTokens:    result.TotalTokens,
		OutputCount:    result.OutputCount,
		OutputUnit:     result.OutputUnit,
		FinalQuota:     result.FinalQuota,
		DurationMs:     durationMs,
		FailureReason:  string(failureReason),
	}); err != nil {
		logger.LogError(c, fmt.Sprintf("failed to finalize model usage, request_id=%s: %s", event.RequestID, err.Error()))
	}
}

func syncModelUsageEvent(c *gin.Context, info *relaycommon.RelayInfo) *model.ModelUsageEvent {
	requestID := info.RequestId
	if requestID == "" {
		requestID = c.GetString(common.RequestIdKey)
	}
	submittedAt := info.StartTime.Unix()
	return &model.ModelUsageEvent{
		EventKey:       "request:" + requestID,
		Kind:           model.ModelUsageKindSync,
		RequestID:      requestID,
		UserID:         info.UserId,
		Username:       common.GetContextKeyString(c, constant.ContextKeyUserName),
		TokenID:        info.TokenId,
		ChannelID:      common.GetContextKeyInt(c, constant.ContextKeyChannelId),
		ModelName:      info.OriginModelName,
		UseGroup:       info.UsingGroup,
		Status:         model.ModelUsageStatusRunning,
		SubmittedAt:    submittedAt,
		LastProgressAt: submittedAt,
		Source:         "live",
	}
}
