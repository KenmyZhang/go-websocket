package servers

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/api"
)

type Client struct {
	Addr        string          `json:"addr"`
	ClientId    string          // 标识ID
	SystemId    string          // 系统ID
	Socket      *websocket.Conn // 用户连接
	ConnectTime uint64          // 首次连接时间
	IsDeleted   bool            // 是否删除或下线
	UserId      string          // 业务端标识用户ID
	Extend      string          // 扩展字段，用户可以自定义
	GroupList   []string
}

type SendData struct {
	Code int
	Msg  string
	Data *interface{}
}

func NewClient(clientId string, addr, systemId string, socket *websocket.Conn) *Client {
	return &Client{
		Addr:        addr,
		ClientId:    clientId,
		SystemId:    systemId,
		Socket:      socket,
		ConnectTime: uint64(time.Now().Unix()),
		IsDeleted:   false,
	}
}

func (c *Client) Read(clientAddr string) {
	traceId := uuid.New()
	logger := logrus.WithFields(log.Fields{"trace_id": traceId})
	go func() {
		for {
			if c.Socket == nil {
				logrus.Error("无效socket")
				return
			}
			messageType, receive, err := c.Socket.ReadMessage()
			if err != nil {
				if messageType == -1 && websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
					Manager.DisConnect <- c
					return
				} else if messageType != websocket.PingMessage {
					return
				}
			}

			var typeMsg TypeMessage
			json.Unmarshal(receive, &typeMsg)
			if typeMsg.Type == "KeepLive" {
				data := api.ConnSuc{
					Type: "KeepLive",
					Data: "Succeed",
				}
				strData, _ := json.Marshal(&data)
				c.Socket.WriteMessage(websocket.TextMessage, strData)
				if err != nil {
					logger.WithFields(log.Fields{"messageType": messageType, "receive": string(receive), "client_ip": clientAddr}).Info("Pong失败")
				}
			} else {
				logger.WithFields(log.Fields{"client_ip": clientAddr}).Info("其它类型消息")
			}

		}
	}()
}
