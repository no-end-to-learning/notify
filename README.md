# Notify

多渠道通知网关服务，支持飞书和 Telegram。

## 功能

- 统一 API 接口，通过 `channel` 参数切换通知渠道
- 支持飞书卡片消息
- 支持 Telegram Bot
- Grafana 告警集成
- 内置消息队列与自动限频重试

## 快速开始

### 构建

```bash
go build -o notify .
```

### 配置

复制环境变量示例并填入实际值：

```bash
cp .env.example .env
```

使用 [direnv](https://direnv.net/) 自动加载（推荐）：

```bash
cp .env.example .envrc
vim .envrc  # 填入实际值
direnv allow
```

或手动加载：

```bash
source .env
```

### 运行

```bash
./notify
```

```bash
./notify
```

### Docker

镜像由 CI 自动构建并推送到 Docker Hub：

```bash
docker run -p 8000:8000 \
  -e APP_LARK_ID=your_app_id \
  -e APP_LARK_SECRET=your_app_secret \
  -e APP_TELEGRAM_BOT_TOKEN=your_bot_token \
  your_username/notify:latest
```

## API 接口

### 发送消息

```
POST /api/messages
Content-Type: application/json

{
  "channel": "lark",
  "target": "oc_xxx",
  "params": {
    "title": "消息标题",
    "color": "Blue",
    "content": "**Markdown** 内容",
    "note": "备注信息",
    "url": "https://example.com"
  }
}
```

**参数说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel | string | 是 | 通道类型：`lark` / `telegram` |
| target | string | 是 | 接收目标。飞书为 `chat_id`；Telegram 为 `chat_id` 或 `chat_id:thread_id`（支持 Topic）。 |
| params.title | string | 否 | 消息标题 |
| params.color | string | 否 | 标题颜色：Blue/Green/Orange/Grey/Red/Purple (Telegram 消息忽略此字段) |
| params.content | string | 否 | 消息内容（飞书支持 Markdown；Telegram 支持 HTML） |
| params.note | string | 否 | 备注 |
| params.url | string | 否 | 跳转链接 |

### 发送原始消息

直接透传对应平台的原始消息结构，用于发送更复杂的卡片或特殊消息。

```
POST /api/messages/raw
Content-Type: application/json

{
  "channel": "lark",
  "target": "oc_xxx",
  "message": {
    "config": { "wide_screen_mode": true },
    "header": { ... },
    "elements": [ ... ]
  }
}
```

### Grafana 告警

支持直接将 Grafana Webhook 指向此接口。

```
POST /api/webhooks/grafana?channel=lark&target=oc_xxx
Content-Type: application/json
```

**Query 参数**

- `channel`: `lark` 或 `telegram`
- `target`: 接收目标 ID

**Payload 示例**

```json
{
  "state": "alerting",
  "ruleName": "CPU 使用率过高",
  "message": "当前 CPU 使用率超过 90%",
  "evalMatches": [
    { "metric": "cpu_usage", "value": 95.5 }
  ]
}
```

### 获取聊天列表（仅飞书）

```
GET /api/chats?channel=lark
```

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| APP_SERVER_HOST | 服务监听地址 | 0.0.0.0 |
| APP_SERVER_PORT | 服务监听端口 | 8000 |
| APP_SERVER_BASE_URL | 服务基础 URL | http://localhost:8000/ |
| APP_LARK_ID | 飞书应用 App ID | - |
| APP_LARK_SECRET | 飞书应用 App Secret | - |
| APP_TELEGRAM_BOT_TOKEN | Telegram Bot Token | - |
| QUEUE_RATE_LIMIT | 发送速率限制 (个/秒) | 1.0 |
| QUEUE_MAX_RETRIES | 最大重试次数 | 3 |
| QUEUE_RETRY_DELAY | 重试延迟时间 | 1s |

## 限频与重试

为了保护下游服务（飞书、Telegram）不被请求淹没并避免触发其频率限制，本服务内置了针对 **每个目标（Channel + Target）** 的独立限频器。

- **限频策略**：每个 `channel + target` 组合拥有独立的发送队列和限频器。
  - 默认限频速率为 **1条/秒**（可通过 `QUEUE_RATE_LIMIT` 配置）。
  - 例如：同时向飞书群 A 和群 B 发送消息，它们互不影响，各自都能达到 1条/秒的速率。但如果短时间内向群 A 发送大量消息，这些消息会排队并按 1条/秒的速度依次发出。
- **自动重试**：如果发送失败（例如网络波动或 API 临时错误），系统会自动重试。
  - 默认重试 **3次**（`QUEUE_MAX_RETRIES`）。
  - 每次重试间隔 **1秒**（`QUEUE_RETRY_DELAY`）。
  - 超过重试次数仍失败的任务将被丢弃，并记录错误日志。

## 常见问题 (FAQ)

### 1. 飞书消息发送失败，提示权限不足？

请确保你的飞书自建应用已开通以下权限，并**发布了版本**：

- **im:message:send_as_bot** (以应用身份发送消息)：这是发送消息的基础权限。
- **im:chat:list** (获取群组列表)：如果你使用了 `/api/chats` 接口来列出群组，则需要此权限。

### 2. 如何获取 Telegram Chat ID？

你可以将 Bot 添加到群组，然后通过访问 `https://api.telegram.org/bot<YourBOTToken>/getUpdates` 查看更新，在 JSON 响应中找到 `chat.id` 字段（通常以 `-100` 开头）。或者使用第三方工具/Bot（如 `@get_id_bot`）来获取。
