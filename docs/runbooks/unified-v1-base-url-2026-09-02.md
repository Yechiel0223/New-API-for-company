# New API 员工统一 Base URL 说明

日期：2026-09-02

## 结论

员工侧统一只需要填写一个 Base URL：

```text
http://124.174.23.197:3000/v1
```

对应使用自己的 New API 虚拟 Key，不使用火山方舟真实 Key。

## 接口路径

| 场景 | 员工填写的 Base URL | 客户端最终路径 |
| --- | --- | --- |
| Seedream 生图 | `http://124.174.23.197:3000/v1` | `POST /images/generations` |
| Seedance 视频任务创建 | `http://124.174.23.197:3000/v1` | `POST /contents/generations/tasks` |
| Seedance 视频任务查询 | `http://124.174.23.197:3000/v1` | `GET /contents/generations/tasks/{task_id}` |
| 后续语言模型 | `http://124.174.23.197:3000/v1` | `POST /chat/completions` |

## 本次改动

为了避免员工同时记忆 `/v1` 和 `/doubao` 两套 Base URL，Doubao/Seedance 视频任务插件新增了两条 `/v1` 兼容路由：

```text
POST /v1/contents/generations/tasks
GET  /v1/contents/generations/tasks/{task_id}
```

旧入口仍保留：

```text
POST /doubao/api/v3/contents/generations/tasks
GET  /doubao/api/v3/contents/generations/tasks/{task_id}
```

旧入口只是内部/历史兼容入口，不建议再写进员工使用文档。

## 计费和数据影响

这次只统一入口路径，不修改计费规则、渠道、虚拟 Key、任务日志或数据看板逻辑。

Seedance 视频仍沿用已经验收过的任务链路：

```text
提交任务 -> 记录任务 -> 轮询终态 -> 按方舟 actual tokens 结算
```

Seedream 生图仍沿用后付费链路：

```text
请求成功 -> 按输出图片数量、尺寸和场景后扣费
```

## Python 测试脚本

测试脚本：

```text
scripts/test_seedance25_video.py
```

默认 Base URL 已改为：

```text
http://124.174.23.197:3000/v1
```

也兼容输入根地址：

```text
http://124.174.23.197:3000
```

脚本会自动拼成：

```text
http://124.174.23.197:3000/v1/contents/generations/tasks
```
