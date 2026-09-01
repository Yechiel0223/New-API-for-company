# 火山方舟三模型接入与 Seedream 后付费实施记录

> 实施日期：2026-09-01（Asia/Shanghai）
> 工作目录：`D:\new-api`
> 适用环境：Windows 本机开发 + Docker Desktop + Linux 容器运行
> 安全边界：本文不记录真实方舟 Key、New API 虚拟 Key、Authorization 请求头或媒体签名 URL

## 1. 结论

本轮完成了以下三种精确模型 ID 的 New API 接入：

| 产品 | 精确 Model ID | 渠道 | 四把现有虚拟 Key | 计费方式 |
| --- | --- | --- | --- | --- |
| Seedance 2.5 | `doubao-seedance-2-5-260628` | 已放行 | 已放行 | 任务完成后按方舟 actual tokens 结算，沿用已验收规则 |
| Seedance 2.0 | `doubao-seedance-2-0-260128` | 已放行 | 已放行 | 任务完成后按方舟 actual tokens、分辨率和视频输入档位结算 |
| Seedream 5.0 Pro | `doubao-seedream-5-0-pro-260628` | 已放行 | 已放行 | 不预扣；方舟成功响应后按输入图片数、输出图片数和 `data[].size` 后扣 |

同一把公司真实方舟 Key 的上游能力已经由 [ark-multi-model-capability-2026-09-01.md](./ark-multi-model-capability-2026-09-01.md) 中的直连真实请求证明。本轮重点是补齐 New API 适配、计费和虚拟 Key 权限。

完成代码与运行配置后，又使用员工 Token ID 17 真实经过 New API 执行了 1 个 Seedance 2.0 视频、Seedream 文生图、图生图和正式图层拆分。四次均成功，另有一次无效参数失败请求确认不扣费。测试前后用户、虚拟 Key 和渠道账本增量完全一致。

## 2. 为什么这样实现

### 2.1 Seedance 保持现有异步任务链路

视频任务在提交后异步生成，方舟终态会返回实际 completion tokens。New API 已有任务预占、轮询、actual token 差额结算和任务日志，因此 Seedance 2.0 只需要增加精确模型、价格表达式和回归向量，不另建一套计费系统。

### 2.2 Seedream 统一后付费

Seedream 的文生图、图生图和图层拆分全部使用同一后付费链路：

```text
虚拟 Key 请求
  -> 校验模型、Key 和渠道
  -> 标记 Seedream postpaid，跳过预扣
  -> 调用火山方舟
     -> 失败：不扣费
     -> 成功：读取 usage + data[].size
             -> 计算本次人民币金额
             -> 一次性后扣
             -> 写入用量日志和结构化计费明细
```

这样没有预扣、退款和冲正分支，三种图片场景只在单张输出价格上有差异。

## 3. 计费规则

### 3.1 Seedance 2.0 表达式

```text
u("resolution") == "4k"
  ? (u("video_input") == "video"
    ? tier("4k_video_input", u("tokens") * 16 / 1000000)
    : tier("4k_no_video_input", u("tokens") * 26 / 1000000))
  : u("resolution") == "1080p"
    ? (u("video_input") == "video"
      ? tier("1080p_video_input", u("tokens") * 31 / 1000000)
      : tier("1080p_no_video_input", u("tokens") * 51 / 1000000))
    : (u("video_input") == "video"
      ? tier("480p_720p_video_input", u("tokens") * 28 / 1000000)
      : tier("480p_720p_no_video_input", u("tokens") * 46 / 1000000))
```

表达式已经写入 PostgreSQL 的：

- `options.key = billing_setting.billing_mode`
- `options.key = billing_setting.billing_expr`

注意：当前前端表达式编辑器重新打开含 `u(...)` 的任务表达式时，会错误显示 `p * 0 + c * 0`。列表和数据库中的运行配置仍正确。不要在该错误预览上直接再次保存；改价后必须查询数据库并重跑 `TestSeedance20PricingVectors`。

### 3.2 Seedream 5.0 Pro

| 项目 | 规则 |
| --- | --- |
| 输入图片 | 第 1 张免费；第 2 张起 ¥0.02/张 |
| 普通文生图/图生图，输出 ≤ 2,610,000 像素 | ¥0.30/张 |
| 普通文生图/图生图，输出 > 2,610,000 像素 | ¥0.60/张 |
| 图层拆分，输出 ≤ 2,610,000 像素 | ¥0.15/张 |
| 图层拆分，输出 > 2,610,000 像素 | ¥0.30/张 |
| 响应声明有输出但尺寸缺失/无法解析 | 按当前场景高档价保守结算 |

模型定价页中的固定价 `0.60` 只负责让模型通过“已配置价格”校验；成功响应后，代码会用本次真实计算总额覆盖该值，并清除请求图片数量倍率，避免重复相乘。

## 4. 代码改动

### 4.1 请求与响应兼容

- `relaykit/dto/openai_image.go`：增加可空布尔字段 `layer_decomposition`，能区分未传、显式 `false` 和显式 `true`。
- `relaykit/dto/openai_response.go`：解析 `usage.input_images` 和 `usage.generated_images`。
- 图片成功响应继续原样返回；不改变 URL、Base64、尺寸、图层顺序或图层元数据。

### 4.2 后付费结算

- `types/price_data.go`：新增 `Postpaid` 和 `ShouldPreConsume()`。
- `relay/helper/price.go`：精确模型 `doubao-seedream-5-0-pro-260628` 标记后付费，预扣额度固定为 0。
- `controller/relay.go`：后付费模型跳过 `PreConsumeBilling`。
- `relay/channel/openai/volcengine_seedream_billing.go`：按响应图片数量和尺寸计算人民币金额。
- `relay/channel/openai/usage.go`：仅在火山方舟图片成功响应解析完 usage 后执行结算计算。
- `relay/channel/openai/relay_image.go`：结算后不再追加 `n` 倍率，防止多图费用被重复乘以图片数。
- `relay/channel/openai/relay_image.go`：SSE 图片流累计每个完成事件的输出数量和尺寸；客户端提前断开时使用请求数量进行高档兜底，避免只看最后一个 SSE 事件造成漏计或误计。
- `relay/image_handler.go`、`service/log_info_generate.go`：写入人类可读摘要和结构化 `seedream_billing`。

后付费标记同时匹配精确模型 ID 与 `ChannelTypeVolcEngine`。同名模型如果由其他类型渠道提供，仍使用该渠道原有的预扣/结算规则，不会被方舟专用逻辑误伤。

### 4.3 模型清单

- `constant/model.go`：集中定义三个精确模型 ID。
- `relay/channel/volcengine/constants.go`：火山方舟渠道模型列表增加三模型。

## 5. 运行配置实际修改

### 5.1 渠道 #1

渠道 `Ark-Seedance-2.5-POC` 当前启用模型：

```text
doubao-seedance-2-5-260628
doubao-seedance-2-0-260128
doubao-seedream-5-0-pro-260628
```

`abilities` 表中三条记录均为 `enabled=true`、分组 `default`。

### 5.2 四把现有虚拟 Key

以下四把仍有效且开启模型限制的 Key 已统一改成三模型白名单：

| Token ID | 非秘密名称 |
| ---: | --- |
| 2 | `POC-seedance-A` |
| 3 | `POC-seedance-B` |
| 17 | `POC-poc-emp-a-708606-Seedance` |
| 18 | `POC-poc-emp-b-708606-Seedance` |

更新只修改 `tokens.model_limits`；没有读取或修改密钥原文、额度、归属用户、状态、有效期或 IP 限制。Root 页面只能看到 Root 自己的两把 Key，因此员工 Key 使用一次限定条件的数据库事务同步；页面刷新后 Root 两把均显示 `3 model(s)`。

### 5.3 非付费 API 验证

分别使用 Token ID 2（Root 所属）和 Token ID 17（员工所属）调用本地：

```text
GET http://localhost:3000/v1/models
```

两次响应均包含三个精确模型 ID。测试只读取模型列表，不触发火山方舟生成和费用。

## 6. Go 安装记录

### 6.1 安装结果

```text
go version go1.27.0 windows/amd64
GOPROXY=https://goproxy.cn,direct
```

Go 安装位置：

```text
C:\Program Files\Go\bin
```

安装通过 `winget` 的 `GoLang.Go` 包完成；系统 PATH 已包含该目录。当前 Codex 进程启动早于安装，因此本轮命令显式使用完整路径；重开 PowerShell 后可直接执行 `go version`。

### 6.2 独立模块注意事项

仓库中的 `relaykit` 是独立 Go module，必须单独验证：

```powershell
cd D:\new-api\relaykit
$env:GOWORK = 'off'
go test ./dto
go build ./...
```

## 7. 测试证据

### 7.1 为什么测试

| 测试 | 防止的问题 |
| --- | --- |
| DTO true/false 往返 | `layer_decomposition` 被未知字段逻辑丢弃 |
| usage 图片数量解析 | 无法区分输入数、输出数，导致漏计费 |
| 文生图/图生图/图层拆分向量 | 价格档位或输入图加价计算错误 |
| 缺失尺寸兜底 | 响应有图片但账单为 0 或低估 |
| postpaid 预扣测试 | 图片请求仍在成功前扣款 |
| 多图倍率测试 | 已按每张求和后又乘一次 `n` |
| SSE 无 usage、无尺寸、混合尺寸向量 | 流式响应只看最后事件导致漏计或全部按高档误计 |
| 非方舟渠道回归 | 仅凭模型名跳过其他渠道预扣 |
| Seedance 2.0 六档向量 | 分辨率或视频输入档位价格错误 |
| Linux 全包回归 | Windows 与生产 Linux 行为差异、既有功能回归 |

### 7.2 针对性测试命令

```powershell
cd D:\new-api
go test ./types ./relay/helper ./relay/channel/openai ./relay/channel/volcengine ./pkg/billingexpr ./service `
  -run '^(TestPriceDataShouldPreConsume|TestModelPriceHelperMarksSeedreamAsPostpaid|TestApplyVolcengineSeedreamBilling.*|TestUpdateOpenAIImageCountDoesNotMultiplySettledSeedreamPrice|TestSeedance20PricingVectors|TestSeedance25POCPricingVectors|TestGenerateTextOtherInfoIncludesSeedreamBilling|TestModelListIncludesCompanyArkModels)$' `
  -count=1
```

实际结果：六个包全部 `ok`。

`relaykit`：

```powershell
cd D:\new-api\relaykit
$env:GOWORK = 'off'
go test ./dto -run '^(TestImageRequestPreservesLayerDecomposition|TestUsageParsesVolcengineImageCounts)$' -count=1
go build ./...
```

实际结果：DTO 测试 `ok`，独立模块构建成功。

### 7.3 Linux/Go 1.26 完整相关包回归

```powershell
docker run --rm -v 'D:\new-api:/src' -w /src golang:1.26.1-alpine `
  go test ./types ./relay/helper ./relay/channel/openai ./relay/channel/volcengine ./pkg/billingexpr ./service
```

实际结果：六个包全部 `ok`。独立 `relaykit` 在同一 Linux/Go 1.26 镜像中执行 `go test ./dto` 和 `go build ./...` 也全部成功。

第一次把两个 `relaykit` 命令放进 `sh -lc` 时出现 `sh: go: not found`，原因是 Alpine 登录 shell 重置了镜像 PATH；这不是源码或测试失败。改为 Docker 直接执行 `go test`/`go build` 后通过。

### 7.4 Windows 全包已知噪声

Windows/Go 1.27 单独执行 `service` 全包时，既有 Channel Affinity 测试可能因 `time.Now().UnixNano()` 在 Windows 上发生测试键碰撞而互相污染。单测重复执行通过，Linux 全包也通过。本轮没有修改该无关代码；最终生产回归以 Linux/Go 1.26 结果为准。

## 8. 真实 New API 端到端验收

### 8.1 测试范围和安全方式

- 使用员工 A 的 Token ID 17；密钥只在当前 PowerShell 进程内读取并放入本地请求头，未输出或写入文档。
- 所有请求先访问 `http://localhost:3000`，再由渠道 #1 使用同一把公司真实方舟 Key 转发。
- 图片只记录数量、尺寸、usage 和 URL 是否存在，不记录签名 URL。
- 视频只轮询同一个本地任务 ID，没有重复 POST。

### 8.2 Seedance 2.0 文生视频

脱敏请求：720p、4 秒、16:9、纯文本输入。实际结果：

| 字段 | 结果 |
| --- | --- |
| 本地任务 ID | `task_OLGjtaynC4gGQCFJkCpsLgAHzVudhoLz` |
| 终态 | `succeeded` / 数据库 `SUCCESS` |
| completion / total tokens | 87,300 / 87,300 |
| 视频 URL | 存在；未记录 |
| 总耗时 | 220.1 秒 |
| New API 最终 quota | 2,007,900 raw = ¥4.015800 |
| 复算 | `87,300 × ¥46 / 1,000,000 = ¥4.015800` |
| 结论 | `PASS`，actual tokens 与本地最终账单完全一致 |

### 8.3 Seedream 文生图

| 字段 | 结果 |
| --- | --- |
| 输出 | 1 张，2048×2048，URL 存在 |
| usage | `generated_images=1`，output/total tokens 16,384 |
| 日志 ID | 78 |
| 结构化分档 | standard；0 个低档、1 个高档、0 个兜底 |
| 最终 quota | 300,000 raw = ¥0.60 |
| 结论 | `PASS` |

### 8.4 Seedream 图生图

使用 `artifacts\poc\seedream\i2i-input-wikimedia-mio.jpg`，以 Base64 data URI 传输。

| 字段 | 结果 |
| --- | --- |
| 输入 | 1 张 JPEG，203,539 bytes |
| 输出 | 1 张，2048×2048，URL 存在 |
| usage | `input_images=1`、`generated_images=1`、output/total tokens 16,384 |
| 日志 ID | 79 |
| 结构化分档 | standard；1 张输入首张免费；1 个高档输出；0 个兜底 |
| 最终 quota | 300,000 raw = ¥0.60 |
| 结论 | `PASS` |

### 8.5 Seedream 正式图层拆分

请求明确携带 `layer_decomposition=true`。如果字段被 New API 丢弃，方舟只会返回普通单图；实际返回如下：

| 字段 | 结果 |
| --- | --- |
| 输出层 | 4 张 |
| 尺寸 | 2048×2048、1194×1206、1956×948、1581×1559 |
| z-index | 0、1、2、3 |
| 命名业务层 | 3 |
| 带 bounding box 的业务层 | 3 |
| usage | `input_images=1`、`generated_images=4`、output/total tokens 66,941 |
| 日志 ID | 80 |
| 结构化分档 | layer_decomposition；3 个低档、1 个高档、0 个兜底 |
| 复算 | `3 × ¥0.15 + 1 × ¥0.30 = ¥0.75` |
| 最终 quota | 375,000 raw = ¥0.75 |
| 结论 | `PASS`；请求字段透传、多层响应和后付费均正确 |

### 8.6 失败不扣费

向 Seedream 发送无效尺寸 `not-a-valid-size`：

| 字段 | 结果 |
| --- | --- |
| HTTP | 400 |
| 上游错误码 | `InvalidParameter` |
| 请求前用户钱包 | quota 31,727,557；used 18,272,443 |
| 请求后用户钱包 | quota 31,727,557；used 18,272,443 |
| 结论 | `PASS`；失败请求没有扣费 |

### 8.7 四次成功请求的总账本

测试前：用户 A `used_quota=15,289,543`，渠道 #1 `used_quota=123,608,712`，日志 76 条，任务 15 条。

测试后：用户 A `used_quota=18,272,443`，渠道 #1 `used_quota=126,591,612`，日志 80 条，任务 16 条。

```text
Seedance 2.0       2,007,900 raw = ¥4.015800
Seedream 文生图      300,000 raw = ¥0.600000
Seedream 图生图      300,000 raw = ¥0.600000
Seedream 图层拆分    375,000 raw = ¥0.750000
------------------------------------------------
合计               2,982,900 raw = ¥5.965800
```

权威恒等式：

```text
用户 used 增量 = 虚拟 Key used 增量 = 渠道 used 增量 = 2,982,900 raw
```

请求数从 3 增至 7，对应四次成功；任务数只增加 1，因为 Seedance 是异步任务表，Seedream 进入通用用量日志。全部账本一致，未扣到 Root 用户。

### 8.8 管理员页面复核

在 `http://localhost:3000/usage-logs/common` 按模型 `doubao-seedream-5-0-pro-260628` 搜索，页面实际显示 4 行：

- 文生图：¥0.60；
- 图生图：¥0.60；
- 图层拆分：¥0.75；
- 无效尺寸错误：¥0。

模型、员工用户名、虚拟 Key 名称、渠道、tokens、耗时和费用均正确。页面顶部“用量”卡仍显示 ¥0，这是 [employee-account-key-acceptance-2026-09-01.md](./employee-account-key-acceptance-2026-09-01.md) 已记录的既有汇总卡 bug；表格明细、用户/Key/渠道账本和数据库均不受影响，本轮按已确认范围没有扩展到前端汇总卡修复。

## 9. 结果在哪里看

| 要看什么 | 位置 |
| --- | --- |
| 渠道三模型 | `http://localhost:3000/channels` |
| Root 自己的 Key 模型数 | `http://localhost:3000/keys` |
| Seedance/Seedream 价格入口 | `http://localhost:3000/system-settings/billing/model-pricing` |
| 每次图片费用与文字摘要 | `http://localhost:3000/usage-logs/common` |
| 结构化后付费明细 | 对应用量日志的 `other.seedream_billing` |
| 用户汇总 | `http://localhost:3000/dashboard/users` |
| 模型汇总 | `http://localhost:3000/dashboard/models` |
| 服务健康 | `http://localhost:3000/api/status` |

`seedream_billing` 包含：

```text
mode
scene
input_images
generated_images
low_pixel_images
high_pixel_images
fallback_images
total_price_rmb
```

## 10. 如何复核数据库（不读取密钥）

先从 `deploy/.env` 读取数据库用户名与库名到当前 PowerShell 变量，再执行只读查询。不要查询 `channels.key` 或 `tokens.key`。

```sql
select id,name,models,status from channels order by id;

select channel_id,"group",model,enabled,priority,weight
from abilities
where channel_id=1
order by model;

select id,name,model_limits_enabled,model_limits
from tokens
where deleted_at is null
order by id;

select key,value
from options
where key in ('ModelPrice','billing_setting.billing_mode','billing_setting.billing_expr')
order by key;
```

预期：渠道能力、四把虚拟 Key 和价格配置均包含本文件列出的三个精确模型 ID。

## 11. 已知边界

1. 后付费不做余额闸门，可能让员工余额变成负数。这是当前“不阻断生成、成功后结算”的明确选择。
2. 如果火山方舟已经成功，但 New API 在写本地账单前崩溃，本地可能漏记。彻底解决需要持久化请求账本和方舟账单对账任务，当前小规模内部方案暂不增加该复杂度。
3. 方舟声明有输出但不提供可解析尺寸时，平台按高档价保守结算，并在日志中增加 `fallback_images`，方便管理员复核。
4. 当前内部所有 Key 使用 `default` 分组，倍率为 1；因此页面人民币金额等于本文件价格规则计算结果。
5. 本轮已完成 New API 端到端真实付费验收；方舟控制台最终日账单可能延迟生成，到账后仍可把 2026-09-01 的官方账单作为第二层对照追加到本文，但不影响本地逐请求计费与账本恒等式已经通过的结论。

## 12. 参考

- [三模型真实方舟 Key 能力验证](./ark-multi-model-capability-2026-09-01.md)
- [Seedance 2.5 POC 与计费证据](./poc-evidence.md)
- [员工账号、Key 与全局管理员验收](./employee-account-key-acceptance-2026-09-01.md)
- [图片生成 API 正式参考](https://docs.volcengine.com/docs/82379/1541523)
- [火山方舟模型价格](https://docs.volcengine.com/docs/82379/1544106#457edfd0)
