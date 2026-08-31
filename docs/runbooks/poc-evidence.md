# 火山方舟 Seedance 用量网关：本机 POC 验收证据

> 本文件只记录脱敏后的结果、时间、摘要和计算过程。严禁记录火山方舟真实 API Key、任何虚拟 Key、管理员密码、数据库密码、会话密钥、Authorization 请求头或本机代理地址。

## 0. 如何阅读与复核本文件

本文件是本机 POC **所有测试结果的统一入口和权威索引**。后续每项测试都必须有唯一编号，并至少记录以下内容：

1. **为什么测试**：该测试验证什么风险或验收条件。
2. **前置条件**：测试依赖的配置、数据与服务状态。
3. **如何测试**：可由另一名开发者重复执行的页面操作、API 请求或脱敏命令。
4. **预期结果**：执行前定义的通过条件。
5. **实际结果**：观察值、时间和 `PASS`、`FAIL`、`BLOCKED` 或 `NOT RUN` 结论。
6. **结果哪里看**：对应的管理员页面、数据库查询、容器日志或本文件章节。
7. **副作用与清理**：是否创建 Key/任务/数据，是否访问付费上游，产生多少费用，清理后还保留什么审计痕迹。
8. **偏差与限制**：操作与原计划不一致、无法安全复现或证据不足时如实记录，不得把推测写成通过。

状态含义：

| 状态 | 含义 |
| --- | --- |
| `PASS` | 实际结果满足预先定义的全部通过条件，且证据可复核 |
| `FAIL` | 已执行，但至少一个通过条件不满足 |
| `BLOCKED` | 因缺少必要配置、权限或外部条件而未能完成 |
| `NOT RUN` | 尚未执行，不代表通过或失败 |

真实调用会额外记录：脱敏任务 ID、虚拟 Key 的非秘密名称、上游终态、actual usage、预占人民币、最终扣费人民币和费用重算过程。完整 Key、带签名的输出 URL 和认证请求头永不写入证据。若测试可能访问火山方舟并产生费用，会在执行前明确标注；本 POC 已获准进行付费能力测试，但该授权不改变密钥脱敏要求。

## 1. 环境

| 项目 | 结果 | 证据时间（Asia/Shanghai） |
| --- | --- | --- |
| Docker Client / Server | 29.7.2 / 29.7.2，`hello-world` 通过 | 2026-08-31 |
| Docker Compose | 5.4.0 | 2026-08-31 |
| WSL | 2.7.12.0，默认 WSL2，内核 6.18.33.2-2 | 2026-08-31 |
| 虚拟化 | HypervisorPresent = true | 2026-08-31 |
| Redis | 未部署 | 2026-08-31 |
| 批量额度更新 | `BATCH_UPDATE_ENABLED=false` | 2026-08-31 |

## 2. 固定源码与镜像

| 项目 | 不可变标识 | 结果 |
| --- | --- | --- |
| New API 上游源码 | `2b6f1dfefbe217fed31fc0726717cc7de6958e8e`（v1.0.0-rc.29） | PASS：为当前分支祖先 |
| 本项目当前提交 | `4a641bc243bf779fc0e4e8b686642eaaff9e9e37` | PASS |
| 本机构建 New API 镜像 | `sha256:40ebb76b45965163fa7357aeb99a0d5bf458a0617902308d1569966ebaaec2dc` | PASS，amd64 |
| PostgreSQL 镜像 | `postgres@sha256:fe0737ba566a2c5b2a28f34433c0a423261900ec17b9bf7ad115e1aae7e57f1b` | PASS，15.19-alpine3.24 |
| 无 Redis 原子预占测试 | `TestTryReserveQuotaWithoutRedis`、`TestReserveFallsBackToDatabaseWhenRedisIsUnavailable` | PASS |

构建网络备注：Docker Desktop 系统代理链路曾发生外部 DNS/传输中断；通过只在当前进程传入 Docker 预定义代理 build args 完成构建。未修改源码，未记录代理值。

## 3. 运行态与 PostgreSQL 权威账本

| 检查项 | 状态 | 证据/备注 |
| --- | --- | --- |
| PostgreSQL healthy | PASS | Compose 健康检查通过 |
| New API running | PASS | 只监听 `127.0.0.1:3000` |
| `/api/status` 成功 | PASS | `success=true` |
| 实际数据库为 `new_api` | PASS | PostgreSQL 内查询确认 |
| PostgreSQL 版本为 15.x | PASS | 15.19 |
| 未使用 SQLite 账本 | PASS | `/api/setup` 报告 `database_type=postgres`，实时数据位于 PostgreSQL named volume |

## 4. 管理员设置

| 检查项 | 目标 | 状态 | 证据/备注 |
| --- | --- | --- | --- |
| 管理员账户 | 恰好一个 | PASS | `users` 总数 1，root 角色总数 1；不记录用户名或密码 |
| 管理员登录 | 保留密码登录 | PASS | 已登录验证；`password_login_enabled=true` |
| 公开注册 | 关闭 | PASS | `register_enabled=false`、`password_register_enabled=false` |
| OAuth/OIDC | 关闭 | PASS | GitHub、Discord、OIDC、Telegram、LinuxDO、微信和 Passkey 均为 false |
| 在线支付、充值、兑换、邀请 | 关闭 | PASS | 未确认支付合规条款，相关功能保持锁定；新用户/邀请双方额度均为 0，充值链接为空，签到关闭 |
| 公开模型与排行榜入口 | 关闭 | PASS | 顶部导航的模型广场和排行榜均已关闭 |
| `QuotaPerUnit` | 500000 | PASS | `/api/status` 核对 |
| `USDExchangeRate` | 1 | PASS | `/api/status` 核对 |
| 币种显示 | 仅 CNY/RMB | PASS | `display_in_currency=true`、`quota_display_type=CNY` |
| 临时 Token 人民币显示 | `¥1.00` | PASS | 见下方 `POC-CURRENCY-001`；验证后已删除 |

### 4.1 `POC-CURRENCY-001`：虚拟 Key 人民币显示验证

**测试时间：** 2026-08-31 15:36:14 +08:00  
**执行位置：** 管理员页面 `http://localhost:3000/keys`  
**证据位置：** 本节；运行态结果还可通过本节末尾的脱敏 SQL 复核。真实 Key 未记录、未复制、未调用。

**为什么测试**

New API 内部使用整数 quota 记账，而公司管理员只应看到人民币。如果只修改系统设置而不创建实际虚拟 Key，无法证明 Key 列表和编辑详情这两个日常管理页面没有泄露 raw quota、美元或积分单位。因此用一条无调用历史的临时记录验证：

```text
500,000 raw quota = ¥1.00
```

**前置条件**

- `DisplayInCurrencyEnabled=true`；
- `general_setting.quota_display_type=CNY`；
- `QuotaPerUnit=500000`；
- `USDExchangeRate=1`；
- 测试开始前系统中没有虚拟 Key、任务和用量日志。

**如何测试**

1. 在 API 密钥页面创建名为 `poc-currency-display-check` 的临时 Key。
2. 关闭“无限配额”，把页面额度设为 `1` CNY，过期时间保持“永不”。
3. 创建完成后立即禁用该 Key，不复制或展示完整 Key。
4. 在列表页核对状态和额度显示。
5. 打开编辑详情，核对字段名称、币种和数值。
6. 在 PostgreSQL 中只按临时名称查询非秘密字段：状态、剩余额度、已用额度、无限额度标志和过期时间。
7. 确认该 Key 的日志数为 0、系统任务数为 0。
8. 按测试前授权永久删除临时 Key，再核对活跃记录为 0。

**实际结果**

| 断言 | 实际观察 | 结果 |
| --- | --- | --- |
| 列表页币种 | 显示 `¥1 / ¥1`，未显示 `$`、USD、raw quota 或积分 | PASS |
| 列表页状态 | 禁用后显示“已禁用” | PASS |
| 编辑详情字段 | 显示“额度 (CNY)” | PASS |
| 编辑详情数值 | 显示 `1` | PASS |
| PostgreSQL 原始余额 | `remain_quota=500000` | PASS |
| PostgreSQL 已用余额 | `used_quota=0` | PASS |
| PostgreSQL 状态 | `status=2`（禁用） | PASS |
| 上游或本地任务 | 0 | PASS |
| 该 Key 用量日志 | 0 | PASS |
| 删除后活跃记录 | 0 | PASS |
| 删除后审计痕迹 | 1 条软删除记录 | PASS |

**流程偏差与影响**

当前 New API 创建表单没有“创建时禁用”字段，后端新增记录默认状态为启用。因此临时 Key 创建后曾短暂处于启用状态，随后立即被禁用。这与原计划“创建即禁用/从未启用”的理想前置条件不完全一致。实际影响被以下证据限定为零：完整 Key 未被复制或使用，任务数为 0，用量日志为 0，`used_quota=0`。这不影响本测试关于人民币显示与换算的结论，但该偏差必须保留在证据中。

**如何复核结果**

运行态 UI：

- 系统币种设置：`http://localhost:3000/system-settings/billing/currency`
- 虚拟 Key 列表：`http://localhost:3000/keys`

删除后的脱敏数据库复核命令：

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml exec -T postgres `
  psql -U new_api -d new_api -F '|' -Atc "select
    (select count(*) from tokens where name='poc-currency-display-check' and deleted_at is null) as active_tokens,
    (select count(*) from tokens where name='poc-currency-display-check' and deleted_at is not null) as deleted_tokens,
    (select count(*) from logs where token_name='poc-currency-display-check') as token_logs,
    (select count(*) from tasks) as all_tasks;"
```

本次输出为：

```text
0|1|0|0
```

依次表示：活跃临时 Key 0、软删除记录 1、该 Key 日志 0、系统任务 0。

## 5. 官方价格与能力来源

### 5.1 `POC-ARK-DOC-001`：官方模型能力与价格口径核对

**测试时间：** 2026-08-31 15:45:29 +08:00

**执行方式：** 只读访问火山引擎官方文档，没有登录公司方舟账号，没有访问 API，没有产生费用。

**证据位置：** 本节及下列官方页面。

- [火山方舟模型列表](https://docs.volcengine.com/docs/82379/1330310?lang=zh)，页面最近更新时间 `2026.08.24 21:58:51`；
- [火山方舟模型价格](https://docs.volcengine.com/docs/82379/1544106?lang=zh)，页面最近更新时间 `2026.08.28 00:06:09`。

**为什么测试**

模型 ID、可用分辨率、时长和价格都会变化。若依据第三方页面、旧版插件常量或模型名称猜测配置，虚拟 Key 余额可能与方舟真实账单不一致。因此所有计费配置必须先以当前官方文档为准，并保留核对时间。

**如何测试与预期结果**

1. 在官方模型列表定位 Doubao Seedance 2.5，核对完整 Model ID、输入模式、分辨率、帧率、时长、格式和在线限流。
2. 在官方模型价格定位同一模型，核对在线推理人民币单价、限时优惠区间、计费公式、最终用量字段和失败计费规则。
3. 预期 Model ID 与 New API 插件声明完全一致；所有启用能力都有官方依据；价格单位必须是人民币/百万 token。

**实际结果**

| 断言 | 官方页面实际值 | 结果 |
| --- | --- | --- |
| 完整 Model ID | `doubao-seedance-2-5-260628` | PASS |
| 模式 | 文生视频、首帧生视频、首尾帧生视频、全模态参考生视频等 | PASS；第一阶段只验收文生视频与图生视频 |
| 分辨率 | 480p（8bit）、720p（8bit）、1080p（10bit） | PASS；2.5 不支持 4K |
| 帧率 | 24 fps | PASS |
| 输出时长 | 4–30 秒 | PASS |
| 输出格式 | mp4、mov | PASS |
| 企业在线限流 | 最大 600 RPM，最大并发 10 | PASS；真实账号归类仍以公司控制台为准 |
| 480p/720p，无视频输入 | `¥70.00 / 百万 token` | PASS |
| 480p/720p，包含视频输入 | `¥42.00 / 百万 token` | PASS |
| 1080p，无视频输入 | 刊例价 `¥77.00 / 百万 token` | PASS |
| 1080p，包含视频输入 | 刊例价 `¥46.00 / 百万 token` | PASS |
| 1080p 当前优惠 | 2026-08-14 14:00 至 2026-09-17 14:00（UTC+8）按刊例价 72 折，即 `¥55.44` / `¥33.12` 每百万 token | PASS |
| 费用公式 | 单价 × token 用量 | PASS |
| token 估算 | `(输入视频时长 + 输出视频时长) × 输出宽 × 输出高 × 24 / 1024` | PASS |
| 最终权威用量 | API 返回的 `usage.completion_tokens` | PASS |
| 失败任务 | 仅成功生成的视频计费；审核等原因失败不收费 | PASS |
| 含视频输入最低用量 | 存在最低 token 用量，准确值最终仍以 API usage 为准 | PASS；第一阶段不验收视频生视频 |

**结论：** `PASS`。官方资料足以确认第一阶段的模型 ID、文生/图生能力、720p 价格和 1080p 探测价格。第一阶段明确不支持、不配置、不验收视频作为输入；图像输入不等于“包含视频输入”，因此图生视频使用“输入不含视频”单价。4K 必须视为不支持，不能因 New API 通用表单或插件枚举中出现 4K 就对员工宣称可用。

**副作用与清理：** 无。本测试没有创建或修改运行数据，不产生方舟费用。

### 5.2 `POC-BILLING-CODE-001`：New API 内置计费路径适配性审查

**测试时间：** 2026-08-31

**检查位置：** `plugins/tasks/doubao/plugin.js`、`relay/relay_task.go`、`pkg/billingexpr/expr.md`。

**证据位置：** 本节；后续实际表达式保存和结算结果会另建测试编号。

**为什么测试**

需要证明 New API 是按方舟实际 usage 结算，而不是只按请求时长估算；还要确认旧倍率常量是否能精确覆盖当前人民币价格和限时优惠。

**检查方法**

1. 检查 Doubao 插件提交时如何估算 token、完成后读取哪个 usage 字段。
2. 检查分辨率和视频输入倍率常量是否与 5.1 的官方价格完全一致。
3. 沿任务计费链路核对预占、终态重算和退款能力。
4. 检查已有的任务用量表达式能否读取 `tokens`、`resolution`、`video_input` 三个插件事实，并在完成时用 actual usage 覆盖估算值。

**实际结果**

| 检查项 | 实际观察 | 结果 |
| --- | --- | --- |
| 提交时 token 估算 | 插件按 `秒 × 宽 × 高 × 24 / 1024` 估算 | PASS |
| 终态 actual usage | 成功时优先读取 `usage.completion_tokens`，缺失时才回退 `total_tokens` | PASS |
| 失败终态 | 完成用量钩子不返回收费事实，任务结算链路具备退款能力 | PASS；仍需真实失败测试验证 |
| 480p/720p 旧倍率 | 无视频 `1`，含视频 `42/70`，与刊例价比例一致 | PASS |
| 1080p 旧倍率 | 使用 `11.7/10.7` 与 `7.0/10.7`，既不精确等于中国区刊例价 `77/70`、`46/70`，也不含当前 72 折 | FAIL |
| 4K 枚举 | 插件通用 schema 包含 4K，但官方 Seedance 2.5 模型列表不支持 4K | FAIL；上游应拒绝，第一阶段不得使用 |
| 任务表达式 | 可按 `resolution` 分层，用 `tokens × 元/百万 token ÷ 1,000,000` 结算，完成时 actual tokens 覆盖估算 tokens | PASS |

**结论：** 旧的固定价格/倍率模式 `FAIL`，不得用于本项目的 Seedance 2.5 账单；复用 New API 现有的任务用量表达式是可行路径。第一阶段表达式只覆盖“输入不含视频”的 480p/720p 与 1080p，并以 `usage.completion_tokens` 做终态结算。视频输入价格虽然保留在官方资料表中，但不进入第一阶段配置或测试。

当前 1080p 优惠存在固定截止时刻。不能在仍有运行中任务时直接切换价格；切价 SOP 为：停用渠道 → 等全部任务终态 → 记录切价前余额与日志 → 把 1080p 单价从 `¥55.44` 改为刊例价 `¥77` → 运行脱敏表达式检查 → 重新启用渠道。优惠结束前的表达式验证与切价提醒将在后续配置测试中补充。

**副作用与清理：** 本节只审查源码，没有修改计费代码、配置或数据库，没有创建任务，也没有产生方舟费用。

### 5.3 quota 换算口径

- `QuotaPerUnit=500000`，所以一个 raw quota 点为 `¥0.000002`；
- 表达式返回单次请求的人民币费用数值，任务 quota 为 `round(人民币费用 × 500000)`；
- 例如 720p 无视频输入、actual usage 为 108,000 token：`108000 × 70 / 1000000 = ¥7.56`，对应 `3,780,000` quota；
- 终态对账允许的最大本地量化误差为一个 raw quota 点，即 `¥0.000002`；不得把请求时估算值当成最终账单。

### 5.4 `POC-PRICING-EXPR-001`：Seedance 2.5 第一阶段计费表达式

**状态：** `PASS`

**测试对象：** New API 管理员页面中模型 `doubao-seedance-2-5-260628` 的任务用量表达式。

**是否访问方舟/产生费用：** 否。该测试只保存并校验本地计费配置。

**为什么测试**

证明第一阶段可以完全复用 New API 的任务用量表达式，按官方 actual token 和分辨率计算人民币，而不使用不精确的内置 1080p 倍率。第一阶段只覆盖文生视频与图生视频，不考虑视频输入。

**已保存表达式**

```text
u("resolution") == "1080p"
  ? tier("1080p_promo", u("tokens") * 55.44 / 1000000)
  : tier("480p_720p", u("tokens") * 70 / 1000000)
```

**如何测试**

1. 在“系统设置 → 计费与支付 → 模型定价”新增精确 Model ID，选择“表达式 → 表达式编辑器”。
2. 保存上方表达式；保存接口必须通过 Doubao 任务插件的 usage schema 与有限非负向量检查。
3. 在 PostgreSQL `options` 表核对 `billing_setting.billing_mode` 为该模型选择 `tiered_expr`，`billing_setting.billing_expr` 保存的表达式与文档逐字符一致。
4. 用下表脱敏向量计算表达式人民币结果和 quota；不创建真实任务。
5. 在管理员模型定价列表搜索该模型，确认模式显示为表达式计费。

**预期与实际结果**

| 向量 | 预期档位 | 预期人民币 | 预期 quota | 实际结果 |
| --- | --- | --- | --- | --- |
| 480p、48,038 token | `480p_720p` | `¥3.36266` | `1,681,330` | PASS |
| 720p、108,000 token | `480p_720p` | `¥7.56` | `3,780,000` | PASS |
| 1080p、243,000 token | `1080p_promo` | `¥13.47192` | `6,735,960` | PASS |
| 720p、0 token | `480p_720p` | `¥0` | `0` | PASS |

**实际执行与证据**

1. 管理员页面保存返回“设置更新成功”；后端按精确 Model ID 找到 Doubao 任务插件，并通过 `tokens`、`resolution` usage schema 的有限非负向量校验。
2. PostgreSQL `options` 表实际保存：该模型 `billing_mode=tiered_expr`，`billing_expr` 与本节表达式一致。
3. 页面刷新后搜索该模型，列表显示“表达式”“阶梯计费 · 2 档”。
4. Windows 主机首次执行 `go test` 时因未安装 Go 命令而未运行，这一步是环境阻塞，不计为表达式失败。随后复用已存在的 `golang:1.26.1-alpine` Docker 镜像执行同一测试夹具，四个子测试全部 PASS。
5. 为确认结果可复现，又执行一次全新容器复跑。默认 `proxy.golang.org` 下载 `github.com/klauspost/compress` 时发生 `unexpected EOF`，测试尚未编译，属于依赖下载失败；改用临时 `GOPROXY=https://goproxy.cn,direct` 后再次复跑，四个子测试全部 PASS，包耗时 `0.032s`。代理设置只作用于该临时容器，没有写入仓库或运行服务。

```text
=== RUN   TestSeedance25POCPricingVectors/480p
=== RUN   TestSeedance25POCPricingVectors/720p
=== RUN   TestSeedance25POCPricingVectors/1080p_promo
=== RUN   TestSeedance25POCPricingVectors/zero_tokens
--- PASS: TestSeedance25POCPricingVectors
PASS
ok github.com/QuantumNous/new-api/pkg/billingexpr
```

测试命令：

```powershell
docker run --rm -v 'D:\new-api:/src' -w /src golang:1.26.1-alpine `
  go test ./pkg/billingexpr -run '^TestSeedance25POCPricingVectors$' -count=1 -v
```

若默认 Go 模块代理下载失败，可执行等价的临时代理复跑：

```powershell
docker run --rm -e GOPROXY=https://goproxy.cn,direct `
  -v 'D:\new-api:/src' -w /src golang:1.26.1-alpine `
  go test ./pkg/billingexpr -run '^TestSeedance25POCPricingVectors$' -count=1 -v
```

测试夹具保留在 `pkg/billingexpr/seedance_pricing_poc_test.go`，不进入运行镜像、不访问方舟，可直接用上方命令复核。1080p 优惠结束切换到刊例价时，必须同时更新运行配置、该测试中的单价/期望值和本节证据。

数据库复核命令（只读取非秘密计费配置）：

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml exec -T postgres `
  psql -U new_api -d new_api -F '|' -Atc "select key,value from options where key in ('billing_setting.billing_mode','billing_setting.billing_expr') order by key;"
```

**前端限制：** 原始表达式保存正确，但当前“编辑模型定价”弹窗重新载入包含 `u(...)` 的任务表达式时，文本框错误显示为 `p * 0 + c * 0`，前端预览同时不识别 `u`。模型列表与数据库仍是正确配置。管理员不得在该编辑弹窗直接再次保存，否则可能覆盖正确表达式；后续改价必须使用能够保留原始 JSON 的配置路径，并在保存后执行上述数据库查询和四组向量测试。

**结果哪里看**

- 管理员页面：`http://localhost:3000/system-settings/billing/model-pricing`，搜索完整 Model ID；
- 权威持久化：PostgreSQL `options` 表的两个 `billing_setting.*` 项；
- 计算结果：本节测试输出摘要；
- 未来真实用量：管理员任务日志和用量日志，必须再与 Ark 返回的 `usage.completion_tokens` 对账。

**副作用与清理：** 新增一条 Seedance 2.5 本地计费配置和一条不进入运行镜像的本地回归测试；没有创建渠道、Key 或任务，没有访问方舟、没有产生费用。

**结论：** 后端保存校验、数据库持久化、页面列表模式和四组确定性计算全部一致，`POC-PRICING-EXPR-001` 为 `PASS`。前端编辑弹窗缺陷不影响当前运行时读取，但构成明确的运维限制，必须按上述方式规避。

### 5.5 `POC-ARK-CHANNEL-001`：Seedance 2.5 方舟渠道配置

**状态：** `PASS`

**测试时间：** 2026-08-31 16:10:35 +08:00

**是否访问方舟/产生费用：** 否。这里只创建并读取本地渠道配置，没有发起渠道测试或模型任务。

**为什么测试**

证明 New API 可以用一个公司真实方舟 Key 建立上游渠道，并严格把该渠道限制为 Seedance 2.5。还要确认 Base URL 不重复包含 `/api/v3`，避免任务插件最终生成错误地址。

**前置条件**

- New API 和 PostgreSQL 容器正常运行；
- 管理员已登录；
- 真实方舟 Key 由用户本人直接输入浏览器，不经过聊天、自动化脚本、终端或测试文档。

**如何测试**

1. 在“渠道 → 新建”选择“火山方舟”（内部类型 `45`）。
2. 设置名称 `Ark-Seedance-2.5-POC`，Base URL 保持 `https://ark.cn-beijing.volces.com`。
3. 模型白名单只添加 `doubao-seedance-2-5-260628`；不添加文生图、图生图或其他模型。
4. 用户本人在页面输入真实方舟 Key 并保存。
5. 在渠道列表核对名称存在。
6. 用下方脱敏 SQL 只读取非秘密字段和 `key_present` 布尔值；不得读取 `key` 原文。

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml exec -T postgres `
  psql -U new_api -d new_api -F '|' -Atc "select id,name,type,coalesce(base_url,''),models,status,(key is not null and length(key)>0) as key_present,used_quota,test_time,response_time from channels where name='Ark-Seedance-2.5-POC' order by id;"
```

**预期结果**

- 只存在一条同名渠道；
- 类型为 `45`，Base URL 不带 `/api/v3` 或尾部斜杠；
- `models` 只有 Seedance 2.5 的精确 Model ID；
- `status=1`、`key_present=true`；
- 创建时 `used_quota=0`、`test_time=0`、`response_time=0`，证明尚未执行上游测试。

**实际结果**

```text
1|Ark-Seedance-2.5-POC|45|https://ark.cn-beijing.volces.com|doubao-seedance-2-5-260628|1|t|0|0|0
```

所有预期字段一致，渠道列表也显示该名称。自动化准备点击保存时页面已经返回渠道列表、表单不存在，随后数据库证明确实已经保存，因此没有重复提交。第一次脱敏 SQL 使用了当前表中不存在的旧字段名 `group_name`，第二次尝试又因 PowerShell 对 `group` 列引号转义失败；两次都只是只读查询失败，没有改变数据、没有读取密钥，最终去掉非关键分组列后查询成功。

**结果哪里看**

- 管理员页面：`http://localhost:3000/channels`；
- 权威持久化：PostgreSQL `channels` 表，使用上方脱敏 SQL；
- 后续真实连通与结算：本文件将另建付费任务测试，不能用本节代替。

**副作用与清理：** 创建了一条启用状态的上游渠道，真实 Key 仅保存在 New API 数据库的渠道凭证字段中。本节没有创建虚拟 Key、没有发起任务、没有产生方舟费用。渠道需保留供后续测试使用，不清理。

**结论：** 渠道类型、Base URL、唯一模型白名单、启用状态与密钥存在性全部符合预期，`POC-ARK-CHANNEL-001` 为 `PASS`。这只证明本地配置正确，不证明真实 Key 权限或方舟接口连通；后者必须由后续真实任务测试验证。

### 5.6 `POC-VIRTUAL-KEY-001`：两把独立虚拟 Key 的创建与配置

**状态：** `PASS`

**测试时间：** 2026-08-31 16:17:32 +08:00

**是否访问方舟/产生费用：** 否。创建与核验只修改、读取本地 PostgreSQL，没有发起模型请求。

**为什么测试**

证明管理员可以按员工创建独立虚拟 Key，并为每把 Key 单独设置人民币额度、模型白名单、启停状态和有效期。后续日志与用量表含 `token_id`/`token_name`，因此两把 Key 可以分别对账。

**如何测试**

1. 用户在“API 密钥”页面手工创建两把 Key，数量均为 `1`，避免批量创建自动添加随机后缀。
2. 关闭“无限配额”，每把输入人民币额度 `200`；页面按当前 `QuotaPerUnit=500000` 保存为 `100,000,000` raw quota。
3. 有效期选择“永不”，模型限制只选择 `doubao-seedance-2-5-260628`，IP 白名单留空。
4. 保存后刷新页面，核对两个非秘密名称均出现。
5. 用下方 SQL 只读取配置字段和 `key_present` 布尔值，不读取 `key` 原文。

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml exec -T postgres `
  psql -U new_api -d new_api -F '|' -Atc "select id,name,status,remain_quota,used_quota,unlimited_quota,expired_time,model_limits_enabled,model_limits,(key is not null and length(key)>0) as key_present from tokens where name in ('POC-seedance-A','POC-seedance-B') and deleted_at is null order by name;"
```

**预期结果**

- 两把 Key 均存在并启用，且各有非空密钥；
- `remain_quota=100000000`，即 `¥200`；
- `used_quota=0`、`unlimited_quota=false`；
- `expired_time=-1`，即永不过期；
- 模型限制已启用，且只允许 Seedance 2.5。

**实际结果**

```text
2|POC-seedance-A|1|100000000|0|f|-1|t|doubao-seedance-2-5-260628|t
3|POC-seedance-B|1|100000000|0|f|-1|t|doubao-seedance-2-5-260628|t
```

页面刷新后两个名称各出现一次。用户实际输入的名称为 `POC-seedance-A/B`，与操作说明中的 `POC-Seedance-A/B` 只有字母大小写差异，不影响 Key 身份、路由、额度或日志归属，因此保留实际名称，不做无必要重命名。第一次按说明中的大小写精确查询返回零行；改为实际名称后查询成功，该偏差没有改变任何数据。

**结果哪里看**

- 管理员页面：`http://localhost:3000/keys`；
- 权威持久化：PostgreSQL `tokens` 表，使用上方脱敏 SQL；
- 单 Key 用量：后续真实任务完成后查看管理员用量日志中的 Key 名称，并按 `token_id` 对账。

**副作用与清理：** 新增两把启用状态的 POC 虚拟 Key，每把初始额度 ¥200。完整 Key 未写入聊天、终端输出、Git 或本文档。两把 Key 需保留用于后续隔离、余额与真实任务测试，暂不清理。

**结论：** 两把 Key 的独立额度、模型白名单、有效期、启用状态与密钥存在性全部符合预期，`POC-VIRTUAL-KEY-001` 为 `PASS`。本节只证明本地 Key 管理正确，尚未证明真实请求能按 Key 路由和扣费。

### 5.7 `POC-TEST-SCRIPTS-001`：文生视频调用与人民币对账脚本

**状态：** `PASS`

**测试时间：** 2026-08-31 16:37:57 +08:00

**是否访问方舟/产生费用：** 否。本节使用 Dry Run、本机临时 HTTP 服务和临时 PostgreSQL 夹具验证脚本；真实任务另建测试编号。

**为什么测试**

后续会执行多个付费任务。如果靠临时命令手工提交和抄账，容易泄露 Key、保存视频签名 URL、混淆预估和 actual usage，或无法复现。因此先固化一条可重复、默认脱敏的调用与对账路径。

**实现文件**

- `scripts/test-seedance-text-video.ps1`：固定精确 Model ID，只接受 480p/720p/1080p、4–30 秒；支持 Dry Run；真实模式从进程环境变量读取虚拟 Key，提交后轮询到终态并保存脱敏证据。
- `scripts/show-seedance-reconciliation.ps1`：按公开任务 ID 只读查询必要字段，按 `quota / 500000` 换算人民币，再按 actual tokens 和当前分辨率单价重算。
- `scripts/tests/test-seedance-text-video.tests.ps1`：验证请求构造、官方参数边界、缺少 Key、真实 HTTP 提交/轮询和证据脱敏。
- `scripts/tests/show-seedance-reconciliation.tests.ps1`：验证未知任务、720p ¥70 档和 1080p 当前 ¥55.44 活动价档；临时数据库数据在 `finally` 清理。

**如何测试**

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File scripts/tests/test-seedance-text-video.tests.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File scripts/tests/show-seedance-reconciliation.tests.ps1
```

**预期结果**

- Dry Run 生成正确 endpoint、Model ID、文本内容、画质和时长，且不输出 Key；
- 4K、3 秒和 31 秒在请求前被拒绝；
- 真实模式缺少 `NEW_API_KEY` 时在网络请求前被拒绝；
- 本地 HTTP 夹具返回成功终态后，脚本保留 `completion_tokens=108000` 和 `video_url_present=true`，但证据中没有 Key 或签名 URL；
- 未知任务明确返回 `Task not found`；
- 720p 夹具：108,000 tokens、3,780,000 quota、¥7.56、重算差额 0；
- 1080p 活动价夹具：243,000 tokens、6,735,960 quota、¥13.47192、重算差额 0；
- 测试完成后数据库临时 token/task 均为 0。

**实际结果**

```text
PASS seedance dry-run request
PASS seedance official parameter boundaries
PASS seedance missing-key validation
PASS seedance submit, poll, and sanitized evidence
PASS seedance reconciliation unknown-task behavior
PASS seedance reconciliation CNY and usage behavior
PASS seedance 1080p promotional reconciliation behavior
tokens|0
tasks|0
```

全部断言通过。各生产行为均先观察到预期红灯，再写最小实现变绿。

**偏差与排障记录**

1. 第一版本地 HTTP 夹具在 Windows PowerShell 5.1 使用 `Start-Job` + 阻塞 `AcceptTcpClient()`，清理时 `Stop-Job` 卡住；最小复现证明 PowerShell 7 可停止而 5.1 不可靠。诊断进程被终止，夹具改为隐藏运行、处理两次请求后自行退出的 Python 本地进程，生产脚本未因此改变。
2. PostgreSQL 夹具最初因 Windows→Docker 参数层改写保留字 `group` 的双引号而语法失败；该非关键列被移除。
3. 直接传 JSON 字符串也被同一参数层改写；改为 PostgreSQL `json_build_object(...)` 原生构造。一次已创建但未进入清理块的测试 token（`id=4`、名称 `reconciliation-fixture`）随后被精确删除，最终复核临时 token/task 均为 0。

**结果哪里看**

- 自动化断言：两个 `scripts/tests/*.tests.ps1` 文件及上方最终输出；
- 真实调用脱敏证据：后续产生于 Git 忽略目录 `artifacts/poc/`；
- 操作步骤：[`local-poc.md`](./local-poc.md) 的“Seedance 2.5 文生视频测试脚本”。

**副作用与清理：** 新增两个运行脚本和两个测试脚本；测试只使用本机回环 HTTP 与已清理的数据库夹具。没有创建方舟任务、没有产生费用、没有留下临时 token/task，也没有写入任何真实 Key。

**结论：** 调用、轮询、参数限制、脱敏证据和人民币对账路径具备可重复的本地测试证据，`POC-TEST-SCRIPTS-001` 为 `PASS`。下一步才可用两把 POC 虚拟 Key 执行受控真实矩阵。

## 6. 能力矩阵

| 模式 | 分辨率 | 创建 | 终态 | actual usage | 账单核对 | 结论 |
| --- | --- | --- | --- | --- | --- | --- |
| 文生视频 | 720p | PENDING | PENDING | PENDING | PENDING | PENDING |
| 图生视频 | 720p | PENDING | PENDING | PENDING | PENDING | PENDING |
| 文生视频能力探测 | 1080p | PENDING | PENDING | PENDING | PENDING | PENDING |

## 7. 真实调用案例

每个案例只记录虚拟 Key 名称、脱敏任务 ID/证据文件、时间、终态、actual usage、预占人民币、最终扣费人民币和剩余人民币。不得记录 Key 值或完整输出 URL 查询参数。

### 7.1 720p 文生视频

PENDING

### 7.2 720p 图生视频

PENDING

### 7.3 1080p 能力探测

PENDING

### 7.4 同步拒绝与退款

PENDING

### 7.5 终态失败与退款

PENDING；若无法安全、确定地复现，必须明确记录“not reproducible safely”，不得伪造证据。

### 7.6 额度不足且未访问 Ark

PENDING

## 8. 跨 Key 隔离

- Key B 查询 Key A 任务：PENDING
- 响应不得包含任务内容、输出地址或 usage：PENDING
- Key A 查询自身任务：PENDING

## 9. 重启恢复与结算幂等

- 运行中任务在 New API 重启后继续：PENDING
- 同一终态任务重复查询 20 次余额不变：PENDING
- 容器 stop/start 后任务、日志和余额仍存在：PENDING

## 10. 并发原子预占

- 请求数：20
- 可支付保守预占次数：1
- 实际接受数：PENDING
- 实际 Ark 上游任务数：PENDING
- 是否出现负余额：PENDING
- 最终结算次数：PENDING

## 11. 余额对账

| 案例 | 预估 usage | 预占 RMB | actual usage | 官方公式重算 RMB | 实扣 RMB | 差异 | 结果 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 720p 文生视频 | PENDING | PENDING | PENDING | PENDING | PENDING | PENDING | PENDING |
| 720p 图生视频 | PENDING | PENDING | PENDING | PENDING | PENDING | PENDING | PENDING |
| 1080p 探测 | PENDING | PENDING | PENDING | PENDING | PENDING | PENDING | PENDING |

允许的最大舍入差为一个 raw quota 点，即 `¥0.000002`。最终费用不得高于保守预占。

## 12. 最终 GO / NO-GO

PENDING：全部硬门槛完成后，只能写入计划规定的 GO 或 NO-GO 原句。
