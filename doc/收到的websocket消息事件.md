# websocket连接成功收到的event

    {
        "code":0,
        "msg":"success",
        "data":{
            "clientId":"xxx123"
        }
    }

# 接收消息的event结构
    messageId: 消息ID
    Data：消息内容
    Type: 消息类型

## Example

    {
        "messageId":"ce9d48f69d53a08c",
        "Type":"1",
        "Data":"123"
    }