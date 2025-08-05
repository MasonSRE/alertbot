package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func Logging(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 读取请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建响应体写入器
		w := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = w

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logEntry := logger.WithFields(logrus.Fields{
			"status_code": statusCode,
			"latency":     latency,
			"client_ip":   clientIP,
			"method":      method,
			"path":        path,
			"user_agent":  c.Request.UserAgent(),
		})

		// 记录请求体（排除敏感信息）
		if len(requestBody) > 0 && len(requestBody) < 1024 {
			logEntry = logEntry.WithField("request_body", string(requestBody))
		}

		// 记录响应体（仅在出错时）
		if statusCode >= 400 && w.body.Len() < 1024 {
			logEntry = logEntry.WithField("response_body", w.body.String())
		}

		if statusCode >= 500 {
			logEntry.Error("Server error")
		} else if statusCode >= 400 {
			logEntry.Warn("Client error")
		} else {
			logEntry.Info("Request processed")
		}
	}
}