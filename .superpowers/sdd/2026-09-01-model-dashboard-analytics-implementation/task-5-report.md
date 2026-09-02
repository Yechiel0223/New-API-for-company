# Task 5 report: model dashboard analytics APIs

## Changes

- Added `GET /api/data/model-analytics`, protected by `middleware.AdminAuth()`.
  - Requires positive canonical Unix-second `start_timestamp` and `end_timestamp` values with `end >= start`.
  - Accepts only `hour`, `day`, or `week` when `granularity` is supplied; omitted granularity is delegated to `service.QueryModelAnalytics` for its existing automatic selection.
  - Accepts an exact `username` and comma-separated exact model IDs, rejecting surrounding whitespace and empty model IDs.
  - Returns the service result unchanged inside the standard success envelope, including the service-provided `Asia/Shanghai` range timezone.
- Added `GET /api/data/model-analytics/health`, protected by `middleware.AdminAuth()`.
  - Defaults to 24 hours when omitted and accepts only canonical `1`, `24`, or `168` values when supplied.
- Added focused controller contract tests for invalid analytics parameters, the successful analytics response fields/timezone, and allowed/rejected health windows.

## Verification

- RED: `go test ./controller -run 'TestGetModelUsageAnalytics|TestGetModelUsageHealth' -count=1` failed because both handlers were initially undefined.
- GREEN: `go test ./controller -run 'TestGetModelUsageAnalytics|TestGetModelUsageHealth' -count=1` passed.
- `go test ./controller ./router -run 'TestGetModelUsage|Test.*DataRoute|Test.*Auth' -count=1` passed.
- `go test ./controller ./router -count=1` passed.
- `git diff --check` passed.

## Concerns

- The handlers intentionally do not duplicate aggregation or health logic; all data access, automatic granularity selection, and Shanghai timezone output remain in the existing service methods.
- No paid generation or external APIs were called.
