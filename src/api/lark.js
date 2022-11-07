const Router = require('koa-router');
const logger = require('../lib/logger')
const larkService = require('../service/lark')

var router = new Router({
    prefix: '/api/lark'
});

router.get('/chats', async (ctx, next) => {
    const res = await larkService.listChats()
    if (res.code !== 0) {
        ctx.throw(400, `${res.code} - ${res.msg}`);
    }
    ctx.body = res.data.items
})

router.post('/message/send', async (ctx, next) => {
    const { receive_id, params } = ctx.request.body
    const res = await larkService.sendCardMessageToChat(receive_id, params)
    if (res.code !== 0) {
        ctx.throw(400, `${res.code} - ${res.msg}`);
    }
    ctx.body = { message_id: res.data.message_id }
})

router.post('/message/send/grafana', async (ctx, next) => {
    const { body } = ctx.request
    logger.info("grafana alert %s", JSON.stringify(body));

    let params
    if (body.state === 'alerting') {
        params = {
            color: "Orange",
            title: body.ruleName,
        }
    } else if (body.state === "ok") {
        params = {
            color: "Green",
            title: '✅ ' + body.ruleName
        }
    } else {
        params = {
            color: "Grey",
            title: body.ruleName,
        }
    }

    body.evalMatches.sort((a,b) => a.metric - b.metric);
    params.content = body.evalMatches.map(item => `${item.metric}: ${item.value}`).join("\n")

    params.note = body.message;
    if (!params.content && !params.note) {
        params.note = new Date().toString()
    }

    params.url = body.ruleUrl

    if (body.imageUrl) {
        let imageRes = await larkService.uploadImage(body.imageUrl)
        params.image = imageRes?.data?.image_key
    }

    const { receive_id } = ctx.request.query
    const res = await larkService.sendCardMessageToChat(receive_id, params)
    if (res.code !== 0) {
        ctx.throw(400, `${res.code} - ${res.msg}`);
    }
    ctx.body = { message_id: res.data.message_id }
})

router.post('/message/send/raw', async (ctx, next) => {
    const { receive_id, message } = ctx.request.body
    const res = await larkService.sendRawCardMessageToChat(receive_id, message)
    if (res.code !== 0) {
        ctx.throw(400, `${res.code} - ${res.msg}`);
    }
    ctx.body = { message_id: res.data.message_id }
})

module.exports = router
