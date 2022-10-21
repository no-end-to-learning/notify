const config = require('config')
const axios = require('axios')
const lark = require("@larksuiteoapi/allcore");
const logger = require('../lib/logger')

const larkConfig = config.get('lark')
const appSettings = lark.newInternalAppSettings(larkConfig)

const conf = lark.newConfig(lark.Domain.LarkSuite, appSettings, {
    loggerLevel: lark.LoggerLevel.ERROR,
})

exports.listChats = async () => {
    const req = lark.api.newRequest("/open-apis/im/v1/chats", "GET", lark.api.AccessTokenType.Tenant)
    return lark.api.sendRequest(conf, req)
}

exports.sendCardMessageToChat = async (chatId, params) => {
    if (params.html) {
        params.url = larkService.HTMLContentToURL(params.html)
    }

    const message = {
        "config": {
            "wide_screen_mode": true
        },
        "elements": []
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

    if (params.content) {
        message['elements'].push({
                "tag": "markdown",
                "content": params.content
        })
    }

    if (params.url) {
        message['elements'].push({
                "tag": "action",
                "actions": [{
                    "tag": "button",
                    "text": {
                        "tag": "plain_text",
                        "content": "View Details"
                    },
                    "url": params.url
                }]
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
    const messageBody = {
        "receive_id": chatId,
        "msg_type": "interactive",
        "content": JSON.stringify(message)
    }

    const req = lark.api.newRequest("/open-apis/im/v1/messages?receive_id_type=chat_id", "POST", lark.api.AccessTokenType.Tenant, messageBody)
    return lark.api.sendRequest(conf, req)
}


exports.HTMLContentToURL = async (html) => {
    axios.post('https://bin.qiujun.xyz', {
        headers: {
            'Content-Type': 'text/html; charset=utf-8'
        },
        data: html,
    })
}
