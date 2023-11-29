package routers

import (
	"net/http"

	"github.com/woodylan/go-websocket/api/bind2group"
	"github.com/woodylan/go-websocket/api/clientstatistics"
	"github.com/woodylan/go-websocket/api/closeclient"
	"github.com/woodylan/go-websocket/api/getonlinelist"
	"github.com/woodylan/go-websocket/api/register"
	"github.com/woodylan/go-websocket/api/send2client"
	"github.com/woodylan/go-websocket/api/send2clients"
	"github.com/woodylan/go-websocket/api/send2group"
	"github.com/woodylan/go-websocket/servers"
)

func Init() {
	registerHandler := &register.Controller{}
	sendToClientHandler := &send2client.Controller{}
	sendToClientsHandler := &send2clients.Controller{}
	sendToGroupHandler := &send2group.Controller{}
	bindToGroupHandler := &bind2group.Controller{}
	getGroupListHandler := &getonlinelist.Controller{}
	closeClientHandler := &closeclient.Controller{}

	clientstatisticsHandler := &clientstatistics.Controller{}

	http.HandleFunc("/api/register", AccessLogMiddleware(registerHandler.Run))
	http.HandleFunc("/api/send_to_client", AccessLogMiddleware(sendToClientHandler.Run))
	http.HandleFunc("/api/send_to_clients", AccessLogMiddleware(sendToClientsHandler.Run))
	http.HandleFunc("/api/send_to_group", AccessLogMiddleware(sendToGroupHandler.Run))
	http.HandleFunc("/api/bind_to_group", AccessLogMiddleware(bindToGroupHandler.Run))
	http.HandleFunc("/api/get_online_list", AccessLogMiddleware(getGroupListHandler.Run))
	http.HandleFunc("/api/close_client", AccessLogMiddleware(closeClientHandler.Run))
	http.HandleFunc("/api/account/client/count", AccessLogMiddleware(clientstatisticsHandler.Run))
	//http.HandleFunc("/api/send_to_client", AccessTokenMiddleware(sendToClientHandler.Run))
	//http.HandleFunc("/api/send_to_clients", AccessTokenMiddleware(sendToClientsHandler.Run))
	//http.HandleFunc("/api/send_to_group", AccessTokenMiddleware(sendToGroupHandler.Run))
	//http.HandleFunc("/api/bind_to_group", AccessTokenMiddleware(bindToGroupHandler.Run))
	//http.HandleFunc("/api/get_online_list", AccessTokenMiddleware(getGroupListHandler.Run))
	//http.HandleFunc("/api/close_client", AccessTokenMiddleware(closeClientHandler.Run))

	servers.StartWebSocket()

	go servers.WriteMessage()
}
