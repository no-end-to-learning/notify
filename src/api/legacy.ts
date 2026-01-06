// 旧版 API 兼容层，待废弃后删除
import Router from 'koa-router'
import { getService } from '../services/base.js'
import { logger } from '../lib/logger.js'
import { GrafanaAlertSchema, MessageParamsSchema } from '../schemas/notify.js'

const router = new Router({ prefix: '/api/lark' })

const larkService = getService('lark')

router.get('/chats', async (ctx) => {
  if (!larkService.listChats) {
    ctx.throw(501, 'Not implemented')
    return
  }
  const chats = await larkService.listChats()
  // 转换为旧版数据结构
  ctx.body = chats.map(chat => ({
    chat_id: chat.chatId,
    name: chat.name,
    description: chat.description
  }))
})

router.post('/message/send', async (ctx) => {
  const { receive_id, params } = ctx.request.body as { receive_id: string; params: unknown }
  const validatedParams = MessageParamsSchema.parse(params)
  const result = await larkService.sendMessage(receive_id, validatedParams)
  ctx.body = { message_id: result.messageId }
})

router.post('/message/send/grafana', async (ctx) => {
  const body = GrafanaAlertSchema.parse(ctx.request.body)
  const receive_id = ctx.request.query.receive_id as string

  logger.info({ body }, 'Grafana alert received (legacy)')

  let params: { color?: string; title: string; content?: string; note?: string; image?: string }
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

  if (body.imageUrl && larkService.uploadImage) {
    params.image = await larkService.uploadImage(body.imageUrl)
  }

  const result = await larkService.sendMessage(receive_id, params as Parameters<typeof larkService.sendMessage>[1])
  ctx.body = { message_id: result.messageId }
})

router.post('/message/send/raw', async (ctx) => {
  const { receive_id, message } = ctx.request.body as { receive_id: string; message: unknown }
  const result = await larkService.sendRawMessage(receive_id, message)
  ctx.body = { message_id: result.messageId }
})

export default router
