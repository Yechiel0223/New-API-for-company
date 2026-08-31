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
| 本项目修复提交 | `b227cf5`、`e0fb2a1` | PASS；分别修复跨虚拟 Key 查询隔离，以及 Seedance 预占终点帧和查询断线恢复 |
| 本机构建 New API 镜像 | `sha256:2cefce1389ba110509cafd840caf65ebad53e9658ec0987f62b10ebac0e7ad70` | PASS，包含上述两项最小修复；旧镜像为 `sha256:ba17b5970edf48e3adbae8c79f636b8754cb14d94ad9cc8a53b99f617db32de9` |
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
| 最终本机账本快照 | PASS | 10 个任务且活动任务 0；52 条日志；渠道/管理员已用均为 `97672169` quota = ¥195.344338；Key A/B 已用合计与其完全一致 |

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
| 管理员基础额度 | 作为基础设施上限，远高于各虚拟 Key | PASS | 管理员页面把剩余额度覆盖为 ¥1,000,000；单 Key 额度仍是日常业务限制 |
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

### 4.2 `POC-OWNER-QUOTA-001`：共享管理员基础额度

**状态：** `PASS`

**为什么测试：** New API 会同时扣虚拟 Key 额度和其所属用户额度。公司只创建一个管理员，若管理员额度与单 Key 同量级，不同员工即使各自 Key 仍有余额，也会因共享用户额度耗尽而互相阻塞。

**实际发现：** 管理员初始额度为 ¥200。完成前六笔真实任务后，其剩余额度只有 ¥108.685224；第一次 1080p/60 秒诊断请求因此先命中共享用户额度，而没有命中预期的 Key B 额度限制。该请求没有创建任务或产生方舟费用，但证明管理员额度不是纯展示字段。

**处理与验证：** 在管理员“用户管理 → 编辑 root → 调整额度”中选择覆盖，把管理员**剩余额度**设为 ¥1,000,000。页面提示“调整额度成功”。此后合法 1080p/30 秒请求能够准确命中 Key B 自身余额限制；20 路并发也只由 Key B 的原子预占决定。管理员页面“总额度”是剩余额度与历史已用额度之和，所以设置后显示约 ¥1,000,091.31 是正常口径，不是多充。

**当前复核（完成全部本文测试后）：** 管理员 `quota=499947985219`、`used_quota=97672169`，即剩余 ¥999,895.970438、已用 ¥195.344338；两者随成功任务同步变化。管理员额度仅作为高水位基础设施上限，按员工统计仍以 `tokens`、`tasks` 和 `logs` 为准。生产运行必须监控这个上限，不能误以为它是无限额度。

### 4.3 `POC-KEY-GUARDS-001`：禁用、过期和模型白名单本地拦截

**状态：** `PASS`

**测试时间：** 2026-09-01 00:11（Asia/Shanghai）

**是否访问方舟/产生费用：** 否。三种请求均在 Token 鉴权/权限阶段被本地拒绝。

**为什么测试：** 管理员对单 Key 的停用、到期和模型限制必须先于方舟调用生效，否则即使页面能编辑这些字段，也不能作为真实管理手段。

**如何测试：** 使用 Key A 的合法 720p/4 秒请求作为探针。依次临时禁用 Key、把到期时间设为过去、恢复后改用白名单外模型 `doubao-seedance-2-0-260128`。每次只记录状态码和脱敏错误；原状态、到期时间均在 `finally` 恢复。过期 Key 首次请求后，New API 会自动把状态改为 `3`，因此恢复时必须同时恢复 `status=1` 与 `expired_time=-1`。

**实际结果：** 禁用和过期均返回 HTTP 401 `Invalid token`；恢复后，白名单外模型返回 HTTP 403 `permission_denied`，消息为该 Token 无权访问对应模型。测试前后均为 Key A `remain=84645612`、`used=15354388`、任务数 8、该 Key 日志数 7、渠道累计 `89184781`；没有任务、余额、用户额度、渠道用量或计费日志变化。最终状态复核为 `status=1`、`expired_time=-1`、模型白名单仍仅包含 `doubao-seedance-2-5-260628`。

**结论：** 三项单 Key 管理动作均在访问方舟前可靠生效，且测试状态已完整恢复。

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
| 模式 | 文生视频、首帧生视频、首尾帧生视频、全模态参考生视频等 | PASS；当前只验收文生视频，图像输入延后 |
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

**结论：** `PASS`。官方资料足以确认当前模型 ID、文生视频能力、720p 价格和 1080p 探测价格。当前明确不配置、不验收视频或图像作为输入；4K 必须视为不支持，不能因 New API 通用表单或插件枚举中出现 4K 就对员工宣称可用。

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
| 提交时 token 估算 | 上游原实现按 `秒 × 宽 × 高 × 24 / 1024`；真实六组任务一致证明输出还包含终点帧，已修正为 `(秒 × 24 + 1) × 宽 × 高 / 1024` | PASS；见 `e0fb2a1` 与三档修复后复测 |
| 终态 actual usage | 成功时优先读取 `usage.completion_tokens`，缺失时才回退 `total_tokens` | PASS |
| 失败终态 | 完成用量钩子不返回收费事实，任务结算链路具备 CAS 防重退款 | PASS（代码级）；真实失败无法安全、确定复现，见 7.9 |
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
- 例如 720p/5 秒无视频输入、actual usage 为 108,900 token：`108900 × 70 / 1000000 = ¥7.623`，对应 `3,811,500` quota；
- 终态对账允许的最大本地量化误差为一个 raw quota 点，即 `¥0.000002`；不得把请求时估算值当成最终账单。
- 提交预占使用修正后的终点帧公式；成功终态仍以方舟 actual usage 为唯一费用依据。480p 最大像素计算会产生小数 token，必须保留小数到最终 quota 四舍五入，不能提前截断。

### 5.4 `POC-PRICING-EXPR-001`：Seedance 2.5 第一阶段计费表达式

**状态：** `PASS`

**测试对象：** New API 管理员页面中模型 `doubao-seedance-2-5-260628` 的任务用量表达式。

**是否访问方舟/产生费用：** 否。该测试只保存并校验本地计费配置。

**为什么测试**

证明当前文生视频范围可以完全复用 New API 的任务用量表达式，按官方 actual token 和分辨率计算人民币，而不使用不精确的内置 1080p 倍率。图像输入和视频输入均不在本轮范围内。

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
| 480p、48,437.8125 token | `480p_720p` | `¥3.390646875` | `1,695,323` | PASS |
| 720p、108,900 token | `480p_720p` | `¥7.623` | `3,811,500` | PASS |
| 1080p、245,025 token | `1080p_promo` | `¥13.584186` | `6,792,093` | PASS |
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

表达式夹具保留在 `pkg/billingexpr/seedance_pricing_poc_test.go`。插件预占夹具保留在 `plugins/doubao_usage_test.go`，覆盖 480p/5 秒、720p/4/5/10/30 秒和 1080p/5 秒六组真实观察值。两者均不访问方舟；插件测试会进入运行镜像对应源码。1080p 优惠结束切换到刊例价时，必须同时更新运行配置、表达式测试中的单价/期望值和本节证据。

插件预占回归命令：

```powershell
docker run --rm -e GOPROXY=https://goproxy.cn,direct `
  -v 'D:\new-api:/src' -w /src golang:1.26.1-alpine `
  go test ./plugins -run '^TestDoubaoSeedanceSubmitEstimateIncludesTerminalFrame$' -count=1 -v
```

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
- 720p 夹具：提交估算 108,000 tokens / ¥7.56、终态 actual 109,000 tokens / ¥7.63，对账必须优先 actual usage 且重算差额为 0；
- 1080p 活动价夹具：243,000 tokens、6,735,960 quota、¥13.47192、重算差额 0；
- 测试完成后数据库临时 token/task 均为 0。

**实际结果**

```text
PASS seedance dry-run request
PASS seedance official parameter boundaries
PASS seedance missing-key validation
PASS seedance submit, transient query retry, poll, and sanitized evidence
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
4. 第一笔真实任务证明 `private_data.billing_context.tiered_snapshot.usage_facts.tokens` 保留提交时估算，而 actual usage 位于 `tasks.data.usage.completion_tokens`。旧对账脚本因此把 48,037.5 的估算误当 actual，并错误报告 ¥0.027965 差异；运行时任务 quota 和差额结算日志实际正确。新增“估算 108,000、actual 109,000”回归夹具，先观察失败，再改为优先读取 `tasks.data.usage.completion_tokens`、其次 `total_tokens`、最后才回退估算快照。真实任务复核后差异为 0。
5. 720p / 30 秒真实任务轮询期间，PostgreSQL 容器发生一次干净重建，一次 GET 恰好在数据库约 2 秒不可用窗口内返回 500。任务本身仍在方舟运行，New API 后台轮询在数据库恢复后继续并完成差额结算。针对该事实新增“首次 GET 返回 500、第二次成功”的测试夹具，先观察脚本失败，再实现仅对查询阶段 5xx 的自动重试；断言实际请求序列必须为 `POST,GET,GET`，确保不会重复创建付费任务。脚本证据新增 `query_retry_count`。容器事件能证明重建发生，但不能从现有证据确定是谁或哪个进程发起，因此不做无依据归因。
6. New API 运行中任务重启测试进一步证明，轮询还会遇到“没有 HTTP 响应”的连接错误。脚本现对查询阶段的无响应或 HTTP 5xx 重试，但仍绝不自动重试 POST。真实 720p/4 秒任务在约 4.6 秒重启窗口内累计 `query_retry_count=4`，恢复后成功且只创建、结算一次；见第 9 节。
7. 修复前六组真实任务的 actual usage 均比原 `秒 × 24` 估算多一个输出帧：720p 每次多 900 tokens，1080p 多 2,025 tokens，480p 约多一个 854×480 帧。新增六向量红灯测试后，将提交估算改为 `秒 × 24 + 1` 帧；修复后三档真实复测均未低估，见第 11 节。

**结果哪里看**

- 自动化断言：两个 `scripts/tests/*.tests.ps1` 文件及上方最终输出；
- 真实调用脱敏证据：后续产生于 Git 忽略目录 `artifacts/poc/`；
- 操作步骤：[`local-poc.md`](./local-poc.md) 的“Seedance 2.5 文生视频测试脚本”。

**副作用与清理：** 新增两个运行脚本和两个测试脚本；测试只使用本机回环 HTTP 与已清理的数据库夹具。没有创建方舟任务、没有产生费用、没有留下临时 token/task，也没有写入任何真实 Key。

**结论：** 调用、轮询、参数限制、脱敏证据和人民币对账路径具备可重复的本地测试证据，`POC-TEST-SCRIPTS-001` 为 `PASS`。下一步才可用两把 POC 虚拟 Key 执行受控真实矩阵。

## 6. 能力矩阵

| 模式 | 分辨率 | 创建 | 终态 | actual usage | 账单核对 | 结论 |
| --- | --- | --- | --- | --- | --- | --- |
| 文生视频 | 480p | PASS | PASS | 48,437 tokens | ¥3.39059，差异 0 | PASS |
| 文生视频 | 720p | PASS | PASS | 4 秒 87,300；5 秒 108,900；10 秒 216,900；30 秒 648,900 tokens | 四笔差异均为 0 | PASS |
| 图生视频 | 720p | DEFERRED | DEFERRED | DEFERRED | DEFERRED | 当前明确只验收文生视频；启用前必须单独测试 |
| 文生视频能力探测 | 1080p | PASS | PASS | 245,025 tokens | ¥13.584186，差异 0 | PASS |

## 7. 真实调用案例

每个案例只记录虚拟 Key 名称、脱敏任务 ID/证据文件、时间、终态、actual usage、预占人民币、最终扣费人民币和剩余人民币。不得记录 Key 值或完整输出 URL 查询参数。

### 7.1 `POC-T2V-480P-5S-001`：480p / 5 秒文生视频

**状态：** `PASS`

**虚拟 Key：** `POC-seedance-A`（`token_id=2`）

**时间：** 2026-08-31 16:40:49 至 16:43:18（Asia/Shanghai），约 149 秒。

**是否访问方舟/产生费用：** 是。创建一条真实 Seedance 2.5 文生视频任务，最终 New API 扣费 ¥3.39059；按官方 actual tokens 与 ¥70/百万 token 重算同为 ¥3.39059。

**为什么测试**

验证公司真实方舟 Key 权限、类型 45 渠道路由、Seedance 插件路径、虚拟 Key 鉴权、异步轮询、视频产物、actual usage 差额结算和人民币余额形成完整闭环。

**请求条件**

- Model ID：`doubao-seedance-2-5-260628`；
- 画质：480p；
- 时长：5 秒；
- 模式：纯文本输入，无图片、无视频输入；
- 提示词：`A red paper airplane flying smoothly across a clean white studio background, fixed camera, no text, no logo`。

**实际结果与证据**

- 公开任务 ID：`task_mqkYajIjhq1D30OQPihfoY0UJTxmkzBG`；
- 终态：`SUCCESS` / API `succeeded`；
- `video_url_present=true`，不保存完整签名 URL；
- 脱敏调用证据：`artifacts/poc/seedance-task_mqkYajIjhq1D30OQPihfoY0UJTxmkzBG.json`（Git 忽略）；
- 渠道：`channel_id=1`；动作：`text_to_video`；模型、Key 名称和 token_id 均正确。

| 指标 | 值 |
| --- | ---: |
| 提交估算 tokens | 48,037.5 |
| 预占 raw quota | 1,681,313 |
| 预占人民币 | ¥3.362626 |
| `usage.completion_tokens` | 48,437 |
| 最终 raw quota | 1,695,295 |
| 最终扣费 | ¥3.39059 |
| official usage 重算 | `48437 × 70 / 1000000 = ¥3.39059` |
| 最终重算差异 | ¥0.000000 |
| Key A 剩余 | ¥196.60941 |

差额结算日志包含初始消费 `1,681,313` quota 和补扣 `13,982` quota，并记录 `pre_consumed_quota=1681313`、`actual_quota=1695295`。两者合计等于任务最终 quota；没有重复结算。

**测试中发现并修复的问题：** 初版对账脚本从计费快照读取 48,037.5 的估算，而不是从任务数据读取 48,437 actual tokens，因此曾错误显示账单不匹配。运行时扣费本身一直正确。修复过程和回归测试见 5.7；修复后本案例 `usage_source=task.data.usage.completion_tokens`、`billing_matches=true`。

**副作用与清理：** 保留一条成功任务、两条消费/差额日志和 ¥3.39059 的真实用量审计记录，不删除、不退款；这是 POC 的有效账单证据。完整 Key 与签名 URL均未进入 Git、聊天或终端输出。

**结论：** 真实 480p 文生视频、视频产物、actual usage 差额结算和人民币对账全部通过，`POC-T2V-480P-5S-001` 为 `PASS`。

### 7.2 `POC-T2V-720P-5S-001`：720p / 5 秒文生视频

**状态：** `PASS`

**虚拟 Key：** `POC-seedance-A`（`token_id=2`）

**时间：** 2026-08-31 16:47:22 至 16:50:15（Asia/Shanghai），约 174 秒。

**为什么测试：** 与 480p 案例保持相同提示词和 5 秒时长，只改变分辨率，验证 720p 路由、产物、actual usage 和 ¥70/百万 tokens 账单。

**如何测试：** 使用 `scripts/test-seedance-text-video.ps1`，参数为 `-Resolution 720p -Duration 5`；完成后将公开任务 ID 传给 `scripts/show-seedance-reconciliation.ps1`。调用时仅通过进程环境变量传入 Key，执行结束立即移除。

**预期结果：** 任务成功且存在视频；最终费用必须以 `usage.completion_tokens` 重算，New API 实扣和官方公式差异不超过 ¥0.000002。

**实际结果与证据：**

- 任务 ID：`task_0ebJokMerRTUVTH0EGQzLloZQJqjY3am`；终态 `SUCCESS`；`video_url_present=true`；
- 脱敏证据：`artifacts/poc/seedance-task_0ebJokMerRTUVTH0EGQzLloZQJqjY3am.json`（Git 忽略）；
- 估算 108,000 tokens，预占 ¥7.56；actual 108,900 tokens；
- 最终实扣 `108900 × 70 / 1000000 = ¥7.623`，重算差异 ¥0；
- 本任务结算后 Key A 剩余 ¥188.98641，累计已用 ¥11.01359。

**副作用与清理：** 产生一条真实成功任务和 ¥7.623 方舟用量；任务与账单保留用于审计，不退款、不删除。完整 Key 与签名 URL未保存。

**结论：** 创建、终态、产物、actual usage 和人民币对账全部通过。

### 7.3 `POC-T2V-720P-4S-001`：720p / 4 秒最短时长

**状态：** `PASS`

**虚拟 Key：** `POC-seedance-B`（`token_id=3`）

**时间：** 2026-08-31 16:58:45 至 17:01:18（Asia/Shanghai），约 153 秒。

**为什么测试：** 验证官方允许的 4 秒下边界可以通过网关创建、完成并按实际用量结算，同时开始验证第二把虚拟 Key 的独立归属和余额。

**如何测试：** 使用与 5 秒案例相同提示词运行 `scripts/test-seedance-text-video.ps1 -Resolution 720p -Duration 4`，再使用对账脚本按任务 ID读取脱敏账单。

**预期结果：** 4 秒请求被接受并成功生成；任务必须归属 Key B；费用按 ¥70/百万 actual tokens 结算，差异不超过一个 raw quota 点。

**实际结果与证据：**

- 任务 ID：`task_ppb1LyIKAx6T5F9osO09tS9pr9Ie74BD`；终态 `SUCCESS`；`video_url_present=true`；
- 脱敏证据：`artifacts/poc/seedance-task_ppb1LyIKAx6T5F9osO09tS9pr9Ie74BD.json`（Git 忽略）；
- 估算 86,400 tokens，预占 ¥6.048；actual 87,300 tokens；
- 最终实扣 `87300 × 70 / 1000000 = ¥6.111`，重算差异 ¥0；
- `token_id=3`、名称 `POC-seedance-B` 均正确；任务后 Key B 剩余 ¥193.889。

**副作用与清理：** 产生一条真实成功任务和 ¥6.111 方舟用量；保留任务与账单审计记录。

**结论：** 4 秒下边界、第二把 Key 归属、产物和人民币对账全部通过。

### 7.4 `POC-T2V-720P-10S-001`：720p / 10 秒文生视频

**状态：** `PASS`

**虚拟 Key：** `POC-seedance-B`（`token_id=3`）

**时间：** 2026-08-31 17:01:58 至 17:04:47（Asia/Shanghai），约 168 秒。

**为什么测试：** 在相同画质和提示词下增加输出时长，验证 token 用量、预占与最终扣费会随时长增长，并且较长任务仍能稳定轮询到终态。

**如何测试：** 运行 `scripts/test-seedance-text-video.ps1 -Resolution 720p -Duration 10`，随后按任务 ID运行人民币对账脚本。

**预期结果：** 任务成功并返回视频；actual usage 大于同条件 4/5 秒任务；实扣必须与官方 actual usage 公式完全一致或仅有一个 raw quota 点量化误差。

**实际结果与证据：**

- 任务 ID：`task_g5zFGbOqwtwaYL9XChDMLJUhYQ04C2pr`；终态 `SUCCESS`；`video_url_present=true`；
- 脱敏证据：`artifacts/poc/seedance-task_g5zFGbOqwtwaYL9XChDMLJUhYQ04C2pr.json`（Git 忽略）；
- 估算 216,000 tokens，预占 ¥15.12；actual 216,900 tokens；
- 最终实扣 `216900 × 70 / 1000000 = ¥15.183`，重算差异 ¥0；
- actual usage 高于同画质 4 秒和 5 秒案例；任务后 Key B 剩余 ¥178.706，累计已用 ¥21.294。

**副作用与清理：** 产生一条真实成功任务和 ¥15.183 方舟用量；保留任务与账单审计记录。

**结论：** 10 秒任务稳定完成，用量增长关系合理，actual usage 与人民币扣费完全一致。

### 7.5 `POC-T2V-720P-30S-001`：720p / 30 秒最长时长与瞬时数据库中断恢复

**状态：** `PASS`

**虚拟 Key：** `POC-seedance-B`（`token_id=3`）

**时间：** 2026-08-31 17:05:43 至 17:10:29（Asia/Shanghai），约 286 秒。

**为什么测试：** 验证官方允许的 30 秒上边界、最大时长下的预占与 actual usage 结算。真实执行中又遇到 PostgreSQL 短时重建，因此同时验证任务不会因一次本地查询 500 丢失或重复提交，数据库恢复后后台能够继续结算。

**如何测试：** 使用同一提示词运行 `scripts/test-seedance-text-video.ps1 -Resolution 720p -Duration 30`。脚本已成功 POST 并持续 GET；17:08:54 的一次 GET 因数据库拒绝连接返回 500。没有重新 POST，而是先通过任务表和容器日志定位原任务，再在数据库恢复后使用相同任务 ID查询终态，并运行人民币对账脚本。随后以真实故障为夹具补齐查询 5xx 自动重试回归测试。

**预期结果：** 30 秒请求成功产生视频；最终按 actual tokens 结算；瞬时本地查询失败不得导致第二次 POST、任务丢失、重复扣费或未结算。

**实际结果与证据：**

- 任务 ID：`task_O9uurate4hsEgYkmoFvEzWkwcEkyheL7`；终态 `SUCCESS`；`video_url_present=true`；
- 脱敏证据：`artifacts/poc/seedance-task_O9uurate4hsEgYkmoFvEzWkwcEkyheL7.json`（Git 忽略）；该文件在故障恢复后由任务行和脱敏 API 结果重建，明确标记 `evidence_recovered_after_transient_query_error=true`；
- 估算 648,000 tokens，预占 ¥45.36；actual 648,900 tokens；
- 最终实扣 `648900 × 70 / 1000000 = ¥45.423`，重算差异 ¥0；
- 任务后 Key B 剩余 ¥133.283，累计已用 ¥66.717；
- New API 请求日志只出现一次该任务 POST；查询序列在 17:08:54 出现一次 500，17:10:02 恢复为 200；17:10:29 后台记录一次差额结算 `¥0.063`；
- PostgreSQL 使用原有 named volume 恢复，任务、Key、余额和历史日志均未丢失；容器 `OOMKilled=false`，重建原因的发起方未能从事件记录确定。

**脚本修复验证：** 本地 HTTP 夹具固定返回 `POST 200 → GET 500 → GET 200 succeeded`。修复前脚本在首次 GET 500 处失败；修复后输出 `query_retry_count=1`，请求日志严格为 `POST,GET,GET`，证明只恢复查询而不重复创建付费任务。

**副作用与清理：** 产生一条真实成功任务和 ¥45.423 方舟用量；数据库容器发生一次计划外干净重建，但持久化数据未丢失。任务和账单保留；临时 HTTP 夹具已删除。

**结论：** 30 秒上边界、actual usage 结算、数据库短时不可用后的任务恢复和防重复 POST 全部满足通过条件。

### 7.6 720p 图生视频

DEFERRED。当前范围明确为 Seedance 2.5 文生视频，图像输入暂不启用。本项不作为当前本机文生视频 POC 的阻塞项，但未来开放图生视频前必须重新执行创建、产物、actual usage、预占、终态费用和跨 Key 查询的完整验收，不能沿用文生视频结论。

### 7.7 `POC-T2V-1080P-5S-001`：1080p / 5 秒能力与活动价探测

**状态：** `PASS`

**虚拟 Key：** `POC-seedance-A`（`token_id=2`）

**时间：** 2026-08-31 16:50:46 至 16:58:15（Asia/Shanghai），约 449 秒。

**为什么测试：** 验证 Seedance 2.5 的 1080p 能力真实可用，并验证当前限时活动单价 ¥55.44/百万 tokens 已被正确应用，而不是错误使用 New API 旧倍率或刊例价 ¥77。

**如何测试：** 使用同一提示词运行 `scripts/test-seedance-text-video.ps1 -Resolution 1080p -Duration 5`，随后用对账脚本读取任务的 actual usage、分辨率档位、最终 quota 和余额。

**预期结果：** 任务成功、有视频产物；计费档位为 1080p 活动价；最终实扣与 `actual tokens × 55.44 / 1000000` 一致。

**实际结果与证据：**

- 任务 ID：`task_iwUUvEhIEongrQNXrRDRK8zdEPvPd0wW`；终态 `SUCCESS`；`video_url_present=true`；
- 脱敏证据：`artifacts/poc/seedance-task_iwUUvEhIEongrQNXrRDRK8zdEPvPd0wW.json`（Git 忽略）；
- 估算 243,000 tokens，预占 ¥13.47192；actual 245,025 tokens；
- 最终实扣 `245025 × 55.44 / 1000000 = ¥13.584186`，重算差异 ¥0；
- 本任务后 Key A 剩余 ¥175.402224，累计已用 ¥24.597776。

**副作用与清理：** 产生一条真实成功任务和 ¥13.584186 方舟用量；任务与账单保留。活动价有截止时间，后续切价必须遵守 5.2 的停单 SOP。

**结论：** 1080p 实际能力、视频产物、活动价分档和最终对账全部通过。

### 7.8 `POC-SYNC-REJECT-001`：上游同步拒绝与全额返还

**状态：** `PASS`

**是否访问方舟/产生费用：** 请求到达方舟参数校验，但没有创建任务，因此没有生成费用。

**为什么测试：** New API 会先预占再向方舟提交。如果上游同步拒绝请求，预占必须完整返还，不能留下幽灵任务或渠道用量。

**如何测试：** 通过 Key B 直接提交 Seedance 2.5 不支持的 4K/5 秒请求，前后读取 Key 余额、任务数、该 Key 日志数和渠道累计用量。

**实际结果：** HTTP 400，错误码 `invalid_request`，上游原因为分辨率参数无效。New API 日志先记录预占 ¥68.04，随后明确记录“请求失败, 返还预扣费”。前后均为 Key B `remain=66641500`、`used=33358500`、任务数 6、渠道累计 `45657388`；该 Key 日志数从 6 增至 7，只增加一条 `quota=0` 的失败日志。没有任务、余额或渠道费用变化。

**结论：** 上游同步拒绝会全额返还预占且不创建任务，`PASS`。

### 7.9 `POC-TERMINAL-FAILURE-001`：终态失败与幂等退款

**状态：** `NOT RUN / not reproducible safely`（真实方舟）；`PASS`（代码级确定性测试）

真实 Seedance 终态失败无法在不故意触发内容安全、破坏网络或制造不可控上游费用的情况下稳定复现，因此没有伪造真实失败证据，也不把同步 400 冒充异步终态失败。

代码级测试覆盖 CAS 退款胜者/败者、阶梯计费失败返回调用方退款、后台轮询四种终态，以及失败任务只退款一次：

```powershell
docker run --rm -e GOPROXY=https://goproxy.cn,direct `
  -v new-api-go-mod-cache:/go/pkg/mod `
  -v new-api-go-build-cache:/root/.cache/go-build `
  -v 'D:\new-api:/src' -w /src golang:1.26.1-alpine `
  go test ./service -run '^(TestUpdateBatchTasksSettlesTieredUsageForTerminalStates|TestUpdateBatchTasksRefundsFailedTieredTask|TestCASGuardedRefund_Win|TestCASGuardedRefund_Lose|TestSettle_TieredFailureReturnsFalseForCallerRefund)$' -count=1 -v
```

上述五项及其子测试全部 `PASS`。这证明本地结算实现，但不替代未来自然发生失败任务时的真实方舟复核；生产观察到第一笔真实失败后，必须补记方舟终态、actual usage、退款和重复轮询证据。

### 7.10 `POC-QUOTA-DENY-001`：合法请求额度不足且未访问 Ark

**状态：** `PASS`

**为什么测试：** 单 Key 余额必须是实际硬限制；余额不足的合法请求应在方舟任务创建前被拒绝，且不改变任何账务累计。

**如何测试：** 修正预占公式后，Key B 剩余 ¥52.339214。提交官方合法边界内的 1080p/30 秒文生视频，所需预占为 1,460,025 tokens × ¥55.44/百万 = ¥80.943786。前后读取 Key 余额、任务数、Key 日志数、渠道与管理员额度。

**实际结果：** HTTP 403 `permission_denied`，错误明确显示剩余 ¥52.339214、需要 ¥80.943786。前后均为 Key B `remain=26169607`、`used=73830393`、任务数 8、渠道累计 `89184781`、管理员 `quota=499956472607` / `used=89184781`；Key B 日志数仅从 30 增至 31，新增日志 `id=49`、`type=5`、`quota=0`。无任务、无余额变化、无渠道用量变化，证明没有产生方舟任务或费用。

此前曾用 1080p/60 秒得到同类本地拒绝，但 60 秒超出官方 30 秒上限，故只保留为诊断记录，不作为本项验收证据。

### 7.11 `POC-PRECHARGE-FRAME-001`：终点帧修复后三档真实复测

**状态：** `PASS`

**测试时间：** 2026-09-01 00:03 至 00:15（Asia/Shanghai）

**为什么测试：** 修复前所有真实任务的 actual usage 均比预占多一个输出帧，违反“合法请求最终费用不得高于预占”的硬门槛。单元测试不足以证明方舟真实计量，必须重建镜像后覆盖全部三档画质。

**如何测试：** 使用镜像 `sha256:2cefce1389ba110509cafd840caf65ebad53e9658ec0987f62b10ebac0e7ad70`，分别运行 480p/5 秒、720p/4 秒和 1080p/5 秒文生视频；480p 与 1080p 并行提交并使用不同提示词。逐笔用 `show-seedance-reconciliation.ps1` 对账。

| 分辨率/时长 | 脱敏任务 ID | 预占 | actual / 实扣 | 结果 |
| --- | --- | --- | --- | --- |
| 480p/5 秒 | `task_dINjWtEyshIPAIgtYyPCa1otNl3SiwE2` | 48,437.8125 tokens；1,695,323 quota = ¥3.390646 | 48,437；1,695,295 quota = ¥3.390590；退回 ¥0.000056 | PASS |
| 720p/4 秒 | `task_vKSCicqBR3xZmP5Cc2s8NqQgDZj4Vdbu` | 87,300 tokens；¥6.111000 | 87,300；¥6.111000 | PASS |
| 1080p/5 秒 | `task_DC03132iMUYI5sbPQZlmP5PaY33uPnhu` | 245,025 tokens；¥13.584186 | 245,025；¥13.584186 | PASS |

三笔均为 `SUCCESS`、`video_url_present=true`、`billing_matches=true`。480p 保留小数 token 到最终 quota 四舍五入，因此产生 28 raw quota（¥0.000056）的保守返还；720p 和 1080p 预占与 actual 完全相等。三档均未再发生超额扣款。

## 8. 跨 Key 隔离

### 8.1 `POC-KEY-ISOLATION-001`：同一管理员下的虚拟 Key 任务隔离

**状态：** `PASS`（发现问题、回归测试、最小修复、完整测试和真实复测均完成）

**测试时间：** 2026-08-31 17:18 至 17:34（Asia/Shanghai）。

**是否访问方舟/产生费用：** 否。只查询已完成任务；没有创建上游任务。

**为什么测试**

公司所有虚拟 Key 都归属于同一个 New API 管理员用户。仅按 `user_id` 隔离不足以防止员工 B 用已知任务 ID读取员工 A 的广告视频、状态和 usage，因此必须验证并强制按虚拟 `token_id` 隔离。

**如何测试**

1. 使用 Key B 查询由 Key A 创建的 `task_mqkYajIjhq1D30OQPihfoY0UJTxmkzBG`，只记录 HTTP 状态和响应是否含任务 ID、usage/视频字段，不保存完整响应或签名 URL。
2. 使用 Key A 查询同一任务，确认所有者仍能读取成功终态、48,437 actual tokens 和视频存在性。
3. 在单元测试中增加同一 `user_id`、同一插件平台但不同 `token_id` 的任务，以及没有 token_id 的旧任务；两者都必须返回与不存在任务相同的 404。
4. 最小修改查询边界：保留原有 `user_id + platform + task_id` 条件，再要求任务私有计费归属的 `TokenId` 等于当前认证 Key 的 token_id；不修改管理员页面、计费或渠道逻辑。
5. 运行定向测试和完整 `go test ./middleware -count=1`，重建镜像并仅替换 New API 容器，再重复步骤 1–2。

**预期结果：** Key B 得到不泄露任务是否存在的 404，响应不含目标任务 ID、usage 或视频字段；Key A 查询仍为 200。完整 middleware 测试必须通过。

**实际结果**

| 阶段 | Key B 查 Key A | 响应泄露 | Key A 自查 | 结果 |
| --- | --- | --- | --- | --- |
| 修复前真实服务 | HTTP 200 | 包含状态、usage 和视频结果 | HTTP 200 | FAIL，确认缺口 |
| 新增回归测试 | 不同 token / 无 token 任务均实际返回 200 | 返回任务 ID | 所有者 200 | 预期红灯 |
| 修复后单元测试 | 两类均 404 | 与不存在任务响应一致 | 所有者 200 | PASS |
| 修复后真实服务 | HTTP 404 | `contains_task_id=false`、`contains_usage_or_video=false` | `succeeded`、48,437 tokens、视频存在 | PASS |

定向测试和完整 middleware 包均通过：

```text
--- PASS: TestPrepareTaskPluginStaticQueryHidesTaskExistenceAndSanitizesErrors
PASS
ok github.com/QuantumNous/new-api/middleware

ok github.com/QuantumNous/new-api/middleware 0.362s
```

**代码与运行证据：** 修复提交 `b227cf5`；运行镜像 `sha256:ba17b5970edf48e3adbae8c79f636b8754cb14d94ad9cc8a53b99f617db32de9`。真实 Key 只在进程变量中使用，未输出或写入文档。

**副作用与清理：** 重建并替换一次 New API 容器；PostgreSQL 容器和数据卷未改变。没有产生方舟费用。修复保留为本项目相对上游的必要补丁，升级 New API 时必须重新验证或确认上游已包含等价修复。

**结论：** 虚拟 Key 之间的异步任务读取已从真实 FAIL 修复为 PASS；员工只能用自己的 Key 查询自己的任务。

## 9. 重启恢复与结算幂等

### 9.1 `POC-NEWAPI-RESTART-001`：运行中重启 New API

**状态：** `PASS`

**测试时间：** 2026-09-01 00:03:40 至 00:07:28（Asia/Shanghai）

**方法与结果：** Key A 提交 720p/4 秒任务 `task_vKSCicqBR3xZmP5Cc2s8NqQgDZj4Vdbu`。数据库先记录 `NOT_START`、预占 3,055,500 quota；任务进入生成后，于 00:04:08.565 重启 New API，00:04:13.210 `/api/status` 恢复，服务窗口约 4.6 秒。重启后原任务为 `IN_PROGRESS`，脚本在查询阶段累计 `query_retry_count=4`，没有重新 POST；00:07:28.880 原任务成功。

最终只有 1 条同任务 ID 记录、1 条计费日志、0 条活动任务；actual 为 87,300 tokens，预占与实扣均为 ¥6.111，差额 0。Key A 从 `87701112|12298888` 变为 `84645612|15354388`，正好只结算 3,055,500 quota；管理员和渠道累计也各只增加同额。脱敏脚本证据位于 Git 忽略目录 `artifacts/poc/seedance-task_vKSCicqBR3xZmP5Cc2s8NqQgDZj4Vdbu.json`。

### 9.2 其他恢复与幂等证据

- 运行中任务在 PostgreSQL 干净重建、约 2 秒不可用后继续并完成一次结算：`PASS`，见 `POC-T2V-720P-30S-001`。
- 同一终态任务重复查询 20 次余额不变：`PASS`，实际连续执行两轮共 40 次；每轮 20/20 响应一致，标准取证轮前后均为 `87701112|12298888|1695295|6`，依次是 Key A 剩余 quota、已用 quota、任务 quota、Key A 日志数。
- 旧镜像替换为隔离修复镜像后数据保持：`PASS`；前后均为 `6|22|87701112|66641500`，依次是任务数、日志数、Key A/B 剩余 quota。
- 本次从 `sha256:ba17…2de9` 替换为 `sha256:2cef…ad70` 前后也保持：用户 1、任务 7、日志 47、渠道累计 `86129281`、管理员 `499959528107|86129281`、Key A `87701112|12298888`、Key B `26169607|73830393`；`/api/status success=true`。

## 10. 并发原子预占

**测试编号：** `POC-CONCURRENT-RESERVE-001`

**状态：** `PASS`

使用 Key B 在同一时刻并发提交 20 个 1080p/30 秒请求。当时 Key B 只能支付一次预占。

- 请求数：20；HTTP 200 恰好 1 个，HTTP 403 恰好 19 个；
- 实际方舟任务：恰好 1 个，`task_nlS2jrvriCdfCsAvcbvEFDcAPpoPxwmi`；
- 被拒请求：19 条本地错误日志，`quota=0`，没有对应任务；
- 接受任务：预估 1,458,000 tokens / ¥80.83152，actual 1,460,025 tokens / ¥80.943786，最终只结算一次；
- Key B 从 ¥133.283000 变为 ¥52.339214，没有瞬时或最终负余额；数据库只增加 1 条任务和 1 条成功计费记录。

该测试同时暴露修复前预估少一个 1080p 帧（2,025 tokens / ¥0.112266），直接促成 `e0fb2a1`。修复后的同参数预占已变为 1,460,025 tokens / ¥80.943786，并由单元向量与三档真实复测确认。

## 11. 余额对账

| 案例 | 预估 usage | 预占 RMB | actual usage | 官方公式重算 RMB | 实扣 RMB | 差异 | 结果 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 480p / 5 秒文生视频 | 48,037.5 | ¥3.362626 | 48,437 | ¥3.39059 | ¥3.39059 | ¥0 | PASS |
| 720p / 5 秒文生视频 | 108,000 | ¥7.56 | 108,900 | ¥7.623 | ¥7.623 | ¥0 | PASS |
| 720p / 4 秒文生视频 | 86,400 | ¥6.048 | 87,300 | ¥6.111 | ¥6.111 | ¥0 | PASS |
| 720p / 10 秒文生视频 | 216,000 | ¥15.12 | 216,900 | ¥15.183 | ¥15.183 | ¥0 | PASS |
| 720p / 30 秒文生视频 | 648,000 | ¥45.36 | 648,900 | ¥45.423 | ¥45.423 | ¥0 | PASS |
| 720p 图生视频 | DEFERRED | DEFERRED | DEFERRED | DEFERRED | DEFERRED | DEFERRED | 当前不启用 |
| 1080p / 5 秒探测 | 243,000 | ¥13.47192 | 245,025 | ¥13.584186 | ¥13.584186 | ¥0 | PASS |
| 1080p / 30 秒并发胜者（修复前） | 1,458,000 | ¥80.83152 | 1,460,025 | ¥80.943786 | ¥80.943786 | ¥0 | PASS；发现预占不足 |
| 480p / 5 秒（修复后） | 48,437.8125 | ¥3.390646 | 48,437 | ¥3.39059 | ¥3.39059 | ¥0 | PASS；返还 ¥0.000056 |
| 720p / 4 秒重启恢复（修复后） | 87,300 | ¥6.111 | 87,300 | ¥6.111 | ¥6.111 | ¥0 | PASS |
| 1080p / 5 秒（修复后） | 245,025 | ¥13.584186 | 245,025 | ¥13.584186 | ¥13.584186 | ¥0 | PASS |

允许的最终实扣与官方公式重算最大舍入差为一个 raw quota 点，即 `¥0.000002`。修复前历史任务的最终实扣准确，但预占普遍少一个输出帧，不满足硬额度要求；`e0fb2a1` 已修复。修复后三档真实任务均满足“最终费用不高于预占”，其中 480p 因小数 token 四舍五入产生 ¥0.000056 的保守返还。未来任何合法请求若再次出现最终费用高于预占，必须停止生产验收并修正公式。

## 12. 最终 GO / NO-GO

PENDING：本机文生视频的计费、隔离、并发、Key 限制和重启硬门槛已通过；仍需在方舟控制台按相同时间窗口核对官方总任务数、总 usage 与人民币账单后，才能给出最终 GO / NO-GO。图生视频当前明确延后，不纳入本轮结论；真实异步失败为 `not reproducible safely`，首笔自然失败发生后必须补录。
