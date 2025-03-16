package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"time"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

// PerfStats 性能统计中间件
func PerfStats() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 logger
		logger := utils.GetLogger()

		// 开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(start)

		// 记录性能统计信息
		logger.Debug("请求性能统计",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Duration("duration", duration),
			zap.Int("status", c.Writer.Status()),
		)

		// 记录到性能统计工具
		perfStats := utils.GetPerfStats()
		perfStats.RecordMetric(c.Request.URL.Path, duration)
	}
}
