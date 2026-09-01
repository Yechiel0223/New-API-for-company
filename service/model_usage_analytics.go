package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/model"
)

type AnalyticsGranularity string

const (
	AnalyticsGranularityHour AnalyticsGranularity = "hour"
	AnalyticsGranularityDay  AnalyticsGranularity = "day"
	AnalyticsGranularityWeek AnalyticsGranularity = "week"
)

type ModelAnalyticsQuery struct {
	StartTimestamp int64
	EndTimestamp   int64
	Granularity    AnalyticsGranularity
	Username       string
	Models         []string
}

type AnalyticsOptions struct {
	Start           int64
	End             int64
	Granularity     AnalyticsGranularity
	Location        *time.Location
	RequestedModels []string
	Now             time.Time
}

type AnalyticsRange struct {
	Start       int64                `json:"start"`
	End         int64                `json:"end"`
	Granularity AnalyticsGranularity `json:"granularity"`
	Timezone    string               `json:"timezone"`
}

type AnalyticsSummary struct {
	TotalCalls   int64   `json:"total_calls"`
	SuccessCalls int64   `json:"success_calls"`
	FailureCalls int64   `json:"failure_calls"`
	RunningCalls int64   `json:"running_calls"`
	TotalQuota   int64   `json:"total_quota"`
	TotalTokens  int64   `json:"total_tokens"`
	PeakRPM      float64 `json:"peak_rpm"`
	PeakRPMAt    int64   `json:"peak_rpm_at"`
	PeakTPM      float64 `json:"peak_tpm"`
	PeakTPMAt    int64   `json:"peak_tpm_at"`
}

type AnalyticsBucket struct {
	BucketStart  int64            `json:"bucket_start"`
	BucketEnd    int64            `json:"bucket_end"`
	ModelName    string           `json:"model_name"`
	TotalCalls   int64            `json:"total_calls"`
	SuccessCalls int64            `json:"success_calls"`
	FailureCalls int64            `json:"failure_calls"`
	RunningCalls int64            `json:"running_calls"`
	Quota        int64            `json:"quota"`
	Tokens       int64            `json:"tokens"`
	OutputCounts map[string]int64 `json:"output_counts"`
}

type AnalyticsModelTotal struct {
	ModelName    string           `json:"model_name"`
	TotalCalls   int64            `json:"total_calls"`
	SuccessCalls int64            `json:"success_calls"`
	FailureCalls int64            `json:"failure_calls"`
	RunningCalls int64            `json:"running_calls"`
	Quota        int64            `json:"quota"`
	Tokens       int64            `json:"tokens"`
	OutputCounts map[string]int64 `json:"output_counts"`
}

type ModelAnalyticsResult struct {
	Range           AnalyticsRange        `json:"range"`
	Summary         AnalyticsSummary      `json:"summary"`
	Series          []AnalyticsBucket     `json:"series"`
	Models          []AnalyticsModelTotal `json:"models"`
	AvailableModels []string              `json:"available_models"`
	UpdatedAt       int64                 `json:"updated_at"`
}

type ModelHealthStatus string

const (
	ModelHealthNoData  ModelHealthStatus = "no_data"
	ModelHealthHealthy ModelHealthStatus = "healthy"
	ModelHealthWarning ModelHealthStatus = "warning"
	ModelHealthFault   ModelHealthStatus = "fault"
)

type HealthOverall struct {
	Status       ModelHealthStatus `json:"status"`
	TotalCalls   int64             `json:"total_calls"`
	SuccessCalls int64             `json:"success_calls"`
	FailureCalls int64             `json:"failure_calls"`
	RunningCalls int64             `json:"running_calls"`
	StuckCalls   int64             `json:"stuck_calls"`
	SuccessRate  float64           `json:"success_rate"`
}

type CurrentLoad struct {
	RPM           float64 `json:"rpm"`
	TPM           float64 `json:"tpm"`
	WindowMinutes int     `json:"window_minutes"`
}

type ModelHealthRow struct {
	ModelName                 string            `json:"model_name"`
	Status                    ModelHealthStatus `json:"status"`
	TotalCalls                int64             `json:"total_calls"`
	SuccessCalls              int64             `json:"success_calls"`
	FailureCalls              int64             `json:"failure_calls"`
	RunningCalls              int64             `json:"running_calls"`
	StuckCalls                int64             `json:"stuck_calls"`
	SuccessRate               float64           `json:"success_rate"`
	P50Ms                     *int64            `json:"p50_ms"`
	P95Ms                     *int64            `json:"p95_ms"`
	SuccessfulDurationSamples int64             `json:"successful_duration_samples"`
	LatestFailureAt           int64             `json:"latest_failure_at"`
	LatestFailureReason       string            `json:"latest_failure_reason"`
}

type ModelHealthResult struct {
	WindowHours int              `json:"window_hours"`
	Overall     HealthOverall    `json:"overall"`
	CurrentLoad CurrentLoad      `json:"current_load"`
	Models      []ModelHealthRow `json:"models"`
	UpdatedAt   int64            `json:"updated_at"`
}

type HealthOptions struct {
	Now         time.Time
	WindowHours int
	Abilities   []model.Ability
	Attempts    []model.ModelUsageAttempt
}

func QueryModelAnalytics(ctx context.Context, query ModelAnalyticsQuery) (ModelAnalyticsResult, error) {
	if query.EndTimestamp < query.StartTimestamp {
		return ModelAnalyticsResult{}, fmt.Errorf("model analytics end timestamp must not precede start timestamp")
	}
	if query.Granularity != "" && !validAnalyticsGranularity(query.Granularity) {
		return ModelAnalyticsResult{}, fmt.Errorf("invalid model analytics granularity")
	}

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return ModelAnalyticsResult{}, err
	}
	events, err := model.ListModelUsageEventsForAnalytics(ctx, model.ModelUsageQuery{
		StartTimestamp: query.StartTimestamp,
		EndTimestamp:   query.EndTimestamp,
		Username:       query.Username,
	})
	if err != nil {
		return ModelAnalyticsResult{}, err
	}

	availableModels := modelNamesFromEvents(events)
	filteredEvents := events
	if query.Models != nil {
		requested := make(map[string]struct{}, len(query.Models))
		for _, modelName := range query.Models {
			requested[modelName] = struct{}{}
		}
		filteredEvents = make([]model.ModelUsageEvent, 0, len(events))
		for _, event := range events {
			if _, ok := requested[event.ModelName]; ok {
				filteredEvents = append(filteredEvents, event)
			}
		}
	}

	now := time.Now()
	result := AggregateModelUsage(filteredEvents, AnalyticsOptions{
		Start:           query.StartTimestamp,
		End:             query.EndTimestamp,
		Granularity:     query.Granularity,
		Location:        location,
		RequestedModels: query.Models,
		Now:             now,
	})
	result.AvailableModels = availableModels
	return result, nil
}

func QueryModelHealth(ctx context.Context, hours int, now time.Time) (ModelHealthResult, error) {
	if hours != 1 && hours != 24 && hours != 168 {
		return ModelHealthResult{}, fmt.Errorf("model health hours must be 1, 24, or 168")
	}
	if now.IsZero() {
		now = time.Now()
	}
	start := now.Add(-time.Duration(hours) * time.Hour).Unix()
	events, err := model.ListModelUsageEventsForHealth(ctx, start, now.Unix())
	if err != nil {
		return ModelHealthResult{}, err
	}
	abilities, err := model.ListEnabledAbilitiesForModelUsage(ctx)
	if err != nil {
		return ModelHealthResult{}, err
	}
	attempts, err := model.ListModelUsageAttemptsForHealth(ctx, start, now.Unix())
	if err != nil {
		return ModelHealthResult{}, err
	}
	return BuildModelHealth(events, HealthOptions{Now: now, WindowHours: hours, Abilities: abilities, Attempts: attempts}), nil
}

func AggregateModelUsage(events []model.ModelUsageEvent, options AnalyticsOptions) ModelAnalyticsResult {
	location := options.Location
	if location == nil {
		location = time.UTC
	}
	granularity := options.Granularity
	if granularity == "" {
		granularity = automaticAnalyticsGranularity(options.Start, options.End)
	}
	updatedAt := options.Now.Unix()
	if options.Now.IsZero() {
		updatedAt = time.Now().Unix()
	}
	result := ModelAnalyticsResult{
		Range: AnalyticsRange{
			Start: options.Start, End: options.End,
			Granularity: granularity, Timezone: location.String(),
		},
		Series:    make([]AnalyticsBucket, 0),
		Models:    make([]AnalyticsModelTotal, 0),
		UpdatedAt: updatedAt,
	}

	uniqueEvents := uniqueModelUsageEvents(events)
	selectedEvents := make([]model.ModelUsageEvent, 0, len(uniqueEvents))
	for _, event := range uniqueEvents {
		if event.SubmittedAt < options.Start || event.SubmittedAt > options.End {
			continue
		}
		selectedEvents = append(selectedEvents, event)
	}

	modelSet := make(map[string]struct{})
	for _, event := range selectedEvents {
		modelSet[event.ModelName] = struct{}{}
	}
	for _, modelName := range options.RequestedModels {
		modelSet[modelName] = struct{}{}
	}
	modelNames := sortedKeys(modelSet)
	result.AvailableModels = append(result.AvailableModels, modelNames...)

	bucketStarts := analyticsBucketStarts(options.Start, options.End, granularity, location)
	buckets := make(map[string]map[int64]*AnalyticsBucket, len(modelNames))
	modelTotals := make(map[string]*AnalyticsModelTotal, len(modelNames))
	for _, modelName := range modelNames {
		buckets[modelName] = make(map[int64]*AnalyticsBucket, len(bucketStarts))
		modelTotals[modelName] = &AnalyticsModelTotal{ModelName: modelName, OutputCounts: make(map[string]int64)}
		for _, bucketStart := range bucketStarts {
			bucketEnd := nextAnalyticsBucket(time.Unix(bucketStart, 0).In(location), granularity).Unix() - 1
			if bucketEnd > options.End {
				bucketEnd = options.End
			}
			buckets[modelName][bucketStart] = &AnalyticsBucket{
				BucketStart:  bucketStart,
				BucketEnd:    bucketEnd,
				ModelName:    modelName,
				OutputCounts: make(map[string]int64),
			}
		}
	}

	minuteCalls := make(map[int64]int64)
	minuteTokens := make(map[int64]int64)
	for _, event := range selectedEvents {
		result.Summary.TotalCalls++
		result.Summary.TotalQuota += int64(event.FinalQuota)
		result.Summary.TotalTokens += event.TotalTokens
		addAnalyticsStatus(&result.Summary.SuccessCalls, &result.Summary.FailureCalls, &result.Summary.RunningCalls, event.Status)

		modelTotal := modelTotals[event.ModelName]
		modelTotal.TotalCalls++
		modelTotal.Quota += int64(event.FinalQuota)
		modelTotal.Tokens += event.TotalTokens
		addAnalyticsStatus(&modelTotal.SuccessCalls, &modelTotal.FailureCalls, &modelTotal.RunningCalls, event.Status)
		if event.OutputCount > 0 && event.OutputUnit != "" {
			modelTotal.OutputCounts[event.OutputUnit] += int64(event.OutputCount)
		}

		bucketStart := analyticsBucketStart(time.Unix(event.SubmittedAt, 0).In(location), granularity).Unix()
		bucket := buckets[event.ModelName][bucketStart]
		bucket.TotalCalls++
		bucket.Quota += int64(event.FinalQuota)
		bucket.Tokens += event.TotalTokens
		addAnalyticsStatus(&bucket.SuccessCalls, &bucket.FailureCalls, &bucket.RunningCalls, event.Status)
		if event.OutputCount > 0 && event.OutputUnit != "" {
			bucket.OutputCounts[event.OutputUnit] += int64(event.OutputCount)
		}

		submitMinute := analyticsMinuteStart(event.SubmittedAt, location)
		minuteCalls[submitMinute]++
	}
	for _, event := range uniqueEvents {
		if isTerminalModelUsageStatus(event.Status) && event.CompletedAt >= options.Start && event.CompletedAt <= options.End {
			completionMinute := analyticsMinuteStart(event.CompletedAt, location)
			minuteTokens[completionMinute] += event.TotalTokens
		}
	}

	for _, modelName := range modelNames {
		result.Models = append(result.Models, *modelTotals[modelName])
		for _, bucketStart := range bucketStarts {
			result.Series = append(result.Series, *buckets[modelName][bucketStart])
		}
	}
	for minute, calls := range minuteCalls {
		if float64(calls) > result.Summary.PeakRPM || float64(calls) == result.Summary.PeakRPM && (result.Summary.PeakRPMAt == 0 || minute < result.Summary.PeakRPMAt) {
			result.Summary.PeakRPM = float64(calls)
			result.Summary.PeakRPMAt = minute
		}
	}
	for minute, tokens := range minuteTokens {
		if float64(tokens) > result.Summary.PeakTPM || float64(tokens) == result.Summary.PeakTPM && (result.Summary.PeakTPMAt == 0 || minute < result.Summary.PeakTPMAt) {
			result.Summary.PeakTPM = float64(tokens)
			result.Summary.PeakTPMAt = minute
		}
	}
	result.Summary.PeakRPM = roundAnalyticsRate(result.Summary.PeakRPM)
	result.Summary.PeakTPM = roundAnalyticsRate(result.Summary.PeakTPM)
	return result
}

func BuildCurrentLoad(events []model.ModelUsageEvent, now time.Time) CurrentLoad {
	const windowMinutes = 5
	windowStart := now.Add(-windowMinutes * time.Minute).Unix()
	nowTimestamp := now.Unix()
	var submittedCalls int64
	var completedTokens int64
	for _, event := range uniqueModelUsageEvents(events) {
		if event.SubmittedAt >= windowStart && event.SubmittedAt <= nowTimestamp {
			submittedCalls++
		}
		if isTerminalModelUsageStatus(event.Status) && event.CompletedAt >= windowStart && event.CompletedAt <= nowTimestamp {
			completedTokens += event.TotalTokens
		}
	}
	return CurrentLoad{
		RPM:           roundAnalyticsRate(float64(submittedCalls) / windowMinutes),
		TPM:           roundAnalyticsRate(float64(completedTokens) / windowMinutes),
		WindowMinutes: windowMinutes,
	}
}

func BuildModelHealth(events []model.ModelUsageEvent, options HealthOptions) ModelHealthResult {
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	windowHours := options.WindowHours
	if windowHours == 0 {
		windowHours = 24
	}
	windowStart := now.Add(-time.Duration(windowHours) * time.Hour).Unix()
	nowTimestamp := now.Unix()
	currentStart := now.Add(-15 * time.Minute).Unix()
	stuckBefore := currentStart

	uniqueEvents := uniqueModelUsageEvents(events)
	windowEvents := make([]model.ModelUsageEvent, 0, len(uniqueEvents))
	currentEvents := make([]model.ModelUsageEvent, 0, len(uniqueEvents))
	for _, event := range uniqueEvents {
		if event.SubmittedAt > nowTimestamp {
			continue
		}
		if modelUsageEventActiveInWindow(event, windowStart, nowTimestamp) || event.Status == model.ModelUsageStatusRunning {
			windowEvents = append(windowEvents, event)
		}
		if modelUsageEventActiveInWindow(event, currentStart, nowTimestamp) || event.Status == model.ModelUsageStatusRunning {
			currentEvents = append(currentEvents, event)
		}
	}
	windowAttempts := make([]model.ModelUsageAttempt, 0, len(options.Attempts))
	currentAttempts := make([]model.ModelUsageAttempt, 0, len(options.Attempts))
	for _, attempt := range uniqueModelUsageAttempts(options.Attempts) {
		if attempt.CompletedAt >= windowStart && attempt.CompletedAt <= nowTimestamp {
			windowAttempts = append(windowAttempts, attempt)
		}
		if attempt.CompletedAt >= currentStart && attempt.CompletedAt <= nowTimestamp {
			currentAttempts = append(currentAttempts, attempt)
		}
	}

	result := ModelHealthResult{
		WindowHours: windowHours,
		CurrentLoad: BuildCurrentLoad(uniqueEvents, now),
		Models:      make([]ModelHealthRow, 0),
		UpdatedAt:   nowTimestamp,
	}
	modelEvents := make(map[string][]model.ModelUsageEvent)
	for _, event := range windowEvents {
		result.Overall.TotalCalls++
		modelEvents[event.ModelName] = append(modelEvents[event.ModelName], event)
		switch event.Status {
		case model.ModelUsageStatusSuccess:
			result.Overall.SuccessCalls++
		case model.ModelUsageStatusFailure:
			result.Overall.FailureCalls++
		case model.ModelUsageStatusRunning:
			result.Overall.RunningCalls++
			if modelUsageEventIsStuck(event, stuckBefore) {
				result.Overall.StuckCalls++
			}
		}
	}
	result.Overall.SuccessRate = terminalSuccessRate(result.Overall.SuccessCalls, result.Overall.FailureCalls)
	modelAttempts := make(map[string][]model.ModelUsageAttempt)
	for _, attempt := range windowAttempts {
		modelAttempts[attempt.ModelName] = append(modelAttempts[attempt.ModelName], attempt)
	}

	enabledChannels := enabledModelChannels(options.Abilities)
	availabilityKnown := options.Abilities != nil
	modelNames := make(map[string]struct{}, len(modelEvents))
	for modelName := range modelEvents {
		modelNames[modelName] = struct{}{}
	}
	for _, modelName := range sortedKeys(modelNames) {
		row := buildModelHealthRow(modelName, modelEvents[modelName], modelAttempts[modelName], enabledChannels[modelName], availabilityKnown, stuckBefore)
		result.Models = append(result.Models, row)
	}

	currentByModel := make(map[string][]model.ModelUsageEvent)
	for _, event := range currentEvents {
		currentByModel[event.ModelName] = append(currentByModel[event.ModelName], event)
	}
	currentAttemptsByModel := make(map[string][]model.ModelUsageAttempt)
	for _, attempt := range currentAttempts {
		currentAttemptsByModel[attempt.ModelName] = append(currentAttemptsByModel[attempt.ModelName], attempt)
	}
	hasUnavailableModel := false
	if availabilityKnown {
		for modelName := range currentByModel {
			if len(enabledChannels[modelName]) == 0 {
				hasUnavailableModel = true
				break
			}
		}
	}
	hasFault := hasUnavailableModel
	hasFailureOrStuck := false
	hasCompleted := false
	for modelName, modelCurrentEvents := range currentByModel {
		if modelUsageEventsHaveFault(modelCurrentEvents, currentAttemptsByModel[modelName], enabledChannels[modelName], availabilityKnown) {
			hasFault = true
		}
		for _, event := range modelCurrentEvents {
			if event.Status == model.ModelUsageStatusFailure || modelUsageEventIsStuck(event, stuckBefore) {
				hasFailureOrStuck = true
			}
			if isTerminalModelUsageStatus(event.Status) {
				hasCompleted = true
			}
		}
	}
	switch {
	case hasFault:
		result.Overall.Status = ModelHealthFault
	case len(currentEvents) == 0:
		result.Overall.Status = ModelHealthNoData
	case hasFailureOrStuck || !hasCompleted:
		result.Overall.Status = ModelHealthWarning
	default:
		result.Overall.Status = ModelHealthHealthy
	}
	return result
}

func buildModelHealthRow(modelName string, events []model.ModelUsageEvent, attempts []model.ModelUsageAttempt, enabledChannels map[int]struct{}, availabilityKnown bool, stuckBefore int64) ModelHealthRow {
	row := ModelHealthRow{ModelName: modelName}
	durations := make([]int64, 0, len(events))
	for _, event := range events {
		row.TotalCalls++
		switch event.Status {
		case model.ModelUsageStatusSuccess:
			row.SuccessCalls++
			if event.DurationMs > 0 {
				durations = append(durations, event.DurationMs)
			}
		case model.ModelUsageStatusFailure:
			row.FailureCalls++
			failureAt := event.CompletedAt
			if failureAt == 0 {
				failureAt = event.SubmittedAt
			}
			if failureAt >= row.LatestFailureAt {
				row.LatestFailureAt = failureAt
				row.LatestFailureReason = event.FailureReason
			}
		case model.ModelUsageStatusRunning:
			row.RunningCalls++
			if modelUsageEventIsStuck(event, stuckBefore) {
				row.StuckCalls++
			}
		}
	}
	row.SuccessRate = terminalSuccessRate(row.SuccessCalls, row.FailureCalls)
	sort.Slice(durations, func(i int, j int) bool { return durations[i] < durations[j] })
	row.SuccessfulDurationSamples = int64(len(durations))
	if len(durations) > 0 {
		p50 := nearestRank(durations, 0.50)
		row.P50Ms = &p50
	}
	if len(durations) >= 3 {
		p95 := nearestRank(durations, 0.95)
		row.P95Ms = &p95
	}

	switch {
	case modelUsageEventsHaveFault(events, attempts, enabledChannels, availabilityKnown):
		row.Status = ModelHealthFault
	case row.FailureCalls > 0 || row.StuckCalls > 0 || row.RunningCalls > 0 && row.SuccessCalls == 0:
		row.Status = ModelHealthWarning
	case row.SuccessCalls > 0:
		row.Status = ModelHealthHealthy
	default:
		row.Status = ModelHealthNoData
	}
	return row
}

func modelUsageEventsHaveFault(events []model.ModelUsageEvent, attempts []model.ModelUsageAttempt, enabledChannels map[int]struct{}, availabilityKnown bool) bool {
	if availabilityKnown && len(enabledChannels) == 0 {
		return true
	}
	terminalEvents := make([]model.ModelUsageEvent, 0, len(events))
	for _, event := range events {
		if isTerminalModelUsageStatus(event.Status) {
			terminalEvents = append(terminalEvents, event)
		}
	}
	sort.Slice(terminalEvents, func(i int, j int) bool {
		return modelUsageTerminalTime(terminalEvents[i]) < modelUsageTerminalTime(terminalEvents[j])
	})
	if len(terminalEvents) >= 3 {
		last := terminalEvents[len(terminalEvents)-3:]
		if last[0].Status == model.ModelUsageStatusFailure && last[1].Status == model.ModelUsageStatusFailure && last[2].Status == model.ModelUsageStatusFailure {
			return true
		}
	}
	if !availabilityKnown || len(enabledChannels) == 0 {
		return false
	}
	latestByChannel := make(map[int]model.ModelUsageAttempt, len(enabledChannels))
	for _, attempt := range attempts {
		if _, enabled := enabledChannels[attempt.ChannelID]; !enabled {
			continue
		}
		latest, exists := latestByChannel[attempt.ChannelID]
		if !exists || attempt.CompletedAt >= latest.CompletedAt {
			latestByChannel[attempt.ChannelID] = attempt
		}
	}
	if len(latestByChannel) != len(enabledChannels) {
		return false
	}
	for channelID := range enabledChannels {
		if latestByChannel[channelID].Status != model.ModelUsageStatusFailure {
			return false
		}
	}
	return true
}

func uniqueModelUsageAttempts(attempts []model.ModelUsageAttempt) []model.ModelUsageAttempt {
	type attemptKey struct {
		eventKey  string
		channelID int
		fallback  int
	}
	unique := make(map[attemptKey]model.ModelUsageAttempt, len(attempts))
	order := make([]attemptKey, 0, len(attempts))
	for index, attempt := range attempts {
		key := attemptKey{eventKey: attempt.EventKey, channelID: attempt.ChannelID}
		if attempt.EventKey == "" || attempt.ChannelID <= 0 {
			key.fallback = index + 1
		}
		previous, exists := unique[key]
		if !exists {
			order = append(order, key)
		}
		if !exists || attempt.CompletedAt >= previous.CompletedAt {
			unique[key] = attempt
		}
	}
	result := make([]model.ModelUsageAttempt, 0, len(unique))
	for _, key := range order {
		result = append(result, unique[key])
	}
	return result
}

func enabledModelChannels(abilities []model.Ability) map[string]map[int]struct{} {
	channels := make(map[string]map[int]struct{})
	for _, ability := range abilities {
		if !ability.Enabled {
			continue
		}
		if channels[ability.Model] == nil {
			channels[ability.Model] = make(map[int]struct{})
		}
		channels[ability.Model][ability.ChannelId] = struct{}{}
	}
	return channels
}

func uniqueModelUsageEvents(events []model.ModelUsageEvent) []model.ModelUsageEvent {
	unique := make(map[string]model.ModelUsageEvent, len(events))
	order := make([]string, 0, len(events))
	for index, event := range events {
		key := event.EventKey
		if key == "" {
			key = fmt.Sprintf("missing:%d", index)
		}
		previous, exists := unique[key]
		if !exists {
			order = append(order, key)
		}
		if !exists || event.UpdatedAt >= previous.UpdatedAt {
			unique[key] = event
		}
	}
	result := make([]model.ModelUsageEvent, 0, len(unique))
	for _, key := range order {
		result = append(result, unique[key])
	}
	return result
}

func analyticsBucketStarts(start int64, end int64, granularity AnalyticsGranularity, location *time.Location) []int64 {
	if end < start {
		return nil
	}
	starts := make([]int64, 0)
	for bucket := analyticsBucketStart(time.Unix(start, 0).In(location), granularity); bucket.Unix() <= end; bucket = nextAnalyticsBucket(bucket, granularity) {
		starts = append(starts, bucket.Unix())
	}
	return starts
}

func analyticsBucketStart(timestamp time.Time, granularity AnalyticsGranularity) time.Time {
	year, month, day := timestamp.Date()
	switch granularity {
	case AnalyticsGranularityDay:
		return time.Date(year, month, day, 0, 0, 0, 0, timestamp.Location())
	case AnalyticsGranularityWeek:
		weekdayOffset := (int(timestamp.Weekday()) + 6) % 7
		return time.Date(year, month, day-weekdayOffset, 0, 0, 0, 0, timestamp.Location())
	default:
		return time.Date(year, month, day, timestamp.Hour(), 0, 0, 0, timestamp.Location())
	}
}

func nextAnalyticsBucket(bucket time.Time, granularity AnalyticsGranularity) time.Time {
	switch granularity {
	case AnalyticsGranularityDay:
		return bucket.AddDate(0, 0, 1)
	case AnalyticsGranularityWeek:
		return bucket.AddDate(0, 0, 7)
	default:
		return bucket.Add(time.Hour)
	}
}

func automaticAnalyticsGranularity(start int64, end int64) AnalyticsGranularity {
	duration := time.Duration(end-start) * time.Second
	if duration <= 48*time.Hour {
		return AnalyticsGranularityHour
	}
	if duration <= 90*24*time.Hour {
		return AnalyticsGranularityDay
	}
	return AnalyticsGranularityWeek
}

func validAnalyticsGranularity(granularity AnalyticsGranularity) bool {
	return granularity == AnalyticsGranularityHour || granularity == AnalyticsGranularityDay || granularity == AnalyticsGranularityWeek
}

func addAnalyticsStatus(success *int64, failure *int64, running *int64, status model.ModelUsageStatus) {
	switch status {
	case model.ModelUsageStatusSuccess:
		(*success)++
	case model.ModelUsageStatusFailure:
		(*failure)++
	case model.ModelUsageStatusRunning:
		(*running)++
	}
}

func modelNamesFromEvents(events []model.ModelUsageEvent) []string {
	names := make(map[string]struct{})
	for _, event := range events {
		names[event.ModelName] = struct{}{}
	}
	return sortedKeys(names)
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func analyticsMinuteStart(timestamp int64, location *time.Location) int64 {
	value := time.Unix(timestamp, 0).In(location)
	return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), value.Minute(), 0, 0, location).Unix()
}

func isTerminalModelUsageStatus(status model.ModelUsageStatus) bool {
	return status == model.ModelUsageStatusSuccess || status == model.ModelUsageStatusFailure
}

func modelUsageEventIsStuck(event model.ModelUsageEvent, stuckBefore int64) bool {
	return event.Status == model.ModelUsageStatusRunning && event.LastProgressAt <= stuckBefore
}

func modelUsageEventActiveInWindow(event model.ModelUsageEvent, start int64, end int64) bool {
	if event.SubmittedAt >= start && event.SubmittedAt <= end {
		return true
	}
	return isTerminalModelUsageStatus(event.Status) && event.CompletedAt >= start && event.CompletedAt <= end
}

func modelUsageTerminalTime(event model.ModelUsageEvent) int64 {
	if event.CompletedAt > 0 {
		return event.CompletedAt
	}
	return event.SubmittedAt
}

func terminalSuccessRate(success int64, failure int64) float64 {
	terminal := success + failure
	if terminal == 0 {
		return 0
	}
	return roundAnalyticsRate(float64(success) * 100 / float64(terminal))
}

func nearestRank(sortedValues []int64, percentile float64) int64 {
	rank := int(math.Ceil(percentile * float64(len(sortedValues))))
	if rank < 1 {
		rank = 1
	}
	return sortedValues[rank-1]
}

func roundAnalyticsRate(value float64) float64 {
	return math.Round(value*100) / 100
}
