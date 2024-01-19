package servers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/api"
)

const (
	// 最大的消息大小
	maxMessageSize = 8192
)

type Controller struct {
}

type renderData struct {
	ClientId string `json:"clientId"`
}

func (c *Controller) Run(w http.ResponseWriter, r *http.Request) {
	conn, err := (&websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允许所有CORS跨域请求
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(w, r, nil)
	if err != nil {
		log.Errorf("upgrade error: %v", err)
		http.NotFound(w, r)
		return
	}

	//设置读取消息大小上线
	conn.SetReadLimit(maxMessageSize)

	//解析参数
	systemId := r.FormValue("systemId")
	if len(systemId) == 0 {
		systemId = r.FormValue("token")
		if len(systemId) == 0 {
			_ = Render(conn, "", "", "retcode.SYSTEM_ID_ERROR", "系统ID不能为空", []string{})
			_ = conn.Close()
			return
		}
	}
	logger := logrus.WithFields(log.Fields{"token": systemId})

	//clientId := util.GenClientId()
	clientId := systemId

	clientSocket := NewClient(clientId, systemId, conn)

	Manager.AddClient2SystemClient(systemId, clientSocket)
	logger.Info("添加客户端成功")
	//读取客户端消息
	clientSocket.Read()

	if err = api.ConnRender(conn, renderData{ClientId: clientId}); err != nil {
		_ = conn.Close()
		return
	}

	logger.Info("开始发送用户连接事件")
	// 用户连接事件
	Manager.Connect <- clientSocket
	logger.Info("用户连接事件发送成功")

	for {
		//接受消息
		//var receive []byte
		messageType, receive, err := conn.ReadMessage()
		if err != nil {
			logrus.WithFields(log.Fields{"messageType": messageType, "err": err}).Info("客户端下线")
			conn.Close()
			return
		}

		logrus.WithFields(log.Fields{"messageType": messageType, "receive": string(receive), "client_ip": r.RemoteAddr}).Info("收到客户端心跳消息")
		if messageType == 1 {
			var typeMsg TypeMessage
			json.Unmarshal(receive, &typeMsg)
			if typeMsg.Type == "KeepLive" {
				err = conn.WriteJSON(api.ConnSuc{
					Type: "KeepLive",
					Data: "Succeed",
				})
				if err != nil {
					logrus.WithFields(log.Fields{"messageType": messageType, "receive": string(receive), "client_ip": r.RemoteAddr}).Info("Pong失败")
				} else {
					logrus.WithFields(log.Fields{"messageType": messageType, "receive": string(receive), "client_ip": r.RemoteAddr}).Info("Pong成功")
				}
			}
		}

	}
}

type TypeMessage struct {
	Type interface{} `json:"Type"`
	Data interface{} `json:"Data"`
}
