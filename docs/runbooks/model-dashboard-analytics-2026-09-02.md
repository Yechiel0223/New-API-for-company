# 模型数据看板：实施与验证手册

日期：2026-09-02
适用分支：`codex/model-dashboard-analytics`
产品依据：`docs/superpowers/specs/2026-09-01-model-dashboard-analytics-prd.md`
技术依据：`docs/superpowers/specs/2026-09-01-model-dashboard-analytics-technical-design.md`

## 目的

为管理员模型看板建立可去重、可回溯的调用统计：所选范围的调用、最终消费、实际 Token 与模型趋势，以及固定窗口的 RPM/TPM 和模型健康。统计独立于原计费链路；不改变任务、日志或最终扣费记录。

## 已实施内容

- 新增 `model_usage_events` 统一调用事实表：同步请求以 `request:<request_id>`、异步任务以 `task:<task_id>` 唯一；开始时为 `running`，完成后原地写入状态、最终 quota、实际 Token、输出数和端到端耗时。
- 新增 `model_usage_attempts`，按 `(event_key, channel_id)` 保存同步渠道尝试结果，用于健康故障判断，不参与调用数或费用汇总。
- 同步 Relay 与异步任务生命周期均写入事实；统计写入失败只记录日志，不影响上游调用、计费或退款。
- 新增管理员接口：`GET /api/data/model-analytics`（选定范围统计）和 `GET /api/data/model-analytics/health?hours=1|24|168`（固定窗口健康）。接口仅返回聚合数据，不返回真实上游 Key 或完整虚拟 Key。
- 新增本地幂等回填工具，从 `tasks`、`logs`、`perf_metrics` 重建历史事实；不访问火山方舟或任何付费模型接口。

截至本手册编写时，后端事实、聚合、接口和回填已提交。前端看板交互及完整浏览器验收仍由实施分支继续推进；不要将本手册中的待验收步骤视为已完成证据。

## 配置与使用

1. 按本仓库既有方式配置应用主数据库与日志数据库；回填程序会调用 `common.InitEnv()`、`model.InitDB()` 和 `model.InitLogDB()`，因此必须明确指向要验收的本地数据库。
2. 先备份该数据库，并记录数据库/容器身份、应用提交、源表行数和备份位置。不要在生产数据库或无法恢复的副本上首次执行。
3. 后端启动后，以 Root 或具备全局看板读取权限的管理员调用接口或打开 `/dashboard/models`。范围接口要求正 Unix 秒 `start_timestamp`、`end_timestamp`（结束不早于开始）；`granularity` 仅允许 `hour`、`day`、`week`，省略时自动选择。`username` 为精确匹配，`models` 使用逗号分隔的完整 Model ID。
4. 时间显示与桶边界使用 `Asia/Shanghai`。业务卡片/图表按首次提交时间归属；当前 RPM 为最近 5 分钟提交数除以 5，当前 TPM 为最近 5 分钟完成且有实际 usage 的 Token 除以 5，均不跟随业务范围。

## 历史回填

本次 PRD 基准截止点为 `2026-09-01 14:44:00 Asia/Shanghai`，Unix 秒 `1788245040`。该截止点隔离基准数据，避免新调用改变验收结果；批次名必须唯一且非空。

```powershell
$analyticsCutoff = 1788245040
$analyticsBatch = "model-dashboard-20260901"

# 只读预览：必须先执行
go run ./scripts/model-usage-backfill.go --dry-run --cutoff $analyticsCutoff --batch $analyticsBatch

# 正式写入：确认 dry-run 的 JSON 报告后才执行
go run ./scripts/model-usage-backfill.go --dry-run=false --cutoff $analyticsCutoff --batch $analyticsBatch

# 幂等复核：第二次正式执行的新增数必须为 0
go run ./scripts/model-usage-backfill.go --dry-run=false --cutoff $analyticsCutoff --batch $analyticsBatch

# 回滚：仅在需要撤销该批次时执行；会删除该批次的事实与关联尝试记录
go run ./scripts/model-usage-backfill.go --rollback-batch $analyticsBatch --confirm-rollback
```

不要在成功验收后为了“测试回滚”执行最后一条命令。它不修改 `tasks`、`logs`、`quota_data` 或原始最终计费记录，但仍是对统计表的破坏性操作。回填使用 `event_key` 去重；历史 `perf_metrics` 仅补齐调用数量差额，无法恢复的 quota、Token 和耗时保留为 0。

基准切片的预期值为：20 次调用、19 成功、1 失败、4,226,311 Token、人民币总消耗 `¥253.18`（quota 换算与展示精度允许绝对误差不超过 `¥0.05`）。实际回填 JSON 和数据库核对结果应保存，而不是仅口头记录。

## 测试与其证明内容

以下命令均为已记录的后端验证；重新执行前不需要、也不得调用付费模型接口。

| 命令 | 证明内容 |
| --- | --- |
| `go test ./model -run 'TestModelUsageEvent|TestListModelUsageEventsForAnalytics|TestModelUsageAttempt' -count=1` | 唯一事实/尝试行、终态更新、回填批次删除保护，以及提交或完成时间窗口的读取行为。 |
| `go test ./service -run 'TestRecordSyncModelUsage|TestBackfillModelUsageEvents|TestRollbackModelUsageBatch|TestBackfillAndRollback|TestBackfillReconstructs|TestBackfillRejects' -count=1` | 同步生命周期不重复记录；dry-run 不写入；tasks/logs/perf 的回填、幂等与指定批次回滚。 |
| `go test ./service -run 'TestQueryModelAnalytics|TestQueryModelHealth|TestBuildModelHealth|TestRecordTaskModelUsage|TestRecordSyncModelUsageAttempt' -count=1` | 真实时间桶与空桶、选定范围总计、固定 5 分钟 RPM/TPM、加权成功率、P50/P95、卡住任务、异步实际 Token 和渠道尝试健康证据。 |
| `go test ./controller ./router -run 'TestGetModelUsage|Test.*DataRoute|Test.*Auth' -count=1` | 管理员路由、参数校验、范围/健康接口契约和未授权隔离。 |
| `go test ./model ./service ./controller -count=1` | 已修改后端包的完整回归。实施记录中该命令在最新健康统计修复后通过。 |
| `go test ./service -count=1` | 回填提交时的 service 包完整回归。 |

完整仓库 `go test ./...` 目前不能作为绿灯：已知基线包含缺失的 `web/dist`（根包 `go:embed` 所需）及 Windows HTTP/2 `GOAWAY` fixture 失败。前端构建 `web/dist` 后应重新运行，并将这两个已知基线与新增失败明确区分。

## 验收与结果位置

在 `artifacts/model-dashboard-analytics/`（运行时证据，不应提交）保存：

- `baseline-cutoff.txt`、数据库备份位置/身份/行数记录；
- dry-run、第一次 apply、第二次 apply 的 JSON 输出；
- 管理员接口响应：`/api/data/model-analytics?start_timestamp=<start>&end_timestamp=1788245040&granularity=hour` 和 `/api/data/model-analytics/health?hours=24`；
- 桌面与窄屏截图，以及今天、24 小时、7 天、30 天、自定义范围的浏览器检查结果。

核对 API 时，确认：卡片总计等于同筛选条件 series 总计；真实的 8 月 31 日/9 月 1 日桶存在且空桶为 0；缩放、平移、重置和图例隐藏不改变顶部总计；健康行包含 Seedance 2.5、Seedance 2.0、Seedream 5.0 Pro，且基准 Seedream 为成功 `3/4 · 75%`。同时检查响应与截图中均没有 Key 明文。

## 已知限制与待完成的 UI 打磨

- 尚未完成 live MySQL 5.7.44 与 PostgreSQL 9.6.24 的全新、升级、重复迁移矩阵；当前已覆盖 SQLite 和通用 GORM 路径。
- 历史 `perf_metrics` 合成行不含精确 Token、消费、端到端耗时，不能进入精确 P50/P95；已存在的历史事实也没有重试渠道尝试明细，渠道全失败规则只从新的可信尝试证据生效。
- P50/P95 在服务内存中排序，适合当前数据规模；事实表达到百万级或 90 天查询明显超过 500ms 时再评估日汇总或数据库分位能力。
- 健康状态反映 New API 实际观察到的调用与渠道尝试，不等同于火山方舟官方 SLA。
- 仍需完成并记录前端的快捷范围/自定义范围保留、缩放与平移、键盘和移动端等价操作、全 ID 复制、空态/错误态/最后更新时间，以及前端单测、类型检查、lint、生产构建和浏览器验收。
