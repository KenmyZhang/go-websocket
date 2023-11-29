package api

import (
	"encoding/json"
	"io"
	"net/http"

	zhongwen "github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
	zh2 "gopkg.in/go-playground/validator.v9/translations/zh"
)

type RetData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ConnSuc struct {
	Type string `json:"Type"`
	Data string `json:"Data"`
}

func ConnRender(conn *websocket.Conn, data interface{}) (err error) {
	err = conn.WriteJSON(ConnSuc{
		Type: "Link",
		Data: "连接成功",
	})

	return
}

func Render(w http.ResponseWriter, code int, msg string, data interface{}) (str string) {
	var retData RetData

	retData.Code = code
	retData.Msg = msg
	retData.Data = data

	retJson, _ := json.Marshal(retData)
	str = string(retJson)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = io.WriteString(w, str)
	return
}

func Validate(inputData interface{}) error {

	validate := validator.New()
	zh := zhongwen.New()
	uni := ut.New(zh, zh)
	trans, _ := uni.GetTranslator("zh")

	_ = zh2.RegisterDefaultTranslations(validate, trans)

	err := validate.Struct(inputData)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			return errors.New(err.Translate(trans))
		}
	}

	return nil
}
