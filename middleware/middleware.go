package middleware

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/zhany/ops-go/config"
	"github.com/zhany/ops-go/models"
	"io"
	"strconv"
	"time"
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
		go func() {
			sysLog := models.SysLog{}
			sysLog.IpAddr = c.ClientIP()
			sysLog.RequestUri = url.RequestURI()
			sysLog.Method = c.Request.Method
			sysLog.Params = string(reqBody)
			sysLog.Resp = writer.body.String()
			sysLog.StatusCode = strconv.Itoa(c.Writer.Status())
			sysLog.CostTimeMs = time.Since(start).Milliseconds()
			config.DB.Create(&sysLog)
		}()
	}
}
