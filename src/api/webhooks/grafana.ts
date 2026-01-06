import Router from '@koa/router'
import { getService } from '../../services/base.js'
import { logger } from '../../lib/logger.js'
import { GrafanaAlertSchema, GrafanaQuerySchema, type GrafanaAlert, type Channel } from '../../schemas/notify.js'

const router = new Router({ prefix: '/api/webhooks' })

router.post('/grafana', async (ctx) => {
  const query = GrafanaQuerySchema.parse(ctx.request.query)
  const body = GrafanaAlertSchema.parse(ctx.request.body)

  logger.info({ body }, 'Grafana alert received')

  const service = getService(query.channel)
  const message = buildMessage(query.channel, body)
  const result = await service.sendRawMessage(query.to, message)

  ctx.body = result
})

function buildMessage(channel: Channel, alert: GrafanaAlert): unknown {
  if (channel === 'wecom') {
    return buildWecomMessage(alert)
  }
  return buildLarkMessage(alert)
}

function buildWecomMessage(alert: GrafanaAlert) {
  // 使用 markdown 消息
  const stateEmoji = {
    alerting: '⚠️',
    ok: '✅',
    default: '📢'
  }
  const emoji = stateEmoji[alert.state as keyof typeof stateEmoji] || stateEmoji.default

  const parts: string[] = []

  // 标题
  parts.push(`### ${emoji} ${alert.ruleName}`)

  // 指标数据
  if (alert.evalMatches && alert.evalMatches.length > 0) {
    parts.push('<font color="comment">──────────</font>')
    const items = alert.evalMatches.map(item => `${item.metric}: ${item.value}`)
    parts.push(items.join('\n'))
  }

  // 消息（灰色）
  if (alert.message) {
    parts.push('<font color="comment">──────────</font>')
    const lines = alert.message.split('\n').map(line => {
      const trimmed = line.replace(/^- /, '')
      return `<font color="comment">${trimmed}</font>`
    })
    parts.push(lines.join('\n'))
  }

  // 如果没有内容，显示时间
  if (parts.length === 1) {
    parts.push(`> ${new Date().toString()}`)
  }

  return {
    msgtype: 'markdown',
    markdown: {
      content: parts.join('\n')
    }
  }
}

function buildLarkMessage(alert: GrafanaAlert) {
  const elements: Array<{ tag: string; [key: string]: unknown }> = []

  let template: string
  let title: string
  if (alert.state === 'alerting') {
    template = 'Orange'
    title = alert.ruleName
  } else if (alert.state === 'ok') {
    template = 'Green'
    title = '✅ ' + alert.ruleName
  } else {
    template = 'Grey'
    title = alert.ruleName
  }

  if (alert.evalMatches && alert.evalMatches.length > 0) {
    elements.push({
      tag: 'markdown',
      content: alert.evalMatches
        .map(item => `**${item.metric}**: ${item.value}`)
        .join('\n')
    })
  }

  if (alert.message) {
    if (elements.length > 0) {
      elements.push({ tag: 'hr' })
    }
    elements.push({
      tag: 'note',
      elements: [{ tag: 'plain_text', content: alert.message }]
    })
  }

  if (elements.length === 0) {
    elements.push({
      tag: 'note',
      elements: [{ tag: 'plain_text', content: new Date().toString() }]
    })
  }

  return {
    config: { wide_screen_mode: true },
    header: {
      title: { tag: 'plain_text', content: title },
      template
    },
    ...(alert.ruleUrl && { card_link: { url: alert.ruleUrl } }),
    elements
  }
}

export default router
