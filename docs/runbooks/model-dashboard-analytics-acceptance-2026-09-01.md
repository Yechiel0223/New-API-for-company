# 模型数据看板验收记录

日期：2026-09-02
分支：`codex/model-dashboard-analytics`

## 数据库迁移

新增 opt-in 测试：

```powershell
go test ./model -run TestModelUsageEventMigration -count=1
```

结果：通过。默认覆盖 SQLite，并在未配置外部 DSN 时跳过外部数据库子项。

MySQL/PostgreSQL 手动验证命令：

```powershell
$env:MODEL_USAGE_MIGRATION_DIALECT="mysql"
$env:MODEL_USAGE_MIGRATION_DSN="root:password@tcp(127.0.0.1:3306)/newapi_model_usage_test?charset=utf8mb4&parseTime=True&loc=Local"
go test ./model -run TestModelUsageEventMigrationConfiguredDatabase -count=1

$env:MODEL_USAGE_MIGRATION_DIALECT="postgres"
$env:MODEL_USAGE_MIGRATION_DSN="host=127.0.0.1 user=postgres password=password dbname=newapi_model_usage_test port=5432 sslmode=disable TimeZone=Asia/Shanghai"
go test ./model -run TestModelUsageEventMigrationConfiguredDatabase -count=1
```

2026-09-02 Docker 实测（临时容器已在测试后删除）：

```powershell
docker run -d --rm --name codex-model-usage-mysql-validation `
  -e MYSQL_ROOT_PASSWORD=password `
  -e MYSQL_DATABASE=newapi_model_usage_test `
  -p 13306:3306 mysql:5.7.44
$env:MODEL_USAGE_MIGRATION_DIALECT="mysql"
$env:MODEL_USAGE_MIGRATION_DSN="root:password@tcp(127.0.0.1:13306)/newapi_model_usage_test?charset=utf8mb4&parseTime=True&loc=Local"
go test ./model -run TestModelUsageEventMigrationConfiguredDatabase -count=1 -v

docker run -d --rm --name codex-model-usage-postgres-validation `
  -e POSTGRES_PASSWORD=password `
  -e POSTGRES_DB=newapi_model_usage_test `
  -p 15432:5432 postgres:9.6.24
$env:MODEL_USAGE_MIGRATION_DIALECT="postgres"
$env:MODEL_USAGE_MIGRATION_DSN="host=127.0.0.1 user=postgres password=password dbname=newapi_model_usage_test port=15432 sslmode=disable TimeZone=Asia/Shanghai"
go test ./model -run TestModelUsageEventMigrationConfiguredDatabase -count=1 -v
```

结果：MySQL 5.7.44（1.05s）与 PostgreSQL 9.6.24（1.10s）均通过 `TestModelUsageEventMigrationConfiguredDatabase`。容器使用隔离端口 13306、15432，测试结束后通过 `docker rm -f` 清理。

该测试证明：

- `model_usage_events` 与 `model_usage_attempts` 可由 GORM 创建并重复迁移；
- `event_key` 与 `(event_key, channel_id)` 去重生效；
- 终态更新保留 quota、Token、输出数、耗时；
- 已有 `tasks` 与 `quota_data` 行不会被迁移破坏。

## 后端回归

已执行：

```powershell
go test ./model ./service ./controller ./router -count=1
```

结果：通过。

## 注意

本文件记录代码级验收。真实浏览器看板验收、生产构建和本地回填输出会在最终交付前继续追加到总 runbook。
