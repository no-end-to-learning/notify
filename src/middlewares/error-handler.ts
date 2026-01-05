import type { Context, Next } from 'koa'
import { ZodError } from 'zod'
import { AppError } from '../lib/errors.js'
import { logger } from '../lib/logger.js'

export async function errorHandler(ctx: Context, next: Next) {
  try {
    await next()
  } catch (err) {
    if (err instanceof ZodError) {
      ctx.status = 400
      ctx.body = {
        error: 'VALIDATION_ERROR',
        message: err.errors.map(e => `${e.path.join('.')}: ${e.message}`).join(', ')
      }
      return
    }

    if (err instanceof AppError) {
      ctx.status = err.statusCode
      ctx.body = {
        error: err.code || 'ERROR',
        message: err.message
      }
      return
    }

    logger.error(err, 'Unhandled error')
    ctx.status = 500
    ctx.body = {
      error: 'INTERNAL_ERROR',
      message: 'Internal server error'
    }
  }
}
