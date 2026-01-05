export class AppError extends Error {
  constructor(
    public statusCode: number,
    message: string,
    public code?: string
  ) {
    super(message)
    this.name = 'AppError'
  }
}

export class ValidationError extends AppError {
  constructor(message: string) {
    super(400, message, 'VALIDATION_ERROR')
    this.name = 'ValidationError'
  }
}

export class NotFoundError extends AppError {
  constructor(message: string) {
    super(404, message, 'NOT_FOUND')
    this.name = 'NotFoundError'
  }
}

export class ServiceError extends AppError {
  constructor(message: string, public service: string) {
    super(502, message, 'SERVICE_ERROR')
    this.name = 'ServiceError'
  }
}
