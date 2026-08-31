# 火山方舟 Seedance 2.5 用量网关本机 POC Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to execute this plan. Use superpowers:test-driven-development for scripts or source changes, and superpowers:verification-before-completion before reporting success.

**Goal:** 在 `D:\new-api` 基于固定 New API 提交构建一个可在 Windows 本机运行的 Seedance 2.5 中转网关，管理员能按虚拟 Key 设置人民币余额、停用 Key、查看任务与实际用量，并通过真实方舟调用证明配额、结算、隔离、并发和重启恢复行为正确。

**Architecture:** 单实例 New API 作为唯一 API 和管理员入口，PostgreSQL 作为权威账本，火山方舟真实 API Key 只保存在 New API 渠道配置中。普通使用者只拿虚拟 Key 和统一 Base URL。第一阶段不部署 Redis，不启用批量额度更新，不修改 New API 核心计费逻辑；只有 POC 以证据证明现成功能缺失时，才另开最小补丁任务。

**Tech Stack:** New API commit `2b6f1dfefbe217fed31fc0726717cc7de6958e8e`、Go 1.26.1、React/Bun（均沿用上游 Dockerfile）、PostgreSQL `15.19-alpine3.24`、Docker Desktop + WSL2 + Docker Compose、PowerShell 7/Windows PowerShell。

**Spec:** `D:\new-api\docs\superpowers\specs\2026-08-31-ark-seedance-usage-gateway-design.md`

## Global Constraints

- 本计划只交付本机 POC；云服务器、域名、HTTPS、自动备份和公网加固在本机验收通过后单独规划。
- 所有版本化文件必须位于 `D:\new-api`。PostgreSQL 的 Docker named volume 属于本机运行时数据，不提交 Git。
- 不把火山方舟真实 API Key、虚拟 Key、管理员密码、数据库密码或会话密钥写入 Git、文档、命令输出和验收截图。
- 真实方舟 Key 只能由管理员在 `http://localhost:3000` 的渠道配置页面手工录入。
- 固定上游完整提交哈希；不得使用 `latest` 镜像，也不得只依赖可移动标签。
- `BATCH_UPDATE_ENABLED=false`；不设置 `REDIS_CONN_STRING`；Compose 中不得存在 Redis 服务。
- 内部人民币换算固定为 `500,000 quota = ¥1`；界面设置为 CNY，汇率固定为 1。
- 第一阶段只允许模型 `doubao-seedance-2-5-260628`。
- 720p 是必测档位。1080p 先作为能力探测：只有方舟真实接口接受、终态返回实际 usage、最终费用不超过预占且金额与官方规则一致，才能列入正式支持范围；否则把第一期支持范围改为方舟实际通过的档位，并同步修正规格。
- 若终态缺失实际 usage、出现重复结算、跨 Key 查询成功、并发超扣、余额为负、实际费用高于保守预占或重启后任务无法继续结算，立即判定 POC 失败，不得用估算值或人工补账掩盖。
- 不删除或改写 New API / QuantumNous 许可证、NOTICE 或品牌标识。
- 任何源代码变更都必须先单独说明证据和必要性；当前计划默认源代码零改动。

---

## Task 1: 准备 Windows 容器运行环境

**Files:**

- Read: `D:\new-api\docs\superpowers\plans\2026-08-31-ark-seedance-gateway-implementation.md`
- Create by installer: Docker Desktop application and one WSL2 Linux distribution

**Success criteria:** `docker version` 同时显示 Client/Server，`docker compose version` 成功，`wsl --status` 显示 WSL2 可用。

- [ ] **Step 1: 记录当前基线**

  Run in PowerShell:

  ```powershell
  docker version
  docker compose version
  wsl --status
  wsl --list --verbose
  ```

  Expected before installation on the current machine: `docker` 未识别；`wsl` 命令存在。把结果摘要写入后续的 `docs/runbooks/poc-evidence.md`，不要粘贴包含用户名或机器敏感信息的完整环境转储。

- [ ] **Step 2: 安装并启用 Docker Desktop 的 WSL2 后端**

  Install Docker Desktop from Docker's official installer. During setup:

  - 使用 WSL2 backend。
  - 不启用 Kubernetes。
  - 保持 Linux containers 模式。
  - 若安装器要求重启，完成重启后再继续，不绕过。

- [ ] **Step 3: 验证运行环境**

  ```powershell
  docker version
  docker compose version
  docker run --rm hello-world
  ```

  Expected: Client 和 Server 都可连接；Compose 为 v2；`hello-world` 退出码为 0。

- [ ] **Step 4: 设置 Docker 资源下限**

  在 Docker Desktop 为 WSL2/Docker 保证至少 4 CPU、8 GiB 内存和 30 GiB 可用磁盘。若当前电脑无法提供，停止执行并记录为环境阻断，不降低 PostgreSQL 数据可靠性来换取启动。

---

## Task 2: 将固定 New API 提交落到 `D:\new-api`

**Files:**

- Preserve: `D:\new-api\docs\superpowers\specs\2026-08-31-ark-seedance-usage-gateway-design.md`
- Preserve: `D:\new-api\docs\superpowers\plans\2026-08-31-ark-seedance-gateway-implementation.md`
- Fetch: upstream repository tree at commit `2b6f1dfefbe217fed31fc0726717cc7de6958e8e`

**Success criteria:** `D:\new-api` 成为 Git 工作树，HEAD 精确等于固定提交，现有规格和计划仍在，分支名为 `codex/ark-seedance-gateway`。

- [ ] **Step 1: 检查目录不会覆盖现有文件**

  ```powershell
  Set-Location 'D:\new-api'
  Get-ChildItem -Force
  git status
  ```

  Expected: 目前只有 `docs`；`git status` 报告不是 Git 仓库。

- [ ] **Step 2: 初始化仓库并获取固定标签目标**

  ```powershell
  git init
  git remote add upstream https://github.com/QuantumNous/new-api.git
  git fetch --depth 1 upstream refs/tags/v1.0.0-rc.29
  git rev-parse "FETCH_HEAD^{commit}"
  ```

  Expected exact output:

  ```text
  2b6f1dfefbe217fed31fc0726717cc7de6958e8e
  ```

  If the hash differs, stop. Do not continue with the new tag target.

- [ ] **Step 3: 确认上游不占用现有设计文档路径**

  ```powershell
  git ls-tree -r --name-only "FETCH_HEAD^{commit}" -- docs/superpowers
  ```

  Expected: no output. Any output is a collision and must be reviewed before checkout.

- [ ] **Step 4: 创建工作分支并检出源码**

  ```powershell
  git switch -c codex/ark-seedance-gateway "FETCH_HEAD^{commit}"
  git rev-parse HEAD
  Test-Path 'docs\superpowers\specs\2026-08-31-ark-seedance-usage-gateway-design.md'
  Test-Path 'docs\superpowers\plans\2026-08-31-ark-seedance-gateway-implementation.md'
  ```

  Expected: HEAD 为固定哈希；两个 `Test-Path` 均为 `True`。

- [ ] **Step 5: 阅读仓库规则并确认关键实现仍存在**

  ```powershell
  Get-Content -Raw 'AGENTS.md'
  rg -n "TryReserveTokenQuota|remain_quota >=|PreConsumeTokenQuota" model service
  rg -n "doubao-seedance-2-5-260628|extractUsageOnComplete|completion_tokens" plugins/tasks/doubao/plugin.js
  ```

  Expected:

  - `model/quota_reserve.go` 有 `TryReserveTokenQuota`。
  - 无 Redis 路径包含 `remain_quota >= ?` 的条件更新。
  - `service/quota.go` 的预消费调用原子预占入口。
  - 豆包任务插件声明目标模型并从终态提取 `completion_tokens`/`total_tokens`。

- [ ] **Step 6: 提交已确认文档**

  ```powershell
  git add docs/superpowers/specs/2026-08-31-ark-seedance-usage-gateway-design.md docs/superpowers/plans/2026-08-31-ark-seedance-gateway-implementation.md
  git commit -m "docs: 确认方舟用量网关规格与实施计划"
  ```

  Expected: one documentation commit; no source changes.

---

## Task 3: 建立可审计的本机部署配置

**Files:**

- Modify: `D:\new-api\.gitignore`
- Create: `D:\new-api\deploy\compose.yaml`
- Create: `D:\new-api\deploy\.env.example`
- Create: `D:\new-api\scripts\initialize-local-env.ps1`
- Create: `D:\new-api\scripts\verify-local-compose.ps1`
- Test: `D:\new-api\scripts\verify-local-compose.ps1`

**Interfaces:**

- `initialize-local-env.ps1 -ExamplePath <path> -OutputPath <path>` creates one secret local `.env` and refuses to overwrite an existing file.
- `verify-local-compose.ps1 -ComposePath <path> -EnvPath <path>` exits non-zero unless the rendered Compose config has one New API service, one PostgreSQL service, no Redis, no PostgreSQL host port, localhost-only New API binding, PostgreSQL DSN and `BATCH_UPDATE_ENABLED=false`.

**Success criteria:** `docker compose config` passes; verifier passes; secrets are ignored; rendered config contains neither `latest` nor Redis.

- [ ] **Step 1: 先写部署配置验证脚本，并确认它会失败**

  Implement `scripts/verify-local-compose.ps1` with mandatory `ComposePath` and `EnvPath` parameters. It must:

  1. Fail if either file does not exist.
  2. Run `docker compose --env-file $EnvPath -f $ComposePath config` and capture rendered text.
  3. Assert services are exactly `new-api` and `postgres`.
  4. Assert rendered New API port contains `127.0.0.1` and `3000`.
  5. Assert `postgres` has no `ports` mapping.
  6. Assert `BATCH_UPDATE_ENABLED` renders as string `false`.
  7. Assert a PostgreSQL `SQL_DSN` is present.
  8. Assert `REDIS_CONN_STRING`, a `redis` service and `:latest` do not appear.
  9. Print only invariant names, never full rendered environment values.

  Run:

  ```powershell
  & 'D:\new-api\scripts\verify-local-compose.ps1' -ComposePath 'D:\new-api\deploy\compose.yaml' -EnvPath 'D:\new-api\deploy\.env'
  ```

  Expected: non-zero because the deployment files do not exist yet.

- [ ] **Step 2: 创建 `.env.example` 和安全初始化脚本**

  `deploy/.env.example` must contain only these non-secret/sentinel values:

  ```dotenv
  POSTGRES_DB=new_api
  POSTGRES_USER=new_api
  POSTGRES_PASSWORD=SET_BY_INITIALIZE_LOCAL_ENV
  SESSION_SECRET=SET_BY_INITIALIZE_LOCAL_ENV
  ```

  Implement `scripts/initialize-local-env.ps1` so it:

  - validates both sentinel fields exist exactly once;
  - generates independent 32-byte cryptographically random lowercase hex values with `RandomNumberGenerator.GetBytes(32)`;
  - replaces the two sentinel values;
  - writes UTF-8 without BOM to `deploy/.env`;
  - refuses overwrite when the output already exists;
  - prints only the output path, never the generated values.

- [ ] **Step 3: 创建精简 Compose 配置**

  `deploy/compose.yaml` must implement this shape:

  ```yaml
  name: ark-seedance-gateway

  services:
    new-api:
      build:
        context: ..
        dockerfile: Dockerfile
      image: company/new-api:rc29-2b6f1df
      command: --log-dir /app/logs
      restart: unless-stopped
      ports:
        - "127.0.0.1:3000:3000"
      volumes:
        - ./data:/data
        - ./logs:/app/logs
      environment:
        SQL_DSN: postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}
        TZ: Asia/Shanghai
        ERROR_LOG_ENABLED: "true"
        BATCH_UPDATE_ENABLED: "false"
        UPDATE_TASK: "true"
        SESSION_SECRET: ${SESSION_SECRET}
        NODE_NAME: local-poc
        SESSION_COOKIE_SECURE: "false"
        TRUSTED_PROXIES: none
      depends_on:
        postgres:
          condition: service_healthy

    postgres:
      image: postgres:15.19-alpine3.24
      restart: unless-stopped
      environment:
        POSTGRES_DB: ${POSTGRES_DB}
        POSTGRES_USER: ${POSTGRES_USER}
        POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
        TZ: Asia/Shanghai
      volumes:
        - pg_data:/var/lib/postgresql/data
      healthcheck:
        test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
        interval: 5s
        timeout: 5s
        retries: 20

  volumes:
    pg_data:
  ```

  Do not add Redis, reverse proxy, HTTPS, a PostgreSQL host port, or a second New API instance.

- [ ] **Step 4: 更新忽略规则**

  Append only these project-specific entries to `.gitignore`:

  ```gitignore
  /deploy/.env
  /deploy/data/
  /deploy/logs/
  /artifacts/poc/
  ```

- [ ] **Step 5: 生成本机秘密配置并运行验证**

  ```powershell
  Set-Location 'D:\new-api'
  & '.\scripts\initialize-local-env.ps1' -ExamplePath '.\deploy\.env.example' -OutputPath '.\deploy\.env'
  & '.\scripts\verify-local-compose.ps1' -ComposePath '.\deploy\compose.yaml' -EnvPath '.\deploy\.env'
  git check-ignore deploy/.env
  docker compose --env-file deploy/.env -f deploy/compose.yaml config --quiet
  ```

  Expected: verifier exits 0; `git check-ignore` prints `deploy/.env`; Compose config exits 0.

- [ ] **Step 6: 提交部署骨架**

  ```powershell
  git add .gitignore deploy/.env.example deploy/compose.yaml scripts/initialize-local-env.ps1 scripts/verify-local-compose.ps1
  git commit -m "feat: 添加本机方舟网关部署配置"
  ```

---

## Task 4: 验证上游原子预占并构建固定镜像

**Files:**

- Read: `D:\new-api\model\quota_reserve.go`
- Read/Test: `D:\new-api\model\quota_reserve_test.go`
- Read: `D:\new-api\service\quota.go`
- Read: `D:\new-api\Dockerfile`
- Create runtime evidence: Docker images only

**Success criteria:** 上游无 Redis 原子预占测试通过；应用镜像从固定工作树构建成功；镜像标签和源提交可对应。

- [ ] **Step 1: 运行上游额度预占测试**

  ```powershell
  Set-Location 'D:\new-api'
  docker run --rm `
    -v "${PWD}:/src" `
    -w /src `
    -e CGO_ENABLED=0 `
    -e GOWORK=off `
    -e GOEXPERIMENT=greenteagc `
    golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 `
    go test ./model -run 'TestTryReserveQuotaWithoutRedis|TestReserveFallsBackToDatabaseWhenRedisIsUnavailable' -count=1
  ```

  Expected: `ok github.com/QuantumNous/new-api/model`; both named tests pass. A failure blocks the POC; do not patch before identifying the cause.

- [ ] **Step 2: 构建应用镜像**

  ```powershell
  docker compose --env-file deploy/.env -f deploy/compose.yaml build --pull new-api
  docker image inspect company/new-api:rc29-2b6f1df --format '{{.Id}}'
  docker image inspect postgres:15.19-alpine3.24 --format '{{index .RepoDigests 0}}'
  ```

  Expected: build exits 0; both inspect commands return immutable digests. Record the digests in `docs/runbooks/poc-evidence.md`, not in Compose.

- [ ] **Step 3: 验证构建源提交未漂移**

  ```powershell
  git rev-parse HEAD
  git status --short
  ```

  Expected: HEAD history contains upstream `2b6f1df...`; only intended committed project files exist; no secrets or runtime data appear.

---

## Task 5: 启动 New API 与 PostgreSQL 并完成初始化

**Files:**

- Create: `D:\new-api\docs\runbooks\local-poc.md`
- Create: `D:\new-api\docs\runbooks\poc-evidence.md`
- Runtime only: `D:\new-api\deploy\.env`
- Runtime only: Docker volume `ark-seedance-gateway_pg_data`

**Success criteria:** 两个容器运行，`/api/status` 正常，管理员可登录，注册和支付入口关闭，界面金额只显示人民币。

- [ ] **Step 1: 创建运行手册与证据结构**

  `docs/runbooks/local-poc.md` must include exact commands for:

  - start: `docker compose --env-file deploy/.env -f deploy/compose.yaml up -d`
  - status: `docker compose --env-file deploy/.env -f deploy/compose.yaml ps`
  - logs: `docker compose --env-file deploy/.env -f deploy/compose.yaml logs --tail 200 new-api postgres`
  - stop without data deletion: `docker compose --env-file deploy/.env -f deploy/compose.yaml stop`
  - restart: `docker compose --env-file deploy/.env -f deploy/compose.yaml restart new-api`
  - explicit warning that `down -v` deletes PostgreSQL POC data and is not an ordinary operation.
  - admin URL `http://localhost:3000` and API Base URL `http://localhost:3000`.
  - balance edit SOP: disable token → wait for all tasks terminal → set absolute RMB balance → re-enable.

  `docs/runbooks/poc-evidence.md` must have fixed sections for environment, source commit, image digests, admin settings, official price source/time, capability matrix, each real-call case, concurrency, restart, isolation, balance reconciliation and final go/no-go. It must explicitly prohibit recording any key value.

- [ ] **Step 2: 启动服务**

  ```powershell
  Set-Location 'D:\new-api'
  docker compose --env-file deploy/.env -f deploy/compose.yaml up -d
  docker compose --env-file deploy/.env -f deploy/compose.yaml ps
  Invoke-RestMethod -Uri 'http://localhost:3000/api/status' -Method Get
  ```

  Expected: PostgreSQL is healthy; New API is running; status endpoint returns a successful JSON response.

- [ ] **Step 3: 在浏览器完成首次管理员初始化**

  Open `http://localhost:3000` and create exactly one administrator account. Use a password manager-generated password. Do not store the password in project files.

- [ ] **Step 4: 关闭与内部网关无关的产品功能**

  In the admin system settings:

  - disable new-user registration;
  - disable email/password registration and verification flows not needed after admin creation;
  - disable OAuth/OIDC login providers;
  - disable online payment, recharge, redemption, invitation and public model marketplace entries wherever switches exist;
  - retain administrator login, token management, channel management, task records and usage logs.

  Verify in a private/incognito browser session that there is no usable public registration flow.

- [ ] **Step 5: 配置人民币显示和额度单位**

  In admin settings set:

  ```text
  QuotaPerUnit = 500000
  USDExchangeRate = 1
  DisplayInCurrencyEnabled = true
  general_setting.quota_display_type = CNY
  ```

  Restart New API only if the UI indicates a restart is required. Then create a temporary disabled token with a displayed balance of `¥1.00` and verify the admin list/detail page displays `¥1.00`, not `500000 quota`, `$1`, `1 USD`, or token points. Delete this temporary token only because it was never enabled and has no task history.

  If raw quota leaks in the required admin token screens, record the exact screen and stop Task 5. That is the only condition that opens a later minimal UI patch task.

- [ ] **Step 6: 验证 PostgreSQL 是实际账本**

  ```powershell
  docker compose --env-file deploy/.env -f deploy/compose.yaml exec -T postgres `
    psql -U new_api -d new_api -c "select current_database(), version();"
  ```

  Expected: database is `new_api`; version reports PostgreSQL 15.x. No SQLite database may be used as the live ledger.

- [ ] **Step 7: 提交不含秘密的运行手册**

  ```powershell
  git add docs/runbooks/local-poc.md docs/runbooks/poc-evidence.md
  git commit -m "docs: 添加本机运行与验收手册"
  ```

---

## Task 6: 配置火山方舟渠道、模型和官方价格

**Files:**

- Read: `D:\new-api\plugins\tasks\doubao\plugin.js`
- Modify with observations only: `D:\new-api\docs\runbooks\poc-evidence.md`
- Runtime database configuration: New API channel/model/pricing settings

**Success criteria:** 只有一个可用火山方舟渠道和一个允许模型；真实 Key 不落盘到项目；官方价格有来源时间；提交前预估公式可复算。

- [ ] **Step 1: 从当前方舟控制台确认模型能力和价格**

  在公司火山方舟控制台中打开 Seedance 2.5 的模型文档/计费页面，记录：

  - 控制台显示的准确模型 ID；
  - 文生视频与图生视频支持的时长、分辨率、比例；
  - 每个相关分辨率档位的人民币单价和计价单位；
  - 官方页面标题、访问日期 `2026-08-31` 及内部可追溯截图文件名。

  截图放公司认可的安全位置，不放 Git。`poc-evidence.md` 只记录价格、单位、页面标题和时间，不记录账号、项目或 Key。

  If the exact model ID differs from `doubao-seedance-2-5-260628`, stop: this fixed New API plugin cannot be assumed compatible without a separate mapping review.

- [ ] **Step 2: 复核插件预估与终态 usage 行为**

  ```powershell
  rg -n "estimateTokens|resolutionMaxPixels|videoInputRatio|extractUsageOnComplete|completion_tokens|total_tokens" plugins/tasks/doubao/plugin.js
  ```

  Confirm for the selected legal request:

  ```text
  estimated_tokens = duration_seconds × max_width × max_height × 24 / 1024
  estimated_RMB = estimated_tokens × official_RMB_per_million_tokens / 1,000,000 × applicable_ratio
  reserved_quota = ceil(estimated_RMB × 500,000)
  ```

  The rounding unit is one raw quota point, equal to `¥0.000002`.

- [ ] **Step 3: 在管理员页面创建唯一方舟渠道**

  Create one VolcEngine/Ark-compatible channel:

  - name: `ark-seedance-2.5-company`
  - channel type: the New API VolcEngine type supported by the Doubao task plugin;
  - Base URL: the official Ark API Base URL shown by the company console;
  - API Key: paste the real company key directly into the secret field;
  - models: only `doubao-seedance-2-5-260628`;
  - status: enabled;
  - priority/weight: defaults, because there is only one channel.

  Never use browser developer tools, screenshots or copied database rows that expose the key.

- [ ] **Step 4: 配置官方原价，不加价**

  Use New API's task-plugin/model pricing UI to enter the exact official RMB price for every phase-1 usage dimension the plugin requests, including the legal resolution tiers and text/image input ratios. Because `USDExchangeRate=1`, values entered into fields internally named USD are treated numerically as RMB for this private deployment.

  Do not invent a blended price. Do not enable group multipliers, user multipliers or channel surcharges. Record the exact configured values and their official source in the evidence file.

- [ ] **Step 5: 渠道连通性验证**

  Use New API's channel test only if it supports the task plugin without generating a paid video. Otherwise skip the generic test and use Task 8's explicit paid call. A generic chat test against a video-only model is not evidence of channel failure.

---

## Task 7: 创建 POC 虚拟 Key 并验证管理动作

**Files:**

- Modify with observations only: `D:\new-api\docs\runbooks\poc-evidence.md`
- Runtime database only: two POC virtual tokens

**Success criteria:** 两把 Key 均只允许 Seedance 2.5；管理员可设置人民币绝对余额、停用/启用、查看各自日志；Key 值不进入项目文件。

- [ ] **Step 1: 创建两个专用 POC Token**

  Under the single administrator/internal service account create:

  - `poc-employee-a`
  - `poc-employee-b`

  For both:

  - expiration: never;
  - model restriction: only `doubao-seedance-2-5-260628`;
  - IP restriction: none for the POC;
  - group: default with multiplier 1;
  - initial balance: enough for all approved real tests, entered as displayed RMB.

  Copy each token once into the current PowerShell process only when running tests. Do not save it in shell profiles, history notes, `.env`, scripts or docs.

- [ ] **Step 2: 验证停用和绝对余额设置**

  For `poc-employee-b`:

  1. disable the token;
  2. set its displayed balance to exactly `¥10.00`;
  3. verify list and detail pages both show `¥10.00`;
  4. re-enable it;
  5. verify used quota remains zero.

- [ ] **Step 3: 验证禁用 Key 不可调用**

  Disable `poc-employee-b`, make one syntactically valid task submission with that key, and require a non-2xx authentication/token-status response with no task ID. Re-enable it after verification. This request must not reach Ark and must not alter balance.

---

## Task 8: 编写真实任务冒烟脚本

**Files:**

- Create: `D:\new-api\scripts\poc-smoke.ps1`
- Create: `D:\new-api\scripts\tests\poc-smoke.contract.tests.ps1`
- Runtime ignored: `D:\new-api\artifacts\poc\`

**Interface:**

```powershell
./scripts/poc-smoke.ps1 `
  -BaseUrl http://localhost:3000 `
  -Token <secure-process-value> `
  -Mode text|image `
  -Model doubao-seedance-2-5-260628 `
  -Resolution 480p|720p|1080p `
  -DurationSeconds 5 `
  [-ImageUrl <https-url>] `
  [-TimeoutMinutes 30]
```

The script submits `POST /doubao/api/v3/contents/generations/tasks`, polls `GET /doubao/api/v3/contents/generations/tasks/{public_task_id}`, returns non-zero on timeout/failure, and writes a sanitized JSON result under `artifacts/poc/`. It must never print or persist the Authorization header or token.

**Success criteria:** Contract tests pass without Ark; one real 720p text task and one real 720p image task reach terminal success and produce sanitized evidence.

- [ ] **Step 1: 先写不访问网络的脚本契约测试**

  The contract test must parse/read the script and assert:

  - all declared parameters exist;
  - image mode rejects a missing `ImageUrl` before network access;
  - model accepts only the fixed Seedance 2.5 ID;
  - resolution accepts only the three POC probe values;
  - duration accepts only the official range recorded in Task 6;
  - request route and query route are exact;
  - persisted result excludes `Token`, `Authorization` and request headers;
  - timeout produces a non-zero exit.

  Run before implementation and expect failure:

  ```powershell
  pwsh -NoProfile -File scripts/tests/poc-smoke.contract.tests.ps1
  ```

- [ ] **Step 2: 实现最小冒烟脚本**

  Request JSON for text mode:

  ```json
  {
    "model": "doubao-seedance-2-5-260628",
    "content": [{"type": "text", "text": "A small red paper airplane glides over a quiet white studio, fixed camera."}],
    "duration": 5,
    "resolution": "720p",
    "ratio": "16:9"
  }
  ```

  Image mode adds one first content item shaped as:

  ```json
  {"type": "image_url", "image_url": {"url": "https://..."}}
  ```

  The image URL must be a company-approved public test asset, not a local file path and not a data URL. The script must:

  - use `Authorization: Bearer <Token>` only in memory;
  - poll every 10 seconds;
  - recognize the plugin's terminal success/failure values;
  - persist only task ID, mode, model, resolution, requested duration, HTTP status, public task status, returned usage fields, output URL host/path metadata, timestamps and elapsed seconds;
  - avoid downloading the generated video during billing tests.

- [ ] **Step 3: 运行契约测试并提交脚本**

  ```powershell
  pwsh -NoProfile -File scripts/tests/poc-smoke.contract.tests.ps1
  git add scripts/poc-smoke.ps1 scripts/tests/poc-smoke.contract.tests.ps1
  git commit -m "test: 添加 Seedance 真实任务冒烟脚本"
  ```

  Expected: contract test exits 0.

- [ ] **Step 4: 运行 720p 文生视频**

  Load `poc-employee-a` into a process variable without echoing it, then run:

  ```powershell
  & '.\scripts\poc-smoke.ps1' `
    -BaseUrl 'http://localhost:3000' `
    -Token $pocEmployeeAKey `
    -Mode text `
    -Model 'doubao-seedance-2-5-260628' `
    -Resolution '720p' `
    -DurationSeconds 5 `
    -TimeoutMinutes 30
  ```

  Expected: terminal success, task ID, video URL and positive actual usage.

- [ ] **Step 5: 运行 720p 图生视频**

  Use the same script with `-Mode image` and the approved public test image URL. Expected: terminal success and positive actual usage.

- [ ] **Step 6: 核对两次真实任务账务**

  For each task, record from New API admin:

  - token name;
  - submit time and terminal time;
  - pre-estimated usage/cost;
  - actual terminal usage;
  - final deducted RMB;
  - remaining RMB;
  - task status and log status.

  Recalculate final cost from actual usage and Task 6's official price. Require equality within one quota point (`¥0.000002`) and require final cost not to exceed the reserved amount.

---

## Task 9: 探测 1080p、失败退款和额度不足

**Files:**

- Modify with observations only: `D:\new-api\docs\runbooks\poc-evidence.md`
- Runtime ignored: `D:\new-api\artifacts\poc\`

**Success criteria:** 能力范围由真实接口确定；同步拒绝和终态失败均不遗留错误扣费；额度不足在访问 Ark 前拒绝。

- [ ] **Step 1: 进行一次 1080p 文生视频能力探测**

  Only if Task 6's current official console lists 1080p for the exact model, run one 5-second 1080p text task. Apply the same usage and billing checks as Task 8.

  - If accepted and fully reconciled, mark 1080p supported.
  - If the official console does not list it or Ark rejects it, mark 1080p unsupported and change the design spec's phase-1 test statement from mandatory 1080p coverage to the actual supported maximum before final acceptance.
  - A rejected capability probe is not itself a platform failure if no balance remains deducted.

- [ ] **Step 2: 验证同步上游拒绝退款**

  Submit a request that is syntactically valid to New API but violates one official Seedance parameter boundary recorded in Task 6. Require:

  - no successful public task creation, or a task marked failed;
  - no final charge;
  - any pre-reserved amount is fully returned;
  - one failure log exists with no secret material.

- [ ] **Step 3: 验证终态失败退款**

  Use an approved test input that Ark accepts at creation but deterministically fails processing, if the official service provides such a safe test case. If no deterministic safe case exists, record this case as “not reproducible safely” and do not fabricate evidence; synchronous rejection coverage remains required, but production rollout must initially monitor real failures until one can be reconciled.

- [ ] **Step 4: 验证额度不足不访问 Ark**

  Follow the approved balance SOP for `poc-employee-b`:

  1. disable token;
  2. verify no in-flight tasks;
  3. set remaining balance below the deterministic conservative precharge for the chosen 720p/5s request;
  4. re-enable token;
  5. submit that exact request.

  Require a quota-insufficient response, no public task ID, unchanged token balance, and no new upstream Ark task. Confirm the last condition using New API channel logs plus the Ark task console around the test timestamp.

---

## Task 10: 验证跨 Key 隔离、重启恢复和幂等结算

**Files:**

- Modify with observations only: `D:\new-api\docs\runbooks\poc-evidence.md`
- Runtime ignored: `D:\new-api\artifacts\poc\`

**Success criteria:** Key B 不能查询 Key A 任务；应用重启后任务继续；重复轮询不重复扣费。

- [ ] **Step 1: 跨 Key 查询隔离**

  Create one task with `poc-employee-a`. While it exists, query its public task ID with `poc-employee-b`:

  ```text
  GET /doubao/api/v3/contents/generations/tasks/{employee_a_public_task_id}
  Authorization: Bearer {employee_b_key}
  ```

  Require non-2xx or an explicit not-found/forbidden response containing no task payload, output URL or usage. Then query with A and require success. Cross-Key visibility is a production blocker.

- [ ] **Step 2: 重启恢复**

  1. submit a valid task with A;
  2. after receiving the public task ID and before terminal state, run `docker compose --env-file deploy/.env -f deploy/compose.yaml restart new-api`;
  3. wait for `/api/status` to recover;
  4. query the same task with A until terminal;
  5. verify one and only one final settlement.

- [ ] **Step 3: 重复轮询幂等性**

  After terminal success, record A's balance. Query the same public task 20 times. Require all queries to return the same terminal result and the balance to remain exactly unchanged.

- [ ] **Step 4: PostgreSQL 持久性抽查**

  Stop both containers without deleting volumes, start them again, and confirm tokens, tasks, logs and balances remain present. Do not run `down -v`.

---

## Task 11: 编写并执行真实 HTTP 并发超扣测试

**Files:**

- Create: `D:\new-api\scripts\poc-concurrency.ps1`
- Create: `D:\new-api\scripts\tests\poc-concurrency.contract.tests.ps1`
- Runtime ignored: `D:\new-api\artifacts\poc\`

**Interface:**

```powershell
./scripts/poc-concurrency.ps1 `
  -BaseUrl http://localhost:3000 `
  -Token <secure-process-value> `
  -Model doubao-seedance-2-5-260628 `
  -Resolution 720p `
  -DurationSeconds 5 `
  -RequestCount 20 `
  -ExpectedAcceptedCount 1
```

The script releases 20 HTTP submissions concurrently using one token, records only sanitized status/task IDs, and exits non-zero unless exactly one request is accepted.

**Success criteria:** 20 simultaneous requests with balance for one conservative precharge create exactly one Ark task; remaining balance never becomes negative; rejected requests do not charge; accepted task settles once.

- [ ] **Step 1: 先写并发脚本契约测试**

  Assert without network access that the future script:

  - validates `RequestCount` from 10 through 20;
  - requires `ExpectedAcceptedCount`;
  - uses a shared start gate and concurrent HTTP tasks rather than a sequential loop;
  - counts accepted responses only when a 2xx response contains a task ID;
  - persists no token or authorization header;
  - exits non-zero on accepted-count mismatch.

  Run and expect failure before implementation:

  ```powershell
  pwsh -NoProfile -File scripts/tests/poc-concurrency.contract.tests.ps1
  ```

- [ ] **Step 2: 实现最小并发提交脚本**

  Use the same deterministic 720p/5s text request as Task 8. Use `System.Net.Http.HttpClient`, create all request tasks first, release them through one start gate, and await all results. Do not poll terminal results inside this script; output accepted public task IDs for the smoke poller/admin reconciliation.

- [ ] **Step 3: 运行契约测试并提交**

  ```powershell
  pwsh -NoProfile -File scripts/tests/poc-concurrency.contract.tests.ps1
  git add scripts/poc-concurrency.ps1 scripts/tests/poc-concurrency.contract.tests.ps1
  git commit -m "test: 添加虚拟 Key 并发超扣验收脚本"
  ```

- [ ] **Step 4: 准备余额恰好等于一次预占**

  For `poc-employee-b`, follow the balance SOP and set the displayed balance to the exact deterministic conservative precharge derived in Task 6 for the 720p/5s request. Because the UI displays RMB, enter the exact value rounded upward to the nearest `¥0.000002` if the control accepts six decimals; otherwise enter the smallest UI-supported value that permits exactly one request and record the rounding behavior.

- [ ] **Step 5: 发起 20 路并发请求**

  ```powershell
  & '.\scripts\poc-concurrency.ps1' `
    -BaseUrl 'http://localhost:3000' `
    -Token $pocEmployeeBKey `
    -Model 'doubao-seedance-2-5-260628' `
    -Resolution '720p' `
    -DurationSeconds 5 `
    -RequestCount 20 `
    -ExpectedAcceptedCount 1
  ```

  Require exactly one accepted public task ID and 19 quota-insufficient rejections. Verify Ark console/channel logs show exactly one upstream task for the timestamp window.

- [ ] **Step 6: 完成并发任务结算核对**

  Poll the one accepted task to terminal. Require:

  - balance never negative;
  - only that task changes used quota;
  - final charge is no greater than reserved amount;
  - any reservation difference is returned;
  - repeat polling does not change balance.

---

## Task 12: 完成本机 POC 验收与交付判断

**Files:**

- Modify: `D:\new-api\docs\runbooks\poc-evidence.md`
- Possibly modify only if evidence requires: `D:\new-api\docs\superpowers\specs\2026-08-31-ark-seedance-usage-gateway-design.md`
- Review: all committed project files

**Success criteria:** 所有硬门槛有证据；无秘密进入 Git；无未经解释的源代码改动；形成明确 go/no-go 结论。

- [ ] **Step 1: 完成证据矩阵**

  Mark each item pass/fail with timestamp and evidence reference:

  - fixed upstream commit and image digest;
  - PostgreSQL 15 live ledger;
  - no Redis and batch update disabled;
  - CNY-only admin display and `500,000 quota = ¥1`;
  - token enable/disable and absolute balance setting;
  - 720p text-to-video success;
  - 720p image-to-video success;
  - 1080p supported or explicitly excluded by actual capability evidence;
  - actual usage present and cost reproducible;
  - actual final cost never exceeds precharge;
  - rejected/failed request refund behavior;
  - insufficient quota rejection before Ark;
  - cross-Key task isolation;
  - restart recovery;
  - repeat-poll settlement idempotency;
  - 20-way atomic quota concurrency;
  - container stop/start persistence.

- [ ] **Step 2: 运行最终静态和运行态验证**

  ```powershell
  Set-Location 'D:\new-api'
  & '.\scripts\verify-local-compose.ps1' -ComposePath '.\deploy\compose.yaml' -EnvPath '.\deploy\.env'
  pwsh -NoProfile -File scripts/tests/poc-smoke.contract.tests.ps1
  pwsh -NoProfile -File scripts/tests/poc-concurrency.contract.tests.ps1
  docker compose --env-file deploy/.env -f deploy/compose.yaml ps
  Invoke-RestMethod -Uri 'http://localhost:3000/api/status' -Method Get
  git grep -n -I -E 'sk-[A-Za-z0-9_-]{16,}|Bearer [A-Za-z0-9._-]{16,}' -- . ':!docs/superpowers/plans/2026-08-31-ark-seedance-gateway-implementation.md'
  git status --short
  ```

  Expected:

  - all verifier/contract commands exit 0;
  - services are running and status is healthy;
  - secret scan prints nothing;
  - Git shows only the intended evidence/spec edits.

- [ ] **Step 3: 复核没有不必要的核心改动**

  ```powershell
  git diff "2b6f1dfefbe217fed31fc0726717cc7de6958e8e" -- '*.go' 'web/**' 'plugins/**'
  ```

  Expected: no source diff. If there is a diff, each line must be backed by a failed POC requirement and its own tests; otherwise revert only that unapproved change.

- [ ] **Step 4: 写出明确结论并提交证据**

  The final section of `poc-evidence.md` must say exactly one of:

  - `GO：本机 POC 全部硬门槛通过，可以开始云服务器迁移实施计划。`
  - `NO-GO：本机 POC 未通过，阻断项见本节，禁止迁移到生产。`

  Then:

  ```powershell
  git add docs/runbooks/poc-evidence.md docs/superpowers/specs/2026-08-31-ark-seedance-usage-gateway-design.md
  git commit -m "docs: 记录方舟网关本机 POC 验收结果"
  git status --short
  ```

  Expected: clean working tree. Do not create the cloud deployment plan unless the result is GO and the user explicitly asks to proceed to the server stage.

---

## Out of Scope for This Plan

- Linux cloud server deployment, domain, HTTPS, reverse proxy and firewall.
- Daily `pg_dump`, seven-day retention and cloud disk snapshots.
- Multi-instance New API, Redis, Kubernetes, ClickHouse or external observability stack.
- Public user portal, employee login, department hierarchy, online payment, recharge ledger, manual deduction or reversal.
- Real Ark Key lifecycle management.
- Historical Ark usage import.
- Video-to-video, first/last-frame workflows and models other than Seedance 2.5.

These are not silently omitted. Cloud deployment and backup become a separate implementation plan only after this local POC returns GO.

## Plan Self-Review

- Every confirmed phase-1 requirement maps to a task and a verifiable check.
- The plan pins the upstream commit, runtime model, database family and PostgreSQL version.
- No secret value, real Key, virtual Key or password is embedded.
- No unresolved implementation placeholder or speculative core patch remains.
- 1080p uncertainty is resolved by official-console evidence plus an actual capability probe, not by assumption.
- Billing acceptance distinguishes estimate, reservation, actual usage and official delayed reconciliation.
- The plan stops at the local POC boundary requested by the user; cloud production work is intentionally separate.
