import { z } from 'zod'

export const ChannelSchema = z.enum(['lark', 'wecom'])
export type Channel = z.infer<typeof ChannelSchema>

export const ColorSchema = z.enum(['Blue', 'Green', 'Orange', 'Grey', 'Red', 'Purple'])
export type Color = z.infer<typeof ColorSchema>

// 将 null 转换为 undefined，使 null 和不传字段行为一致
const nullable = <T extends z.ZodTypeAny>(schema: T) =>
  schema.nullish().transform((val) => val ?? undefined)

export const MessageParamsSchema = z.object({
  title: nullable(z.string()),
  color: nullable(ColorSchema),
  content: nullable(z.string()),
  image: nullable(z.string()),
  url: nullable(z.string().url()),
  note: nullable(z.string())
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
