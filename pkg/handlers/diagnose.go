package handlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

// DiagnoseRequest 诊断请求结构
type DiagnoseRequest struct {
	Name      string `json:"name" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
}

// Diagnose 处理诊断请求
func Diagnose(c *gin.Context) {
	var req DiagnoseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model := c.DefaultQuery("model", "gpt-4o")
	cluster := c.DefaultQuery("cluster", "default")

	// TODO: 实现实际的诊断逻辑
	result := fmt.Sprintf("Diagnosing pod %s in namespace %s using model %s on cluster %s",
		req.Name, req.Namespace, model, cluster)

	c.JSON(http.StatusOK, gin.H{
		"message": result,
		"status":  "success",
	})
} 