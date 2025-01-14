package servers

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/pkg/setting"
	"github.com/woodylan/go-websocket/tools/util"
)

// channel通道
var ToClientChan = make(chan clientInfo, 10000)

// channel通道结构体
type clientInfo struct {
	ClientId   string
	SendUserId string
	MessageId  string
	Code       string
	Msg        string
	Data       *string
}

func (o *clientInfo) GetData() string {
	if o.Data == nil {
		return ""
	}
	return *o.Data
}

type RetData struct {
	MessageId  string `json:"messageId"`
	SendUserId string `json:"-"`
	Code       string `json:"Type"`
	Msg        string `json:"-"`
	//Code       int         `json:"code"`
	//Msg        string      `json:"msg"`
	Data interface{} `json:"Data"`
}

// 心跳间隔
var heartbeatInterval = 25 * time.Second

func init() {
	ToClientChan = make(chan clientInfo, 1000)
}

var Manager = NewClientManager() // 管理者

func StartWebSocket() {
	websocketHandler := &Controller{}
	http.HandleFunc("/ws", websocketHandler.Run)

	go Manager.Start()
}

// 发送信息到指定客户端
func SendMessage2Client(clientId string, sendUserId string, code string, msg string, data *string) (messageId string) {
	messageId = util.GenUUID()
	if util.IsCluster() {
		addr, _, _, isLocal, err := util.GetAddrInfoAndIsLocal(clientId)
		if err != nil {
			log.Errorf("%s", err)
			return
		}
		log.WithFields(log.Fields{"addr": addr, "isLocal": isLocal}).Info("发送到机器位置")
		//如果是本机则发送到本机
		if isLocal {
			SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
		} else {
			//发送到指定机器 TODO
			log.WithFields(log.Fields{"addr": addr}).Info("发送到指定机器")
			// SendRpc2Client(addr, messageId, sendUserId, clientId, code, msg, data)
		}
	} else {
		log.WithFields(log.Fields{"data": data, "clientId": clientId, "sendUserId": sendUserId}).Info("单机发送")

		//如果是单机服务，则只发送到本机
		go SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
	}

	return
}

// 关闭客户端
func CloseClient(clientId, systemId, clientAddr string) {
	if util.IsCluster() {
		addr, _, _, isLocal, err := util.GetAddrInfoAndIsLocal(clientId)
		if err != nil {
			log.Errorf("%s", err)
			return
		}

		//如果是本机则发送到本机
		if isLocal {
			CloseLocalClient(clientId, systemId, clientAddr)
		} else {
			//发送到指定机器
			CloseRpcClient(addr, clientId, systemId)
		}
	} else {
		//如果是单机服务，则只发送到本机
		CloseLocalClient(clientId, systemId, clientAddr)
	}

	return
}

// 连接统计
func ClientStatistics(platform, account string) ([]string, int, error) {
	return Manager.PlatformAccountCount(platform, account)
}

// 添加客户端到分组
func AddClient2Group(systemId string, groupName string, clientId string, userId string, extend string) {
	//如果是集群则用redis共享数据
	if util.IsCluster() {
		//	//判断key是否存在
		//	addr, _, _, isLocal, err := util.GetAddrInfoAndIsLocal(clientId)
		//	if err != nil {
		//		log.Errorf("%s", err)
		//		return
		//	}
		//
		//	if isLocal {
		//		if client, err := Manager.GetByClientId(clientId); err == nil {
		//			//添加到本地
		//			Manager.AddClient2LocalGroup(groupName, client, userId, extend)
		//		} else {
		//			log.Error(err)
		//		}
		//	} else {
		//		//发送到指定的机器
		//		SendRpcBindGroup(addr, systemId, groupName, clientId, userId, extend)
		//	}
	} else {
		//if client, err := Manager.GetByClientId(clientId); err == nil {
		//	//如果是单机，就直接添加到本地group了
		//	Manager.AddClient2LocalGroup(groupName, client, userId, extend)
		//}
	}
}

// 发送信息到指定分组
func SendMessage2Group(systemId, sendUserId, groupName string, code int, msg string, data *string) (messageId string) {
	messageId = util.GenUUID()
	if util.IsCluster() {
		//发送分组消息给指定广播
		go SendGroupBroadcast(systemId, messageId, sendUserId, groupName, code, msg, data)
	} else {
		//如果是单机服务，则只发送到本机
		// TODO	Manager.SendMessage2LocalGroup(systemId, messageId, sendUserId, groupName, code, msg, data)
	}
	return
}

// 发送信息到指定系统
func SendMessage2System(systemId, sendUserId string, code int, msg string, data string) {
	messageId := util.GenUUID()
	if util.IsCluster() {
		//发送到系统广播
		SendSystemBroadcast(systemId, messageId, sendUserId, code, msg, &data)
	} else {
		//如果是单机服务，则只发送到本机
		// TODO Manager.SendMessage2LocalSystem(systemId, messageId, sendUserId, code, msg, &data)
	}
}

// 获取分组列表
func GetOnlineList(systemId *string, groupName *string) map[string]interface{} {
	var clientList []string
	if util.IsCluster() {
		//发送到系统广播
		clientList = GetOnlineListBroadcast(systemId, groupName)
	} else {
		//如果是单机服务，则只发送到本机
		retList := Manager.GetGroupClientList(util.GenGroupKey(*systemId, *groupName))
		clientList = append(clientList, retList...)
	}

	return map[string]interface{}{
		"count": len(clientList),
		"list":  clientList,
	}
}

// 通过本服务器发送信息
func SendMessage2LocalClient(messageId, clientId string, sendUserId string, code string, msg string, data *string) {
	log.WithFields(log.Fields{
		"host":      setting.GlobalSetting.LocalHost,
		"port":      setting.CommonSetting.HttpPort,
		"messageId": messageId,
		"clientId":  clientId,
	}).Info("发送到通道")
	ToClientChan <- clientInfo{ClientId: clientId, MessageId: messageId, SendUserId: sendUserId, Code: code, Msg: msg, Data: data}
	return
}

// 发送关闭信号
func CloseLocalClient(clientId, systemId, addr string) {
	if addrToClientMap, err := Manager.GetByClientId(clientId); err == nil && addrToClientMap != nil {
		if conn, ok := addrToClientMap[addr]; ok {
			if conn.SystemId != systemId {
				return
			}
			Manager.DisConnect <- conn
			log.WithFields(log.Fields{
				"host":        setting.GlobalSetting.LocalHost,
				"port":        setting.CommonSetting.HttpPort,
				"client_addr": addr,
				"clientId":    clientId,
			}).Info("主动踢掉客户端")
		}
	}
	return
}

// 监听并发送给客户端信息
func WriteMessage() {
	for {
		clientInfo := <-ToClientChan
		log.WithFields(log.Fields{
			"host":       setting.GlobalSetting.LocalHost,
			"port":       setting.CommonSetting.HttpPort,
			"clientId":   clientInfo.ClientId,
			"messageId":  clientInfo.MessageId,
			"sendUserId": clientInfo.SendUserId,
			"code":       clientInfo.Code,
			"msg":        clientInfo.Msg,
			"data":       clientInfo.GetData(),
		}).Info("发送到本机")
		if addrToConn, err := Manager.GetByClientId(clientInfo.ClientId); err == nil && addrToConn != nil {
			log.WithFields(log.Fields{"addrToConn": addrToConn}).Info("同一client_id对应的所有client")
			for addr, conn := range addrToConn {
				if err := Render(conn.Socket, clientInfo.MessageId, clientInfo.SendUserId, clientInfo.Code, clientInfo.Msg, clientInfo.Data); err != nil {
					Manager.DisConnect <- conn
					log.WithFields(log.Fields{
						"host":      setting.GlobalSetting.LocalHost,
						"port":      setting.CommonSetting.HttpPort,
						"messageId": clientInfo.MessageId,
						"clientId":  clientInfo.ClientId,
						"addr":      addr,
						"msg":       clientInfo.Msg,
					}).Error("客户端异常离线：" + err.Error())
				} else {
					log.WithFields(log.Fields{
						"messageId": clientInfo.MessageId,
						"host":      setting.GlobalSetting.LocalHost,
						"port":      setting.CommonSetting.HttpPort,
						"addr":      addr,
						"clientId":  clientInfo.ClientId}).Info("发送成功")
				}
			}
		} else {
			log.WithFields(log.Fields{"err": err,
				"messageId": clientInfo.MessageId,
				"host":      setting.GlobalSetting.LocalHost,
				"port":      setting.CommonSetting.HttpPort,
				"clientId":  clientInfo.ClientId}).Info("发送失败")
		}
	}
}

func Render(conn *websocket.Conn, messageId string, sendUserId string, code string, message string, data interface{}) error {
	return conn.WriteJSON(RetData{
		Code:       code,
		MessageId:  messageId,
		SendUserId: sendUserId,
		Msg:        message,
		Data:       data,
	})
}

// 启动定时器进行心跳检测
func PingTimer() {
	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(log.Fields{"debug": string(debug.Stack())}).Error("PingTimer失败")
			}
		}()
		for {
			<-ticker.C
			//发送心跳
			if Manager == nil {
				logrus.Info("Manager为空，找不到客户端连接")
				continue
			}
			for clientId, addrToClientMap := range Manager.AllClient() {
				for addr, conn := range addrToClientMap {
					if conn == nil {
						logrus.WithFields(log.Fields{"client_id": clientId, "addr": addr}).Info("客户端已经被下线删除")
						continue
					}
					if err := conn.Socket.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
						Manager.DisConnect <- conn
						log.Infof("发送心跳失败: %s, 关闭链接,总连接数：%d", clientId, Manager.Count())
					}
				}
			}
		}

	}()
}
