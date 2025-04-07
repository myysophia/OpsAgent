package middleware

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/myysophia/OpsAgent/pkg/audit"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"go.uber.org/zap"
	"time"
)

// AuditMiddleware 审计中间件
func AuditMiddleware(auditLogger *audit.AuditLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果审计日志记录器为空（未启用），直接跳过
		if auditLogger == nil {
			c.Next()
			return
		}

		// 只对特定路径进行审计
		if c.Request.URL.Path != "/api/execute" && c.Request.URL.Path != "/api/diagnose" && c.Request.URL.Path != "/api/analyze" {
			c.Next()
			return
		}

		// 获取或创建会话ID
		var sessionID uuid.UUID
		sessionIDStr, err := c.Cookie("session_id")
		if err != nil || sessionIDStr == "" {
			sessionID = uuid.New()
			c.SetCookie("session_id", sessionID.String(), 86400, "/", "", false, true)
		} else {
			sessionID, err = uuid.Parse(sessionIDStr)
			if err != nil {
				sessionID = uuid.New()
				c.SetCookie("session_id", sessionID.String(), 86400, "/", "", false, true)
			}
		}

		// 创建交互ID
		interactionID := uuid.New()

		// 将ID注入上下文
		c.Set("session_id", sessionID)
		c.Set("interaction_id", interactionID)

		// 获取日志记录器
		logger := utils.GetLogger()
		logger.Debug("开始审计请求",
			zap.String("path", c.Request.URL.Path),
			zap.String("session_id", sessionID.String()),
			zap.String("interaction_id", interactionID.String()),
		)

		// 记录开始时间
		startTime := time.Now()

		// 创建响应体写入器
		blw := &bodyLogWriter{body: []byte{}, ResponseWriter: c.Writer}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算总耗时
		totalDuration := time.Since(startTime)

		// 从上下文中获取其他需要记录的信息
		var (
			question          string
			modelName         string
			provider          string
			baseURL           string
			cluster           string
			thought           string
			finalAnswer       string
			status            string
			toolCalls         []audit.ToolCall
			perfMetrics       map[string]time.Duration
			assistantDuration time.Duration
			parseDuration     time.Duration
		)

		// 从上下文中提取数据
		if v, exists := c.Get("audit_data"); exists {
			if data, ok := v.(map[string]interface{}); ok {
				if q, ok := data["question"].(string); ok {
					question = q
				}
				if m, ok := data["model_name"].(string); ok {
					modelName = m
				}
				if p, ok := data["provider"].(string); ok {
					provider = p
				}
				if b, ok := data["base_url"].(string); ok {
					baseURL = b
				}
				if cl, ok := data["cluster"].(string); ok {
					cluster = cl
				}
				if t, ok := data["thought"].(string); ok {
					thought = t
				}
				if f, ok := data["final_answer"].(string); ok {
					finalAnswer = f
				}
				if s, ok := data["status"].(string); ok {
					status = s
				}
				if tc, ok := data["tool_calls"].([]audit.ToolCall); ok {
					toolCalls = tc
				}
				if pm, ok := data["perf_metrics"].(map[string]time.Duration); ok {
					perfMetrics = pm
				}
				if ad, ok := data["assistant_duration"].(time.Duration); ok {
					assistantDuration = ad
				}
				if pd, ok := data["parse_duration"].(time.Duration); ok {
					parseDuration = pd
				}
			}
		}

		// 如果没有从上下文中获取到问题，尝试从请求体中获取
		if question == "" {
			// 尝试从请求体中提取问题
			if v, exists := c.Get("request_body"); exists {
				if reqBody, ok := v.(map[string]interface{}); ok {
					if instructions, ok := reqBody["instructions"].(string); ok {
						question = instructions
					}
				}
			}
		}

		// 如果没有从上下文中获取到最终答案，尝试从响应体中获取
		if finalAnswer == "" && len(blw.body) > 0 {
			// 尝试从响应体中提取最终答案
			var respBody map[string]interface{}
			if err := json.Unmarshal(blw.body, &respBody); err == nil {
				if message, ok := respBody["message"].(string); ok {
					finalAnswer = message
				}
				if s, ok := respBody["status"].(string); ok {
					status = s
				}
			} else {
				// 如果解析失败，尝试清理后再解析
				cleanedJSON := utils.CleanJSON(string(blw.body))
				if err := json.Unmarshal([]byte(cleanedJSON), &respBody); err == nil {
					if message, ok := respBody["message"].(string); ok {
						finalAnswer = message
					}
					if s, ok := respBody["status"].(string); ok {
						status = s
					}
				}
			}
		}

		// 创建日志条目
		entry := audit.LogEntry{
			SessionID:         sessionID,
			InteractionID:     interactionID,
			UserID:            c.GetString("username"),
			ClientIP:          c.ClientIP(),
			UserAgent:         c.Request.UserAgent(),
			Question:          question,
			ModelName:         modelName,
			Provider:          provider,
			BaseURL:           baseURL,
			Cluster:           cluster,
			Thought:           thought,
			FinalAnswer:       finalAnswer,
			Status:            status,
			ToolCalls:         toolCalls,
			PerfMetrics:       perfMetrics,
			TotalDuration:     totalDuration,
			AssistantDuration: assistantDuration,
			ParseDuration:     parseDuration,
		}

		// 异步记录日志
		auditLogger.LogInteraction(entry)

		logger.Debug("完成审计请求",
			zap.String("path", c.Request.URL.Path),
			zap.String("interaction_id", interactionID.String()),
			zap.Duration("duration", totalDuration),
		)
	}
}

// bodyLogWriter 是一个自定义的响应体写入器，用于捕获响应体
type bodyLogWriter struct {
	gin.ResponseWriter
	body []byte
}

// Write 实现 ResponseWriter 接口
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return w.ResponseWriter.Write(b)
}

// WriteString 实现 ResponseWriter 接口
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body = append(w.body, s...)
	return w.ResponseWriter.WriteString(s)
}
