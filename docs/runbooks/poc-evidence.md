# 火山方舟 Seedance 用量网关：本机 POC 验收证据

> 本文件只记录脱敏后的结果、时间、摘要和计算过程。严禁记录火山方舟真实 API Key、任何虚拟 Key、管理员密码、数据库密码、会话密钥、Authorization 请求头或本机代理地址。

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

- 核对时间：PENDING
- 官方来源：PENDING
- 模型：`doubao-seedance-2-5-260628`
- 官方价格维度与人民币单价：PENDING
- 合法时长、分辨率、比例和图像输入限制：PENDING
- 预占计算及一 quota 点（`¥0.000002`）舍入示例：PENDING

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
