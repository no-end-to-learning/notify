import Router from 'koa-router'
import { getService } from '../services/base.js'
import { SendMessageSchema, SendRawMessageSchema } from '../schemas/notify.js'

const router = new Router({ prefix: '/api/messages' })

// 发送格式化消息
router.post('/', async (ctx) => {
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

export default router
