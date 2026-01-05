import config from 'config'
import axios from 'axios'
import type { NotifyService, SendResult } from './base.js'
import type { MessageParams, Channel, Color } from '../schemas/notify.js'
import { logger } from '../lib/logger.js'
import { ServiceError } from '../lib/errors.js'

interface WecomConfig {
  webhookUrl: string
}

interface WecomMarkdownMessage {
  msgtype: 'markdown'
  markdown: {
    content: string
  }
}

interface WecomNewsMessage {
  msgtype: 'news'
  news: {
    articles: Array<{
      title: string
      description?: string
      url?: string
      picurl?: string
    }>
  }
}

type WecomMessage = WecomMarkdownMessage | WecomNewsMessage

const COLOR_EMOJI: Record<Color, string> = {
  Blue: 'ℹ️',
  Green: '✅',
  Orange: '⚠️',
  Grey: '⏸️',
  Red: '❌',
  Purple: '🔮'
}

export class WecomService implements NotifyService {
  readonly channel: Channel = 'wecom'
  private webhookUrl: string

  constructor() {
    const wecomConfig = config.get<WecomConfig>('wecom')
    this.webhookUrl = wecomConfig.webhookUrl
  }

  async sendMessage(to: string, params: MessageParams): Promise<SendResult> {
    const message = this.buildMessage(params)
    return this.sendRawMessage(to, message)
  }

  async sendRawMessage(to: string, message: unknown): Promise<SendResult> {
    const url = `${this.webhookUrl}?key=${to}`
    logger.info({ message, to }, 'Sending WeCom message')

    const res = await axios.post(url, message)

    if (res.data.errcode !== 0) {
      throw new ServiceError(`${res.data.errcode} - ${res.data.errmsg}`, 'wecom')
    }

    return { success: true }
  }

  private buildMessage(params: MessageParams): WecomMessage {
    if (params.image || params.url) {
      return this.buildNewsMessage(params)
    }
    return this.buildMarkdownMessage(params)
  }

  private buildMarkdownMessage(params: MessageParams): WecomMarkdownMessage {
    const parts: string[] = []

    if (params.title) {
      const emoji = params.color ? COLOR_EMOJI[params.color] : ''
      parts.push(`### ${emoji} ${params.title}`)
    }

    if (params.content) {
      parts.push(params.content)
    }

    if (params.note) {
      parts.push(`> ${params.note}`)
    }

    return {
      msgtype: 'markdown',
      markdown: {
        content: parts.join('\n\n')
      }
    }
  }

  private buildNewsMessage(params: MessageParams): WecomNewsMessage {
    return {
      msgtype: 'news',
      news: {
        articles: [{
          title: params.title || 'Notification',
          description: [params.content, params.note].filter(Boolean).join('\n\n'),
          url: params.url,
          picurl: params.image
        }]
      }
    }
  }
}
