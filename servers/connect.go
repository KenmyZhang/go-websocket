package servers

import (
	"fmt"
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

	//解析参数
	systemId := r.FormValue("systemId")
	logger := logrus.WithFields(log.Fields{"token": systemId, "client_addr": r.RemoteAddr})

	// 打印头部信息
	for name, values := range r.Header {
		for _, value := range values {
			logger.Info(fmt.Sprintf("%s: %s\n", name, value))
		}
	}

	// 打印请求参数信息
	r.ParseForm()
	for name, values := range r.Form {
		for _, value := range values {
			logger.Info(fmt.Sprintf("%s: %s\n", name, value))
		}
	}

	conn, err := (&websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// 允许所有CORS跨域请求
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("upgrade error: %v", err)
		http.NotFound(w, r)
		return
	}

	//设置读取消息大小上线
	conn.SetReadLimit(maxMessageSize)

	if len(systemId) == 0 {
		systemId = r.FormValue("token")
		if len(systemId) == 0 {
			logger.Error("系统ID不能为空")
			_ = Render(conn, "", "", "retcode.SYSTEM_ID_ERROR", "系统ID不能为空", []string{})
			_ = conn.Close()
			return
		}
	}

	//clientId := util.GenClientId()
	clientId := systemId

	clientSocket := NewClient(clientId, r.RemoteAddr, systemId, conn)

	Manager.AddClient2SystemClient(systemId, r.RemoteAddr, clientSocket)
	logger.Info("添加客户端成功")
	//读取客户端消息
	clientSocket.Read(r.RemoteAddr)

	if err = api.ConnRender(conn, renderData{ClientId: clientId}); err != nil {
		_ = conn.Close()
		return
	}
	logger.Info("开始发送用户连接事件")
	// 用户连接事件
	Manager.Connect <- clientSocket
	logger.Info("用户连接事件发送成功")

}

type TypeMessage struct {
	Type interface{} `json:"Type"`
	Data interface{} `json:"Data"`
}
