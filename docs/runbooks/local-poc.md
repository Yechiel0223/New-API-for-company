# 火山方舟 Seedance 用量网关：本机 POC 运行手册

本手册只适用于 `D:\new-api` 的 Windows 本机 POC。管理员页面和 API Base URL 均为 `http://localhost:3000`，只监听本机回环地址。

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

## PostgreSQL 账本核验

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml exec -T postgres `
  psql -U new_api -d new_api -c "select current_database(), version();"
```

输出必须显示数据库 `new_api` 和 PostgreSQL 15.x。PostgreSQL 是本 POC 的唯一权威账本；不得切换为 SQLite。

## 本机网络排障备注

若 Docker 构建时出现 Docker Hub、`proxy.golang.org` 的 DNS 或 `unexpected EOF`，先确认 Docker Desktop 仍在使用 Windows 系统代理。构建时可把现有回环代理转换为 `host.docker.internal` 后，通过 Docker 预定义的 `HTTP_PROXY`/`HTTPS_PROXY` build args 临时传入。代理地址和端口不得写入本文件或 Git。
