package service

import (
	"fmt"
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
