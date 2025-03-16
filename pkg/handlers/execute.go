package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
	"net/http"
	"strings"

	"github.com/myysophia/OpsAgent/pkg/assistants"
	"github.com/myysophia/OpsAgent/pkg/utils"
)

// ExecuteRequest 执行请求结构
type ExecuteRequest struct {
	Instructions   string   `json:"instructions" binding:"required"`
	Args           string   `json:"args" binding:"required"`
	Provider       string   `json:"provider"`
	BaseUrl        string   `json:"baseUrl"`
	CurrentModel   string   `json:"currentModel"`
	Cluster        string   `json:"cluster"`
	SelectedModels []string `json:"selectedModels"`
}

// AIResponse AI 响应结构
type AIResponse struct {
	Question string `json:"question"`
	Thought  string `json:"thought"`
	Action   struct {
		Name  string `json:"name"`
		Input string `json:"input"`
	} `json:"action"`
	Observation string `json:"observation"`
	FinalAnswer string `json:"final_answer"`
}

// 添加工具历史记录结构
type ToolHistory struct {
	Name        string `json:"name"`
	Input       string `json:"input"`
	Observation string `json:"observation"`
}

const executeSystemPrompt_cn = `您是Kubernetes和云原生网络的技术专家，您的任务是遵循链式思维方法，确保彻底性和准确性，同时遵守约束。

可用工具：
- kubectl：用于执行 Kubernetes 命令。必须使用正确语法（例如 'kubectl get pods' 而非 'kubectl get pod'），避免使用 -o json/yaml 全量输出。
- python：用于复杂逻辑或调用 Kubernetes Python SDK。输入：Python 脚本，输出：通过 print(...) 返回。
- trivy：用于扫描镜像漏洞。输入：镜像名称，输出：漏洞报告。
- jq：用于处理 JSON 数据。输入：有效的 jq 表达式，始终使用 'test()' 进行名称匹配。

您采取的步骤如下：
1. 问题识别：清楚定义问题，描述目标。
2. 诊断命令：根据问题选择工具，优先使用 kubectl 获取数据。若涉及 JSON 处理，使用 jq 并确保语法一致。
3. 输出解释：分析工具输出，描述结果。如果输出为空，必须明确告知用户未找到相关信息。
4. 故障排除策略：根据输出制定策略。
5. 可行解决方案：提出解决方案，确保命令准确。

严格约束：
- 始终使用 'kubectl get pods'（复数形式）获取 Pod 信息，禁止使用 'kubectl get pod'。
- 避免使用 -o json/yaml 全量输出，优先使用 jsonpath 或 custom-columns 进行精确查询。
- 使用 --no-headers 选项减少不必要的输出。
- jq 表达式中，名称匹配必须使用 'test()'，避免使用 '=='。
- Bash 命令需正确转义特殊字符（如 []、"），必要时用单引号包裹参数。
- 检查 Bash 转义并修复错误，确保命令正确执行。
- 当工具执行结果为空时，必须在final_answer中明确告知用户"未找到相关信息"，不要返回示例或虚构的结果。
示例：
- 问题："查看名称包含nginx的pod的镜像版本是什么？"
  - 正确：'kubectl get pods --no-headers | grep nginx | awk "{print $1}" | xargs -I {} kubectl get pod {} -o jsonpath="{.spec.containers[*].image}"'
  - 错误：'kubectl get pods -o json | jq -r ".items[] | select(.metadata.name | test(\"nginx\")) | .spec.containers[].image"'

重要提示：始终使用以下 JSON 格式返回响应：
{
  "question": "<用户的输入问题>",
  "thought": "<您的分析和思考过程>",
  "action": {
    "name": "<工具名称>",
    "input": "<工具输入>"
  },
  "observation": "",
  "final_answer": "<最终答案，使用Markdown格式。如果工具执行结果为空，必须返回'未找到相关信息'>"
}

注意：
1. observation字段必须保持为空字符串，不要填写任何内容，系统会自动填充
2. final_answer必须是有意义的回答，不能包含模板文本或占位符
3. 如果需要执行工具，填写action字段；如果已经得到答案，可以直接在final_answer中回复
4. 禁止在任何字段中使用类似"<工具执行结果，由外部填充>"这样的模板文本
5. 当工具执行结果为空时，不要直接返回"未找到相关信息"，而是：
   - 分析可能的原因
   - 提供改进建议
   - 询问用户是否需要进一步澄清

## 示例
问题："查看名称包含nginx的pod的镜像版本是什么？"

当结果为空时，应该这样处理：
1. 首先尝试使用更宽松的查询：
   kubectl get pods --no-headers | grep -i nginx | awk "{print $1}" | xargs -I {} kubectl get pod {} -o jsonpath="{.spec.containers[*].image}"

2. 如果仍然为空，在 final_answer 中提供：
   - 当前查询条件说明
   - 可能的原因（如命名空间问题、权限问题等）
   - 建议的解决方案
   - 是否需要用户提供更多信息
目标：
在 Kubernetes 和云原生网络领域内识别问题根本原因，提供清晰、可行的解决方案，同时保持诊断和故障排除的运营约束。`

// Execute 处理执行请求
func Execute(c *gin.Context) {
	// 获取性能统计工具
	perfStats := utils.GetPerfStats()
	// 开始整体执行计时
	defer perfStats.TraceFunc("execute_total")()

	// 获取 logger
	logger := utils.GetLogger()

	// 获取是否显示思考过程的配置
	// 首先尝试从URL参数获取
	showThoughtStr := c.DefaultQuery("show-thought", "")
	var showThought bool

	if showThoughtStr != "" {
		// 如果URL参数中有指定，则使用URL参数的值
		showThought = showThoughtStr == "true"
	} else {
		// 否则尝试从全局变量中获取配置
		if value, exists := utils.GetGlobalVar("showThought"); exists {
			showThought = value.(bool)
		} else {
			// 默认不显示思考过程
			showThought = false
		}
	}

	logger.Debug("Execute处理请求",
		zap.Bool("show-thought", showThought),
	)

	// 获取API Key
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		logger.Error("缺少 API Key")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing API Key"})
		return
	}

	// 解析请求体
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Debug("Execute 请求解析失败",
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("请求格式错误: %v", err)})
		return
	}

	// 记录请求信息
	logger.Debug("Execute 接口收到请求",
		zap.String("instructions", req.Instructions),
		zap.String("args", req.Args),
		zap.String("provider", req.Provider),
		zap.String("baseUrl", req.BaseUrl),
		zap.String("currentModel", req.CurrentModel),
		zap.Strings("selectedModels", req.SelectedModels),
		zap.String("cluster", req.Cluster),
		zap.String("apiKey", "***"),
	)

	// 确定使用的模型
	executeModel := req.CurrentModel
	if executeModel == "" {
		executeModel = "gpt-4"
	}

	// 构建执行指令
	instructions := req.Instructions
	if req.Args != "" && !strings.Contains(instructions, req.Args) {
		instructions = fmt.Sprintf("%s %s", req.Instructions, req.Args)
	}

	// 清理指令
	cleanInstructions := strings.TrimPrefix(instructions, "execute")
	cleanInstructions = strings.TrimSpace(cleanInstructions)
	logger.Debug("Execute 执行参数",
		zap.String("model", executeModel),
		zap.String("instructions", cleanInstructions),
		zap.String("baseUrl", req.BaseUrl),
		zap.String("cluster", req.Cluster),
	)

	// 构建 OpenAI 消息
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: executeSystemPrompt_cn,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: cleanInstructions,
		},
	}

	// 开始 AI 助手执行计时
	perfStats.StartTimer("execute_assistant")

	// 调用 AI 助手
	response, chatHistory, err := assistants.AssistantWithConfig(executeModel, messages, 8192, true, true, 10, apiKey, req.BaseUrl)

	// 停止 AI 助手执行计时
	assistantDuration := perfStats.StopTimer("execute_assistant")
	logger.Info("AI助手执行完成",
		zap.Duration("duration", assistantDuration),
	)

	if err != nil {
		logger.Error("Execute 执行失败",
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("执行失败: %v", err),
		})
		return
	}

	// 提取工具使用历史
	var toolsHistory []ToolHistory
	for i := 0; i < len(chatHistory); i++ {
		if chatHistory[i].Role == openai.ChatMessageRoleUser && i > 0 {
			var toolPrompt map[string]interface{}
			if err := json.Unmarshal([]byte(chatHistory[i].Content), &toolPrompt); err == nil {
				if action, ok := toolPrompt["action"].(map[string]interface{}); ok {
					name, _ := action["name"].(string)
					input, _ := action["input"].(string)
					observation, _ := toolPrompt["observation"].(string)

					if name != "" && input != "" {
						toolsHistory = append(toolsHistory, ToolHistory{
							Name:        name,
							Input:       input,
							Observation: observation,
						})
					}
				}
			}
		}
	}

	// 开始响应解析计时
	perfStats.StartTimer("execute_response_parse")

	// 解析 AI 响应
	var aiResp AIResponse
	err = json.Unmarshal([]byte(response), &aiResp)

	if err != nil {
		logger.Debug("标准JSON解析失败，尝试更健壮的解析方法",
			zap.Error(err),
			zap.String("response", response),
		)

		// 尝试提取 final_answer
		finalAnswer, extractErr := utils.ExtractField(response, "final_answer")
		thought, _ := utils.ExtractField(response, "thought")
		question, _ := utils.ExtractField(response, "question")

		// 尝试提取action和observation
		var action struct {
			Name  string `json:"name"`
			Input string `json:"input"`
		}
		actionStr, _ := utils.ExtractField(response, "action")
		if actionStr != "" {
			json.Unmarshal([]byte(actionStr), &action)
		}
		observation, _ := utils.ExtractField(response, "observation")

		if extractErr == nil && finalAnswer != "" {
			logger.Debug("成功使用工具函数提取final_answer",
				zap.String("final_answer", finalAnswer),
				zap.String("thought", thought),
			)

			parseDuration := perfStats.StopTimer("execute_response_parse")
			logger.Debug("响应解析完成（工具函数提取）",
				zap.Duration("duration", parseDuration),
			)

			responseData := gin.H{
				"message": finalAnswer,
				"status":  "success",
			}

			// 根据showThought配置决定是否返回思考过程和工具历史
			if showThought {
				responseData["thought"] = thought
				responseData["question"] = question
				responseData["action"] = action
				responseData["observation"] = observation
				responseData["tools_history"] = toolsHistory
			}

			c.JSON(http.StatusOK, responseData)
			return
		}

		// 尝试清理 JSON 后解析
		cleanedJSON := utils.CleanJSON(response)
		if err2 := json.Unmarshal([]byte(cleanedJSON), &aiResp); err2 == nil && aiResp.FinalAnswer != "" {
			logger.Debug("成功从清理后的JSON中提取final_answer",
				zap.String("final_answer", aiResp.FinalAnswer),
				zap.String("thought", aiResp.Thought),
			)

			parseDuration := perfStats.StopTimer("execute_response_parse")
			logger.Debug("响应解析完成（清理JSON后解析）",
				zap.Duration("duration", parseDuration),
			)

			responseData := gin.H{
				"message": aiResp.FinalAnswer,
				"status":  "success",
			}

			// 根据showThought配置决定是否返回思考过程和工具历史
			if showThought {
				responseData["thought"] = aiResp.Thought
				responseData["question"] = aiResp.Question
				responseData["action"] = aiResp.Action
				responseData["observation"] = aiResp.Observation
				responseData["tools_history"] = toolsHistory
			}

			c.JSON(http.StatusOK, responseData)
			return
		}

		// 尝试从非标准 JSON 中提取
		var genericResp map[string]interface{}
		if err2 := json.Unmarshal([]byte(response), &genericResp); err2 == nil {
			if finalAnswer, ok := genericResp["final_answer"].(string); ok && finalAnswer != "" {
				logger.Debug("成功从非标准JSON中提取final_answer",
					zap.String("final_answer", finalAnswer),
				)

				parseDuration := perfStats.StopTimer("execute_response_parse")
				logger.Debug("响应解析完成（非标准JSON提取）",
					zap.Duration("duration", parseDuration),
				)

				// 提取其他字段
				thought, _ := genericResp["thought"].(string)
				question, _ := genericResp["question"].(string)
				observation, _ := genericResp["observation"].(string)

				// 提取action
				var action struct {
					Name  string `json:"name"`
					Input string `json:"input"`
				}
				if actionMap, ok := genericResp["action"].(map[string]interface{}); ok {
					if name, ok := actionMap["name"].(string); ok {
						action.Name = name
					}
					if input, ok := actionMap["input"].(string); ok {
						action.Input = input
					}
				}

				responseData := gin.H{
					"message": finalAnswer,
					"status":  "success",
				}

				// 根据showThought配置决定是否返回思考过程和工具历史
				if showThought {
					responseData["thought"] = thought
					responseData["question"] = question
					responseData["action"] = action
					responseData["observation"] = observation
					responseData["tools_history"] = toolsHistory
				}

				c.JSON(http.StatusOK, responseData)
				return
			}
		}

		parseDuration := perfStats.StopTimer("execute_response_parse")
		logger.Debug("所有解析方法均失败，返回原始响应",
			zap.Duration("duration", parseDuration),
		)

		responseData := gin.H{
			"message":      response,
			"raw_response": true,
			"status":       "success",
		}

		// 即使在解析失败的情况下，也根据showThought配置决定是否返回工具历史
		if showThought {
			responseData["tools_history"] = toolsHistory
		}

		c.JSON(http.StatusOK, responseData)
		return
	}

	parseDuration := perfStats.StopTimer("execute_response_parse")
	logger.Debug("响应解析完成（标准格式）",
		zap.Duration("duration", parseDuration),
	)

	if aiResp.FinalAnswer != "" {
		responseData := gin.H{
			"message": aiResp.FinalAnswer,
			"status":  "success",
		}

		// 根据showThought配置决定是否返回思考过程和工具历史
		if showThought {
			responseData["thought"] = aiResp.Thought
			responseData["question"] = aiResp.Question
			responseData["action"] = aiResp.Action
			responseData["observation"] = aiResp.Observation
			responseData["tools_history"] = toolsHistory
		}

		c.JSON(http.StatusOK, responseData)
	} else {
		responseData := gin.H{
			"message": "指令正在执行中，请稍候...",
			"status":  "processing",
		}

		// 根据showThought配置决定是否返回思考过程和工具历史
		if showThought {
			responseData["thought"] = aiResp.Thought
			responseData["question"] = aiResp.Question
			responseData["action"] = aiResp.Action
			responseData["observation"] = aiResp.Observation
			responseData["tools_history"] = toolsHistory
		}

		c.JSON(http.StatusOK, responseData)
	}
}
