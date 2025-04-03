package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/myysophia/OpsAgent/pkg/kubectl"
	"github.com/myysophia/OpsAgent/pkg/rag"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"go.uber.org/zap"
)

type RAGRequest struct {
	Query      string                 `json:"query" binding:"required"`
	Parameters map[string]interface{} `json:"parameters"`
}

type RAGResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type ContextSwitchData struct {
	ContextSwitch struct {
		Command string `json:"command"`
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"context_switch"`
	QueryResult interface{} `json:"query_result"`
}

// SwitchContext 处理RAG上下文切换请求
func SwitchContext(c *gin.Context) {
	// 从环境变量获取配置
	ragAPIKey := os.Getenv("RAG_API_KEY")
	ragAppID := os.Getenv("RAG_APP_ID")

	if ragAPIKey == "" || ragAppID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "RAG_API_KEY and RAG_APP_ID environment variables are required",
		})
		return
	}

	// 读取原始请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.Error("Failed to read request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Failed to read request body",
		})
		return
	}
	// 将请求体放回，以便后续绑定
	c.Request.Body = io.NopCloser(io.NopCloser(bytes.NewBuffer(body)))

	// 打印原始请求体
	utils.Debug("Received request body", zap.String("body", string(body)))

	var req RAGRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error("Failed to bind JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	// 打印解析后的请求
	utils.Debug("Parsed request",
		zap.String("query", req.Query),
		zap.Any("parameters", req.Parameters),
	)

	ragService := rag.NewRAGService(ragAPIKey, ragAppID)
	kubectlExecutor := kubectl.NewExecutor()

	// 调用RAG服务获取推荐命令
	recommendCmd, err := ragService.GetRecommendCommand(req.Query)
	if err != nil {
		utils.Error("Failed to get recommend command", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("Failed to get recommend command: %v", err),
		})
		return
	}

	// 执行kubectl命令
	output, err := kubectlExecutor.ExecuteCommand(recommendCmd)
	if err != nil {
		utils.Error("Failed to execute command", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("Failed to execute command: %v", err),
		})
		return
	}

	// 构建响应
	response := RAGResponse{
		Code:    200,
		Message: "success",
		Data: ContextSwitchData{
			ContextSwitch: struct {
				Command string `json:"command"`
				Status  string `json:"status"`
				Message string `json:"message"`
			}{
				Command: recommendCmd,
				Status:  "success",
				Message: output,
			},
			QueryResult: nil, // 这里可以添加用户查询的执行结果
		},
	}

	c.JSON(http.StatusOK, response)
}
