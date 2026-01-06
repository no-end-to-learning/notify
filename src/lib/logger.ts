import pino from 'pino'
import pretty from 'pino-pretty'

const stream = pretty({
  colorize: true,
  singleLine: true
})

export const logger = pino(stream)
