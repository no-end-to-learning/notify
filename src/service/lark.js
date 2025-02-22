const config = require('config')
const axios = require('axios')
const lark = require("@larksuiteoapi/node-sdk");
const logger = require('../lib/logger')

const larkConfig = config.get('lark')
const larkClient = new lark.Client({
    appType: lark.AppType.SelfBuild,
    domain: lark.Domain.Feishu,
    loggerLevel: lark.LoggerLevel.error,
    ...larkConfig
})

exports.listChats = async () => {
    return larkClient.im.chat.list({
        params: {
            page_size: 100
        },
    }, lark.withTenantToken(""))
}

exports.sendCardMessageToChat = async (chatId, params) => {
    if (params.html) {
        params.url = this.HTMLContentToURL(params.html)
    }

    const message = {
        "config": {
            "wide_screen_mode": true
        },
        "elements": []
    }

    if (params.url) {
        message['card_link'] = {
            "url": params.url
        }
    }

    if (params.title) {
        message['header'] = {
            "title": {
                "tag": "plain_text",
                "content": params.title
            },
            "template": params.color || 'Blue'
        }
    }

    if (params.image) {
        message['elements'].push({
            "tag": "img",
            "img_key": params.image,
            "alt": {
                "tag": "plain_text",
                "content": params.title || 'image'
            }
        })
    }

    if (params.content) {
        message['elements'].push({
            "tag": "markdown",
            "content": params.content
        })
    }

    if (params.note) {
        if (params.content || params.url) {
            message['elements'].push({
                "tag": "hr"
            })
        }
        message['elements'].push({
            "tag": "note",
            "elements": [{
                "tag": "plain_text",
                "content": params.note
            }]
        })
    }

    return this.sendRawCardMessageToChat(chatId, message)
}

exports.sendRawCardMessageToChat = async (chatId, message) => {
    logger.info("send %s to %s", JSON.stringify(message), chatId)
    return larkClient.im.message.create({
        params: {
            receive_id_type: 'chat_id',
        },
        data: {
            receive_id: chatId,
            msg_type: 'interactive',
            content: JSON.stringify(message)
        },
    }, lark.withTenantToken(""));
}


exports.HTMLContentToURL = async (html) => {
    axios.post('https://bin.qiujun.xyz', {
        headers: {
            'Content-Type': 'text/html; charset=utf-8'
        },
        data: html,
    })
}
