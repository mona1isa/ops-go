package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/controllers/system/api"
	"github.com/zhany/ops-go/models"
	"io"
)

// 自定义 ResponseWriter
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b) // 捕获响应体
	return w.ResponseWriter.Write(b)
}

func LogMiddleware() gin.HandlerFunc {

	return func(c *gin.Context) {

		userId, userIdExist := c.Get("userId")
		deptId, deptIdExists := c.Get("deptId")
		username, userNameExists := c.Get("userName")

		// 记录请求体（需重置读取位置）
		reqBody, _ := c.GetRawData()
		c.Request.Body = io.NopCloser(bytes.NewReader(reqBody)) // 放回请求体

		// 包装 ResponseWriter 以捕获响应体
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = writer

		url := c.Request.URL
		start := time.Now()
		c.Next()
		costTime := time.Since(start).Milliseconds()
		go func() {
			sysLog := models.SysLog{}
			sysLog.IpAddr = c.ClientIP()
			sysLog.RequestUri = url.RequestURI()
			sysLog.Method = c.Request.Method
			sysLog.Params = string(reqBody)
			if strings.Contains(sysLog.RequestUri, "login") {
				r := api.LoginRequest{}
				err := json.Unmarshal(reqBody, &r)
				if err != nil {
					return
				}
				r.Password = "******"
				marshal, err := json.Marshal(&r)
				sysLog.Params = string(marshal)
			}

			respBody := writer.body.String()
			sysLog.Resp = respBody
			sysLog.StatusCode = strconv.Itoa(c.Writer.Status())
			sysLog.CostTimeMs = costTime
			sysLog.UserAgent = truncate(c.Request.UserAgent(), 512)
			sysLog.OperateAt = start.Format("2006-01-02 15:04:05")
			module, operType, sensitive := models.ClassifyOper(c.Request.Method, url.Path)
			sysLog.OperModule = module
			sysLog.OperType = operType
			sysLog.Sensitive = sensitive
			sysLog.BizCode, sysLog.BizMsg = parseBizResult(respBody)

			if userIdExist {
				sysLog.CreateBy = userId.(string)
			}

			if deptIdExists {
				id, _ := strconv.Atoi(deptId.(string))
				sysLog.DeptId = id
			}

			if userNameExists {
				sysLog.CreateUser = username.(string)
			}

			models.DB.Create(&sysLog)
		}()
	}
}

// parseBizResult 从统一响应体中提取业务码与提示，HTTP 200 但业务失败的操作同样留痕
func parseBizResult(body string) (code string, msg string) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
		return "", ""
	}
	var envelope struct {
		Code any `json:"code"`
		Msg  any `json:"msg"`
	}
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return "", ""
	}
	if envelope.Code != nil {
		code = truncate(fmt.Sprint(envelope.Code), 16)
	}
	if envelope.Msg != nil {
		msg = truncate(fmt.Sprint(envelope.Msg), 255)
	}
	return code, msg
}

func truncate(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
