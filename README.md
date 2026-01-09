# Notify

多渠道通知网关服务，支持飞书、企业微信和 Telegram。

## 功能

- 统一 API 接口，通过 `channel` 参数切换通知渠道
- 支持飞书卡片消息
- 支持企业微信 Webhook 机器人
- 支持 Telegram Bot
- Grafana 告警集成

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
  "to": "oc_xxx",
  "params": {
    "title": "消息标题",
    "color": "Blue",
    "content": "**Markdown** 内容",
    "note": "备注信息",
    "url": "https://example.com",
    "image": "image_key"
  }
}
```

**参数说明**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| channel | string | 是 | 通道类型：`lark` / `wecom` / `telegram` |
| to | string | 是 | 飞书群聊 ID / 企业微信 Webhook Key / Telegram chat_id |
| params.title | string | 否 | 消息标题 |
| params.color | string | 否 | 标题颜色：Blue/Green/Orange/Grey/Red/Purple |
| params.content | string | 否 | Markdown 内容 |
| params.note | string | 否 | 备注 |
| params.url | string | 否 | 跳转链接 |
| params.image | string | 否 | 图片（飞书为 image_key，其他为图片 URL） |

### 发送原始消息

```
POST /api/messages/raw
Content-Type: application/json

{
  "channel": "lark",
  "to": "oc_xxx",
  "message": {
    "config": { "wide_screen_mode": true },
    "header": { ... },
    "elements": [ ... ]
  }
}
```

### Grafana 告警

```
POST /api/webhooks/grafana?channel=lark&to=oc_xxx
Content-Type: application/json

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
| APP_WECOM_WEBHOOK_URL | 企业微信 Webhook 基础 URL | https://qyapi.weixin.qq.com/cgi-bin/webhook/send |
| APP_TELEGRAM_BOT_TOKEN | Telegram Bot Token | - |

## License

ISC
