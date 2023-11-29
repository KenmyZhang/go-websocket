package routers

import (
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/woodylan/go-websocket/api"
	"github.com/woodylan/go-websocket/define"
	"github.com/woodylan/go-websocket/define/retcode"
	"github.com/woodylan/go-websocket/pkg/etcd"
	"github.com/woodylan/go-websocket/servers"
	"github.com/woodylan/go-websocket/tools/util"
)

func AccessTokenMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		//检查header是否设置SystemId
		systemId := r.Header.Get("SystemId")
		if len(systemId) == 0 {
			api.Render(w, retcode.FAIL, "系统ID不能为空", []string{})
			return
		}

		//判断是否被注册
		if util.IsCluster() {
			resp, err := etcd.Get(define.ETCD_PREFIX_ACCOUNT_INFO + systemId)
			if err != nil {
				api.Render(w, retcode.FAIL, "etcd服务器错误", []string{})
				return
			}

			if resp.Count == 0 {
				api.Render(w, retcode.FAIL, "系统ID无效", []string{})
				return
			}
		} else {
			if _, ok := servers.SystemMap.Load(systemId); !ok {
				api.Render(w, retcode.FAIL, "系统ID无效", []string{})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func AccessLogMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceId := r.Header.Get("Trace-Id")
		if traceId == "" {
			traceId = uuid.New().String()
		}
		httpRequest, _ := httputil.DumpRequest(r, true)
		httpRequests := strings.Replace(string(httpRequest), "\r\n", " ", -1)
		httpRequests = strings.Replace(httpRequests, "\n", " ", -1)
		logger := logrus.WithFields(logrus.Fields{"trace_id": traceId})
		logger.WithFields(logrus.Fields{"httpRequests": httpRequests}).Info("请求报文")
		// 开始时间
		startTime := time.Now()
		next.ServeHTTP(w, r)

		// 结束时间
		endTime := time.Now()

		// 执行时间
		latencyTime := endTime.Sub(startTime)

		// 请求方式
		reqMethod := r.Method

		// 请求路由
		reqUri := r.RequestURI

		// 请求IP
		clientIP := r.RemoteAddr

		//日志格式
		logger.Infof("| %3d  | %15s | %s | %s |",
			latencyTime,
			clientIP,
			reqMethod,
			reqUri,
		)

	})
}
