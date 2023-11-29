# 发送websocket消息
## Content-Type
    application/json
## 请求参数
    clientId：接收方id
    data：消息内容
    type: 消息类型

## 响应参数

    code 成功返回0，否则返回其它错误码
    msg 成功返回success，否则返回其它错误信息
    messageId 消息id

## Example
    curl -X POST "http://13.213.8.98:7800/api/send_to_client" -i -d '{"clientId":"xxx123", "data":"123", "type":"1"}'
    HTTP/1.1 200 OK
    Content-Type: application/json; charset=utf-8
    Date: Wed, 29 Nov 2023 16:34:22 GMT
    Content-Length: 66

    {
        "code":0,
        "msg":"success",
        "data":{
            "messageId":"af6c4055af578007"
        }
    }


