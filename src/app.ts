import Koa from 'koa'
import bodyParser from 'koa-bodyparser'
import { errorHandler } from './middlewares/error-handler.js'
import notifyRouter from './api/notify.js'

const app = new Koa()

app.use(errorHandler)
app.use(bodyParser())
app.use(notifyRouter.routes())
app.use(notifyRouter.allowedMethods())

export default app
