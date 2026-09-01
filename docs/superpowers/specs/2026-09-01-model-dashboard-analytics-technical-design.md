# 公司模型数据看板技术方案

日期：2026-09-01

状态：待技术评审

对应 PRD：`docs/superpowers/specs/2026-09-01-model-dashboard-analytics-prd.md`

实施计划：`docs/superpowers/plans/2026-09-01-model-dashboard-analytics-implementation.md`

## 1. 方案结论

本次不继续使用现有 `quota_data` 和 `perf_metrics` 直接拼装模型看板，而是在主数据库新增一张统一调用事实表 `model_usage_events`：

- 一个同步请求只对应一条事实记录。
- 一个异步任务只对应一条事实记录。
- 调用开始时写入 `running`，完成后原地更新成功/失败、最终消费、实际 Token、输出数量和端到端耗时。
- 现有计费、日志、任务和渠道逻辑保持不变；新表只做统计，不参与扣费。
- 使用本地 `tasks`、`logs`、`perf_metrics` 做一次幂等历史回填，不访问火山方舟。
- 新增两个管理员接口，分别服务“所选时间范围统计”和“固定窗口性能健康”。

这是当前最稳妥的方案。直接改造 `quota_data` 会继续受到预扣、补扣、退款等多条计费事件影响；直接使用 `perf_metrics` 又无法覆盖 Seedance 异步任务和精确 P50/P95。

## 2. 系统边界

本方案负责：

- 模型调用去重统计。
- 最终人民币消费汇总。
- Seedance、Seedream 和未来语言模型的实际 Token 汇总。
- 时间范围、自动粒度、真实时间轴、缩放与筛选。
- 当前 RPM/TPM。
- 按模型成功率、P50、P95、运行中和疑似卡住状态。
- 上线前本地历史数据回填。

本方案不负责：

- 修改已经验收的 Seedance/Seedream 计费规则。
- 替代 `logs`、`tasks`、`quota_data` 或 `perf_metrics`。
- 对接火山方舟账单 API。
- 提供员工自助后台。
- 构建完整 APM 或实时日志流。

## 3. 总体架构

```mermaid
flowchart LR
    VK[员工虚拟 Key] --> RELAY[New API Relay]
    RELAY --> SYNC[同步模型调用]
    RELAY --> TASK[异步任务提交/轮询]
    SYNC --> BILL[现有计费与日志]
    TASK --> BILL
    SYNC --> FACT[(model_usage_events)]
    TASK --> FACT
    LOG[(logs)] --> BACKFILL[本地幂等回填]
    TASKDB[(tasks)] --> BACKFILL
    PERF[(perf_metrics)] --> BACKFILL
    BACKFILL --> FACT
    FACT --> RANGE[/model-analytics]
    FACT --> HEALTH[/model-analytics/health]
    RANGE --> UI[管理员模型看板]
    HEALTH --> UI
```

设计原则：

1. 计费系统决定“扣多少钱”，统计系统只复制已经结算的结果。
2. 调用身份由 `event_key` 唯一约束，重试、轮询和重复回填都不能重复计数。
3. 新统计写入失败只能记录错误，不能影响模型调用、扣费、退款和用户响应。
4. 时间归属按首次提交时间，任务完成后回写原记录，不把 Token 和消费移动到完成时间桶。
5. 所有报表时间统一使用 `Asia/Shanghai`。

## 4. 为什么现有数据不能直接满足需求

### 4.1 `quota_data`

`quota_data` 以计费事件为中心。一个异步任务可能产生预扣、差额结算或退款，多条事件中的 `count` 不能天然代表一个模型调用，因此会造成总调用数重复。

### 4.2 `perf_metrics`

`perf_metrics` 是按时间桶聚合的同步请求性能数据：

- 当前没有完整覆盖异步视频任务。
- 只有聚合耗时，无法还原每次调用的准确 P50/P95。
- 不包含最终消费和全部 actual tokens。

### 4.3 `logs` 与 `tasks`

二者是可靠来源，但分别描述同步计费事件和异步任务生命周期。每次页面查询临时跨库拼接会复杂、慢且难保证幂等，因此只用于事实表实时写入的来源校验和一次性历史回填。

## 5. 统一调用事实表

表名：`model_usage_events`

| 字段 | 类型口径 | 作用 |
|---|---|---|
| `id` | bigint | 主键 |
| `event_key` | varchar(191)，唯一 | 同步为 `request:<request_id>`，异步为 `task:<task_id>` |
| `kind` | `sync` / `task` | 调用形态，仅用于内部记录，前端不按它分组 |
| `request_id` / `task_id` | varchar | 与原日志或任务对账 |
| `user_id` / `username` | 用户快照 | 用户筛选和离职后历史可读 |
| `token_id` | int | 虚拟 Key 归属，不保存 Key 明文 |
| `channel_id` | int | 渠道对账 |
| `model_name` | varchar(128) | 完整 Model ID |
| `use_group` | varchar(64) | 调用时分组快照 |
| `status` | running/success/failure | 当前最终状态 |
| `submitted_at` | Unix 秒 | 首次提交时间，决定业务时间桶 |
| `completed_at` | Unix 秒 | 终态时间，供 TPM 和健康统计 |
| `last_progress_at` | Unix 秒 | 疑似卡住判断 |
| `total_tokens` | bigint | 最终实际 Token；未知或失败无 usage 时为 0 |
| `output_count` | int | 图片/视频输出数量；不存在时为 0 |
| `output_unit` | image/video/空 | 输出数量单位，不混加不同单位 |
| `final_quota` | int | 复制现有结算后的最终 quota，不重新计价 |
| `duration_ms` | bigint | 端到端耗时；无可靠样本时为 0 |
| `failure_reason` | text | 脱敏并截断至 512 个 UTF-8 字符 |
| `source` | varchar(24) | live/backfill_task/backfill_log/backfill_perf |
| `backfill_batch` | varchar(64) | 精确回滚本次历史回填 |
| `recorded_at` / `updated_at` | Unix 秒 | 统计记录时间 |

关键索引：

- `event_key` 唯一索引，保证调用不重复。
- `(model_name, submitted_at)` 联合索引，服务时间范围与模型筛选。
- `submitted_at`、`completed_at`、`last_progress_at`、`username`、`status`、`backfill_batch` 普通索引。

兼容性要求：只使用 GORM 通用字段和索引，不使用数据库专属 JSON、生成列或日期函数，必须同时支持 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。

## 6. 实时写入流程

### 6.1 同步请求

1. 请求完成本地鉴权、模型检查和请求体校验。
2. 第一次真正进入上游处理前创建 `running` 事实；渠道重试不重复创建。
3. 成功完成现有计费与 `RecordConsumeLog` 后，写入最终 quota、Token、图片数量和耗时。
4. 所有渠道重试结束仍失败时，只终结一次，quota/Token 为 0，并保存脱敏原因。
5. 未进入上游的鉴权、权限和参数错误不计入调用。

Seedream 输出数量优先使用标准化响应中的 `usage.generated_images`。只有正数才写 `output_count=1..n` 和 `output_unit=image`。

### 6.2 异步任务

1. 火山返回任务且本地 `tasks` 插入成功后，立即创建 `running` 事实。
2. 轮询结果只有在状态、进度、开始/结束时间、失败原因、结果地址或数据发生变化时更新 `last_progress_at`；空轮询不刷新时间。
3. 成功/失败终态由 CAS 获胜的轮询进程处理，先执行现有结算或退款，再写最终事实。
4. 超时清理器将任务转为失败并退款后，同样终结事实。
5. 立即返回终态的任务在提交计费完成后直接终结，不调用轮询内部结算函数。

Seedance Token 取值顺序：

1. `TaskInfo.TotalTokens`。
2. `TaskInfo.CompletionTokens`。
3. 正数 `UsageFacts["tokens"]`。
4. 都不存在时为 0，并不影响调用、消费和耗时展示。

成功视频记 `output_count=1`、`output_unit=video`；失败任务不记输出。

## 7. 历史数据回填

回填工具只在本机运行，并支持：

```powershell
go run ./scripts/model-usage-backfill.go --dry-run --cutoff 1788245040 --batch model-dashboard-20260901
go run ./scripts/model-usage-backfill.go --cutoff 1788245040 --batch model-dashboard-20260901
go run ./scripts/model-usage-backfill.go --rollback-batch model-dashboard-20260901 --confirm-rollback
```

处理顺序：

1. 从 `tasks` 按 `task_id` 重建异步事实，读取状态、时间、最终 quota、usage 和失败原因。
2. 从 `LOG_DB.logs` 按非空 `request_id` 分组重建同步事实，排除带 `is_task=true` 或 `task_id` 的任务日志。
3. 使用 `perf_metrics` 只补齐无法从旧日志还原的历史调用数量差额；补齐记录 quota/Token/耗时为 0，不参与 P50/P95。
4. 如果重建事实数量已经大于旧聚合计数，阻止正式应用并要求人工核对，避免反向制造重复。

幂等与回滚：

- dry-run 只读，输出预计新增、跳过、拒绝和汇总值。
- 正式执行依赖 `event_key` 唯一索引；同一批执行第二次新增数必须为 0。
- 回滚只删除精确匹配 `backfill_batch` 的事实，不删除实时数据和源数据。
- 回填前备份数据库并记录应用提交、数据库/容器身份和源表行数。

基准截止时间采用 PRD 证据时间 `2026-09-01 14:44:00 Asia/Shanghai`，Unix 秒为 `1788245040`。预期结果：20 次调用、19 成功、1 失败、4,226,311 Token、总消耗 ¥253.18（允许 quota 换算及展示精度导致绝对误差不超过 ¥0.05）。

## 8. 后端接口

### 8.1 所选范围统计

`GET /api/data/model-analytics`

参数：

- `start_timestamp`：必填，Unix 秒。
- `end_timestamp`：必填，且不小于开始时间。
- `granularity`：可选，`hour | day | week`；省略时自动选择。
- `username`：可选，用户名精确匹配。
- `models`：可选，逗号分隔的完整 Model ID。

返回：

- `range`：实际时间范围、粒度、时区。
- `summary`：所选范围总调用、成功/失败/运行中、总消费、总 Token、峰值 RPM/TPM。
- `series`：按时间桶和模型的调用、消费、Token、输出数量。
- `models`：按模型汇总，用于卡片明细、占比和排行。
- `available_models`：筛选器候选 Model ID。
- `updated_at`：数据生成时间。

### 8.2 性能健康

`GET /api/data/model-analytics/health?hours=24`

只允许 `hours=1|24|168`。返回：

- 最近 15 分钟整体当前状态。
- 最近 5 分钟 RPM/TPM。
- 所选 1 小时/24 小时/7 天的加权成功率、失败、运行中和疑似卡住数量。
- 每个模型的成功数/总调用、成功率、P50、P95、样本数和最近失败。

两个接口都挂在 `middleware.AdminAuth()` 后，Root 和普通管理员可访问，普通用户不可访问；响应不包含虚拟 Key 明文。

## 9. 统计算法

### 9.1 时间范围总计

所有卡片都筛选 `submitted_at`：

- 总调用：事实记录数。
- 总消费：`sum(final_quota)`，再使用现有 quota→人民币格式化逻辑。
- 总 Token：`sum(total_tokens)`。
- 成功/失败/运行中：按事实当前状态统计。

### 9.2 时间桶

- 今天、24 小时、≤48 小时自定义：小时。
- 7 天、30 天、>48 小时且≤90 天自定义：天。
- >90 天自定义：周，周一 00:00 开始。
- 后端用 Go 和 `Asia/Shanghai` 生成全部桶并补 0，不依赖数据库日期函数。
- 同一调用的最终消费和 Token 仍归到 `submitted_at` 所在桶。

### 9.3 RPM/TPM

- 当前 RPM：最近 5 分钟 `submitted_at` 事实数 ÷ 5。
- 当前 TPM：最近 5 分钟按 `completed_at` 获得的实际 Token 合计 ÷ 5。
- 两者固定使用滚动窗口，不跟随页面业务范围。
- 峰值 RPM/TPM 来自所选业务范围，卡片悬浮显示峰值和发生时间。

### 9.4 性能健康

- `无调用`：最近 15 分钟没有新调用、运行中调用和明确不可用模型。
- `健康`：存在完成调用，且没有失败、卡住或不可用模型。
- `注意`：存在失败或卡住，但未达到故障条件。
- `故障`：同一模型最近 3 次终态均失败、该模型没有可用渠道，或所有已启用渠道最后一次尝试均失败且没有更晚成功。
- 成功率：`成功总数 ÷ (成功总数 + 失败总数)`，运行中不进入分母，不能对各模型百分比做简单平均。
- P50/P95：只使用成功且 `duration_ms>0` 的样本，采用 nearest-rank；P50 至少 1 个样本，P95 至少 3 个样本。
- 成功耗时样本少于 20 时悬浮提示“小样本，仅供参考”。
- 疑似卡住：非终态且 `last_progress_at <= 当前时间-15分钟`。

## 10. 前端方案

页面由 `ModelAnalyticsSection` 统一拥有查询状态，避免卡片、健康和图表各自请求导致时间口径不一致。

顶部常驻：

- 今天、24 小时、7 天、30 天、自定义。
- 当前准确范围和聚合粒度。
- 手动刷新、筛选、偏好设置。

五张卡片：

- 总调用：所选范围；副文本显示成功/失败/运行中。
- 总消耗：所选范围；文案明确为“总消耗”。
- 总 Token：所选范围；悬浮按模型展示 Token，并可复制完整 Model ID。
- 当前 RPM：近 5 分钟；悬浮展示所选范围峰值。
- 当前 TPM：近 5 分钟；悬浮展示所选范围峰值。

性能健康直接按模型展示，例如：

```text
Seedream 5.0 Pro   成功 3/4 · 75%   P50 50s   P95 55s
```

图表：

- 使用后端返回的真实 Unix 时间桶，不再调用旧 `fillTimePoints` 生成最近 7 个时间点。
- 后端返回完整空桶，前端只负责格式化。
- 时间序列超过 12 个点时显示底部缩放条。
- 支持滚轮缩放、拖动平移、双击恢复，并提供键盘可用的放大/缩小/重置按钮。
- 缩放只改变可视范围，不改变卡片总计。
- 图例显示短名称；悬浮/点击显示完整 Model ID、准确时间、调用、人民币消费、占比、Token 或图片/视频数量。
- 移动端通过点击保留与鼠标悬浮等价的详情。

刷新策略：

- 所选范围统计每 60 秒刷新。
- 性能健康和 RPM/TPM 每 30 秒刷新。
- 查询失败显示 `--` 和重试，不用旧数据伪装成最新数据。

## 11. 一致性、并发与故障处理

- 实时创建使用 `ON CONFLICT DO NOTHING`，终态更新使用同一个 `event_key`。
- 异步终态只由任务状态 CAS 获胜者写入；重复写入仍是幂等更新。
- 分析写入不使用携带 Gin 生命周期的异步 goroutine。
- `final_quota`、Token、输出数量、耗时都拒绝负数；输出单位只允许空、image、video。
- 失败原因先使用现有脱敏错误，再按 UTF-8 截断。
- 分析接口失败不影响模型接口；页面明确降级并允许重试。
- 主表暂不引入定时物化汇总。当前约 20 名用户，在 Go 服务层聚合筛选后的事实行足够简单可靠。

扩容观察点：当事实表达到百万级、单次 90 天查询明显超过 500ms 或数据库 CPU 因看板持续升高时，再增加日级汇总表或缓存；本期不提前引入 Redis 和复杂聚合任务。

## 12. 发布、验证与回滚

发布顺序：

1. 备份本地 PostgreSQL，记录基准截止时间。
2. 在 SQLite、MySQL 5.7.44、PostgreSQL 9.6.24 上分别验证全新迁移、旧版本升级迁移和重复迁移。
3. 发布后端事实表与实时写入。
4. dry-run 历史回填并核对预期统计。
5. 正式回填，再执行第二次确认新增 0 条。
6. 发布新接口和前端。
7. 按 24 小时、7 天、30 天、自定义范围做浏览器验收并保存截图和查询证据。

回滚：

- 前端可回退到旧看板，新表不会影响原接口。
- 新接口可停止路由，不影响计费和模型调用。
- 历史回填按 `backfill_batch` 精确删除。
- 事实表可以保留；若最终需要删除，必须在确认无任何新代码读取后单独执行数据库变更，不在应用回滚中自动删表。

验证证据统一写入：`docs/runbooks/model-dashboard-analytics-acceptance-2026-09-01.md`。禁止为了验证调用任何付费模型接口。

## 13. 已知取舍

1. 历史 `perf_metrics` 只能补齐调用数量差额，无法反推出精确 Token、消费和分位耗时，因此合成记录的这些字段保持 0。
2. 当前 P50/P95 在 Go 内存中对所选事实排序，适合当前规模；数据量明显增长后再使用数据库分位函数或汇总表，但那会增加三数据库差异。
3. 当前状态依赖 New API 实际观察到的请求和渠道状态，不等同于火山方舟官方 SLA。
4. Seedream Token 只作为调用规模参考，不作为官方计费单位；人民币消费仍以已结算 quota 为准。

## 14. 技术验收标准

- 相同请求或任务无论重试、补扣、轮询多少次，总调用只增加 1。
- 所选范围的总调用、总消费、总 Token 与同筛选条件图表一致。
- Seedance actual tokens 被计入，Seedream Token 和图片数量可展示。
- 当前 RPM/TPM 始终为最近 5 分钟，业务范围改变时不变化。
- Seedance 2.5、Seedance 2.0、Seedream 5.0 Pro 均进入性能健康。
- 基准快照结果达到 20 次调用、19 成功、1 失败、4,226,311 Token、¥253.18±¥0.05。
- 8 月 31 日和 9 月 1 日的真实时间桶均能显示，空桶为 0，缩放不改变卡片总计。
- SQLite、MySQL 5.7.44、PostgreSQL 9.6.24 全新/升级/重复迁移均通过。
- 回填 dry-run 不写库，正式回填重复执行不新增，回滚只删除指定批次。
- 全过程不访问任何付费生成接口。
