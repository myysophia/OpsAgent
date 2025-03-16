package middleware

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"io"
	"time"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

// RequestLogger 请求日志中间件
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 请求开始时间
		startTime := time.Now()

		// 读取请求体
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = c.GetRawData()
			// 将请求体放回，以便后续中间件使用
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 获取 logger
		logger := utils.GetLogger()

		// 记录请求信息
		logger.Debug("收到请求",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("body", string(bodyBytes)),
		)

		// 处理请求
		c.Next()

		// 请求结束时间
		duration := time.Since(startTime)

		logger.Debug("请求处理完成",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
		)
	}
}

// Logger 注入 logger 到 Gin 上下文
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取全局 logger
		logger := utils.GetLogger()

		// 注入 logger 到上下文
		c.Set("logger", logger)

		// 记录请求信息
		logger.Debug("收到请求",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("remote_addr", c.ClientIP()),
		)

		c.Next()
	}
}
