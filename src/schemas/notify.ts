import { z } from 'zod'

export const ChannelSchema = z.enum(['lark', 'wecom'])
export type Channel = z.infer<typeof ChannelSchema>

export const ColorSchema = z.enum(['Blue', 'Green', 'Orange', 'Grey', 'Red', 'Purple'])
export type Color = z.infer<typeof ColorSchema>

export const MessageParamsSchema = z.object({
  title: z.string().optional(),
  color: ColorSchema.optional(),
  content: z.string().optional(),
  image: z.string().optional(),
  url: z.string().url().optional(),
  note: z.string().optional()
})
export type MessageParams = z.infer<typeof MessageParamsSchema>

export const SendMessageSchema = z.object({
  channel: ChannelSchema,
  to: z.string().min(1),
  params: MessageParamsSchema
})
export type SendMessageInput = z.infer<typeof SendMessageSchema>

export const SendRawMessageSchema = z.object({
  channel: ChannelSchema,
  to: z.string().min(1),
  message: z.record(z.unknown())
})
export type SendRawMessageInput = z.infer<typeof SendRawMessageSchema>

export const GrafanaAlertSchema = z.object({
  state: z.string(),
  ruleName: z.string(),
  ruleUrl: z.string().optional(),
  message: z.string().optional(),
  imageUrl: z.string().optional(),
  evalMatches: z.array(z.object({
    metric: z.string(),
    value: z.number()
  })).optional()
})
export type GrafanaAlert = z.infer<typeof GrafanaAlertSchema>

export const GrafanaQuerySchema = z.object({
  channel: ChannelSchema,
  to: z.string().min(1)
})

export const ChatsQuerySchema = z.object({
  channel: z.literal('lark')
})
