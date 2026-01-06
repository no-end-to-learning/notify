import Router from 'koa-router'
import { getService } from '../services/base.js'
import { ValidationError } from '../lib/errors.js'
import { ChatsQuerySchema } from '../schemas/notify.js'

const router = new Router({ prefix: '/api/chats' })

// 获取聊天列表
router.get('/', async (ctx) => {
  const query = ChatsQuerySchema.parse(ctx.request.query)
  const service = getService(query.channel)

  if (!service.listChats) {
    throw new ValidationError(`Channel ${query.channel} does not support listing chats`)
  }

  const chats = await service.listChats()
  ctx.body = chats
})

export default router
