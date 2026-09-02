package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

type BackfillOptions struct {
	DryRun bool
	Cutoff int64
	Batch  string
}

type BackfillReport struct {
	Batch       string `json:"batch"`
	DryRun      bool   `json:"dry_run"`
	WouldInsert int    `json:"would_insert"`
	Inserted    int    `json:"inserted"`
	Skipped     int    `json:"skipped"`
	Rejected    int    `json:"rejected"`
	TotalCalls  int64  `json:"total_calls"`
	TotalQuota  int64  `json:"total_quota"`
	TotalTokens int64  `json:"total_tokens"`
}

func BackfillModelUsageEvents(ctx context.Context, options BackfillOptions) (BackfillReport, error) {
	report := BackfillReport{Batch: options.Batch, DryRun: options.DryRun}
	if options.Cutoff <= 0 {
		return report, fmt.Errorf("cutoff is required")
	}
	if strings.TrimSpace(options.Batch) == "" {
		return report, fmt.Errorf("batch is required")
	}

	candidates, attempts, err := buildBackfillCandidates(ctx, options)
	if err != nil {
		return report, err
	}
	report.TotalCalls = int64(len(candidates))
	for _, candidate := range candidates {
		report.TotalQuota += int64(candidate.FinalQuota)
		report.TotalTokens += candidate.TotalTokens
	}

	existing, err := existingBackfillEventKeys(ctx, candidates)
	if err != nil {
		return report, err
	}
	for _, candidate := range candidates {
		if existing[candidate.EventKey] {
			report.Skipped++
			continue
		}
		if options.DryRun {
			report.WouldInsert++
			continue
		}
		event := candidate
		if err := model.CreateModelUsageEvent(&event); err != nil {
			report.Rejected++
			return report, err
		}
		report.Inserted++
	}

	if !options.DryRun {
		for _, attempt := range attempts {
			if existing[attempt.EventKey] {
				continue
			}
			if err := model.UpsertModelUsageAttempt(&attempt); err != nil {
				report.Rejected++
				return report, err
			}
		}
	}

	return report, nil
}

func RollbackModelUsageBatch(ctx context.Context, batch string) (int64, error) {
	if strings.TrimSpace(batch) == "" {
		return 0, fmt.Errorf("model usage backfill batch is required")
	}
	var eventKeys []string
	if err := model.DB.WithContext(ctx).Model(&model.ModelUsageEvent{}).
		Where("backfill_batch = ?", batch).
		Pluck("event_key", &eventKeys).Error; err != nil {
		return 0, err
	}
	if len(eventKeys) > 0 {
		if err := model.DB.WithContext(ctx).Where("event_key IN ?", eventKeys).Delete(&model.ModelUsageAttempt{}).Error; err != nil {
			return 0, err
		}
	}
	return model.DeleteModelUsageBackfillBatch(batch)
}

func buildBackfillCandidates(ctx context.Context, options BackfillOptions) ([]model.ModelUsageEvent, []model.ModelUsageAttempt, error) {
	var candidates []model.ModelUsageEvent
	var attempts []model.ModelUsageAttempt

	taskEvents, err := buildTaskBackfillEvents(ctx, options)
	if err != nil {
		return nil, nil, err
	}
	candidates = append(candidates, taskEvents...)

	logEvents, logAttempts, err := buildLogBackfillEvents(ctx, options)
	if err != nil {
		return nil, nil, err
	}
	candidates = append(candidates, logEvents...)
	attempts = append(attempts, logAttempts...)

	perfEvents, err := buildLegacyPerfBackfillEvents(ctx, options, logEvents)
	if err != nil {
		return nil, nil, err
	}
	candidates = append(candidates, perfEvents...)

	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].EventKey < candidates[j].EventKey })
	sort.SliceStable(attempts, func(i, j int) bool {
		if attempts[i].EventKey == attempts[j].EventKey {
			return attempts[i].ChannelID < attempts[j].ChannelID
		}
		return attempts[i].EventKey < attempts[j].EventKey
	})
	return candidates, attempts, nil
}

func buildTaskBackfillEvents(ctx context.Context, options BackfillOptions) ([]model.ModelUsageEvent, error) {
	var tasks []model.Task
	if err := model.DB.WithContext(ctx).Where("submit_time > 0 AND submit_time <= ?", options.Cutoff).Find(&tasks).Error; err != nil {
		return nil, err
	}
	events := make([]model.ModelUsageEvent, 0, len(tasks))
	for _, task := range tasks {
		if task.TaskID == "" {
			continue
		}
		status, terminal := taskBackfillStatus(task.Status)
		if !terminal {
			continue
		}
		modelName := firstNonEmpty(task.Properties.OriginModelName, task.Properties.UpstreamModelName)
		if modelName == "" {
			modelName = string(task.Platform)
		}
		requestID := ""
		if task.PrivateData.Execution != nil {
			requestID = task.PrivateData.Execution.RequestID
		}
		duration := int64(0)
		if task.FinishTime > task.SubmitTime {
			duration = (task.FinishTime - task.SubmitTime) * 1000
		}
		outputCount, outputUnit := 0, ""
		if task.Status == model.TaskStatusSuccess {
			outputCount, outputUnit = 1, "video"
		}
		events = append(events, model.ModelUsageEvent{
			EventKey:      "task:" + task.TaskID,
			Kind:          model.ModelUsageKindTask,
			RequestID:     requestID,
			TaskID:        task.TaskID,
			UserID:        task.UserId,
			Username:      lookupBackfillUsername(ctx, task.UserId),
			TokenID:       task.PrivateData.TokenId,
			ChannelID:     task.ChannelId,
			ModelName:     modelName,
			UseGroup:      task.Group,
			Status:        status,
			SubmittedAt:   task.SubmitTime,
			CompletedAt:   task.FinishTime,
			TotalTokens:   extractPositiveJSONInt64(task.Data, "usage.total_tokens", "usage.completion_tokens", "data.usage.total_tokens", "response.usage.total_tokens"),
			OutputCount:   outputCount,
			OutputUnit:    outputUnit,
			FinalQuota:    task.Quota,
			DurationMs:    duration,
			FailureReason: task.FailReason,
			Source:        "backfill_task",
			BackfillBatch: options.Batch,
			RecordedAt:    common.GetTimestamp(),
			UpdatedAt:     common.GetTimestamp(),
		})
	}
	return events, nil
}

func buildLogBackfillEvents(ctx context.Context, options BackfillOptions) ([]model.ModelUsageEvent, []model.ModelUsageAttempt, error) {
	var logs []model.Log
	if err := model.LOG_DB.WithContext(ctx).
		Where("created_at <= ? AND request_id <> ''", options.Cutoff).
		Where("type IN ?", []int{model.LogTypeConsume, model.LogTypeError, model.LogTypeRefund}).
		Order("request_id ASC, created_at ASC, id ASC").
		Find(&logs).Error; err != nil {
		return nil, nil, err
	}
	grouped := map[string][]model.Log{}
	for _, log := range logs {
		if logLooksLikeTask(log) {
			continue
		}
		grouped[log.RequestId] = append(grouped[log.RequestId], log)
	}

	var events []model.ModelUsageEvent
	var attempts []model.ModelUsageAttempt
	for requestID, requestLogs := range grouped {
		event, eventAttempts, ok := buildLogRequestBackfillEvent(options, requestID, requestLogs)
		if !ok {
			continue
		}
		events = append(events, event)
		attempts = append(attempts, eventAttempts...)
	}
	return events, attempts, nil
}

func buildLogRequestBackfillEvent(options BackfillOptions, requestID string, logs []model.Log) (model.ModelUsageEvent, []model.ModelUsageAttempt, bool) {
	if len(logs) == 0 {
		return model.ModelUsageEvent{}, nil, false
	}
	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].CreatedAt == logs[j].CreatedAt {
			return logs[i].Id < logs[j].Id
		}
		return logs[i].CreatedAt < logs[j].CreatedAt
	})

	var finalConsume *model.Log
	var first model.Log
	var hasFirst bool
	quota := 0
	maxUseTime := 0
	submittedAt := int64(math.MaxInt64)
	attemptsByChannel := map[int]model.ModelUsageAttempt{}

	for i := range logs {
		log := logs[i]
		if !hasFirst {
			first, hasFirst = log, true
		}
		if log.UseTime > maxUseTime {
			maxUseTime = log.UseTime
		}
		candidateSubmitted := log.CreatedAt - int64(max(0, log.UseTime))
		if candidateSubmitted > log.CreatedAt {
			candidateSubmitted = log.CreatedAt
		}
		if candidateSubmitted < submittedAt {
			submittedAt = candidateSubmitted
		}
		switch log.Type {
		case model.LogTypeConsume:
			finalConsume = &logs[i]
			quota += log.Quota
			if log.ChannelId > 0 {
				attemptsByChannel[log.ChannelId] = model.ModelUsageAttempt{
					EventKey: requestEventKey(requestID), ChannelID: log.ChannelId, ModelName: log.ModelName,
					Status: model.ModelUsageStatusSuccess, CompletedAt: log.CreatedAt,
				}
			}
		case model.LogTypeRefund:
			quota -= log.Quota
		case model.LogTypeError:
			if log.ChannelId > 0 {
				attemptsByChannel[log.ChannelId] = model.ModelUsageAttempt{
					EventKey: requestEventKey(requestID), ChannelID: log.ChannelId, ModelName: log.ModelName,
					Status: model.ModelUsageStatusFailure, CompletedAt: log.CreatedAt,
				}
			}
		}
	}
	if quota < 0 {
		quota = 0
	}
	if submittedAt == math.MaxInt64 {
		submittedAt = first.CreatedAt
	}

	status := model.ModelUsageStatusFailure
	completedAt := logs[len(logs)-1].CreatedAt
	totalTokens := int64(0)
	outputCount, outputUnit := 0, ""
	modelName := first.ModelName
	userID, username, tokenID, channelID, useGroup := first.UserId, first.Username, first.TokenId, first.ChannelId, first.Group
	if finalConsume != nil {
		status = model.ModelUsageStatusSuccess
		completedAt = finalConsume.CreatedAt
		totalTokens = int64(finalConsume.PromptTokens + finalConsume.CompletionTokens)
		outputCount = parseGeneratedImages(finalConsume.Other)
		if outputCount > 0 {
			outputUnit = "image"
		}
		modelName = finalConsume.ModelName
		userID, username, tokenID, channelID, useGroup = finalConsume.UserId, finalConsume.Username, finalConsume.TokenId, finalConsume.ChannelId, finalConsume.Group
	}

	attempts := make([]model.ModelUsageAttempt, 0, len(attemptsByChannel))
	for _, attempt := range attemptsByChannel {
		if attempt.ModelName == "" {
			attempt.ModelName = modelName
		}
		attempts = append(attempts, attempt)
	}
	return model.ModelUsageEvent{
		EventKey:      requestEventKey(requestID),
		Kind:          model.ModelUsageKindSync,
		RequestID:     requestID,
		UserID:        userID,
		Username:      username,
		TokenID:       tokenID,
		ChannelID:     channelID,
		ModelName:     modelName,
		UseGroup:      useGroup,
		Status:        status,
		SubmittedAt:   submittedAt,
		CompletedAt:   completedAt,
		TotalTokens:   totalTokens,
		OutputCount:   outputCount,
		OutputUnit:    outputUnit,
		FinalQuota:    quota,
		DurationMs:    int64(maxUseTime) * 1000,
		Source:        "backfill_log",
		BackfillBatch: options.Batch,
		RecordedAt:    common.GetTimestamp(),
		UpdatedAt:     common.GetTimestamp(),
	}, attempts, true
}

func buildLegacyPerfBackfillEvents(ctx context.Context, options BackfillOptions, reconstructed []model.ModelUsageEvent) ([]model.ModelUsageEvent, error) {
	var metrics []model.PerfMetric
	if err := model.DB.WithContext(ctx).Where("bucket_ts <= ?", options.Cutoff).Find(&metrics).Error; err != nil {
		return nil, err
	}
	type counts struct{ requests, successes int64 }
	reconstructedCounts := map[string]counts{}
	for _, event := range reconstructed {
		if event.Kind != model.ModelUsageKindSync {
			continue
		}
		key := legacyPerfKey(event.ModelName, hourBucket(event.SubmittedAt))
		count := reconstructedCounts[key]
		count.requests++
		if event.Status == model.ModelUsageStatusSuccess {
			count.successes++
		}
		reconstructedCounts[key] = count
	}

	var events []model.ModelUsageEvent
	for _, metric := range metrics {
		key := legacyPerfKey(metric.ModelName, metric.BucketTs)
		count := reconstructedCounts[key]
		requestGap := metric.RequestCount - count.requests
		successGap := metric.SuccessCount - count.successes
		if requestGap < 0 || successGap < 0 || successGap > requestGap {
			return nil, fmt.Errorf("negative legacy performance gap for model %s bucket %d", metric.ModelName, metric.BucketTs)
		}
		for i := int64(0); i < successGap; i++ {
			events = append(events, legacyPerfEvent(options, metric, model.ModelUsageStatusSuccess, i))
		}
		for i := int64(0); i < requestGap-successGap; i++ {
			events = append(events, legacyPerfEvent(options, metric, model.ModelUsageStatusFailure, i))
		}
	}
	return events, nil
}

func legacyPerfEvent(options BackfillOptions, metric model.PerfMetric, status model.ModelUsageStatus, ordinal int64) model.ModelUsageEvent {
	return model.ModelUsageEvent{
		EventKey:      fmt.Sprintf("legacy-perf:%s:%d:%s:%d", metric.ModelName, metric.BucketTs, status, ordinal),
		Kind:          model.ModelUsageKindSync,
		ModelName:     metric.ModelName,
		UseGroup:      metric.Group,
		Status:        status,
		SubmittedAt:   metric.BucketTs,
		CompletedAt:   metric.BucketTs,
		Source:        "backfill_perf",
		BackfillBatch: options.Batch,
		RecordedAt:    common.GetTimestamp(),
		UpdatedAt:     common.GetTimestamp(),
	}
}

func existingBackfillEventKeys(ctx context.Context, candidates []model.ModelUsageEvent) (map[string]bool, error) {
	result := map[string]bool{}
	if len(candidates) == 0 {
		return result, nil
	}
	keys := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		keys = append(keys, candidate.EventKey)
	}
	var existing []string
	if err := model.DB.WithContext(ctx).Model(&model.ModelUsageEvent{}).Where("event_key IN ?", keys).Pluck("event_key", &existing).Error; err != nil {
		return nil, err
	}
	for _, key := range existing {
		result[key] = true
	}
	return result, nil
}

func taskBackfillStatus(status model.TaskStatus) (model.ModelUsageStatus, bool) {
	switch status {
	case model.TaskStatusSuccess:
		return model.ModelUsageStatusSuccess, true
	case model.TaskStatusFailure:
		return model.ModelUsageStatusFailure, true
	default:
		return "", false
	}
}

func lookupBackfillUsername(ctx context.Context, userID int) string {
	if userID <= 0 {
		return ""
	}
	var user model.User
	if err := model.DB.WithContext(ctx).Select("username").Where("id = ?", userID).First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			common.SysLog("model usage backfill username lookup failed: " + err.Error())
		}
		return ""
	}
	return user.Username
}

func extractPositiveJSONInt64(raw json.RawMessage, paths ...string) int64 {
	if len(raw) == 0 {
		return 0
	}
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return 0
	}
	for _, path := range paths {
		value, ok := lookupJSONPath(root, strings.Split(path, "."))
		if !ok {
			continue
		}
		if parsed, ok := jsonNumberToPositiveInt64(value); ok {
			return parsed
		}
	}
	return 0
}

func lookupJSONPath(value any, path []string) (any, bool) {
	current := value
	for _, segment := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func jsonNumberToPositiveInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		if typed <= 0 || math.IsInf(typed, 0) || math.IsNaN(typed) || typed > math.MaxInt64 || typed != math.Trunc(typed) {
			return 0, false
		}
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil && parsed > 0
	default:
		return 0, false
	}
}

func parseGeneratedImages(other string) int {
	if other == "" {
		return 0
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(other), &payload); err != nil {
		return 0
	}
	value, ok := jsonNumberToPositiveInt64(payload["generated_images"])
	if !ok || value > math.MaxInt32 {
		return 0
	}
	return int(value)
}

func logLooksLikeTask(log model.Log) bool {
	if log.Other == "" {
		return false
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(log.Other), &payload); err != nil {
		return strings.Contains(log.Other, "task_id") || strings.Contains(log.Other, "is_task")
	}
	if value, ok := payload["is_task"].(bool); ok && value {
		return true
	}
	if value, ok := payload["task_id"].(string); ok && value != "" {
		return true
	}
	return false
}

func requestEventKey(requestID string) string {
	return "request:" + requestID
}

func legacyPerfKey(modelName string, bucket int64) string {
	return modelName + ":" + strconv.FormatInt(bucket, 10)
}

func hourBucket(ts int64) int64 {
	return ts - ts%3600
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
