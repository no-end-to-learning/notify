# Notify

多渠道通知网关服务，支持飞书和企业微信。

## 功能

- 统一 API 接口，通过 `channel` 参数切换通知渠道
- 支持飞书卡片消息
- 支持企业微信 Webhook 机器人
- Grafana 告警集成
- Zod 参数校验
- TypeScript 类型安全

## 快速开始

### 安装依赖

```bash
npm install
```

### 配置

复制配置文件并填入实际凭证：

```bash
cp config/default.json config/local.json
```

编辑 `config/local.json`：

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": 8000,
    "baseURL": "http://localhost:8000/"
  },
  "lark": {
    "appId": "your_lark_app_id",
    "appSecret": "your_lark_app_secret"
  },
  "wecom": {
    "webhookUrl": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send"
  }
}
```

### 运行

```bash
# 开发模式
npm run dev

# 生产构建
npm run build
npm start
```

### Docker

```bash
docker build -t notify .
docker run -p 8000:8000 \
  -e APP_LARK_ID=your_app_id \
  -e APP_LARK_SECRET=your_app_secret \
  notify
```

## API 接口

### 发送消息

```
POST /api/notify/send
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
| channel | string | 是 | 通道类型：`lark` / `wecom` |
| to | string | 是 | 飞书群聊 ID 或企业微信 Webhook Key |
| params.title | string | 否 | 消息标题 |
| params.color | string | 否 | 标题颜色：Blue/Green/Orange/Grey/Red/Purple |
| params.content | string | 否 | Markdown 内容 |
| params.note | string | 否 | 备注 |
| params.url | string | 否 | 跳转链接 |
| params.image | string | 否 | 图片（飞书为 image_key，企业微信为图片 URL） |

### 发送原始消息

```
POST /api/notify/raw
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
POST /api/notify/grafana?channel=lark&to=oc_xxx
Content-Type: application/json

{
  "state": "alerting",
  "ruleName": "CPU 使用率过高",
  "message": "当前 CPU 使用率超过 90%",
  "evalMatches": [
    { "metric": "cpu_usage", "value": 95.5 }
  ],
  "imageUrl": "https://..."
}
```

### 获取聊天列表（仅飞书）

```
GET /api/notify/chats?channel=lark
```

## 环境变量

| 变量 | 说明 |
|------|------|
| APP_SERVER_HOST | 服务监听地址 |
| APP_SERVER_PORT | 服务监听端口 |
| APP_LARK_ID | 飞书应用 App ID |
| APP_LARK_SECRET | 飞书应用 App Secret |
| APP_WECOM_WEBHOOK_URL | 企业微信 Webhook 基础 URL |

## License

ISC
