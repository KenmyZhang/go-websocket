package clientstatistics

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/woodylan/go-websocket/api"
	"github.com/woodylan/go-websocket/define/retcode"
	"github.com/woodylan/go-websocket/servers"
)

type Controller struct {
}

type inputData struct {
	Platform string `json:"platform"`
	Account  string `json:"account"`
}

func (c *Controller) Run(w http.ResponseWriter, r *http.Request) {
	var inputData inputData
	if err := json.NewDecoder(r.Body).Decode(&inputData); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := api.Validate(inputData)
	if err != nil {
		api.Render(w, retcode.FAIL, err.Error(), []string{})
		return
	}

	clientIds, count, err := servers.ClientStatistics(inputData.Platform, inputData.Account)
	logrus.WithFields(logrus.Fields{"platform": inputData.Platform, "account": inputData.Account, "count": count, "clientIds": clientIds}).Info("账号在线数量")
	api.Render(w, retcode.SUCCESS, "success", map[string]interface{}{
		"clientIds": clientIds,
		"count":     count,
	})
	return
}
