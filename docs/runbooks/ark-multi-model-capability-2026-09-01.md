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
| Seedream 5.0 Pro | `doubao-seedream-5-0-pro-260628` | `PASS`；2K 文生图、2K 图生图和正式 `layer_decomposition` 图层拆分均真实成功；拆层返回 1 张底图和 3 个可独立编辑层 |

需要特别注意：`Seedream 5.0 Pro` 的本 Key 实际模型 ID 是 `doubao-seedream-5-0-pro-260628`，不能用旧的 `doubao-seedream-5-0-260128` 代替，也不能把产品显示名称直接当作 API model 参数。

本轮只证明真实方舟 Key 的上游能力，**不代表 2.0 和 Seedream 已经完成 New API 接入**。当前 New API 渠道和员工虚拟 Key 仍只允许 Seedance 2.5；新增模型的文生视频、文生图、图生图和图层拆分均直接访问方舟，因此没有进入 New API 的任务、日志、员工余额或渠道累计。

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

### 5.4 图生图：真实参考图编辑

**为什么测试：** 文生图成功只能证明模型能出图，不能证明该 Key 的图片输入权限、参考图下载/编码、主体保持和图生图 usage 均正常。这里使用公开、无敏感内容的单个红苹果图片，要求保留苹果位置和轮廓，只修改布景与质感。

参考图来源：[`Mio.jpg`（Wikimedia Commons）](https://upload.wikimedia.org/wikipedia/commons/1/1f/Mio.jpg)。测试时已下载到 Git 忽略目录：

```text
D:\new-api\artifacts\poc\seedream\i2i-input-wikimedia-mio.jpg
```

第一次把 Wikimedia URL 直接放进 `image`，方舟返回 HTTP `400` / `InvalidParameter`，明确说明服务端下载该 URL 超时。这证明失败点是方舟拉取外部 URL，而不是模型或 Key 无图生图权限。只改变图片传输方式，把同一文件转为 Base64 data URI 后重试；模型、提示词、尺寸和其他参数不变。

脱敏成功请求：

```json
{
  "model": "doubao-seedream-5-0-pro-260628",
  "prompt": "Keep the same single red apple as the unmistakable main subject and preserve its position and silhouette. Replace the plain white background with a premium dark navy advertising studio, add realistic cold condensation droplets on the apple, dramatic rim lighting, no text, no logo.",
  "image": "data:image/jpeg;base64,[省略]",
  "size": "2K",
  "response_format": "url",
  "watermark": false
}
```

| 字段 | 实际结果 |
| --- | --- |
| 开始时间 | 2026-09-01 10:37:50（Asia/Shanghai） |
| 完成时间 | 2026-09-01 10:39:28（Asia/Shanghai） |
| HTTP | `200` |
| 输入 | JPEG，1391×1391，203,539 bytes |
| 输出 | JPEG，2048×2048，378,209 bytes |
| 图片数 | `1` |
| output tokens | `16,384` |
| total tokens | `16,384` |
| 视觉检查 | 苹果仍是同一明确主体，位置和轮廓保留；白底变为深蓝广告棚，出现冷凝水珠和轮廓光；无文字、无 logo |
| 结论 | `PASS` |

输出文件：

```text
D:\new-api\artifacts\poc\seedream\i2i-output-seedream-5-pro.jpg
SHA256 6A7BAEB0826C6BDFCA9CCE16BD2B132593056F64D8F84B59E496E6130F80CE70
```

参考图 SHA256 为 `20B93799A9F729E69EBB5BD6CD7CB46CBB139792E6633F36DC514D3955695371`。这次结果说明生产接入时应优先允许客户端传 Base64 或使用公司可被方舟稳定访问的对象存储 URL；不能假设任意公网图片 URL 都能被方舟拉取。

### 5.5 图层拆分：正式 `layer_decomposition` 能力

**为什么测试：** 图层拆分不是“让模型分别画几张类似图片”，而是方舟 Seedream 5.0 Pro 的正式 API 能力。验收必须同时满足：请求使用 `layer_decomposition=true`；响应给出 z-index、图层名称和边界框；业务层文件具有透明通道；拆开后可以独立重组和替换背景。

继续使用同一苹果参考图。由于上一项已经确认 Wikimedia URL 会被方舟下载超时，本项直接使用本地文件的 Base64，不重复触发已知失败。

脱敏请求：

```json
{
  "model": "doubao-seedream-5-0-pro-260628",
  "image": "data:image/jpeg;base64,[省略]",
  "layer_decomposition": true,
  "prompt": "将红苹果主体、苹果投影和白色背景拆分为可独立编辑的图层，保持原始构图。",
  "size": "2K",
  "response_format": "url",
  "output_format": "png",
  "watermark": false
}
```

实际请求从 2026-09-01 10:45:06 至 10:46:12（Asia/Shanghai），HTTP `200`。方舟一次返回 4 张图：z-index 0 的底图，以及 3 个有名称、有描述、有边界框的业务层。

| z-index | 方舟层名 | 下载 PNG 尺寸 | absolute bounding box | Alpha 实测 | 文件大小 | SHA256 |
| ---: | --- | --- | --- | --- | ---: | --- |
| 0 | 底图（响应未命名） | 2048×2048 | 无 | 索引色 PNG，无 Alpha | 1,378,661 | `077F68F47B90526873053627CDB2E685C266CE366C209CBA95C83DACE5AE4DEA` |
| 1 | 白色背景底板 | 1297×1462 | `377,199,1674,1661` | RGBA；10 个全透明、1,861,356 个半透明像素 | 2,847,841 | `26AA897362DECE9F1972D3BA54BCFCD060E917E88301F630B2DE5938677D33F1` |
| 2 | 苹果投影 | 1989×919 | `570,1170,1923,1795` | RGBA；541,370 个全透明、1,279,240 个半透明像素 | 2,153,458 | `5D0686905759E5A44788BBEB31F4747364D39272658CCE3D8AA673AB7998F1E4` |
| 3 | 红苹果主体 | 1502×1492 | `403,404,1617,1605` | RGBA；471,192 个全透明、1,698,556 个半透明像素 | 3,748,370 | `7D64A842B49084490F993221B1BBF6F83AED2CA005CA228E024AF45D6B50CECD` |

方舟对三层的描述分别为：完整白色背景且不含苹果和投影；苹果在白色平面上的柔和阴影且不含主体；完整带果柄红苹果且不含投影和背景。文件位于：

```text
D:\new-api\artifacts\poc\seedream\layer-decomposition\z00.png
D:\new-api\artifacts\poc\seedream\layer-decomposition\z01.png
D:\new-api\artifacts\poc\seedream\layer-decomposition\z02.png
D:\new-api\artifacts\poc\seedream\layer-decomposition\z03.png
```

本地验收又做了两步，不只看响应字段：

1. 按 `bounding_box.absolute` 把各层映射到 2048×2048 底图并依 z-index 叠加，得到 `composite-reconstructed.png`。视觉上恢复为红苹果、白底和独立投影。
2. 去掉“白色背景底板”，把底色换成深蓝，再叠加苹果投影与苹果主体，得到 `editability-demo-navy.png`。苹果和投影仍能独立存在，证明返回的不是几张不可编辑的平面成图。

```text
D:\new-api\artifacts\poc\seedream\layer-decomposition\composite-reconstructed.png
SHA256 5AE8B1489A6E550A17A0A0DA59D5062103262B32938637E8A220D845D815EE25

D:\new-api\artifacts\poc\seedream\layer-decomposition\editability-demo-navy.png
SHA256 651A36A113B6E20C6814824238BEF21D6B0E9BD51BA4A7C775C6B8D4DAE43B25
```

usage 实际返回：

| 字段 | 值 |
| --- | ---: |
| `input_images` | 1 |
| `generated_images` | 4 |
| `output_tokens` | 65,472 |
| `total_tokens` | 65,472 |

**计费异常点必须保留：** 官方价格规则写明图层拆分按每张输出的实际像素分别计价：不超过 261 万像素 ¥0.15/张，超过则 ¥0.30/张；同一请求内各层可以落在不同档。按本次下载文件尺寸推算，z0 为 ¥0.30，z1–z3 各 ¥0.15，输出合计应为 **¥0.75**，单张输入属于首张免费。但下载文件总像素为 10,159,393，按官方 `sum(width × height) / 256` 公式约为 39,685 tokens，与实际返回的 65,472 tokens 明显不一致。API 没有返回人民币字段，因此 ¥0.75 只能标记为“按官方价目规则推算”，**不能标记为已核实实扣**；本次最终人民币金额必须等方舟费用明细到账后核对。

结论：正式图层拆分能力和返回文件可编辑性均为 `PASS`；本次人民币实扣仍为 `PENDING OFFICIAL BILL`。

### 5.6 返回格式与当前平台限制

图层拆分的正式返回不是 PSD 或 ZIP。方舟在 `data` 数组中返回底图和最多 16 个独立层：业务层是带 Alpha 的 PNG，并附 `z_index`、`name`、`description` 和 `bounding_box`；URL 输出通常需要及时下载保存。

当前仓库还不能把该能力直接开放给员工虚拟 Key：`relaykit/dto/openai_image.go` 的 `ImageRequest` 没有 `layer_decomposition` 字段，而且自定义未知字段虽会进入 `Extra`，序列化时明确不会合并回上游请求，因此该开关会被丢弃。非流式图片响应本身会原样转发，响应侧不是主要阻塞点。要接入图层拆分，至少需要增加并透传请求字段、验证多层响应与按层计费，不能只在渠道白名单中增加模型名。

## 6. Seedance 2.5 当前证据

本轮鉴权模型列表继续返回 `doubao-seedance-2-5-260628`。同一 New API 渠道 #1、同一把真实方舟 Key 在此前验收中已完成 15 个真实 Seedance 2.5 视频任务；最终基线为 4,039,299 actual tokens、New API 本地费用 ¥247.217424。逐笔任务和账单证据见：

- [poc-evidence.md](./poc-evidence.md)
- [employee-account-key-acceptance-2026-09-01.md](./employee-account-key-acceptance-2026-09-01.md)

因此没有为重复证明 2.5 权限再创建一条付费视频；本轮“当前列表仍可见 + 同 Key 已有真实成功任务”足以判定 `PASS`。

## 7. 为什么本轮费用不在 New API 页面出现

新增的 2.0 视频，以及 Seedream 文生图、图生图和图层拆分，是为了隔离验证真实方舟 Key 权限，直接调用了方舟 API，没有经过 `http://localhost:3000`。全部测试后数据库仍保持：

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
2. 文生图、图生图和图层拆分各自的正式人民币价格配置。
3. 通过 New API 虚拟 Key 的文生图、图生图和图层拆分请求与费用验收；本轮直连方舟成功不能替代这一层。
4. `ImageRequest` 增加并透传 `layer_decomposition`；当前未知字段在反序列化后不会重新合并，直接经过 New API 会丢失该开关。
5. 图层拆分按实际输出层数和各层像素档结算，并处理本次 `output_tokens` 与官方像素公式不一致的问题。

本轮已从正式价目页确认图层拆分的公开规则：不超过 261 万像素 ¥0.15/张，超过为 ¥0.30/张；同一次请求逐层按实际像素档计费。该规则足以推算测试值，但 API usage 与像素公式出现矛盾，且公司账号可能有具体商品、折扣和生效期，因此仍要用方舟费用明细确认真实实扣。文生图、普通图生图也不能直接套用拆层价格。

## 9. 后续接入顺序

1. 从公司方舟控制台取得 Seedance 2.0 与 Seedream 5.0 Pro 的当前价格截图/导出，记录生效时间和优惠期限。
2. 先新增专用测试虚拟 Key，只放行新增模型；不要立刻扩到所有员工 Key。
3. 为 Seedance 2.0 配置任务 usage 表达式，为 Seedream 分别配置普通生图和按输出层计费，并先跑完全本地的价格向量。
4. 为 New API 图片请求补齐 `layer_decomposition` 透传与测试，再通过 New API 分别执行：2.0 文生视频、Seedream 文生图、Seedream 图生图和图层拆分。
5. 核对虚拟 Key、所属员工、通用日志、模型看板、渠道累计和方舟官方账单。
6. 全部通过后再把模型加入正式员工 Key 白名单。

## 10. 官方接口参考

- [视频生成任务创建 API](https://api.volcengine.com/api-explorer/?action=CreateContentsGenerationsTasks&groupName=%E8%A7%86%E9%A2%91%E7%94%9F%E6%88%90API&serviceCode=ark&version=2024-01-01)
- [视频生成任务查询 API](https://api.volcengine.com/api-explorer/?action=GetContentsGenerationsTask&groupName=%E8%A7%86%E9%A2%91%E7%94%9F%E6%88%90API&serviceCode=ark&version=2024-01-01)
- [图片生成 API](https://api.volcengine.com/api-explorer/?action=ImageGenerations&groupName=%E5%9B%BE%E7%89%87%E7%94%9F%E6%88%90API&serviceCode=ark&version=2024-01-01)
- [Seedream 5.0 Pro 使用教程：图层拆分](https://docs.volcengine.com/docs/82379/2582774#layer_decomposition)
- [图片生成 API 正式参考](https://docs.volcengine.com/docs/82379/1541523)
- [火山方舟模型价格](https://docs.volcengine.com/docs/82379/1544106#457edfd0)
- [火山方舟产品页](https://www.volcengine.com/product/ark)
