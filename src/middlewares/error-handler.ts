import type { Context, Next } from 'koa'
import { ZodError } from 'zod'
import type { $ZodIssue } from 'zod/v4/core'
import { AppError } from '../lib/errors.js'
import { logger } from '../lib/logger.js'

export async function errorHandler(ctx: Context, next: Next) {
  try {
    await next()
  } catch (err) {
    if (err instanceof ZodError) {
      const message = err.issues.map((e: $ZodIssue) => `${e.path.join('.')}: ${e.message}`).join(', ')
      logger.warn({ path: ctx.path }, `Validation error: ${message}`)
      ctx.status = 400
      ctx.body = {
        error: 'VALIDATION_ERROR',
        message
      }
      return
    }

    if (err instanceof AppError) {
      logger.warn({ path: ctx.path, code: err.code }, err.message)
      ctx.status = err.statusCode
      ctx.body = {
        error: err.code || 'ERROR',
        message: err.message
      }
      return
    }

    logger.error({ err, path: ctx.path }, 'Unhandled error')
    ctx.status = 500
    ctx.body = {
      error: 'INTERNAL_ERROR',
      message: 'Internal server error'
    }
  }
}
