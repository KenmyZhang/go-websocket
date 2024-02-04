package servers

import (
	"encoding/json"
	"errors"
	"runtime/debug"
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
	ClientIdMap        map[string]map[string]*Client // 全部的连接
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
		PlatformAccountMap: make(map[string][]string),
		ClientIdMap:        make(map[string]map[string]*Client),
		Connect:            make(chan *Client, 10000),
		DisConnect:         make(chan *Client, 10000),
		Groups:             make(map[string][]string, 100),
		SystemClients:      make(map[string][]string, 100),
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
		"addr":     client.Addr,
		"counts":   Manager.Count(),
	}).Info("客户端已连接")
}

// 断开连接时间
func (manager *ClientManager) EventDisconnect(client *Client) {
	//关闭连接
	_ = client.Socket.Close()
	log.WithFields(log.Fields{"client_id": client.ClientId}).Info("关闭连接")
	manager.DelClient(client)

	mJson, _ := json.Marshal(map[string]string{
		"clientId": client.ClientId,
		"userId":   client.UserId,
		"addr":     client.Addr,
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
		"addr":     client.Addr,
		"counts":   Manager.Count(),
		"seconds":  uint64(time.Now().Unix()) - client.ConnectTime,
	}).Info("客户端已断开")

	//标记销毁
	client.IsDeleted = true
	client = nil
}

// 添加客户端
func (manager *ClientManager) AddClient(client *Client) {
	defer func() {
		if r := recover(); r != nil {
			log.WithFields(log.Fields{"debug": string(debug.Stack())}).Error("AddClient失败")
		}
	}()
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()

	var addrToClientMap map[string]*Client
	data, _ := json.Marshal(&manager.ClientIdMap)
	log.WithFields(log.Fields{"data": string(data)}).Info("添加client前")
	if strings.Contains(client.ClientId, "APP_") {
		delete(manager.ClientIdMap, client.ClientId)
		addrToClientMap = make(map[string]*Client)
		manager.ClientIdMap[client.ClientId] = addrToClientMap
		addrToClientMap[client.Addr] = client
	} else {
		addrToClientMap, ok := manager.ClientIdMap[client.ClientId]
		if ok {
			addrToClientMap[client.Addr] = client
			return
		}

		addrToClientMap = make(map[string]*Client)
		manager.ClientIdMap[client.ClientId] = addrToClientMap
		addrToClientMap[client.Addr] = client
	}
	data, _ = json.Marshal(&manager.ClientIdMap)
	log.WithFields(log.Fields{"data": string(data)}).Info("添加client后")

	for addr := range addrToClientMap {
		log.WithFields(log.Fields{"client_id": client.ClientId, "已有的地址": addr, "待添加的地址": client.Addr}).Info("client_id原有的conn")
	}
	if len(addrToClientMap) < 2 {
		log.WithFields(log.Fields{"client_id": client.ClientId, "addr": client.Addr}).Info("添加客户端")

		tmpClientInfos := strings.Split(client.ClientId, "_")
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
		platformAccounts := len(manager.PlatformAccountMap[key])
		log.WithFields(log.Fields{"在线账号": platformAccounts,
			"client_id": client.ClientId,
			"key":       key}).Info("添加客户端前")
		manager.PlatformAccountMap[key] = append(manager.PlatformAccountMap[key], client.ClientId)

		platformAccounts = len(manager.PlatformAccountMap[key])
		log.WithFields(log.Fields{"在线账号": platformAccounts,
			"client_id": client.ClientId,
			"key":       key}).Info("添加客户端后")
	} else {
		log.WithFields(log.Fields{"client_id": client.ClientId}).Info("重复添加客户端")
	}
}

// 获取所有的客户端
func (manager *ClientManager) AllClient() map[string]map[string]*Client {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()
	result := map[string]map[string]*Client{}
	for key, value := range manager.ClientIdMap {
		addrToClientMap, ok := result[key]
		if !ok {
			addrToClientMap = make(map[string]*Client)
			result[key] = value
		}
		for k, v := range value {
			addrToClientMap[k] = v
		}
	}
	return result
}

// 客户端数量
func (manager *ClientManager) Count() int {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()
	return len(manager.ClientIdMap)
}

// 删除客户端
func (manager *ClientManager) DelClient(client *Client) {
	defer func() {
		if r := recover(); r != nil {
			log.WithFields(log.Fields{"debug": string(debug.Stack())}).Error("DelClient失败")
		}
	}()

	if client == nil {
		logrus.Info("client已被删除")
		return
	}

	if client.ClientId == "" {
		logrus.WithFields(log.Fields{"client_id": client.ClientId}).Error("clientId无效")
		return
	}
	logrus.WithFields(logrus.Fields{"ClientId": client.ClientId}).Info("待删除的客户端信息")
	manager.delClientIdMap(client.ClientId, client.Addr)
	logrus.WithFields(logrus.Fields{"ClientId": client.ClientId}).Info("删除clientmap")

	//删除所在的分组
	if len(client.GroupList) > 0 {
		for _, groupName := range client.GroupList {
			manager.delGroupClient(util.GenGroupKey(client.SystemId, groupName), client.ClientId)
		}
	}

	// 删除系统里的客户端
	manager.delSystemClient(client)
	logrus.WithFields(logrus.Fields{"ClientId": client.ClientId}).Info("删除clientmap")

}

// 删除clientIdMap
func (manager *ClientManager) delClientIdMap(clientId, addr string) {
	manager.ClientIdMapLock.Lock()
	defer manager.ClientIdMapLock.Unlock()
	addrToClientMap, ok := manager.ClientIdMap[clientId]
	if !ok {
		log.WithFields(log.Fields{"client_id": clientId}).Info("已经从clientIdMap删除")
		return
	}
	if _, ok := addrToClientMap[addr]; !ok {
		log.WithFields(log.Fields{"client_id": clientId, "addr": addr}).Info("已经从clientIdMap删除")
		return
	}
	delete(addrToClientMap, clientId)
	if len(addrToClientMap) == 0 {
		delete(manager.ClientIdMap, clientId)
	} else {
		log.WithFields(log.Fields{"client_id": clientId, "addr": addr}).Info("删除client from 成功")
		return
	}
	log.WithFields(log.Fields{"client_id": clientId, "addr": addr}).Info("删除clientIdMap")

	tmpClientInfos := strings.Split(clientId, "_")
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
	platformAccount, ok := manager.PlatformAccountMap[key]
	if !ok {
		log.WithFields(log.Fields{"key": key}).Info("已经从PlatformAccountMap删除")
		return
	}
	count := len(platformAccount)
	log.WithFields(log.Fields{"账号在线数": count,
		"client_id": clientId,
		"key":       key}).Info("删除client前")
	manager.PlatformAccountMap[key] = removeElement(manager.PlatformAccountMap[key], clientId)
	count = len(manager.PlatformAccountMap[key])
	log.WithFields(log.Fields{"账号在线数": count,
		"client_id": clientId,
		"key":       key}).Info("删除client后")

}

// 账号在线数量
func (manager *ClientManager) PlatformAccountCount(platform, account string) ([]string, int, error) {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()
	key := platform + "_" + account
	count := len(manager.PlatformAccountMap[key])
	log.WithFields(log.Fields{"数量": count, "key": key}).Info("查询账户在线数量")
	var result []string
	result = append(result, manager.PlatformAccountMap[key]...)
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
func (manager *ClientManager) GetByClientId(clientId string) (map[string]*Client, error) {
	manager.ClientIdMapLock.RLock()
	defer manager.ClientIdMapLock.RUnlock()
	count := len(manager.ClientIdMap)
	log.WithFields(log.Fields{"数量": count}).Info("连接数的数量")
	addrToClient := map[string]*Client{}
	if tmpAddrToClient, ok := manager.ClientIdMap[clientId]; !ok {
		return nil, errors.New("客户端不存在")
	} else {
		for addr, client := range tmpAddrToClient {
			addrToClient[addr] = client
		}
		return addrToClient, nil
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
			if (index + 1) < len(manager.Groups) {
				manager.Groups[groupKey] = append(manager.Groups[groupKey][:index], manager.Groups[groupKey][index+1:]...)
			} else {
				manager.Groups[groupKey] = manager.Groups[groupKey][:index]
			}
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
func (manager *ClientManager) AddClient2SystemClient(systemId, addr string, client *Client) {
	manager.AddClient(client)
	manager.SystemClientsLock.Lock()
	defer manager.SystemClientsLock.Unlock()
	manager.SystemClients[systemId] = append(manager.SystemClients[systemId], client.ClientId)
	clientCount := len(manager.SystemClients)
	logrus.WithFields(log.Fields{"count": clientCount, "systemId": systemId, "addr": addr}).Info("客户端数量")
}

// 删除系统里的客户端
func (manager *ClientManager) delSystemClient(client *Client) {
	manager.SystemClientsLock.Lock()
	defer manager.SystemClientsLock.Unlock()

	for index, clientId := range manager.SystemClients[client.SystemId] {
		if clientId == client.ClientId {
			if (index + 1) < len(manager.Groups) {
				manager.SystemClients[client.SystemId] = append(manager.SystemClients[client.SystemId][:index], manager.SystemClients[client.SystemId][index+1:]...)
			} else {
				manager.SystemClients[client.SystemId] = manager.SystemClients[client.SystemId][:index]
			}
		}
	}
}

// 获取指定系统的客户端列表
func (manager *ClientManager) GetSystemClientList(systemId string) []string {
	manager.SystemClientsLock.RLock()
	defer manager.SystemClientsLock.RUnlock()
	return manager.SystemClients[systemId]
}
