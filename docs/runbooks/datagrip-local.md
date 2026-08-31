# DataGrip 连接本机 PostgreSQL

本手册只适用于 `D:\new-api` 的 Windows 本机 POC。Compose 将 PostgreSQL 映射到 `127.0.0.1:5432`，仅本机程序可以连接；不得把主机地址改为 `0.0.0.0`。

## DataGrip 字段

| 字段 | 填写值 |
| --- | --- |
| 名称 | `new-api-local`，仅为显示名称，可自定义 |
| 驱动程序 | `PostgreSQL` |
| 连接类型 | `default` |
| 主机 | `127.0.0.1` |
| 端口 | `5432` |
| 身份验证 | 用户与密码 |
| 用户 | `new_api` |
| 密码 | `deploy/.env` 中 `POSTGRES_PASSWORD=` 后面的值，不得写入本文档或截图 |
| 数据库 | `new_api` |
| URL | 由 DataGrip 自动生成 `jdbc:postgresql://127.0.0.1:5432/new_api` |
| SSH | 关闭 |
| SSL | 关闭 |

在 `default` 连接类型下，URL 由主机、端口和数据库自动生成，不需要手动编辑。修改数据库下拉框时，点击输入框，按 `Ctrl+A`，输入 `new_api` 并按 Enter。

## 测试连接

点击 DataGrip 左下角“测试连接”。连接成功后，在“架构”页只选择 `new_api.public` 即可查看业务表。

若提示缺少 PostgreSQL 驱动，先点击 DataGrip 提供的“下载缺失的驱动程序文件”，下载完成后再次测试。

## `Connection refused` 排障

在 PowerShell 中执行：

```powershell
Set-Location 'D:\new-api'
docker compose --env-file deploy/.env -f deploy/compose.yaml ps
Test-NetConnection -ComputerName 127.0.0.1 -Port 5432
```

预期 Docker 端口显示：

```text
127.0.0.1:5432->5432/tcp
```

且 `TcpTestSucceeded` 必须为 `True`。如果 PostgreSQL 容器未运行，执行：

```powershell
docker compose --env-file deploy/.env -f deploy/compose.yaml up -d postgres
```

这会复用现有 `pg_data` 数据卷。不要执行 `docker compose down -v`，否则会删除本机数据库数据。

## 权限提醒

当前为保持 POC 简单，DataGrip 复用应用数据库用户 `new_api`。该用户具有写权限，不只是查看权限；日常查看时只执行 `SELECT`，不要在表格编辑器中提交修改或删除操作。以后若需要把数据库查看权限交给更多人，应另建只读角色，不应共享此账户。
