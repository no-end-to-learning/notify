const Koa = require('koa');
const bodyParser = require('koa-bodyparser');

const larkRouter = require('./api/lark')

const app = new Koa();

app.use(bodyParser());
app.use(larkRouter.routes())

module.exports = app