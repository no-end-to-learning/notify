import Router from 'koa-router'
import { getService } from '../services/base.js'
import { logger } from '../lib/logger.js'
import { ValidationError } from '../lib/errors.js'
import {
  SendMessageSchema,
  SendRawMessageSchema,
  GrafanaAlertSchema,
  GrafanaQuerySchema,
  ChatsQuerySchema,
  type MessageParams
} from '../schemas/notify.js'

const router = new Router({ prefix: '/api/notify' })

// 发送消息
router.post('/send', async (ctx) => {
  const input = SendMessageSchema.parse(ctx.request.body)
  const service = getService(input.channel)
  const result = await service.sendMessage(input.to, input.params)
  ctx.body = result
})

// 发送原始消息
router.post('/raw', async (ctx) => {
  const input = SendRawMessageSchema.parse(ctx.request.body)
  const service = getService(input.channel)
  const result = await service.sendRawMessage(input.to, input.message)
  ctx.body = result
})

// Grafana 告警
router.post('/grafana', async (ctx) => {
  const query = GrafanaQuerySchema.parse(ctx.request.query)
  const body = GrafanaAlertSchema.parse(ctx.request.body)

  logger.info({ body }, 'Grafana alert received')

  const service = getService(query.channel)

  let params: MessageParams
  if (body.state === 'alerting') {
    params = {
      color: 'Orange',
      title: body.ruleName
    }
  } else if (body.state === 'ok') {
    params = {
      color: 'Green',
      title: '✅ ' + body.ruleName
    }
  } else {
    params = {
      color: 'Grey',
      title: body.ruleName
    }
  }

  if (body.evalMatches && body.evalMatches.length > 0) {
    params.content = body.evalMatches
      .map(item => `${item.metric}: ${item.value}`)
      .join('\n')
  }

  params.note = body.message
  if (!params.content && !params.note) {
    params.note = new Date().toString()
  }

  if (body.imageUrl && service.uploadImage) {
    params.image = await service.uploadImage(body.imageUrl)
  } else if (body.imageUrl) {
    params.image = body.imageUrl
  }

  const result = await service.sendMessage(query.to, params)
  ctx.body = result
})

// 获取聊天列表（仅飞书支持）
router.get('/chats', async (ctx) => {
  const query = ChatsQuerySchema.parse(ctx.request.query)
  const service = getService(query.channel)

  if (!service.listChats) {
    throw new ValidationError(`Channel ${query.channel} does not support listing chats`)
  }

  const chats = await service.listChats()
  ctx.body = chats
})

export default router
