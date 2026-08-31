# 火山方舟 Seedance 用量网关：本机 POC 运行手册

本手册只适用于 `D:\new-api` 的 Windows 本机 POC。管理员页面和 API Base URL 均为 `http://localhost:3000`，只监听本机回环地址。

## 测试记录入口

所有测试的目的、步骤、预期结果、实际结果、证据位置、副作用和清理结果统一记录在 [`poc-evidence.md`](./poc-evidence.md)。执行测试前先在该文件中定义通过条件，执行后立即填写实际结果；没有可复核证据的项目不得标记为 `PASS`。真实密钥、虚拟 Key、密码和认证请求头不得写入测试命令或文档。

## 启动与检查

在 PowerShell 中进入项目目录：

```powershell
Set-Location 'D:\new-api'
docker compose --env-file deploy/.env -f deploy/compose.yaml up -d
docker compose --env-file deploy/.env -f deploy/compose.yaml ps
Invoke-RestMethod -Uri 'http://localhost:3000/api/status' -Method Get
```

查看最近日志：

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml logs --tail 200 new-api postgres
```

只重启 New API，不重启 PostgreSQL：

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml restart new-api
```

停止服务但保留数据库数据：

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml stop
```

再次启动已停止的服务：

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml start
```

> 严禁把 `docker compose ... down -v` 当作普通停止命令。`-v` 会删除 PostgreSQL 的本机 POC 数据卷，令管理员、虚拟 Key、任务、日志和余额数据不可恢复。普通停止只使用 `stop`。

## 常用地址

- 管理员页面：`http://localhost:3000`
- 员工调用用 API Base URL：`http://localhost:3000`
- 健康状态：`http://localhost:3000/api/status`

本机 POC 不配置域名和 HTTPS。迁移到云服务器时必须另行配置 HTTPS、反向代理、防火墙与备份，不能直接照搬本机 HTTP 边界。

## 管理员初始化与密钥边界

首次打开管理员页面时只创建一个管理员账户。管理员密码使用密码管理器生成并保存，不得写入本项目、命令、截图或验收文档。

火山方舟真实 API Key 只能在 New API 的渠道页面手工录入。虚拟 Key 也不得保存到项目文件、PowerShell 配置、截图或验收文档。

## 虚拟 Key 余额修改 SOP

每次设置某个虚拟 Key 的绝对人民币余额时，严格按以下顺序操作：

1. 停用该 Key。
2. 确认该 Key 的所有任务均已进入终态，不存在运行中任务。
3. 在管理员页面设置绝对人民币余额。
4. 同时核对列表页与详情页的余额显示。
5. 重新启用该 Key。

第一阶段不做充值流水、人工扣减或冲正；余额管理就是管理员设置绝对值。

## Seedance 2.5 限时价格切换 SOP

当前官方价格页声明：1080p 在 2026-08-14 14:00 至 2026-09-17 14:00（Asia/Shanghai）按刊例价 72 折，480p/720p 不参与折扣。切换到刊例价时不得让新旧价格与运行中任务交叉：

1. 在截止时刻前停用方舟渠道，阻止新任务提交。
2. 等所有已提交任务进入成功或失败终态。
3. 在 `poc-evidence.md` 记录切换前表达式、任务数、余额和用量日志摘要，不记录密钥。
4. 把第一阶段 1080p 无视频输入单价从 `55.44` 改为 `77` 元/百万 token；480p/720p 无视频输入保持 `70`。同步更新 `pkg/billingexpr/seedance_pricing_poc_test.go` 的 1080p 单价、档位名和期望结果。第一阶段不配置视频输入价格。
5. 运行任务表达式的四组脱敏测试向量，确认档位、人民币和 quota 换算全部通过。
6. 重新启用渠道并记录时间。

如果届时官方公告延长或修改优惠，先更新官方证据和预期结果，再修改表达式；不得根据旧文档自动续用折扣。

> 当前“编辑模型定价”弹窗不能正确重新载入 Doubao 任务表达式中的 `u(...)`，会错误显示 `p * 0 + c * 0`。不要在该弹窗直接保存已有 Seedance 2.5 配置。改价时必须从可保留原始表达式的 JSON 配置路径操作，并在保存后用 `poc-evidence.md` 的数据库命令及四组向量测试复核。

## PostgreSQL 账本核验

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml exec -T postgres `
  psql -U new_api -d new_api -c "select current_database(), version();"
```

输出必须显示数据库 `new_api` 和 PostgreSQL 15.x。PostgreSQL 是本 POC 的唯一权威账本；不得切换为 SQLite。

## 本机网络排障备注

若 Docker 构建时出现 Docker Hub、`proxy.golang.org` 的 DNS 或 `unexpected EOF`，先确认 Docker Desktop 仍在使用 Windows 系统代理。构建时可把现有回环代理转换为 `host.docker.internal` 后，通过 Docker 预定义的 `HTTP_PROXY`/`HTTPS_PROXY` build args 临时传入。代理地址和端口不得写入本文件或 Git。
