import type { MessageParams, Channel } from '../schemas/notify.js'

export interface SendResult {
  messageId?: string
  success: boolean
}

export interface ChatItem {
  chatId: string
  name: string
  description?: string
}

export interface NotifyService {
  readonly channel: Channel
  sendMessage(to: string, params: MessageParams): Promise<SendResult>
  sendRawMessage(to: string, message: unknown): Promise<SendResult>
  uploadImage?(imageUrl: string): Promise<string>
  listChats?(): Promise<ChatItem[]>
}

import { LarkService } from './lark.js'
import { WecomService } from './wecom.js'
import { NotFoundError } from '../lib/errors.js'

const services: Record<Channel, NotifyService> = {
  lark: new LarkService(),
  wecom: new WecomService()
}

export function getService(channel: Channel): NotifyService {
  const service = services[channel]
  if (!service) {
    throw new NotFoundError(`Unknown channel: ${channel}`)
  }
  return service
}
