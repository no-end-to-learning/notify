import Koa from 'koa'
import bodyParser from 'koa-bodyparser'
import { errorHandler } from './middlewares/error-handler.js'
import messagesRouter from './api/messages.js'
import chatsRouter from './api/chats.js'
import { grafanaRouter } from './api/webhooks/index.js'
import legacyRouter from './api/legacy.js'

const app = new Koa()

app.use(errorHandler)
app.use(bodyParser())
app.use(messagesRouter.routes())
app.use(messagesRouter.allowedMethods())
app.use(chatsRouter.routes())
app.use(chatsRouter.allowedMethods())
app.use(grafanaRouter.routes())
app.use(grafanaRouter.allowedMethods())
app.use(legacyRouter.routes())
app.use(legacyRouter.allowedMethods())

export default app
