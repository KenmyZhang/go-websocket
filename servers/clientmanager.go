package servers

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/define/retcode"
	"github.com/woodylan/go-websocket/pkg/setting"
	"github.com/woodylan/go-websocket/tools/util"
)

// 连接管理
type ClientManager struct {
	ClientIdMap        map[string]*Client // 全部的连接
	PlatformAccountMap map[string][]string
	ClientIdMapLock    sync.RWMutex // 读写锁

	Connect    chan *Client // 连接处理
	DisConnect chan *Client // 断开连接处理

	GroupLock sync.RWMutex
	Groups    map[string][]string

	SystemClientsLock sync.RWMutex
	SystemClients     map[string][]string
}

func NewClientManager() (clientManager *ClientManager) {
	clientManager = &ClientManager{
		ClientIdMap:   make(map[string]*Client),
		Connect:       make(chan *Client, 10000),
		DisConnect:    make(chan *Client, 10000),
		Groups:        make(map[string][]string, 100),
		SystemClients: make(map[string][]string, 100),
	}

	return
}

// 管道处理程序
func (manager *ClientManager) Start() {
	for {
		select {
		case client := <-manager.Connect:
			// 建立连接事件
			manager.EventConnect(client)
		case conn := <-manager.DisConnect:
			// 断开连接事件
			manager.EventDisconnect(conn)
		}
	}
}

// 建立连接事件
func (manager *ClientManager) EventConnect(client *Client) {
	manager.AddClient(client)

	log.WithFields(log.Fields{
		"host":     setting.GlobalSetting.LocalHost,
		"port":     setting.CommonSetting.HttpPort,
		"clientId": client.ClientId,
		"counts":   Manager.Count(),
	}).Info("客户端已连接")
}

// 断开连接时间
func (manager *ClientManager) EventDisconnect(client *Client) {
	//关闭连接
	_ = client.Socket.Close()
	manager.DelClient(client)

	mJson, _ := json.Marshal(map[string]string{
		"clientId": client.ClientId,
		"userId":   client.UserId,
		"extend":   client.Extend,
	})
	data := string(mJson)
	sendUserId := ""

	//发送下线通知
	if len(client.GroupList) > 0 {
		for _, groupName := range client.GroupList {
			SendMessage2Group(client.SystemId, sendUserId, groupName, retcode.OFFLINE_MESSAGE_CODE, "客户端下线", &data)
		}
	}

	log.WithFields(log.Fields{
		"host":     setting.GlobalSetting.LocalHost,
		"port":     setting.CommonSetting.HttpPort,
		"clientId": client.ClientId,
		"counts":   Manager.Count(),
		"seconds":  uint64(time.Now().Unix()) - client.ConnectTime,
	}).Info("客户端已断开")

	//标记销毁
	client.IsDeleted = true
	client = nil
}

// 添加客户端
func (manager *ClientManager) AddClient(client *Client) {
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()

	if _, ok := manager.ClientIdMap[client.ClientId]; !ok {
		log.WithFields(log.Fields{"client_id": client.ClientId}).Info("重复添加客户端")
		manager.ClientIdMap[client.ClientId] = client

		tmpClientInfos := strings.Split(client.ClientId, "-")
		if len(tmpClientInfos) < 2 {
			log.Error("无效clientId")
			return
		}
		var platform, account string
		if len(tmpClientInfos) == 3 {
			platform = tmpClientInfos[0]
			account = tmpClientInfos[2]
		}
		if len(tmpClientInfos) == 2 {
			platform = tmpClientInfos[0]
			account = tmpClientInfos[1]
		}

		key := platform + "_" + account

		log.WithFields(log.Fields{"在线账号": manager.PlatformAccountMap[key],
			"client_id": client.ClientId,
			"key":       key}).Info("添加客户端前")
		manager.PlatformAccountMap[key] = append(manager.PlatformAccountMap[key], client.ClientId)
		log.WithFields(log.Fields{"在线账号": manager.PlatformAccountMap[key],
			"client_id": client.ClientId,
			"key":       key}).Info("添加客户端后")
	} else {
		log.WithFields(log.Fields{"client_id": client.ClientId}).Info("重复添加客户端")
	}
}

// 获取所有的客户端
func (manager *ClientManager) AllClient() map[string]*Client {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()

	return manager.ClientIdMap
}

// 客户端数量
func (manager *ClientManager) Count() int {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()
	return len(manager.ClientIdMap)
}

// 删除客户端
func (manager *ClientManager) DelClient(client *Client) {
	if client == nil {
		logrus.Info("client已被删除")
		return
	}

	if client.ClientId == "" {
		logrus.Error("clientId无效")
		return
	}
	logrus.WithFields(logrus.Fields{"client": client}).Info("待删除的客户端信息")
	manager.delClientIdMap(client.ClientId)
	logrus.WithFields(logrus.Fields{"client": client}).Info("删除clientmap")

	//删除所在的分组
	if len(client.GroupList) > 0 {
		for _, groupName := range client.GroupList {
			manager.delGroupClient(util.GenGroupKey(client.SystemId, groupName), client.ClientId)
		}
	}

	// 删除系统里的客户端
	manager.delSystemClient(client)
	logrus.WithFields(logrus.Fields{"client": client}).Info("删除clientmap")

}

// 删除clientIdMap
func (manager *ClientManager) delClientIdMap(clientId string) {
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()
	delete(manager.ClientIdMap, clientId)
	log.WithFields(log.Fields{"client_id": clientId}).Info("删除clientIdMap")

	tmpClientInfos := strings.Split(clientId, "-")
	if len(tmpClientInfos) < 2 {
		log.Error("无效clientId")
		return
	}
	var platform, account string
	if len(tmpClientInfos) == 3 {
		platform = tmpClientInfos[0]
		account = tmpClientInfos[2]
	}
	if len(tmpClientInfos) == 2 {
		platform = tmpClientInfos[0]
		account = tmpClientInfos[1]
	}
	key := platform + "_" + account
	log.WithFields(log.Fields{"账号在线数": manager.PlatformAccountMap[key],
		"client_id": clientId,
		"key":       key}).Info("删除client前")
	removeElement(manager.PlatformAccountMap[key], clientId)
	log.WithFields(log.Fields{"账号在线数": manager.PlatformAccountMap[key],
		"client_id": clientId,
		"key":       key}).Info("删除client后")

}

// 账号在线数量
func (manager *ClientManager) PlatformAccountCount(platform, account string) ([]string, int, error) {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()
	key := platform + "_" + account
	log.WithFields(log.Fields{"数量": len(manager.PlatformAccountMap[key]), "key": key}).Info("查询账户在线数量")
	var result []string
	copy(result, manager.PlatformAccountMap[key])
	return result, len(result), nil
}

func removeElement(nums []string, val string) []string {
	var result []string

	for _, num := range nums {
		if num != val {
			result = append(result, num)
		}
	}

	return result
}

// 通过clientId获取
func (manager *ClientManager) GetByClientId(clientId string) (*Client, error) {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()
	log.WithFields(log.Fields{"数量": len(manager.ClientIdMap)}).Info("连接数的数量")
	if client, ok := manager.ClientIdMap[clientId]; !ok {
		return nil, errors.New("客户端不存在")
	} else {
		return client, nil
	}
}

// 发送到本机分组
func (manager *ClientManager) SendMessage2LocalGroup(systemId, messageId, sendUserId, groupName string, code string, msg string, data *string) {
	if len(groupName) > 0 {
		clientIds := manager.GetGroupClientList(util.GenGroupKey(systemId, groupName))
		if len(clientIds) > 0 {
			for _, clientId := range clientIds {
				if _, err := Manager.GetByClientId(clientId); err == nil {
					//添加到本地
					SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
				} else {
					//删除分组
					manager.delGroupClient(util.GenGroupKey(systemId, groupName), clientId)
				}
			}
		}
	}
}

// 发送给指定业务系统
func (manager *ClientManager) SendMessage2LocalSystem(systemId, messageId string, sendUserId string, code string, msg string, data *string) {
	if len(systemId) > 0 {
		clientIds := Manager.GetSystemClientList(systemId)
		if len(clientIds) > 0 {
			for _, clientId := range clientIds {
				SendMessage2LocalClient(messageId, clientId, sendUserId, code, msg, data)
			}
		}
	}
}

// 添加到本地分组
func (manager *ClientManager) AddClient2LocalGroup(groupName string, client *Client, userId string, extend string) {
	//标记当前客户端的userId
	client.UserId = userId
	client.Extend = extend

	//判断之前是否有添加过
	for _, groupValue := range client.GroupList {
		if groupValue == groupName {
			return
		}
	}

	// 为属性添加分组信息
	groupKey := util.GenGroupKey(client.SystemId, groupName)

	manager.addClient2Group(groupKey, client)

	client.GroupList = append(client.GroupList, groupName)

	mJson, _ := json.Marshal(map[string]string{
		"clientId": client.ClientId,
		"userId":   client.UserId,
		"extend":   client.Extend,
	})
	data := string(mJson)
	sendUserId := ""

	//发送系统通知
	SendMessage2Group(client.SystemId, sendUserId, groupName, retcode.ONLINE_MESSAGE_CODE, "客户端上线", &data)
}

// 添加到本地分组
func (manager *ClientManager) addClient2Group(groupKey string, client *Client) {
	manager.GroupLock.Lock()
	defer manager.GroupLock.Unlock()
	manager.Groups[groupKey] = append(manager.Groups[groupKey], client.ClientId)
}

// 删除分组里的客户端
func (manager *ClientManager) delGroupClient(groupKey string, clientId string) {
	manager.GroupLock.Lock()
	defer manager.GroupLock.Unlock()

	for index, groupClientId := range manager.Groups[groupKey] {
		if groupClientId == clientId {
			manager.Groups[groupKey] = append(manager.Groups[groupKey][:index], manager.Groups[groupKey][index+1:]...)
		}
	}
}

// 获取本地分组的成员
func (manager *ClientManager) GetGroupClientList(groupKey string) []string {
	manager.GroupLock.RLock()
	defer manager.GroupLock.RUnlock()
	return manager.Groups[groupKey]
}

// 添加到系统客户端列表
func (manager *ClientManager) AddClient2SystemClient(systemId string, client *Client) {
	manager.AddClient(client)
	manager.SystemClientsLock.Lock()
	defer manager.SystemClientsLock.Unlock()
	manager.SystemClients[systemId] = append(manager.SystemClients[systemId], client.ClientId)
	logrus.WithFields(log.Fields{"count": len(manager.SystemClients)}).Info("客户端数量")
}

// 删除系统里的客户端
func (manager *ClientManager) delSystemClient(client *Client) {
	manager.SystemClientsLock.Lock()
	defer manager.SystemClientsLock.Unlock()

	for index, clientId := range manager.SystemClients[client.SystemId] {
		if clientId == client.ClientId {
			manager.SystemClients[client.SystemId] = append(manager.SystemClients[client.SystemId][:index], manager.SystemClients[client.SystemId][index+1:]...)
		}
	}
}

// 获取指定系统的客户端列表
func (manager *ClientManager) GetSystemClientList(systemId string) []string {
	manager.SystemClientsLock.RLock()
	defer manager.SystemClientsLock.RUnlock()
	return manager.SystemClients[systemId]
}
