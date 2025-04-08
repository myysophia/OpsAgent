package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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

// KubeContextKnowledge 表示 Kubernetes 上下文知识
type KubeContextKnowledge struct {
	ContextName      string `json:"context_name"`
	RegionIdentifier string `json:"region_identifier"`
}

// ContextSwitchData 表示上下文切换数据
type ContextSwitchData struct {
	KubecontextKnowledge []KubeContextKnowledge `json:"kubecontext_knowledge"`
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

	// 调用RAG服务获取上下文信息
	contextInfo, err := ragService.GetContextInfo(req.Query)
	if err != nil {
		utils.Error("Failed to get context info", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("Failed to get context info: %v", err),
		})
		return
	}

	// 解析上下文信息
	var contextData ContextSwitchData
	if err := json.Unmarshal([]byte(contextInfo), &contextData); err != nil {
		utils.Error("Failed to parse context info", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("Failed to parse context info: %v", err),
		})
		return
	}
	// 构建响应
	response := RAGResponse{
		Code:    200,
		Message: "success",
		Data:    contextInfo,
	}

	c.JSON(http.StatusOK, response)
}
