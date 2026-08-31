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
- DataGrip 本机连接：`127.0.0.1:5432/new_api`，详细步骤见 [`datagrip-local.md`](./datagrip-local.md)

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

## Seedance 2.5 文生视频测试脚本

先执行完全本地、不会访问方舟的脚本测试：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File scripts/tests/test-seedance-text-video.tests.ps1
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File scripts/tests/show-seedance-reconciliation.tests.ps1
```

只检查请求、不产生费用时使用 Dry Run：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File scripts/test-seedance-text-video.ps1 `
  -DryRun -Resolution 720p -Duration 5 `
  -Prompt "A red paper airplane flying smoothly across a clean white studio background, fixed camera, no text, no logo"
```

真实测试前，在当前 PowerShell 会话安全输入一把 **New API 虚拟 Key**。不要输入方舟真实 Key，不要把 Key 写进命令、脚本或文档：

```powershell
$pocSecureKey = Read-Host "New API virtual Key" -AsSecureString
$pocCredential = [pscredential]::new("unused", $pocSecureKey)
$env:NEW_API_KEY = $pocCredential.GetNetworkCredential().Password
```

单次真实任务：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File scripts/test-seedance-text-video.ps1 `
  -Resolution 720p -Duration 5 `
  -Prompt "A red paper airplane flying smoothly across a clean white studio background, fixed camera, no text, no logo"
```

脚本自动提交、轮询到终态，并把脱敏 JSON 写入已被 Git 忽略的 `artifacts/poc/`。文件保留任务 ID、画质、时长、状态、actual tokens 和 `query_retry_count`，只记录 `video_url_present`，不保存 Key、Authorization 或视频签名 URL。任务查询阶段若遇到临时 HTTP 5xx，脚本会在总超时范围内继续 GET；不会重新 POST，因此不会因瞬时数据库或网关故障重复创建付费任务。提交 POST 本身失败时不会自动重试，因为客户端无法仅凭网络错误判断上游是否已经创建任务，必须先由管理员按时间和 Key 名称检查任务列表后再决定是否重提。

拿到公开任务 ID 后执行对账：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass `
  -File scripts/show-seedance-reconciliation.ps1 `
  -TaskId "task_替换为公开任务ID" `
  -ComposePath deploy/compose.yaml `
  -EnvPath deploy/.env
```

对账脚本输出最终 raw quota、人民币扣费、actual tokens、分辨率、适用单价、重新计算的人民币和差额；它只选择必要数据库字段，不读取完整 `private_data` 或任何 Key。

完成后清除当前进程中的虚拟 Key：

```powershell
Remove-Item Env:NEW_API_KEY -ErrorAction SilentlyContinue
Remove-Variable pocSecureKey,pocCredential -ErrorAction SilentlyContinue
```

本 POC 的受控真实矩阵使用同一提示词隔离变量：`POC-seedance-A` 运行 480p/5 秒、720p/5 秒、1080p/5 秒以比较画质；`POC-seedance-B` 运行 720p/4 秒、720p/10 秒、720p/30 秒以覆盖最短、常用和最长时长。若某组合暴露新问题，再补充有针对性的组合，不用无差别重复同一断言。

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
