package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

// PerfStats 获取性能统计信息
func PerfStats(c *gin.Context) {
	logger := c.MustGet("logger").(*zap.Logger)
	perfStats := utils.GetPerfStats()

	stats := perfStats.GetStats()
	logger.Debug("获取性能统计信息",
		zap.Any("stats", stats),
	)

	c.JSON(http.StatusOK, gin.H{
		"stats":  stats,
		"status": "success",
	})
}

// ResetPerfStats 重置性能统计信息
func ResetPerfStats(c *gin.Context) {
	logger := c.MustGet("logger").(*zap.Logger)
	perfStats := utils.GetPerfStats()

	perfStats.Reset()
	logger.Info("重置性能统计信息")

	c.JSON(http.StatusOK, gin.H{
		"message": "性能统计信息已重置",
		"status":  "success",
	})
}
