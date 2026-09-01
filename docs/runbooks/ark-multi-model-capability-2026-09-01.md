# 火山方舟真实 Key：Seedance 2.5 / 2.0 / Seedream 5.0 Pro 能力验证

> 测试日期：2026-09-01（Asia/Shanghai）
>
> 测试环境：本机 `D:\new-api`；真实请求直接访问火山方舟，不经过 New API
>
> 安全边界：本文不记录真实方舟 Key、New API 虚拟 Key、Authorization 请求头或带签名的媒体 URL

## 1. 先看结论

同一把公司火山方舟真实 Key 对以下三个精确模型 ID 均有权限：

| 产品名称 | 方舟返回的精确模型 ID | 权限/真实请求结论 |
| --- | --- | --- |
| Seedance 2.5 | `doubao-seedance-2-5-260628` | `PASS`；模型列表当前可见，且此前同一渠道已完成 15 个真实视频任务 |
| Seedance 2.0 | `doubao-seedance-2-0-260128` | `PASS`；本轮真实 720p/4 秒文生视频成功，返回 87,300 tokens 和视频 URL |
| Seedream 5.0 Pro | `doubao-seedream-5-0-pro-260628` | `PASS`；本轮真实 2K 文生图成功，返回 1 张图片和 16,384 output tokens |

需要特别注意：`Seedream 5.0 Pro` 的本 Key 实际模型 ID 是 `doubao-seedream-5-0-pro-260628`，不能用旧的 `doubao-seedream-5-0-260128` 代替，也不能把产品显示名称直接当作 API model 参数。

本轮只证明真实方舟 Key 的上游能力，**不代表 2.0 和 Seedream 已经完成 New API 接入**。当前 New API 渠道和员工虚拟 Key 仍只允许 Seedance 2.5；两次新增模型的真实调用直接访问方舟，因此没有进入 New API 的任务、日志、员工余额或渠道累计。

## 2. 为什么要分三层验证

“模型能用”至少包含三层，不能混为一个结论：

1. 账号授权层：鉴权后的模型列表能否返回模型。
2. 上游生成层：真实创建请求能否成功，并返回媒体和实际 usage。
3. New API 接入层：渠道与虚拟 Key 是否放行、请求是否被正确适配、费用是否按官方规则进入员工和公司账本。

本轮完成第 1、2 层；第 3 层中只有既有 Seedance 2.5 已完成。2.0 和 Seedream 必须在价格口径确认后另做接入与账单验收。

## 3. 模型列表权限探测

### 3.1 方法

从本机 PostgreSQL 的既有渠道 #1 读取真实 Key 到当前 PowerShell 进程变量，只向方舟发送鉴权 GET：

```text
GET https://ark.cn-beijing.volces.com/api/v3/models
Authorization: Bearer [仅存在于进程内，未输出]
```

只保留模型 ID、领域、输入输出模态等非秘密字段；不保存或输出密钥。

### 3.2 结果

- HTTP：`200`
- 方舟共返回：`130` 个模型
- 三个目标模型全部存在

| 模型 ID | domain | input modalities | output modalities |
| --- | --- | --- | --- |
| `doubao-seedance-2-5-260628` | `VideoGeneration` | text / image / video / audio | video |
| `doubao-seedance-2-0-260128` | `VideoGeneration` | text / image / video / audio | video |
| `doubao-seedream-5-0-pro-260628` | `ImageGeneration` | text / image | image |

模型列表是权限强证据，但不能单独替代一次真实生成，所以继续执行第 4、5 节。

## 4. Seedance 2.0 真实视频测试

### 4.1 为什么这样测试

使用短时、标准 16:9 的 720p 文生视频，既能触发真实推理和 usage，又避免把图像/视频输入适配混入本轮权限判断。

### 4.2 脱敏请求

```json
{
  "model": "doubao-seedance-2-0-260128",
  "content": [
    {
      "type": "text",
      "text": "A single red paper airplane glides smoothly across a clean white studio background, fixed camera, no text, no logo"
    }
  ],
  "resolution": "720p",
  "duration": 4,
  "ratio": "16:9"
}
```

创建端点：

```text
POST /api/v3/contents/generations/tasks
```

创建成功后只轮询：

```text
GET /api/v3/contents/generations/tasks/{task_id}
```

没有重复 POST。

### 4.3 实际结果

| 字段 | 结果 |
| --- | --- |
| 提交时间 | 2026-09-01 10:08:41（Asia/Shanghai） |
| 创建 HTTP | `200` |
| 脱敏上游任务 ID | `cgt-20260901100841-544gb` |
| 终态时间 | 2026-09-01 10:11:27（Asia/Shanghai） |
| 终态 | `succeeded` |
| 视频 URL | 存在；未保存 URL |
| completion tokens | `87,300` |
| total tokens | `87,300` |
| 结论 | `PASS` |

## 5. Seedream 5.0 Pro 真实图片测试

### 5.1 脱敏确认请求

```json
{
  "model": "doubao-seedream-5-0-pro-260628",
  "prompt": "Capability probe confirm 20260901-1010: a single red paper airplane centered on a clean white studio background, product photography, no text, no logo",
  "size": "2K",
  "response_format": "url",
  "watermark": false
}
```

端点：

```text
POST /api/v3/images/generations
```

该接口同步返回图片，不使用视频任务查询端点。

### 5.2 实际结果

| 字段 | 结果 |
| --- | --- |
| 开始时间 | 2026-09-01 10:09:51（Asia/Shanghai） |
| 完成时间 | 2026-09-01 10:10:27（Asia/Shanghai） |
| HTTP | `200` |
| 图片数 | `1` |
| 图片 URL | 存在；未保存 URL |
| base64 图片 | 未返回（符合 `response_format=url`） |
| usage | 存在 |
| output tokens | `16,384` |
| total tokens | `16,384` |
| 结论 | `PASS` |

### 5.3 首次不确定请求

在上述确认请求之前，曾以最小 `model + prompt` 请求发起一次同步生成。客户端等待 30 秒后未收到终态输出，无法判断客户端停止等待时上游是否已经完成并计费，因此该次标记为 `INDETERMINATE`，不作为 PASS 证据。为取得确定结果，随后执行了带唯一提示词、2K、URL 输出的确认请求。费用不是本轮限制，但未来自动化脚本仍应给同步图片生成至少 300 秒客户端超时，并把完整响应在进程内脱敏后再退出。

## 6. Seedance 2.5 当前证据

本轮鉴权模型列表继续返回 `doubao-seedance-2-5-260628`。同一 New API 渠道 #1、同一把真实方舟 Key 在此前验收中已完成 15 个真实 Seedance 2.5 视频任务；最终基线为 4,039,299 actual tokens、New API 本地费用 ¥247.217424。逐笔任务和账单证据见：

- [poc-evidence.md](./poc-evidence.md)
- [employee-account-key-acceptance-2026-09-01.md](./employee-account-key-acceptance-2026-09-01.md)

因此没有为重复证明 2.5 权限再创建一条付费视频；本轮“当前列表仍可见 + 同 Key 已有真实成功任务”足以判定 `PASS`。

## 7. 为什么本轮费用不在 New API 页面出现

新增的 2.0 视频和 Seedream 图片是为了隔离验证真实方舟 Key 权限，直接调用了方舟 API，没有经过 `http://localhost:3000`。测试后数据库保持：

| 指标 | 值 |
| --- | ---: |
| New API `tasks` 总数 | 15 |
| New API 成功任务 | 15 |
| New API `logs` 总数 | 71 |
| 渠道 #1 `used_quota` | 123,608,712 raw quota |
| 2.0 / Seedream 本地任务行 | 0 |

这不是漏记账，而是本轮测试路径刻意绕过了 New API。方舟可能对这些直连请求计费；应在后续方舟控制台账单中作为“直连能力测试”与平台正式流量分开核对。

## 8. 当前 New API 接入差距

### 8.1 Seedance 2.0

仓库内建 Doubao 视频任务插件已经包含 `doubao-seedance-2-0-260128`，提交、逐任务查询、actual token 提取和差额结算框架均已存在。当前缺少的是运行配置：

1. 渠道 #1 模型白名单仍只有 Seedance 2.5。
2. 员工虚拟 Key 的模型限制仍只有 Seedance 2.5。
3. `billing_setting.billing_mode` / `billing_expr` 仅配置了 Seedance 2.5。
4. 必须先确认公司方舟控制台中 Seedance 2.0 的当前精细价格，再配置表达式并做预占/actual 对账。

官方公开产品页当前能确认的通用两档是：含视频输入 ¥28/百万 tokens、无视频输入 ¥46/百万 tokens；但公开文本不足以确认各分辨率细分价和优惠有效期，不能凭代码中的历史比例直接作为生产价格。

### 8.2 Seedream 5.0 Pro

现有 VolcEngine 通用图片适配会把 `/v1/images/generations` 转到方舟 `/api/v3/images/generations`，现有图片响应处理也能够读取 `input_tokens` / `output_tokens` / `total_tokens`。但当前仍缺少：

1. 渠道和虚拟 Key 模型白名单。
2. Seedream 5.0 Pro 的正式价格配置。
3. 文生图、图生图各自的请求与费用验收。
4. 输入图片、输出数量、尺寸/像素与最终费用的边界验证。

官方公开当前价目页没有给出 Seedream 5.0 Pro 的可确认现行价格。火山官方开发者社区有“输入 ¥0.02/张、输出 ¥0.30/张”的二级资料，但没有有效期和尺寸规则，不能在未核对公司控制台价格前直接当作生产账单真值。

## 9. 后续接入顺序

1. 从公司方舟控制台取得 Seedance 2.0 与 Seedream 5.0 Pro 的当前价格截图/导出，记录生效时间和优惠期限。
2. 先新增专用测试虚拟 Key，只放行新增模型；不要立刻扩到所有员工 Key。
3. 为 Seedance 2.0 配置任务 usage 表达式，为 Seedream 配置图片价格/usage 结算，并先跑完全本地的价格向量。
4. 通过 New API 分别执行：2.0 文生视频、Seedream 文生图、Seedream 图生图。
5. 核对虚拟 Key、所属员工、通用日志、模型看板、渠道累计和方舟官方账单。
6. 全部通过后再把模型加入正式员工 Key 白名单。

## 10. 官方接口参考

- [视频生成任务创建 API](https://api.volcengine.com/api-explorer/?action=CreateContentsGenerationsTasks&groupName=%E8%A7%86%E9%A2%91%E7%94%9F%E6%88%90API&serviceCode=ark&version=2024-01-01)
- [视频生成任务查询 API](https://api.volcengine.com/api-explorer/?action=GetContentsGenerationsTask&groupName=%E8%A7%86%E9%A2%91%E7%94%9F%E6%88%90API&serviceCode=ark&version=2024-01-01)
- [图片生成 API](https://api.volcengine.com/api-explorer/?action=ImageGenerations&groupName=%E5%9B%BE%E7%89%87%E7%94%9F%E6%88%90API&serviceCode=ark&version=2024-01-01)
- [火山方舟产品页](https://www.volcengine.com/product/ark)
