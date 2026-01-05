import config from 'config'
import axios from 'axios'
import streamifier from 'streamifier'
import * as lark from '@larksuiteoapi/node-sdk'
import type { NotifyService, SendResult, ChatItem } from './base.js'
import type { MessageParams, Channel } from '../schemas/notify.js'
import { logger } from '../lib/logger.js'
import { ServiceError } from '../lib/errors.js'

interface LarkConfig {
  appId: string
  appSecret: string
}

interface LarkCardMessage {
  config?: { wide_screen_mode?: boolean }
  header?: {
    title: { tag: string; content: string }
    template?: string
  }
  card_link?: { url: string }
  elements: Array<{
    tag: string
    [key: string]: unknown
  }>
}

export class LarkService implements NotifyService {
  readonly channel: Channel = 'lark'
  private client: lark.Client

  constructor() {
    const larkConfig = config.get<LarkConfig>('lark')
    this.client = new lark.Client({
      appType: lark.AppType.SelfBuild,
      domain: lark.Domain.Feishu,
      loggerLevel: lark.LoggerLevel.error,
      ...larkConfig
    })
  }

  async sendMessage(to: string, params: MessageParams): Promise<SendResult> {
    const message = this.buildCardMessage(params)
    return this.sendRawMessage(to, message)
  }

  async sendRawMessage(to: string, message: unknown): Promise<SendResult> {
    logger.info({ message, to }, 'Sending Lark message')

    const res = await this.client.im.message.create({
      params: { receive_id_type: 'chat_id' },
      data: {
        receive_id: to,
        msg_type: 'interactive',
        content: JSON.stringify(message)
      }
    }, lark.withTenantToken(''))

    if (res.code !== 0) {
      throw new ServiceError(`${res.code} - ${res.msg}`, 'lark')
    }

    return {
      messageId: res.data?.message_id,
      success: true
    }
  }

  async uploadImage(imageUrl: string): Promise<string> {
    const downloadResponse = await axios.get(imageUrl, { responseType: 'arraybuffer' })
    const imageBuffer = Buffer.from(downloadResponse.data)
    const imageReadableStream = streamifier.createReadStream(imageBuffer)

    const uploadResponse = await this.client.im.v1.image.create({
      data: {
        image_type: 'message',
        image: imageReadableStream as unknown as Buffer
      }
    })

    if (!uploadResponse?.image_key) {
      throw new ServiceError('Failed to upload image', 'lark')
    }

    return uploadResponse.image_key
  }

  async listChats(): Promise<ChatItem[]> {
    const res = await this.client.im.chat.list({
      params: { page_size: 100 }
    }, lark.withTenantToken(''))

    if (res.code !== 0) {
      throw new ServiceError(`${res.code} - ${res.msg}`, 'lark')
    }

    return (res.data?.items || []).map(item => ({
      chatId: item.chat_id!,
      name: item.name!,
      description: item.description
    }))
  }

  private buildCardMessage(params: MessageParams): LarkCardMessage {
    const message: LarkCardMessage = {
      config: { wide_screen_mode: true },
      elements: []
    }

    if (params.url) {
      message.card_link = { url: params.url }
    }

    if (params.title) {
      message.header = {
        title: { tag: 'plain_text', content: params.title },
        template: params.color || 'Blue'
      }
    }

    if (params.image) {
      message.elements.push({
        tag: 'img',
        img_key: params.image,
        alt: { tag: 'plain_text', content: params.title || 'image' }
      })
    }

    if (params.content) {
      message.elements.push({
        tag: 'markdown',
        content: params.content
      })
    }

    if (params.note) {
      if (params.content || params.url) {
        message.elements.push({ tag: 'hr' })
      }
      message.elements.push({
        tag: 'note',
        elements: [{ tag: 'plain_text', content: params.note }]
      })
    }

    return message
  }
}
