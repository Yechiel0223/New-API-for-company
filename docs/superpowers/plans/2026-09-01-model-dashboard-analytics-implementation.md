# Model Dashboard Analytics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an administrator model analytics dashboard whose selected-range totals, real-time load, time series, and per-model health all come from one deduplicated record per synchronous request or asynchronous task.

**Architecture:** Add a derived `model_usage_events` fact table in the main database and keep `logs`, `tasks`, `quota_data`, and `perf_metrics` unchanged as billing/task source records and compatibility data. Live request/task completion paths upsert one fact row, a local idempotent backfill reconstructs historical rows, and two admin-only APIs expose selected-range analytics and fixed-window health. The React dashboard consumes these APIs through React Query and renders exact range totals, real time RPM/TPM, zero-filled timelines, zoom controls, and direct per-model health rows.

**Tech Stack:** Go 1.25.1, Gin, GORM v2, SQLite/MySQL/PostgreSQL, React 19, TypeScript, TanStack Query, VChart 2.1.4, Day.js, Bun/Vitest.

**Spec:** `docs/superpowers/specs/2026-09-01-model-dashboard-analytics-prd.md` and `docs/superpowers/specs/2026-09-01-model-dashboard-analytics-technical-design.md`

## Global Constraints

- Keep `quota_data` and the existing `/api/data`, `/api/data/users`, and `/api/data/flow` behavior intact for the overview, user, and flow dashboards.
- Store analytics in the main database; read synchronous history from `LOG_DB`, which may be a separate database.
- Support SQLite, MySQL >= 5.7.8, and PostgreSQL >= 9.6 with the same GORM model and query behavior.
- Use `common.Marshal`, `common.Unmarshal`, and related wrappers instead of direct JSON marshal/unmarshal calls.
- Analytics write failures must be logged but must never change request responses, task settlement, refunds, wallet balances, or channel accounting.
- Never derive billing again in analytics. `final_quota` copies the already-settled amount from the billing/task source record and must be non-negative.
- Treat `Asia/Shanghai` as the reporting timezone. Hour, day, and week buckets start in that timezone; weeks start Monday at 00:00.
- Top-level calls, consumption, and Token totals follow the selected business range. Current RPM/TPM use the most recent five complete rolling minutes.
- Do not call Seedance, Seedream, Volcengine, or any other paid model while implementing or validating this feature.
- Add no frontend or backend dependency; use the existing VChart, Day.js, GORM, Testify, React Query, and Vitest packages.
- All new user-facing text uses `t()` and is present in English and Simplified Chinese locale files; run the existing i18n synchronizer for other locales.
- Each task below is committed separately with the listed commit message.

---

## File and Responsibility Map

### New backend files

- `model/model_usage_event.go`: fact model, idempotent create/finalize operations, filtered event reads, backfill-batch deletion, and enabled-channel availability lookup.
- `model/model_usage_event_test.go`: GORM behavior and unique-event regression coverage.
- `service/model_usage_recorder.go`: non-fatal live recording adapters for synchronous relays and asynchronous tasks.
- `service/model_usage_recorder_test.go`: request/task deduplication, final quota/Token, and failure recording tests.
- `service/model_usage_analytics.go`: selected-range aggregation, zero-filled buckets, RPM/TPM, weighted health, stuck detection, and percentiles.
- `service/model_usage_analytics_test.go`: deterministic pure aggregation and health tests.
- `service/model_usage_backfill.go`: dry-run, apply, reconciliation, and batch rollback logic using `tasks`, `logs`, and legacy `perf_metrics`.
- `service/model_usage_backfill_test.go`: idempotency and baseline reconstruction tests.
- `controller/model_usage_analytics.go`: strict query parsing and two admin responses.
- `controller/model_usage_analytics_test.go`: authorization-independent handler contract tests using Gin test contexts.
- `scripts/model-usage-backfill.go`: local command entry point; initializes the configured main/log databases and never initializes relay workers.

### New frontend files

- `web/src/features/dashboard/hooks/use-model-analytics.ts`: selected-range and health React Query hooks, 60/30 second refresh, and manual refresh.
- `web/src/features/dashboard/lib/model-analytics.ts`: range presets, Shanghai time formatting, short model names, chart records, and VChart specs.
- `web/src/features/dashboard/lib/__tests__/model-analytics.test.ts`: range, zero-fill, model-name, and chart-spec tests.
- `web/src/features/dashboard/components/models/model-analytics-section.tsx`: one data owner that renders toolbar, cards, health, consumption, and call charts.
- `web/src/features/dashboard/components/models/model-analytics-toolbar.tsx`: persistent quick ranges, current range/granularity, refresh, filter, and preferences controls.
- `web/src/features/dashboard/components/models/__tests__/model-analytics-toolbar.test.tsx`: keyboard and range-selection behavior.
- `web/src/features/dashboard/components/models/__tests__/model-analytics-section.test.tsx`: loading, error, empty, refresh, and filter integration behavior.
- `web/src/features/dashboard/components/models/__tests__/performance-overview.test.tsx`: current status and direct model-row behavior.

### Existing files modified

- `model/main.go`: AutoMigrate `ModelUsageEvent`.
- `service/quota.go`, `service/text_quota.go`, `controller/relay.go`: record one final synchronous event.
- `service/task_billing.go`, `service/task_polling.go`, `controller/relay.go`: record task submit/progress/final states.
- `router/api-router.go`: add admin-only analytics routes.
- `web/src/features/dashboard/types.ts`, `api.ts`, `constants.ts`, `lib/filters.ts`: API and filter contracts.
- `web/src/features/dashboard/index.tsx`: render one model analytics section instead of independent fetchers.
- `web/src/features/dashboard/components/models/log-stat-cards.tsx`: presentation-only selected totals plus current RPM/TPM.
- `web/src/features/dashboard/components/models/performance-overview.tsx`: direct per-model rows and weighted overall state.
- `web/src/features/dashboard/components/models/consumption-distribution-chart.tsx`, `model-charts.tsx`: consume new series and expose zoom/reset behavior.
- `web/src/features/dashboard/components/models/models-filter-dialog.tsx`, `models-chart-preferences.tsx`: model multi-select, new presets, automatic granularity, and retained custom range.
- `web/src/lib/dayjs.ts`: enable existing Day.js UTC/timezone plugins for `Asia/Shanghai` calculations.
- `web/src/i18n/locales/en.json`, `zh.json`, and synchronized locale files: dashboard text.
- `docs/runbooks/model-dashboard-analytics-acceptance-2026-09-01.md`: commands, database reconciliation, browser checks, screenshots, results, and rollback.

---

### Task 1: Add the canonical model usage fact table

**Files:**
- Create: `model/model_usage_event.go`
- Create: `model/model_usage_event_test.go`
- Modify: `model/main.go:329-365`

**Interfaces:**
- Produces: `model.ModelUsageEvent`, `model.ModelUsageQuery`, `model.CreateModelUsageEvent`, `model.FinalizeModelUsageEvent`, `model.TouchModelUsageEvent`, `model.ListModelUsageEvents`, and `model.DeleteModelUsageBackfillBatch`.
- Consumes: existing `model.DB`, GORM clauses, and `common.GetTimestamp()`.

- [ ] **Step 1: Write a failing model test for unique event identity and terminal updates**

```go
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
```

- [ ] **Step 2: Run the focused test and confirm the missing model fails**

Run: `go test ./model -run TestModelUsageEventCreateAndFinalizeIsIdempotent -count=1`

Expected: FAIL because `ModelUsageEvent` and its operations do not exist.

- [ ] **Step 3: Define the cross-database fact model and constants**

```go
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
    ID             int64  `json:"id" gorm:"primaryKey"`
    EventKey       string `json:"event_key" gorm:"size:191;not null;uniqueIndex:uk_model_usage_event_key"`
    Kind           string `json:"kind" gorm:"size:16;not null;index"`
    RequestID      string `json:"request_id" gorm:"size:64;index"`
    TaskID         string `json:"task_id" gorm:"size:191;index"`
    UserID         int    `json:"user_id" gorm:"index"`
    Username       string `json:"username" gorm:"size:64;index"`
    TokenID        int    `json:"token_id" gorm:"index"`
    ChannelID      int    `json:"channel_id" gorm:"index"`
    ModelName      string `json:"model_name" gorm:"size:128;not null;index:idx_model_usage_model_submitted,priority:1"`
    UseGroup       string `json:"use_group" gorm:"column:use_group;size:64;index"`
    Status         ModelUsageStatus `json:"status" gorm:"size:16;not null;index"`
    SubmittedAt    int64  `json:"submitted_at" gorm:"index:idx_model_usage_model_submitted,priority:2;index"`
    CompletedAt    int64  `json:"completed_at" gorm:"index"`
    LastProgressAt int64  `json:"last_progress_at" gorm:"index"`
    TotalTokens    int64  `json:"total_tokens"`
    OutputCount    int    `json:"output_count"`
    OutputUnit     string `json:"output_unit" gorm:"size:16"`
    FinalQuota     int    `json:"final_quota"`
    DurationMs     int64  `json:"duration_ms"`
    FailureReason  string `json:"failure_reason" gorm:"type:text"`
    Source         string `json:"source" gorm:"size:24;not null;index"`
    BackfillBatch  string `json:"backfill_batch" gorm:"size:64;index"`
    RecordedAt     int64  `json:"recorded_at"`
    UpdatedAt      int64  `json:"updated_at"`
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

func CreateModelUsageEvent(event *ModelUsageEvent) error
func FinalizeModelUsageEvent(eventKey string, final ModelUsageFinal) error
func TouchModelUsageEvent(eventKey string, status ModelUsageStatus, progressAt int64) error
func ListModelUsageEvents(query ModelUsageQuery) ([]ModelUsageEvent, error)
func DeleteModelUsageBackfillBatch(batch string) (int64, error)
```

Use `clause.OnConflict{DoNothing: true}` for create and a GORM `Updates(map[string]any{...})` for terminal updates. Reject negative Token/output/quota/duration values before persistence; allow only empty, `image`, or `video` as `output_unit`, and require an empty unit when the count is zero. `TouchModelUsageEvent` updates only `status`, `last_progress_at`, and `updated_at`, and must not turn a terminal event back into `running`.

- [ ] **Step 4: Add the model to `migrateDB`**

Add `&ModelUsageEvent{}` beside `&QuotaData{}` and `&PerfMetric{}` in `model/main.go`. Do not add dialect-specific column types, generated columns, or raw DDL.

- [ ] **Step 5: Run the model tests**

Run: `go test ./model -run 'TestModelUsageEvent' -count=1`

Expected: PASS with exactly one row after duplicate create/finalize calls.

- [ ] **Step 6: Commit the fact table**

```bash
git add model/model_usage_event.go model/model_usage_event_test.go model/main.go
git commit -m "feat: 新增模型调用事实表"
```

---

### Task 2: Record synchronous success and failure exactly once

**Files:**
- Create: `service/model_usage_recorder.go`
- Create: `service/model_usage_recorder_test.go`
- Modify: `service/quota.go:376-391`
- Modify: `service/text_quota.go:535-550`
- Modify: `controller/relay.go:251-263`

**Interfaces:**
- Consumes: `model.CreateModelUsageEvent`, `model.FinalizeModelUsageEvent`, `relaycommon.RelayInfo`, Gin request context, final settled quota, and normalized usage.
- Produces: `service.RecordSyncModelUsageStarted(c, relayInfo)` and `service.RecordSyncModelUsage(c, relayInfo, SyncModelUsageResult)`.

- [ ] **Step 1: Write failing tests for one success row and one post-retry failure row**

```go
func TestRecordSyncModelUsageUpsertsOneTerminalRequest(t *testing.T) {
    setupUsageRecorderDB(t)
    c, info := usageRecorderContext("req-sync-1")
    RecordSyncModelUsageStarted(c, info)
    RecordSyncModelUsage(c, info, SyncModelUsageResult{
        Success: true, TotalTokens: 7, FinalQuota: 123,
    })
    RecordSyncModelUsage(c, info, SyncModelUsageResult{
        Success: true, TotalTokens: 7, FinalQuota: 123,
    })

    var rows []model.ModelUsageEvent
    require.NoError(t, model.DB.Find(&rows).Error)
    require.Len(t, rows, 1)
    assert.Equal(t, "request:req-sync-1", rows[0].EventKey)
    assert.Equal(t, model.ModelUsageStatusSuccess, rows[0].Status)
}
```

Add a second case with `Success: false`, zero quota/Token, and a masked failure reason. Assert the started event is visible as running before finalization, `DurationMs` is derived from `relayInfo.StartTime`, and the event is still counted once after channel retries.

- [ ] **Step 2: Run the recorder tests and confirm they fail**

Run: `go test ./service -run TestRecordSyncModelUsage -count=1`

Expected: FAIL because the recorder does not exist.

- [ ] **Step 3: Implement a non-fatal recorder**

```go
type SyncModelUsageResult struct {
    Success       bool
    TotalTokens   int64
    OutputCount   int
    OutputUnit    string
    FinalQuota    int
    FailureReason string
}

func RecordSyncModelUsageStarted(c *gin.Context, info *relaycommon.RelayInfo)

func RecordSyncModelUsage(c *gin.Context, info *relaycommon.RelayInfo, result SyncModelUsageResult) {
    // Build "request:" + requestID, snapshot identity from c/info, insert if absent,
    // then finalize. Log DB errors with the request ID and return no error.
}
```

The started function inserts one running fact immediately before the first upstream attempt. The final function uses the request start timestamp as `submitted_at`, the recording time as `completed_at`, and `time.Since(info.StartTime).Milliseconds()` as duration. Truncate the stored failure reason to 512 UTF-8 characters after using the existing masked error string. Do not launch a goroutine with a live Gin context.

- [ ] **Step 4: Wire one running event immediately before the first upstream attempt**

In `controller/relay.go`, add a request-local `analyticsStarted` boolean around the channel retry loop. After channel selection and body validation, immediately before the first relay handler switch, call `RecordSyncModelUsageStarted` and set the boolean. Retries do not insert another event. Local validation/body failures that never enter an upstream handler do not call it.

- [ ] **Step 5: Wire successful image/audio/general quota settlement**

Immediately after `RecordConsumeLog` in `service/quota.go`, call the recorder with `usage.TotalTokens`, the final local `quota`, and `usage.GeneratedImages`. When the generated-image count is positive, store `output_unit="image"`; otherwise leave both output fields zero/empty. This path covers Seedream image generation and other non-text usage without inventing a unit when the upstream response reports none.

- [ ] **Step 6: Wire successful text settlement**

Immediately after `RecordConsumeLog` in `service/text_quota.go`, call the recorder with `summary.TotalTokens` and `summary.Quota`. This preserves prompt plus completion Token semantics for future language models.

- [ ] **Step 7: Wire one final relay failure after retries**

In `controller/relay.go`, after the retry loop determines `newAPIError != nil`, call the recorder once with zero quota/Token and `newAPIError.MaskSensitiveErrorWithStatusCode()`. Do not call it from `processChannelError`, because that function runs once per attempted channel.

- [ ] **Step 8: Run focused service/controller tests**

Run: `go test ./service ./controller -run 'TestRecordSyncModelUsage|Test.*Relay.*Failure' -count=1`

Expected: PASS; a retried client request creates one failed fact, not one row per channel.

- [ ] **Step 9: Commit synchronous recording**

```bash
git add service/model_usage_recorder.go service/model_usage_recorder_test.go service/quota.go service/text_quota.go controller/relay.go
git commit -m "feat: 记录同步模型调用事实"
```

---

### Task 3: Record asynchronous task submit, progress, and final settlement

**Files:**
- Modify: `service/model_usage_recorder.go`
- Modify: `service/model_usage_recorder_test.go`
- Modify: `service/task_billing.go:19-85`
- Modify: `service/task_polling.go:289-340,465-585`
- Modify: `controller/relay.go:704-767`

**Interfaces:**
- Produces: `RecordTaskUsageSubmitted`, `TouchTaskUsageProgress`, and `FinalizeTaskUsage`.
- Consumes: persisted `model.Task`, `relaycommon.TaskInfo`, and the final `task.Quota` after settlement/refund.

- [ ] **Step 1: Write failing task lifecycle tests**

```go
func TestTaskUsageLifecycleKeepsOneEventAndActualTokens(t *testing.T) {
    setupUsageRecorderDB(t)
    task := taskUsageFixture("task-1", model.TaskStatusSubmitted, 1_000)
    RecordTaskUsageSubmitted(testGinContext(), relayInfoFixture(), task)
    TouchTaskUsageProgress(task, model.TaskStatusInProgress, 1_700_000_010)
    task.Status = model.TaskStatusSuccess
    task.FinishTime = 1_700_000_050
    task.Quota = 900
    FinalizeTaskUsage(t.Context(), task, &relaycommon.TaskInfo{TotalTokens: 87_300})

    event := requireSingleTaskUsageEvent(t, "task:task-1")
    assert.Equal(t, int64(87_300), event.TotalTokens)
    assert.Equal(t, 900, event.FinalQuota)
    assert.Equal(t, model.ModelUsageStatusSuccess, event.Status)
}
```

Add a failure case in which `RefundTaskQuota` has set `task.Quota = 0`; assert final quota and tokens are zero and the failure reason is stored. Add a timeout-sweeper case so an expired task is finalized as failure after its refund instead of remaining analytically `running`.

- [ ] **Step 2: Run the task lifecycle tests and confirm they fail**

Run: `go test ./service -run 'TestTaskUsageLifecycle|TestTaskUsageFailure' -count=1`

Expected: FAIL because task recorder functions do not exist.

- [ ] **Step 3: Implement task recorder functions**

```go
func RecordTaskUsageSubmitted(c *gin.Context, info *relaycommon.RelayInfo, task *model.Task)
func TouchTaskUsageProgress(task *model.Task, status model.TaskStatus, progressAt int64)
func FinalizeTaskUsage(ctx context.Context, task *model.Task, result *relaycommon.TaskInfo)
```

`FinalizeTaskUsage` chooses `TotalTokens`, then `CompletionTokens`, then a positive numeric `UsageFacts["tokens"]`. A successful video task records `output_count=1` and `output_unit="video"`; failed/cancelled tasks record no output. It uses `FinishTime - SubmitTime` for duration and records `task.Quota` only after billing settlement/refund has finished. Each function logs and returns on analytics errors; none returns an error to the generation flow.

- [ ] **Step 4: Create the running event immediately after durable task submission**

In `controller/relay.go`, call `RecordTaskUsageSubmitted` immediately after `task.InsertWithContext` succeeds and before local billing settlement. At that point the upstream task and local task row both exist, so it is a real model call even if a later local settlement step fails. Do not call it before persistence, and do not place it in `LogTaskConsumption`, because that function is skipped when post-insert settlement returns an error.

- [ ] **Step 5: Finalize immediate terminal tasks**

After `service.LogTaskConsumption(c, relayInfo, task)` in `controller/relay.go`, if `result.Immediate` is terminal, call `FinalizeTaskUsage` directly. `LogTaskConsumption` has already persisted the quota settled by `service.SettleBilling`; do not call the task-polling settlement helper from the controller. Final quota must be the persisted `task.Quota`.

- [ ] **Step 6: Touch progress only when task state actually changes**

In both batch and single polling paths, compare `snap.Equal(task.Snapshot())`. After a winning CAS update, call `TouchTaskUsageProgress` only when status, progress, start time, finish time, failure reason, result URL, or data changed. A no-op poll must not refresh `last_progress_at`, otherwise a truly stuck task never reaches the 15-minute threshold.

- [ ] **Step 7: Finalize after settlement/refund in polling and timeout paths**

In batch and single polling, call `FinalizeTaskUsage` after `settleTaskBillingOnComplete` and the failure refund fallback. In `sweepTimedOutTasks`, call it after the winning status update and `RefundTaskQuota`. CAS losers do not finalize; the winner writes the terminal fact. Repeated polling remains safe because the fact key is `"task:" + task.TaskID`.

- [ ] **Step 8: Run task billing and polling regression tests**

Run: `go test ./service -run 'TestTaskUsage|TestLogTaskConsumption|Test.*TaskPolling|Test.*Task.*Quota' -count=1`

Expected: PASS; existing wallet/subscription/task quota assertions remain unchanged.

- [ ] **Step 9: Commit asynchronous recording**

```bash
git add service/model_usage_recorder.go service/model_usage_recorder_test.go service/task_billing.go service/task_polling.go controller/relay.go
git commit -m "feat: 记录异步任务调用事实"
```

---

### Task 4: Build selected-range aggregation and performance health

**Files:**
- Create: `service/model_usage_analytics.go`
- Create: `service/model_usage_analytics_test.go`
- Modify: `model/model_usage_event.go`
- Modify: `model/model_usage_event_test.go`

**Interfaces:**
- Consumes: filtered `[]model.ModelUsageEvent`, enabled `abilities`, the query clock, reporting location, and quota integers.
- Produces: `ModelAnalyticsResult`, `ModelHealthResult`, `QueryModelAnalytics`, and `QueryModelHealth`.

- [ ] **Step 1: Write failing selected-range aggregation tests**

```go
func TestAggregateModelUsageCountsUniqueCallsAndFillsRealRange(t *testing.T) {
    loc := mustShanghai(t)
    events := []model.ModelUsageEvent{
        usageEvent("request:a", "model-a", 1_700_000_100, 10, 100, model.ModelUsageStatusSuccess),
        usageEvent("task:b", "model-b", 1_700_007_300, 20, 200, model.ModelUsageStatusFailure),
    }
    result := AggregateModelUsage(events, AnalyticsOptions{
        Start: 1_699_999_200, End: 1_700_010_000,
        Granularity: GranularityHour, Location: loc,
    })
    assert.Equal(t, int64(2), result.Summary.TotalCalls)
    assert.Equal(t, int64(30), result.Summary.TotalTokens)
    assert.Equal(t, 300, result.Summary.TotalQuota)
    assertEveryExpectedHourPresent(t, result.Series)
}
```

Add cases proving that a final task is attributed to `submitted_at`, empty buckets are zero, requested models with no data are included, and the series sum equals the top summary.

- [ ] **Step 2: Write failing health and percentile tests**

```go
func TestBuildModelHealthUsesWeightedSuccessAndSuccessfulLatency(t *testing.T) {
    events := healthFixture(
        successful("seedream", 50_000),
        successful("seedream", 55_000),
        successful("seedream", 52_000),
        failed("seedream", 180_000),
        successful("seedance", 130_000),
    )
    result := BuildModelHealth(events, HealthOptions{Now: fixedNow, WindowHours: 24})
    assert.Equal(t, int64(4), result.Overall.SuccessCalls)
    assert.Equal(t, int64(5), result.Overall.TotalCalls)
    seedream := requireHealthModel(t, result, "seedream")
    assert.Equal(t, 75.0, seedream.SuccessRate)
    assert.Equal(t, int64(52_000), seedream.P50Ms)
    assert.Equal(t, int64(55_000), seedream.P95Ms)
}
```

Add cases for no calls, a 15-minute stuck task, three consecutive failures, zero enabled channels, P95 sample shortage below three successful rows, and current RPM/TPM using submit/completion timestamps respectively.

- [ ] **Step 3: Run the new service tests and confirm they fail**

Run: `go test ./service -run 'TestAggregateModelUsage|TestBuildModelHealth|TestCurrentLoad' -count=1`

Expected: FAIL because the analytics service does not exist.

- [ ] **Step 4: Define the response contracts**

```go
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

type AnalyticsRange struct {
    Start       int64                `json:"start"`
    End         int64                `json:"end"`
    Granularity AnalyticsGranularity `json:"granularity"`
    Timezone    string               `json:"timezone"`
}

type AnalyticsSummary struct {
    TotalCalls    int64   `json:"total_calls"`
    SuccessCalls  int64   `json:"success_calls"`
    FailureCalls  int64   `json:"failure_calls"`
    RunningCalls  int64   `json:"running_calls"`
    TotalQuota    int64   `json:"total_quota"`
    TotalTokens   int64   `json:"total_tokens"`
    PeakRPM       float64 `json:"peak_rpm"`
    PeakRPMAt     int64   `json:"peak_rpm_at"`
    PeakTPM       float64 `json:"peak_tpm"`
    PeakTPMAt     int64   `json:"peak_tpm_at"`
}

type AnalyticsBucket struct {
    BucketStart   int64            `json:"bucket_start"`
    BucketEnd     int64            `json:"bucket_end"`
    ModelName     string           `json:"model_name"`
    TotalCalls    int64            `json:"total_calls"`
    SuccessCalls  int64            `json:"success_calls"`
    FailureCalls  int64            `json:"failure_calls"`
    RunningCalls  int64            `json:"running_calls"`
    Quota         int64            `json:"quota"`
    Tokens        int64            `json:"tokens"`
    OutputCounts  map[string]int64 `json:"output_counts"`
}

type AnalyticsModelTotal struct {
    ModelName     string           `json:"model_name"`
    TotalCalls    int64            `json:"total_calls"`
    SuccessCalls  int64            `json:"success_calls"`
    FailureCalls  int64            `json:"failure_calls"`
    RunningCalls  int64            `json:"running_calls"`
    Quota         int64            `json:"quota"`
    Tokens        int64            `json:"tokens"`
    OutputCounts  map[string]int64 `json:"output_counts"`
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
    Status        ModelHealthStatus `json:"status"`
    TotalCalls    int64             `json:"total_calls"`
    SuccessCalls  int64             `json:"success_calls"`
    FailureCalls  int64             `json:"failure_calls"`
    RunningCalls  int64             `json:"running_calls"`
    StuckCalls    int64             `json:"stuck_calls"`
    SuccessRate   float64           `json:"success_rate"`
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
    WindowHours int               `json:"window_hours"`
    Overall     HealthOverall     `json:"overall"`
    CurrentLoad CurrentLoad       `json:"current_load"`
    Models      []ModelHealthRow  `json:"models"`
    UpdatedAt   int64             `json:"updated_at"`
}

func QueryModelAnalytics(ctx context.Context, query ModelAnalyticsQuery) (ModelAnalyticsResult, error)
func QueryModelHealth(ctx context.Context, hours int, now time.Time) (ModelHealthResult, error)
```

`AnalyticsSummary` includes total/success/failure/running calls, total quota, total Token, peak RPM/value time, and peak TPM/value time. `AnalyticsBucket` includes bucket start/end, model, calls by status, quota, Token, output count, and output unit. Aggregate output counts only within the same unit; never add image and video counts into one unlabeled number.

- [ ] **Step 5: Implement Shanghai bucket boundaries without database date functions**

Fetch filtered events with GORM, then group in Go. Hour/day boundaries use `time.Date` in `Asia/Shanghai`; week boundaries subtract `(weekday+6)%7` days to Monday. Generate all expected bucket starts from range start through range end, then emit one row per observed/requested model per bucket. This avoids SQLite/MySQL/PostgreSQL date-expression differences.

- [ ] **Step 6: Implement current load and peaks**

RPM is unique events with `submitted_at` in `[now-5m, now] / 5`. TPM is `total_tokens` for terminal events with `completed_at` in the same window divided by 5. Peak RPM buckets use submit time; peak TPM buckets use completion time. Round displayed rates to two decimal places only in the response, not while accumulating.

- [ ] **Step 7: Implement health rules exactly**

- Overall current status observes the last 15 minutes.
- `no_data`: no submitted/running events and no explicit unavailable model.
- `healthy`: completed calls exist with no failure/stuck/unavailable model.
- `warning`: at least one failure or stuck task, but no fault condition.
- `fault`: the same model's last three terminal calls all failed, or that model has no enabled ability/channel, or every currently enabled channel attempted for that model has a latest failed result with no later success.
- Overall success rate is `sum(success) / sum(total terminal calls)`; running calls do not enter the denominator.
- Model P50/P95 use successful positive `duration_ms` values and nearest-rank `ceil(p*n)-1`; P95 is unavailable when `n < 3`.
- Stuck means nonterminal and `last_progress_at <= now-15m`.

- [ ] **Step 8: Run analytics tests**

Run: `go test ./service -run 'TestAggregateModelUsage|TestBuildModelHealth|TestCurrentLoad' -count=1`

Expected: PASS with weighted `4/5`, Seedream `3/4`, correct P50/P95, and complete time buckets.

- [ ] **Step 9: Commit aggregation and health**

```bash
git add model/model_usage_event.go model/model_usage_event_test.go service/model_usage_analytics.go service/model_usage_analytics_test.go
git commit -m "feat: 聚合模型用量与性能健康"
```

---

### Task 5: Expose strict admin-only analytics APIs

**Files:**
- Create: `controller/model_usage_analytics.go`
- Create: `controller/model_usage_analytics_test.go`
- Modify: `router/api-router.go:316-321`

**Interfaces:**
- Consumes: `service.QueryModelAnalytics`, `service.QueryModelHealth`.
- Produces: `GET /api/data/model-analytics` and `GET /api/data/model-analytics/health`.

- [ ] **Step 1: Write failing handler contract tests**

```go
func TestGetModelUsageAnalyticsRejectsInvalidRange(t *testing.T) {
    r := gin.New()
    r.GET("/analytics", GetModelUsageAnalytics)
    req := httptest.NewRequest(http.MethodGet, "/analytics?start_timestamp=20&end_timestamp=10", nil)
    rec := httptest.NewRecorder()
    r.ServeHTTP(rec, req)
    assert.Equal(t, http.StatusBadRequest, rec.Code)
}
```

Add a valid response test asserting `range`, `summary`, `series`, `models`, `available_models`, and `updated_at`. Add health tests for allowed `hours=1|24|168` and rejection of other values.

- [ ] **Step 2: Run the controller tests and confirm they fail**

Run: `go test ./controller -run 'TestGetModelUsageAnalytics|TestGetModelUsageHealth' -count=1`

Expected: FAIL because handlers are missing.

- [ ] **Step 3: Implement strict query parsing**

Selected-range parameters:

```text
start_timestamp: required positive Unix seconds
end_timestamp: required, greater than or equal to start
granularity: hour | day | week; omitted means server auto-selection
username: optional exact username
models: optional comma-separated exact model IDs
```

Auto-selection is hour for ranges <= 48 hours, day for ranges <= 90 days, and week above 90 days. The response echoes the effective granularity and `Asia/Shanghai` timezone. Health accepts only `1`, `24`, or `168` hours.

- [ ] **Step 4: Register admin-only routes**

```go
dataRoute.GET("/model-analytics", middleware.AdminAuth(), controller.GetModelUsageAnalytics)
dataRoute.GET("/model-analytics/health", middleware.AdminAuth(), controller.GetModelUsageHealth)
```

Keep the existing `/api/perf-metrics` public/pricing routes unchanged because the model square still uses them.

- [ ] **Step 5: Run controller/router regression tests**

Run: `go test ./controller ./router -run 'TestGetModelUsage|Test.*DataRoute|Test.*Auth' -count=1`

Expected: PASS; unauthenticated and ordinary-user requests are rejected by the route middleware.

- [ ] **Step 6: Commit the APIs**

```bash
git add controller/model_usage_analytics.go controller/model_usage_analytics_test.go router/api-router.go
git commit -m "feat: 提供模型数据看板统计接口"
```

---

### Task 6: Add idempotent local history backfill and rollback

**Files:**
- Create: `service/model_usage_backfill.go`
- Create: `service/model_usage_backfill_test.go`
- Create: `scripts/model-usage-backfill.go`

**Interfaces:**
- Consumes: main DB `tasks` and `perf_metrics`, log DB `logs`, and `model.CreateModelUsageEvent`.
- Produces: `BackfillModelUsageEvents(ctx, BackfillOptions) (BackfillReport, error)`, `RollbackModelUsageBatch`, and the local CLI.

```go
type BackfillOptions struct {
    DryRun bool
    Cutoff int64
    Batch  string
}

type BackfillReport struct {
    Batch            string `json:"batch"`
    DryRun           bool   `json:"dry_run"`
    WouldInsert      int    `json:"would_insert"`
    Inserted         int    `json:"inserted"`
    Skipped          int    `json:"skipped"`
    Rejected         int    `json:"rejected"`
    TotalCalls       int64  `json:"total_calls"`
    TotalQuota       int64  `json:"total_quota"`
    TotalTokens      int64  `json:"total_tokens"`
}

func BackfillModelUsageEvents(ctx context.Context, options BackfillOptions) (BackfillReport, error)
func RollbackModelUsageBatch(ctx context.Context, batch string) (int64, error)
```

- [ ] **Step 1: Write failing dry-run and idempotency tests**

```go
func TestBackfillModelUsageEventsDryRunAndApplyAreIdempotent(t *testing.T) {
    mainDB, logDB := setupBackfillDatabases(t)
    seedBackfillTaskAndLogs(t, mainDB, logDB)
    opts := BackfillOptions{DryRun: true, Cutoff: 1_800_000_000, Batch: "baseline"}
    report, err := BackfillModelUsageEvents(t.Context(), opts)
    require.NoError(t, err)
    assert.Equal(t, 2, report.WouldInsert)
    assert.Zero(t, countUsageEvents(t, mainDB))

    opts.DryRun = false
    _, err = BackfillModelUsageEvents(t.Context(), opts)
    require.NoError(t, err)
    _, err = BackfillModelUsageEvents(t.Context(), opts)
    require.NoError(t, err)
    assert.Equal(t, 2, countUsageEvents(t, mainDB))
}
```

- [ ] **Step 2: Run backfill tests and confirm they fail**

Run: `go test ./service -run TestBackfillModelUsageEvents -count=1`

Expected: FAIL because backfill types/functions do not exist.

- [ ] **Step 3: Implement task reconstruction**

For each task submitted at or before the cutoff:

- key `"task:" + task.TaskID`;
- identity/status/time/final quota from task columns;
- Token lookup order: persisted task response JSON paths `usage.total_tokens`, `usage.completion_tokens`, `data.usage.*`, `response.usage.*`, then persisted tiered usage fact `tokens`;
- successful video output: `output_count=1`, `output_unit="video"`; terminal failures/cancellations have no output;
- successful duration from `finish_time - submit_time`;
- source `backfill_task`, batch from CLI.

All JSON path conversions reject non-finite, negative, and overflowing values.

- [ ] **Step 4: Implement synchronous log reconstruction**

Group logs by non-empty `request_id` and exclude any group whose `other` contains `is_task=true` or `task_id`. A group with a consume log is success; a group with only error logs is failure. Derive each candidate submit time as `created_at - use_time`, bounded so it never exceeds `created_at`, then use the earliest candidate in the request group. Sum consume quota minus refund quota, clamp the final result at zero, and sum prompt plus completion Tokens only once from the final consume record. Parse a positive integer `other.generated_images` from the final consume record as `output_count` with `output_unit="image"`; missing, invalid, or zero values remain empty. Use the maximum `use_time` in seconds for duration.

- [ ] **Step 5: Reconcile missing legacy performance counts deterministically**

For each legacy `perf_metrics` model/hour bucket, compare request/success counts against reconstructed synchronous events. For positive gaps, insert zero-quota, zero-Token synthetic facts keyed with `fmt.Sprintf("legacy-perf:%s:%d:%s:%d", modelName, bucket, status, ordinal)`, with source `backfill_perf`. These rows preserve historical call/success totals but have `duration_ms=0` and are excluded from P50/P95. A negative gap is reported as an error and blocks apply because it indicates double counting.

- [ ] **Step 6: Implement dry-run, batch apply, and explicit rollback**

Dry-run performs all reads/reconciliation and writes zero facts. Apply inserts only missing event keys and returns inserted/skipped/rejected counts plus aggregate calls/quota/Token. Rollback deletes only rows whose exact `backfill_batch` matches and requires both `--rollback-batch model-dashboard-20260901` and `--confirm-rollback`.

- [ ] **Step 7: Add the local command**

Supported commands:

```powershell
go run ./scripts/model-usage-backfill.go --dry-run --cutoff 1788245040 --batch model-dashboard-20260901
go run ./scripts/model-usage-backfill.go --cutoff 1788245040 --batch model-dashboard-20260901
go run ./scripts/model-usage-backfill.go --rollback-batch model-dashboard-20260901 --confirm-rollback
```

The command loads `.env`, calls `common.InitEnv()`, initializes logger/main/log databases, writes a JSON report to stdout, and exits. It does not initialize Redis, HTTP clients, task pollers, plugins, or performance flush loops.

- [ ] **Step 8: Run backfill tests twice**

Run: `go test ./service -run 'TestBackfillModelUsageEvents|TestRollbackModelUsageBatch' -count=2`

Expected: PASS in both repetitions; apply number two inserts zero rows.

- [ ] **Step 9: Commit backfill tooling**

```bash
git add service/model_usage_backfill.go service/model_usage_backfill_test.go scripts/model-usage-backfill.go
git commit -m "feat: 增加模型用量历史回填工具"
```

---

### Task 7: Add frontend API types, Shanghai ranges, and query ownership

**Files:**
- Modify: `web/src/features/dashboard/types.ts`
- Modify: `web/src/features/dashboard/api.ts`
- Modify: `web/src/features/dashboard/constants.ts`
- Modify: `web/src/features/dashboard/lib/filters.ts`
- Modify: `web/src/lib/dayjs.ts`
- Create: `web/src/features/dashboard/hooks/use-model-analytics.ts`
- Create: `web/src/features/dashboard/lib/model-analytics.ts`
- Create: `web/src/features/dashboard/lib/__tests__/model-analytics.test.ts`

**Interfaces:**
- Consumes: backend analytics/health JSON.
- Produces: `ModelAnalyticsData`, `ModelHealthData`, `getModelAnalytics`, `getModelHealth`, `useModelAnalytics`, `createRangePreset`, `formatShanghaiRange`, and `shortModelName`.

- [ ] **Step 1: Write failing pure frontend tests**

```ts
it('uses Shanghai midnight for Today and keeps 24 hours rolling', () => {
  const now = new Date('2026-09-01T08:00:00Z')
  expect(createRangePreset('today', now)).toEqual({
    start: 1788192000,
    end: 1788249600,
    granularity: 'hour',
  })
  expect(createRangePreset('24h', now).end - createRangePreset('24h', now).start).toBe(86400)
})

it('formats known Ark models and falls back to the full ID', () => {
  expect(shortModelName('doubao-seedance-2-5-260628')).toBe('Seedance 2.5')
  expect(shortModelName('future-model-x')).toBe('future-model-x')
})
```

- [ ] **Step 2: Run the frontend unit test and confirm it fails**

Run: `cd web && bun run test -- src/features/dashboard/lib/__tests__/model-analytics.test.ts`

Expected: FAIL because the module and contracts do not exist.

- [ ] **Step 3: Add response/filter types**

Add exact TypeScript mirrors of both backend responses:

```ts
export type AnalyticsGranularity = "hour" | "day" | "week";
export type ModelHealthStatus = "no_data" | "healthy" | "warning" | "fault";

export interface ModelAnalyticsQuery {
  start_timestamp: number;
  end_timestamp: number;
  granularity?: AnalyticsGranularity;
  username?: string;
  models?: string[];
}

export interface AnalyticsRange {
  start: number;
  end: number;
  granularity: AnalyticsGranularity;
  timezone: "Asia/Shanghai";
}

export interface AnalyticsSummary {
  total_calls: number;
  success_calls: number;
  failure_calls: number;
  running_calls: number;
  total_quota: number;
  total_tokens: number;
  peak_rpm: number;
  peak_rpm_at: number;
  peak_tpm: number;
  peak_tpm_at: number;
}

export interface AnalyticsBucket {
  bucket_start: number;
  bucket_end: number;
  model_name: string;
  total_calls: number;
  success_calls: number;
  failure_calls: number;
  running_calls: number;
  quota: number;
  tokens: number;
  output_counts: Record<string, number>;
}

export interface AnalyticsModelTotal {
  model_name: string;
  total_calls: number;
  success_calls: number;
  failure_calls: number;
  running_calls: number;
  quota: number;
  tokens: number;
  output_counts: Record<string, number>;
}

export interface ModelAnalyticsData {
  range: AnalyticsRange;
  summary: AnalyticsSummary;
  series: AnalyticsBucket[];
  models: AnalyticsModelTotal[];
  available_models: string[];
  updated_at: number;
}

export interface ModelHealthRow {
  model_name: string;
  status: ModelHealthStatus;
  total_calls: number;
  success_calls: number;
  failure_calls: number;
  running_calls: number;
  stuck_calls: number;
  success_rate: number;
  p50_ms: number | null;
  p95_ms: number | null;
  successful_duration_samples: number;
  latest_failure_at: number;
  latest_failure_reason: string;
}

export interface ModelHealthData {
  window_hours: 1 | 24 | 168;
  overall: {
    status: ModelHealthStatus;
    total_calls: number;
    success_calls: number;
    failure_calls: number;
    running_calls: number;
    stuck_calls: number;
    success_rate: number;
  };
  current_load: { rpm: number; tpm: number; window_minutes: 5 };
  models: ModelHealthRow[];
  updated_at: number;
}
```

Extend `DashboardFilters` with `models?: string[]` and preserve existing username/date/granularity fields for the flow dashboard. Replace `29 Days` with `30 Days`; set the model analytics default to seven days.

- [ ] **Step 4: Enable Day.js timezone support**

Extend the existing singleton with `utc` and `timezone`, then use `dayjs.tz(..., 'Asia/Shanghai')` only in model analytics helpers. Existing consumers of `@/lib/dayjs` keep their current behavior.

- [ ] **Step 5: Add API functions and React Query hooks**

```ts
export function getModelAnalytics(params: ModelAnalyticsQuery): Promise<ModelAnalyticsData>
export function getModelHealth(hours: 1 | 24 | 168): Promise<ModelHealthData>

export function useModelAnalytics(filters: DashboardFilters, enabled: boolean) {
  // selected query: refetchInterval 60_000
  // health query: refetchInterval 30_000
  // return refreshAll() that refetches both promises
}
```

Use query keys containing normalized Unix timestamps, granularity, username, and sorted model IDs. Keep prior data visible only while a new valid filter query loads; on query error render the explicit error state and mark the last update stale.

- [ ] **Step 6: Run frontend pure tests and typecheck**

Run: `cd web && bun run test -- src/features/dashboard/lib/__tests__/model-analytics.test.ts && bun run typecheck`

Expected: PASS.

- [ ] **Step 7: Commit frontend data contracts**

```bash
git add web/src/features/dashboard/types.ts web/src/features/dashboard/api.ts web/src/features/dashboard/constants.ts web/src/features/dashboard/lib/filters.ts web/src/features/dashboard/hooks/use-model-analytics.ts web/src/features/dashboard/lib/model-analytics.ts web/src/features/dashboard/lib/__tests__/model-analytics.test.ts web/src/lib/dayjs.ts
git commit -m "feat: 增加模型看板前端数据契约"
```

---

### Task 8: Build the persistent time toolbar and filters

**Files:**
- Create: `web/src/features/dashboard/components/models/model-analytics-toolbar.tsx`
- Create: `web/src/features/dashboard/components/models/__tests__/model-analytics-toolbar.test.tsx`
- Modify: `web/src/features/dashboard/components/models/models-filter-dialog.tsx`
- Modify: `web/src/features/dashboard/components/models/models-chart-preferences.tsx`
- Modify: `web/src/features/dashboard/constants.ts`
- Modify: `web/src/features/dashboard/lib/filters.ts`

**Interfaces:**
- Consumes: current `DashboardFilters`, available model IDs, preference values, and `refreshAll`.
- Produces: exact selected range changes and retained custom/model filters.

- [ ] **Step 1: Write failing toolbar behavior tests**

```tsx
it('shows the active range and chooses day aggregation for seven days', async () => {
  const user = userEvent.setup()
  render(<ModelAnalyticsToolbar {...toolbarProps()} />)
  await user.click(screen.getByRole('button', { name: /7 days/i }))
  expect(toolbarProps().onFiltersChange).toHaveBeenCalledWith(
    expect.objectContaining({ time_granularity: 'day' })
  )
})
```

Add assertions for Today, 24 hours, 30 days, custom-range retention after dialog reopen, model multi-select, manual refresh, keyboard focus, and accessible pressed state.

- [ ] **Step 2: Run the toolbar test and confirm it fails**

Run: `cd web && bun run test -- src/features/dashboard/components/models/__tests__/model-analytics-toolbar.test.tsx`

Expected: FAIL because the toolbar does not exist.

- [ ] **Step 3: Implement persistent quick ranges and current-range text**

Render `[Today] [24 Hours] [7 Days] [30 Days] [Custom]` as a keyboard-accessible single-selection group. Display a concrete label such as `Current range: 2026-08-26 14:00 – 2026-09-02 14:00 · Day` at all times. Quick selections call the Shanghai range helper and apply hour/hour/day/day respectively.

- [ ] **Step 4: Add model multi-select to the existing filter dialog**

Use the existing `MultiSelect` component with short display labels and full IDs in secondary text. Keep username exact matching. Reopening the dialog copies the currently applied custom dates and selected models; Reset restores saved defaults and clears username/model filters.

- [ ] **Step 5: Update preferences without adding new settings**

Keep default consumption chart, call chart, range, and granularity only. Change the 29-day option to 30 days and set a fresh installation's default to 7 days/day. Preserve existing stored preferences by mapping legacy `29` to `30` and accepting legacy `1`, `7`, and `14` values until the next save.

- [ ] **Step 6: Run toolbar tests and typecheck**

Run: `cd web && bun run test -- src/features/dashboard/components/models/__tests__/model-analytics-toolbar.test.tsx && bun run typecheck`

Expected: PASS with no inaccessible unlabeled controls.

- [ ] **Step 7: Commit toolbar and filtering**

```bash
git add web/src/features/dashboard/components/models/model-analytics-toolbar.tsx web/src/features/dashboard/components/models/__tests__/model-analytics-toolbar.test.tsx web/src/features/dashboard/components/models/models-filter-dialog.tsx web/src/features/dashboard/components/models/models-chart-preferences.tsx web/src/features/dashboard/constants.ts web/src/features/dashboard/lib/filters.ts
git commit -m "feat: 优化模型看板时间筛选"
```

---

### Task 9: Replace top cards and performance health with unified facts

**Files:**
- Create: `web/src/features/dashboard/components/models/model-analytics-section.tsx`
- Create: `web/src/features/dashboard/components/models/__tests__/model-analytics-section.test.tsx`
- Create: `web/src/features/dashboard/components/models/__tests__/performance-overview.test.tsx`
- Modify: `web/src/features/dashboard/components/models/log-stat-cards.tsx`
- Modify: `web/src/features/dashboard/components/models/performance-overview.tsx`
- Modify: `web/src/features/dashboard/index.tsx`

**Interfaces:**
- Consumes: `useModelAnalytics` results and toolbar filters.
- Produces: five cards and direct per-model health rows with one shared refresh lifecycle.

- [ ] **Step 1: Write failing section and health component tests**

```tsx
it('renders selected totals and fixed five-minute load from separate fields', () => {
  render(<LogStatCards analytics={analyticsFixture()} health={healthFixture()} />)
  expect(screen.getByText('20')).toBeVisible()
  expect(screen.getByText('¥253.18')).toBeVisible()
  expect(screen.getByText('4,226,311')).toBeVisible()
  expect(screen.getByText('Current RPM')).toBeVisible()
  expect(screen.getByText('Current TPM')).toBeVisible()
})
```

Add a health test that directly renders `Seedream 5.0 Pro`, `Success 3/4 · 75%`, `P50 50s`, and `P95 55s`, without sync/async headings or throughput.

- [ ] **Step 2: Run the component tests and confirm they fail**

Run: `cd web && bun run test -- src/features/dashboard/components/models/__tests__/model-analytics-section.test.tsx src/features/dashboard/components/models/__tests__/performance-overview.test.tsx`

Expected: FAIL because components still use quota/perf-metric contracts.

- [ ] **Step 3: Make stat cards presentation-only**

Remove the internal `/api/data` fetch and time-range division. Render:

- Total calls, with success/failure/running secondary text.
- Total consumption using existing `formatQuota` and “Selected range”.
- Total Token using the exact selected-range total; its tooltip lists Token by model, shows the full Model ID, and provides a copy action.
- Current RPM and current TPM from `health.current_load`, with “Last 5 minutes”.
- Peak values/timestamps in tooltips from selected-range summary.

Loading uses skeletons. Error uses `--` plus a visible retry action. Exact values remain available via title/tooltip when compact formatting is used.

- [ ] **Step 4: Rewrite performance health as direct model rows**

The section header contains `1h / 24h / 7d`, the 15-minute current-state badge, weighted success/total, failure, running, stuck, and last update. Each model row renders short name, status, success ratio, P50, P95 or sample shortage, running count when nonzero, and latest failure. When the successful duration sample count is below 20, P50/P95 expose a tooltip saying “Small sample, for reference only”. Remove average latency, unified throughput, and model badges.

- [ ] **Step 5: Add log navigation**

Clicking a model row navigates to `/usage-logs/common` with `model`, `start_timestamp`, and `end_timestamp` search parameters for the selected health window. Preserve the full model ID in the URL.

- [ ] **Step 6: Centralize query ownership in `ModelAnalyticsSection`**

This component owns `useModelAnalytics`, toolbar refresh, and child props. Update `dashboard/index.tsx` so the models section renders this single component. Remove `modelData`, `dataLoading`, and the old `onDataUpdate` plumbing; keep flow/user dashboard state unchanged.

- [ ] **Step 7: Run component tests and typecheck**

Run: `cd web && bun run test -- src/features/dashboard/components/models/__tests__/model-analytics-section.test.tsx src/features/dashboard/components/models/__tests__/performance-overview.test.tsx && bun run typecheck`

Expected: PASS; the exact user-approved Seedream row is visible.

- [ ] **Step 8: Commit cards and health**

```bash
git add web/src/features/dashboard/components/models/model-analytics-section.tsx web/src/features/dashboard/components/models/__tests__/model-analytics-section.test.tsx web/src/features/dashboard/components/models/__tests__/performance-overview.test.tsx web/src/features/dashboard/components/models/log-stat-cards.tsx web/src/features/dashboard/components/models/performance-overview.tsx web/src/features/dashboard/index.tsx
git commit -m "feat: 重构模型看板统计与健康状态"
```

---

### Task 10: Rebuild timelines with real buckets and zoom interaction

**Files:**
- Modify: `web/src/features/dashboard/lib/model-analytics.ts`
- Modify: `web/src/features/dashboard/lib/__tests__/model-analytics.test.ts`
- Modify: `web/src/features/dashboard/components/models/consumption-distribution-chart.tsx`
- Modify: `web/src/features/dashboard/components/models/model-charts.tsx`
- Modify: `web/src/features/dashboard/constants.ts`
- Modify: `web/src/features/dashboard/lib/charts.ts`

**Interfaces:**
- Consumes: backend zero-filled `AnalyticsBucket[]` and range metadata.
- Produces: VChart bar/area/call trend/proportion/ranking specs with real timestamps, shared legend names, tooltips, and view-only zoom.

- [ ] **Step 1: Extend failing chart tests for the original seven-point bug**

```ts
it('keeps all real buckets when only five contain data', () => {
  const specs = buildModelAnalyticsCharts(fiveRealBucketsFixture(), chartOptions())
  expect(specs.consumption.data[0].values.map((row) => row.bucketStart)).toEqual(
    fiveRealBucketsFixture().map((row) => row.bucket_start)
  )
})

it('adds data zoom without changing summary totals', () => {
  const specs = buildModelAnalyticsCharts(longRangeFixture(), chartOptions())
  expect(specs.consumption.dataZoom).toBeDefined()
  expect(specs.summary.totalQuota).toBe(longRangeFixtureTotalQuota())
})
```

- [ ] **Step 2: Run chart tests and confirm the first test fails against the old padding logic**

Run: `cd web && bun run test -- src/features/dashboard/lib/__tests__/model-analytics.test.ts`

Expected: FAIL while `MAX_CHART_TREND_POINTS`/`fillTimePoints` still replace real points.

- [ ] **Step 3: Build specs from Unix bucket starts, not formatted labels**

Use numeric `bucket_start` as the x field and format labels/tooltips in Shanghai time. Delete `MAX_CHART_TREND_POINTS` and the model-page `fillTimePoints` path. The backend already emits empty buckets, so the frontend must not synthesize or relocate timestamps.

- [ ] **Step 4: Add VChart zoom and reset behavior without changing data totals**

For hour/day/week series longer than 12 points, enable a bottom `dataZoom` slider spanning the full domain and VChart wheel/drag roam on the x dimension. Register double-click to reset the dataZoom range to 0–100%. Add keyboard-accessible “Zoom in”, “Zoom out”, and “Reset zoom” buttons that update the same visible-domain state. Zoom state filters only the chart viewport; headers always render server summary totals.

- [ ] **Step 5: Improve tooltips and legends**

Consumption tooltip: exact time, short name, copyable full ID, calls, CNY consumption via existing currency config, bucket share, Token, and a labeled image/video output count when positive. When one model has no recorded Token, show “No Token data” instead of hiding or failing the tooltip. Call tooltip: exact time, model, total/success/failure. Legend clicks retain VChart's series visibility behavior. Unknown model IDs display unchanged. Use VChart's click/mark interaction so the same detail remains available by tap on mobile.

- [ ] **Step 6: Run chart tests and component tests**

Run: `cd web && bun run test -- src/features/dashboard/lib/__tests__/model-analytics.test.ts src/features/dashboard/components/models/__tests__/model-analytics-section.test.tsx`

Expected: PASS; no real timestamp is dropped and zoom does not alter card totals.

- [ ] **Step 7: Commit chart repair**

```bash
git add web/src/features/dashboard/lib/model-analytics.ts web/src/features/dashboard/lib/__tests__/model-analytics.test.ts web/src/features/dashboard/components/models/consumption-distribution-chart.tsx web/src/features/dashboard/components/models/model-charts.tsx web/src/features/dashboard/constants.ts web/src/features/dashboard/lib/charts.ts
git commit -m "fix: 修复模型看板时间轴与缩放"
```

---

### Task 11: Complete i18n, errors, accessibility, and frontend verification

**Files:**
- Modify: `web/src/i18n/locales/en.json`
- Modify: `web/src/i18n/locales/zh.json`
- Modify: synchronized files under `web/src/i18n/locales/`
- Modify: model dashboard component tests from Tasks 8–10

**Interfaces:**
- Consumes: all new translation keys and UI states.
- Produces: localized, keyboard-operable final model dashboard.

- [ ] **Step 1: Add English and Simplified Chinese strings**

Cover range names, current range/granularity, total consumption, current RPM/TPM, selected range, last five minutes, last updated, no calls, healthy/warning/fault, running/stuck, sample shortage, small-sample warning, no-Token-data, image/video units, zoom controls, error/retry, model filters, and tooltip labels.

- [ ] **Step 2: Synchronize other locale keys**

Run: `cd web && bun run i18n:sync`

Expected: every locale contains every new key; untranslated locales use the project's normal English fallback generated by the synchronizer.

- [ ] **Step 3: Extend tests for empty, error, stale, and keyboard states**

Assert zero cards plus “no calls” for empty responses, retry button on API failure, gray no-data health status, visible last-update time, keyboard-operable quick ranges/zoom/model rows, tap-accessible chart detail, and matching `aria-pressed`/status text. Do not assert complete Tailwind class strings.

- [ ] **Step 4: Run all affected frontend tests**

Run: `cd web && bun run test -- src/features/dashboard/lib/__tests__/model-analytics.test.ts src/features/dashboard/components/models/__tests__`

Expected: PASS.

- [ ] **Step 5: Run frontend quality gates**

Run:

```powershell
cd web
bun run typecheck
bun run lint
bun run format:check
bun run build
```

Expected: all commands exit 0. Fix only issues introduced or exposed in files changed by this feature.

- [ ] **Step 6: Commit localization and UI hardening**

```bash
git add web/src/i18n/locales web/src/features/dashboard
git commit -m "feat: 完善模型看板交互与多语言"
```

---

### Task 12: Verify migrations on all supported databases

**Files:**
- Create: `model/model_usage_event_migration_test.go`
- Create: `docs/runbooks/model-dashboard-analytics-acceptance-2026-09-01.md`

**Interfaces:**
- Consumes: the finished GORM model/migration and test databases.
- Produces: fresh/upgrade/idempotency evidence for SQLite, MySQL 5.7.44, and PostgreSQL 9.6.24.

- [ ] **Step 1: Add an opt-in migration integration test**

The test reads `MODEL_USAGE_MIGRATION_DSN` and `MODEL_USAGE_MIGRATION_DIALECT`, initializes a main database created by the previous commit, runs current `AutoMigrate` twice, inserts/finalizes the same event twice, and asserts one fact row plus preserved existing `tasks` and `quota_data` rows. Skip only when the two environment variables are absent. `logs` may live in a separate `LOG_DB`; record its before/after row count in the acceptance runbook instead of migrating or mutating it in this test.

- [ ] **Step 2: Verify fresh SQLite and upgrade SQLite**

Run the migration test once against a new temporary SQLite file and once against a file initialized from commit `09b40436`. Run current migration twice in each case. Record commands, database file hashes, and results in the runbook.

- [ ] **Step 3: Verify fresh and upgrade MySQL 5.7.44**

Start an isolated `mysql:5.7.44` container with an explicitly named test database. Initialize the upgrade database using commit `09b40436`, then run the current migration test twice. Confirm the unique index, `TEXT` failure reason, bigint timestamps/Token, and preserved source rows.

- [ ] **Step 4: Verify fresh and upgrade PostgreSQL 9.6.24**

Repeat the same process with `postgres:9.6.24`. Confirm no MySQL quoting or SQLite-only DDL appears and both migration passes are no-ops after the first.

- [ ] **Step 5: Run backend regression tests**

Run:

```powershell
go test ./model ./service ./controller ./router -count=1
go test ./... -count=1
```

Expected: PASS. If the full suite has a pre-existing unrelated failure, record its exact package/test/output and still require every changed package to pass.

- [ ] **Step 6: Commit database verification coverage**

```bash
git add model/model_usage_event_migration_test.go docs/runbooks/model-dashboard-analytics-acceptance-2026-09-01.md
git commit -m "test: 验证模型看板三数据库迁移"
```

---

### Task 13: Backfill the local baseline and perform browser acceptance

**Files:**
- Modify: `docs/runbooks/model-dashboard-analytics-acceptance-2026-09-01.md`
- Create: screenshots/results under `artifacts/model-dashboard-analytics/` (kept out of Git if the repository ignores runtime artifacts).

**Interfaces:**
- Consumes: local PostgreSQL, existing historical tasks/logs/perf metrics, the completed frontend/backend, and PRD baseline values.
- Produces: reproducible local acceptance evidence without paid API calls.

- [ ] **Step 1: Capture the immutable cutoff and database backup**

Use the PRD evidence capture time, 2026-09-01 14:44:00 Asia/Shanghai, as the immutable baseline cutoff. Run the following first, then record the database/container identity, application commit, row counts, and backup file path. Use the verified local PostgreSQL backup procedure from the existing runbooks before applying backfill.

```powershell
$analyticsCutoff = 1788245040
$analyticsStart = 1788105600
New-Item -ItemType Directory -Force -Path artifacts/model-dashboard-analytics | Out-Null
Set-Content -LiteralPath artifacts/model-dashboard-analytics/baseline-cutoff.txt -Value $analyticsCutoff
```

- [ ] **Step 2: Run dry-run and inspect the reconciliation report**

Run:

```powershell
go run ./scripts/model-usage-backfill.go --dry-run --cutoff $analyticsCutoff --batch model-dashboard-20260901
```

Expected for the PRD baseline slice: 20 calls, 19 successes, 1 failure, 4,226,311 Token, and final quota converting to ¥253.18 within ¥0.05. The runbook records the exact value stored in `baseline-cutoff.txt`; any calls submitted after that value are excluded.

- [ ] **Step 3: Apply backfill and immediately rerun it**

Run the same command without `--dry-run`, then run it a second time. The second report must show zero inserts and unchanged totals.

- [ ] **Step 4: Verify API responses directly**

With an authenticated administrator session, save JSON responses for:

```text
/api/data/model-analytics?start_timestamp=$analyticsStart&end_timestamp=$analyticsCutoff&granularity=hour
/api/data/model-analytics/health?hours=24
```

Confirm card totals equal series totals and that full virtual/upstream keys are absent.

- [ ] **Step 5: Verify the dashboard in the browser**

Check Today, 24 hours, 7 days, 30 days, and a custom range. Confirm August 31 data remains visible when included, all empty buckets are zero, zoom/pan/reset work, legend toggles work, and cards do not change while zooming. Capture desktop and narrow/mobile screenshots.

- [ ] **Step 6: Verify the user-approved health layout**

Confirm there are no sync/async sections and no unified throughput. Verify model rows use short names and the baseline Seedream row is `Success 3/4 · 75%` with computed P50/P95. Verify Seedance 2.5 and 2.0 also appear.

- [ ] **Step 7: Document rollback without executing it**

Record the exact command:

```powershell
go run ./scripts/model-usage-backfill.go --rollback-batch model-dashboard-20260901 --confirm-rollback
```

Also record frontend/backend Git rollback commits and database restore instructions. Do not run rollback after a successful acceptance.

- [ ] **Step 8: Final verification commit**

```bash
git add docs/runbooks/model-dashboard-analytics-acceptance-2026-09-01.md
git commit -m "docs: 记录模型看板验收结果"
```

---

## Final Acceptance Gate

Before claiming implementation complete, verify all of the following from current command output and saved evidence:

- `model_usage_events` has one row per unique request/task and duplicate writes are idempotent.
- Selected-range calls, final quota, and Token equal the sum of the returned model/time series.
- Baseline values are 20 calls, 19 successes, 1 failure, 4,226,311 Token, and ¥253.18 ± ¥0.05 at the recorded cutoff.
- Current RPM/TPM remain five-minute rolling values when the business range changes.
- Seedance asynchronous Token and end-to-end durations are present.
- Health uses weighted success, successful-duration P50/P95, explicit sample shortage, and 15-minute stuck detection.
- The chart contains all real August 31/September 1 buckets and zero-filled gaps.
- Zoom, pan, double-click reset, legend toggles, full-ID tooltip/copy, empty/error states, keyboard access, and responsive layout are verified.
- Fresh and upgrade migrations pass twice on SQLite, MySQL 5.7.44, and PostgreSQL 9.6.24.
- Changed Go tests, complete Go tests, frontend tests, typecheck, lint, format check, and production build have current passing evidence.
- No paid model API was called during backfill or acceptance.
