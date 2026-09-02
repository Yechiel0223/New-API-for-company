package service

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustShanghai(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	return location
}

func analyticsEvent(eventKey string, modelName string, submittedAt int64, tokens int64, quota int, status model.ModelUsageStatus) model.ModelUsageEvent {
	return model.ModelUsageEvent{
		EventKey:    eventKey,
		ModelName:   modelName,
		SubmittedAt: submittedAt,
		TotalTokens: tokens,
		FinalQuota:  quota,
		Status:      status,
	}
}

func TestAggregateModelUsageCountsUniqueCallsAndFillsRealRange(t *testing.T) {
	location := mustShanghai(t)
	start := time.Date(2026, 9, 1, 0, 30, 0, 0, location).Unix()
	end := time.Date(2026, 9, 1, 3, 15, 0, 0, location).Unix()
	first := analyticsEvent("request:a", "model-a", time.Date(2026, 9, 1, 0, 45, 0, 0, location).Unix(), 10, 100, model.ModelUsageStatusSuccess)
	first.OutputCount = 2
	first.OutputUnit = "image"
	finalTask := analyticsEvent("task:b", "model-b", time.Date(2026, 9, 1, 2, 5, 0, 0, location).Unix(), 20, 200, model.ModelUsageStatusFailure)
	finalTask.CompletedAt = time.Date(2026, 9, 2, 2, 5, 0, 0, location).Unix()

	result := AggregateModelUsage([]model.ModelUsageEvent{first, first, finalTask}, AnalyticsOptions{
		Start:           start,
		End:             end,
		Granularity:     AnalyticsGranularityHour,
		Location:        location,
		RequestedModels: []string{"model-c"},
		Now:             time.Unix(end, 0),
	})

	assert.Equal(t, int64(2), result.Summary.TotalCalls)
	assert.Equal(t, int64(1), result.Summary.SuccessCalls)
	assert.Equal(t, int64(1), result.Summary.FailureCalls)
	assert.Equal(t, int64(30), result.Summary.TotalTokens)
	assert.Equal(t, int64(300), result.Summary.TotalQuota)
	assert.Equal(t, []string{"model-a", "model-b", "model-c"}, result.AvailableModels)
	require.Len(t, result.Series, 12)

	var seriesCalls int64
	var seriesTokens int64
	var seriesQuota int64
	for _, bucket := range result.Series {
		seriesCalls += bucket.TotalCalls
		seriesTokens += bucket.Tokens
		seriesQuota += bucket.Quota
		if bucket.ModelName == "model-c" {
			assert.Zero(t, bucket.TotalCalls)
			assert.Empty(t, bucket.OutputCounts)
		}
	}
	assert.Equal(t, result.Summary.TotalCalls, seriesCalls)
	assert.Equal(t, result.Summary.TotalTokens, seriesTokens)
	assert.Equal(t, result.Summary.TotalQuota, seriesQuota)

	modelAFirstBucket := result.Series[0]
	assert.Equal(t, time.Date(2026, 9, 1, 0, 0, 0, 0, location).Unix(), modelAFirstBucket.BucketStart)
	assert.Equal(t, int64(1), modelAFirstBucket.TotalCalls)
	assert.Equal(t, int64(2), modelAFirstBucket.OutputCounts["image"])

	modelBThirdBucket := result.Series[6]
	assert.Equal(t, "model-b", modelBThirdBucket.ModelName)
	assert.Equal(t, time.Date(2026, 9, 1, 2, 0, 0, 0, location).Unix(), modelBThirdBucket.BucketStart)
	assert.Equal(t, int64(1), modelBThirdBucket.TotalCalls)
}

func TestAggregateModelUsageAlignsDayAndWeekBucketsInShanghai(t *testing.T) {
	location := mustShanghai(t)
	tests := []struct {
		name        string
		granularity AnalyticsGranularity
		start       time.Time
		end         time.Time
		wantStarts  []time.Time
	}{
		{
			name:        "day",
			granularity: AnalyticsGranularityDay,
			start:       time.Date(2026, 9, 1, 14, 0, 0, 0, location),
			end:         time.Date(2026, 9, 3, 8, 0, 0, 0, location),
			wantStarts: []time.Time{
				time.Date(2026, 9, 1, 0, 0, 0, 0, location),
				time.Date(2026, 9, 2, 0, 0, 0, 0, location),
				time.Date(2026, 9, 3, 0, 0, 0, 0, location),
			},
		},
		{
			name:        "week starts Monday",
			granularity: AnalyticsGranularityWeek,
			start:       time.Date(2026, 9, 2, 14, 0, 0, 0, location),
			end:         time.Date(2026, 9, 15, 8, 0, 0, 0, location),
			wantStarts: []time.Time{
				time.Date(2026, 8, 31, 0, 0, 0, 0, location),
				time.Date(2026, 9, 7, 0, 0, 0, 0, location),
				time.Date(2026, 9, 14, 0, 0, 0, 0, location),
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := AggregateModelUsage(nil, AnalyticsOptions{
				Start: testCase.start.Unix(), End: testCase.end.Unix(),
				Granularity: testCase.granularity, Location: location,
				RequestedModels: []string{"model-a"},
			})
			require.Len(t, result.Series, len(testCase.wantStarts))
			for index, wantStart := range testCase.wantStarts {
				assert.Equal(t, wantStart.Unix(), result.Series[index].BucketStart)
			}
		})
	}
}

func TestAggregateModelUsageCalculatesSubmitAndCompletionPeaks(t *testing.T) {
	location := mustShanghai(t)
	start := time.Date(2026, 9, 1, 10, 0, 0, 0, location)
	events := []model.ModelUsageEvent{
		analyticsEvent("request:a", "model-a", start.Add(10*time.Second).Unix(), 10, 100, model.ModelUsageStatusSuccess),
		analyticsEvent("request:b", "model-a", start.Add(20*time.Second).Unix(), 20, 200, model.ModelUsageStatusSuccess),
		analyticsEvent("request:c", "model-a", start.Add(70*time.Second).Unix(), 25, 300, model.ModelUsageStatusSuccess),
	}
	events[0].CompletedAt = start.Add(2*time.Minute + 10*time.Second).Unix()
	events[1].CompletedAt = start.Add(2*time.Minute + 20*time.Second).Unix()
	events[2].CompletedAt = start.Add(3*time.Minute + 10*time.Second).Unix()

	result := AggregateModelUsage(events, AnalyticsOptions{
		Start: start.Unix(), End: start.Add(4 * time.Minute).Unix(),
		Granularity: AnalyticsGranularityHour, Location: location,
	})

	assert.Equal(t, 2.0, result.Summary.PeakRPM)
	assert.Equal(t, start.Unix(), result.Summary.PeakRPMAt)
	assert.Equal(t, 30.0, result.Summary.PeakTPM)
	assert.Equal(t, start.Add(2*time.Minute).Unix(), result.Summary.PeakTPMAt)
}

func healthEvent(eventKey string, modelName string, submittedAt int64, status model.ModelUsageStatus, durationMs int64) model.ModelUsageEvent {
	completedAt := submittedAt + durationMs/1_000
	return model.ModelUsageEvent{
		EventKey: eventKey, ModelName: modelName, Status: status,
		SubmittedAt: submittedAt, CompletedAt: completedAt,
		LastProgressAt: completedAt, DurationMs: durationMs,
	}
}

func requireHealthModel(t *testing.T, result ModelHealthResult, modelName string) ModelHealthRow {
	t.Helper()
	for _, row := range result.Models {
		if row.ModelName == modelName {
			return row
		}
	}
	require.FailNow(t, "health model not found", modelName)
	return ModelHealthRow{}
}

func TestBuildModelHealthUsesWeightedSuccessAndSuccessfulLatency(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	events := []model.ModelUsageEvent{
		healthEvent("request:a", "seedream", now.Add(-10*time.Minute).Unix(), model.ModelUsageStatusSuccess, 50_000),
		healthEvent("request:b", "seedream", now.Add(-9*time.Minute).Unix(), model.ModelUsageStatusSuccess, 55_000),
		healthEvent("request:c", "seedream", now.Add(-8*time.Minute).Unix(), model.ModelUsageStatusSuccess, 52_000),
		healthEvent("request:d", "seedream", now.Add(-7*time.Minute).Unix(), model.ModelUsageStatusFailure, 180_000),
		healthEvent("task:e", "seedance", now.Add(-6*time.Minute).Unix(), model.ModelUsageStatusSuccess, 130_000),
	}

	result := BuildModelHealth(events, HealthOptions{Now: now, WindowHours: 24})

	assert.Equal(t, int64(4), result.Overall.SuccessCalls)
	assert.Equal(t, int64(5), result.Overall.TotalCalls)
	assert.Equal(t, 80.0, result.Overall.SuccessRate)
	seedream := requireHealthModel(t, result, "seedream")
	assert.Equal(t, int64(3), seedream.SuccessCalls)
	assert.Equal(t, int64(4), seedream.TotalCalls)
	assert.Equal(t, 75.0, seedream.SuccessRate)
	require.NotNil(t, seedream.P50Ms)
	require.NotNil(t, seedream.P95Ms)
	assert.Equal(t, int64(52_000), *seedream.P50Ms)
	assert.Equal(t, int64(55_000), *seedream.P95Ms)
	assert.Equal(t, int64(3), seedream.SuccessfulDurationSamples)
	assert.Equal(t, events[3].CompletedAt, seedream.LatestFailureAt)
}

func TestBuildModelHealthIncludesOldSubmissionCompletedInCurrentWindow(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	event := healthEvent("task:recent-completion", "seedance", now.Add(-2*time.Hour).Unix(), model.ModelUsageStatusSuccess, 7_080_000)

	result := BuildModelHealth([]model.ModelUsageEvent{event}, HealthOptions{Now: now, WindowHours: 1})

	assert.Equal(t, ModelHealthHealthy, result.Overall.Status)
	assert.Equal(t, int64(1), result.Overall.TotalCalls)
	assert.Equal(t, int64(1), result.Overall.SuccessCalls)
	row := requireHealthModel(t, result, "seedance")
	assert.Equal(t, int64(1), row.TotalCalls)
	require.NotNil(t, row.P50Ms)
	assert.Equal(t, event.DurationMs, *row.P50Ms)
}

func TestBuildModelHealthAppliesCurrentStateFaultAndSampleRules(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		events     []model.ModelUsageEvent
		attempts   []model.ModelUsageAttempt
		abilities  []model.Ability
		wantStatus ModelHealthStatus
		wantStuck  int64
		check      func(t *testing.T, result ModelHealthResult)
	}{
		{
			name:       "no calls",
			abilities:  []model.Ability{{Model: "idle-model", ChannelId: 1, Enabled: true}},
			wantStatus: ModelHealthNoData,
		},
		{
			name: "task stuck at boundary",
			events: []model.ModelUsageEvent{{
				EventKey: "task:stuck", ModelName: "seedance", Status: model.ModelUsageStatusRunning,
				SubmittedAt: now.Add(-time.Hour).Unix(), LastProgressAt: now.Add(-15 * time.Minute).Unix(),
			}},
			wantStatus: ModelHealthWarning,
			wantStuck:  1,
		},
		{
			name: "last three terminal calls failed",
			events: []model.ModelUsageEvent{
				healthEvent("request:success", "model-a", now.Add(-14*time.Minute).Unix(), model.ModelUsageStatusSuccess, 1_000),
				healthEvent("request:failure-1", "model-a", now.Add(-12*time.Minute).Unix(), model.ModelUsageStatusFailure, 1_000),
				healthEvent("request:failure-2", "model-a", now.Add(-10*time.Minute).Unix(), model.ModelUsageStatusFailure, 1_000),
				healthEvent("request:failure-3", "model-a", now.Add(-8*time.Minute).Unix(), model.ModelUsageStatusFailure, 1_000),
			},
			wantStatus: ModelHealthFault,
		},
		{
			name: "observed model has zero enabled channels",
			events: []model.ModelUsageEvent{
				healthEvent("request:success", "unavailable", now.Add(-5*time.Minute).Unix(), model.ModelUsageStatusSuccess, 1_000),
			},
			abilities:  []model.Ability{},
			wantStatus: ModelHealthFault,
		},
		{
			name: "all enabled channels latest attempts failed",
			events: []model.ModelUsageEvent{
				{EventKey: "request:canonical", ModelName: "model-a", ChannelID: 1, Status: model.ModelUsageStatusFailure, SubmittedAt: now.Add(-10 * time.Minute).Unix(), CompletedAt: now.Add(-7 * time.Minute).Unix()},
			},
			attempts: []model.ModelUsageAttempt{
				{EventKey: "request:canonical", ModelName: "model-a", ChannelID: 1, Status: model.ModelUsageStatusFailure, CompletedAt: now.Add(-9 * time.Minute).Unix()},
				{EventKey: "request:canonical", ModelName: "model-a", ChannelID: 2, Status: model.ModelUsageStatusFailure, CompletedAt: now.Add(-7 * time.Minute).Unix()},
			},
			abilities: []model.Ability{
				{Model: "model-a", ChannelId: 1, Enabled: true},
				{Model: "model-a", ChannelId: 2, Enabled: true},
			},
			wantStatus: ModelHealthFault,
		},
		{
			name: "later channel success clears all-channel fault",
			events: []model.ModelUsageEvent{
				{EventKey: "request:canonical", ModelName: "model-a", ChannelID: 1, Status: model.ModelUsageStatusFailure, SubmittedAt: now.Add(-10 * time.Minute).Unix(), CompletedAt: now.Add(-7 * time.Minute).Unix()},
			},
			attempts: []model.ModelUsageAttempt{
				{EventKey: "request:older", ModelName: "model-a", ChannelID: 1, Status: model.ModelUsageStatusFailure, CompletedAt: now.Add(-9 * time.Minute).Unix()},
				{EventKey: "request:older", ModelName: "model-a", ChannelID: 2, Status: model.ModelUsageStatusFailure, CompletedAt: now.Add(-8 * time.Minute).Unix()},
				{EventKey: "request:newer", ModelName: "model-a", ChannelID: 2, Status: model.ModelUsageStatusSuccess, CompletedAt: now.Add(-6 * time.Minute).Unix()},
			},
			abilities: []model.Ability{
				{Model: "model-a", ChannelId: 1, Enabled: true},
				{Model: "model-a", ChannelId: 2, Enabled: true},
			},
			wantStatus: ModelHealthWarning,
		},
		{
			name: "higher attempt id wins when channel attempts complete in the same second",
			events: []model.ModelUsageEvent{
				{EventKey: "request:canonical", ModelName: "model-a", ChannelID: 1, Status: model.ModelUsageStatusFailure, SubmittedAt: now.Add(-10 * time.Minute).Unix(), CompletedAt: now.Add(-7 * time.Minute).Unix()},
			},
			attempts: []model.ModelUsageAttempt{
				{ID: 2, EventKey: "request:newer", ModelName: "model-a", ChannelID: 1, Status: model.ModelUsageStatusSuccess, CompletedAt: now.Add(-7 * time.Minute).Unix()},
				{ID: 1, EventKey: "request:older", ModelName: "model-a", ChannelID: 1, Status: model.ModelUsageStatusFailure, CompletedAt: now.Add(-7 * time.Minute).Unix()},
			},
			abilities: []model.Ability{
				{Model: "model-a", ChannelId: 1, Enabled: true},
			},
			wantStatus: ModelHealthWarning,
		},
		{
			name: "P95 needs three successful duration samples",
			events: []model.ModelUsageEvent{
				healthEvent("request:a", "model-a", now.Add(-10*time.Minute).Unix(), model.ModelUsageStatusSuccess, 2_000),
				healthEvent("request:b", "model-a", now.Add(-8*time.Minute).Unix(), model.ModelUsageStatusSuccess, 4_000),
			},
			wantStatus: ModelHealthHealthy,
			check: func(t *testing.T, result ModelHealthResult) {
				row := requireHealthModel(t, result, "model-a")
				require.NotNil(t, row.P50Ms)
				assert.Equal(t, int64(2_000), *row.P50Ms)
				assert.Nil(t, row.P95Ms)
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			result := BuildModelHealth(testCase.events, HealthOptions{
				Now: now, WindowHours: 24, Abilities: testCase.abilities, Attempts: testCase.attempts,
			})
			assert.Equal(t, testCase.wantStatus, result.Overall.Status)
			assert.Equal(t, testCase.wantStuck, result.Overall.StuckCalls)
			if testCase.check != nil {
				testCase.check(t, result)
			}
		})
	}
}

func TestBuildModelHealthCurrentStatusDoesNotChangeWithDetailWindow(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	events := []model.ModelUsageEvent{
		{EventKey: "request:current", ModelName: "active", ChannelID: 1, Status: model.ModelUsageStatusSuccess, SubmittedAt: now.Add(-5 * time.Minute).Unix(), CompletedAt: now.Add(-4 * time.Minute).Unix(), DurationMs: 60_000},
		{EventKey: "request:historical", ModelName: "retired", ChannelID: 2, Status: model.ModelUsageStatusFailure, SubmittedAt: now.Add(-2 * time.Hour).Unix(), CompletedAt: now.Add(-119 * time.Minute).Unix(), DurationMs: 60_000},
	}
	abilities := []model.Ability{{Model: "active", ChannelId: 1, Enabled: true}}

	oneHour := BuildModelHealth(events, HealthOptions{Now: now, WindowHours: 1, Abilities: abilities})
	twentyFourHours := BuildModelHealth(events, HealthOptions{Now: now, WindowHours: 24, Abilities: abilities})

	assert.Equal(t, ModelHealthHealthy, oneHour.Overall.Status)
	assert.Equal(t, oneHour.Overall.Status, twentyFourHours.Overall.Status)
	assert.Equal(t, int64(1), oneHour.Overall.TotalCalls)
	assert.Equal(t, int64(2), twentyFourHours.Overall.TotalCalls)
}

func TestCurrentLoadUsesSubmitAndCompletionWindows(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	events := []model.ModelUsageEvent{
		{EventKey: "request:submit-boundary", Status: model.ModelUsageStatusSuccess, SubmittedAt: now.Add(-5 * time.Minute).Unix(), CompletedAt: now.Add(-10 * time.Minute).Unix(), TotalTokens: 100},
		{EventKey: "request:submit-before", Status: model.ModelUsageStatusSuccess, SubmittedAt: now.Add(-5*time.Minute - time.Second).Unix(), CompletedAt: now.Add(-4 * time.Minute).Unix(), TotalTokens: 200},
		{EventKey: "task:completed-boundary", Status: model.ModelUsageStatusSuccess, SubmittedAt: now.Add(-time.Hour).Unix(), CompletedAt: now.Add(-5 * time.Minute).Unix(), TotalTokens: 300},
		{EventKey: "task:running", Status: model.ModelUsageStatusRunning, SubmittedAt: now.Add(-2 * time.Minute).Unix(), CompletedAt: now.Add(-time.Minute).Unix(), TotalTokens: 400},
		{EventKey: "request:future", Status: model.ModelUsageStatusSuccess, SubmittedAt: now.Add(time.Second).Unix(), CompletedAt: now.Add(time.Second).Unix(), TotalTokens: 500},
	}

	load := BuildCurrentLoad(events, now)

	assert.Equal(t, 0.4, load.RPM)
	assert.Equal(t, 100.0, load.TPM)
	assert.Equal(t, 5, load.WindowMinutes)
}

func TestQueryModelAnalyticsAndHealthUseFactTableFilters(t *testing.T) {
	setupUsageRecorderDB(t)
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	require.NoError(t, model.DB.AutoMigrate(&model.Ability{}))
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "model-a", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "backup", Model: "model-a", ChannelId: 3, Enabled: true}).Error)
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "model-b", ChannelId: 2, Enabled: true}).Error)
	events := []*model.ModelUsageEvent{
		{EventKey: "request:alice-a", Kind: model.ModelUsageKindSync, ModelName: "model-a", Username: "alice", ChannelID: 1, Status: model.ModelUsageStatusSuccess, SubmittedAt: now.Add(-10 * time.Minute).Unix(), CompletedAt: now.Add(-9 * time.Minute).Unix(), DurationMs: 1_000, Source: "live"},
		{EventKey: "request:alice-b", Kind: model.ModelUsageKindSync, ModelName: "model-b", Username: "alice", Status: model.ModelUsageStatusSuccess, SubmittedAt: now.Add(-8 * time.Minute).Unix(), CompletedAt: now.Add(-7 * time.Minute).Unix(), DurationMs: 2_000, Source: "live"},
		{EventKey: "request:bob-a", Kind: model.ModelUsageKindSync, ModelName: "model-a", Username: "bob", ChannelID: 1, Status: model.ModelUsageStatusFailure, SubmittedAt: now.Add(-6 * time.Minute).Unix(), CompletedAt: now.Add(-5 * time.Minute).Unix(), Source: "live"},
	}
	for _, event := range events {
		require.NoError(t, model.CreateModelUsageEvent(event))
	}

	analytics, err := QueryModelAnalytics(context.Background(), ModelAnalyticsQuery{
		StartTimestamp: now.Add(-time.Hour).Unix(), EndTimestamp: now.Unix(),
		Username: "alice", Models: []string{"model-a"},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), analytics.Summary.TotalCalls)
	assert.Equal(t, []string{"model-a", "model-b"}, analytics.AvailableModels)
	assert.Equal(t, AnalyticsGranularityHour, analytics.Range.Granularity)

	health, err := QueryModelHealth(context.Background(), 24, now)
	require.NoError(t, err)
	assert.Equal(t, int64(3), health.Overall.TotalCalls)
	assert.Equal(t, ModelHealthWarning, health.Overall.Status)
	_, err = QueryModelHealth(context.Background(), 721, now)
	require.Error(t, err)
}

func TestQueryModelHealthIncludesOldSubmissionCompletedInWindow(t *testing.T) {
	setupUsageRecorderDB(t)
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	require.NoError(t, model.DB.AutoMigrate(&model.Ability{}))
	require.NoError(t, model.DB.Create(&model.Ability{Group: "default", Model: "seedance", ChannelId: 1, Enabled: true}).Error)
	require.NoError(t, model.CreateModelUsageEvent(&model.ModelUsageEvent{
		EventKey: "task:recent-completion", Kind: model.ModelUsageKindTask,
		ModelName: "seedance", ChannelID: 1, Status: model.ModelUsageStatusSuccess,
		SubmittedAt: now.Add(-2 * time.Hour).Unix(), CompletedAt: now.Add(-5 * time.Minute).Unix(),
		DurationMs: int64((115 * time.Minute) / time.Millisecond), Source: "live",
	}))

	result, err := QueryModelHealth(context.Background(), 1, now)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Overall.TotalCalls)
	assert.Equal(t, ModelHealthHealthy, result.Overall.Status)
	assert.Equal(t, int64(1), requireHealthModel(t, result, "seedance").SuccessCalls)
}

func TestQueryModelAnalyticsPeakTPMIncludesOldSubmissionCompletedInRange(t *testing.T) {
	setupUsageRecorderDB(t)
	location := mustShanghai(t)
	end := time.Date(2026, 9, 2, 12, 0, 0, 0, location)
	start := end.Add(-10 * time.Minute)
	require.NoError(t, model.CreateModelUsageEvent(&model.ModelUsageEvent{
		EventKey: "task:recent-tokens", Kind: model.ModelUsageKindTask,
		ModelName: "seedance", Status: model.ModelUsageStatusSuccess,
		SubmittedAt: end.Add(-time.Hour).Unix(), CompletedAt: end.Add(-2 * time.Minute).Unix(),
		TotalTokens: 300, DurationMs: int64((58 * time.Minute) / time.Millisecond), Source: "live",
	}))

	result, err := QueryModelAnalytics(context.Background(), ModelAnalyticsQuery{
		StartTimestamp: start.Unix(), EndTimestamp: end.Unix(), Granularity: AnalyticsGranularityHour,
	})
	require.NoError(t, err)
	assert.Zero(t, result.Summary.TotalCalls)
	assert.Zero(t, result.Summary.TotalTokens)
	for _, bucket := range result.Series {
		assert.Zero(t, bucket.TotalCalls)
		assert.Zero(t, bucket.Tokens)
	}
	assert.Equal(t, 300.0, result.Summary.PeakTPM)
	assert.Equal(t, end.Add(-2*time.Minute).Truncate(time.Minute).Unix(), result.Summary.PeakTPMAt)
}
