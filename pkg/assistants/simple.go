package assistants

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/myysophia/OpsAgent/pkg/llms"
	"github.com/myysophia/OpsAgent/pkg/tools"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	// 使用新的日志工具包获取日志记录器
	logger = utils.GetLogger()
}

const (
	defaultMaxIterations = 5
)

// Action 表示工具调用的动作
type Action struct {
	Name  string `json:"name"`
	Input string `json:"input"`
}

func Assistant(model string, prompts []openai.ChatCompletionMessage, maxTokens int, countTokens bool, verbose bool, maxIterations int) (result string, chatHistory []openai.ChatCompletionMessage, err error) {
	return AssistantWithConfig(model, prompts, maxTokens, countTokens, verbose, maxIterations, "", "")
}

// AssistantWithConfig is the AI assistant with custom configuration.
func AssistantWithConfig(model string, prompts []openai.ChatCompletionMessage, maxTokens int, countTokens bool, verbose bool, maxIterations int, apiKey string, baseUrl string) (result string, chatHistory []openai.ChatCompletionMessage, err error) {
	// 获取性能统计工具
	perfStats := utils.GetPerfStats()
	// 开始整体执行计时
	defer perfStats.TraceFunc("assistant_total")()

	logger.Info("开始执行 AssistantWithConfig",
		zap.String("model", model),
		zap.Int("maxTokens", maxTokens),
		zap.Bool("countTokens", countTokens),
		zap.Bool("verbose", verbose),
		zap.Int("maxIterations", maxIterations),
		zap.String("baseUrl", baseUrl),
	)

	chatHistory = prompts
	if len(prompts) == 0 {
		logger.Error("提示信息为空")
		return "", nil, fmt.Errorf("prompts cannot be empty")
	}

	// 开始创建客户端计时
	perfStats.StartTimer("assistant_create_client")

	client, err := llms.NewOpenAIClient(apiKey, baseUrl)

	// 停止创建客户端计时
	clientDuration := perfStats.StopTimer("assistant_create_client")
	logger.Debug("创建OpenAI客户端完成",
		zap.Duration("duration", clientDuration),
	)

	if err != nil {
		logger.Error("创建 OpenAI 客户端失败",
			zap.Error(err),
		)
		return "", nil, fmt.Errorf("unable to get OpenAI client: %v", err)
	}

	// 开始第一轮对话计时
	perfStats.StartTimer("assistant_first_chat")

	resp, err := client.Chat(model, maxTokens, chatHistory)

	// 停止第一轮对话计时
	chatDuration := perfStats.StopTimer("assistant_first_chat")
	logger.Debug("第一轮对话完成",
		zap.Duration("duration", chatDuration),
		zap.String("response", resp),
	)
	if err != nil {
		logger.Error("对话完成失败",
			zap.Error(err),
		)
		return "", chatHistory, fmt.Errorf("chat completion error: %v", err)
	}

	chatHistory = append(chatHistory, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: string(resp),
	})

	// 开始解析工具提示计时
	perfStats.StartTimer("assistant_parse_tool_prompt")

	var toolPrompt tools.ToolPrompt
	var action Action

	// 首先尝试解析完整的ToolPrompt
	if err = json.Unmarshal([]byte(resp), &toolPrompt); err != nil {
		// 如果完整解析失败，尝试只解析action部分
		var actionOnly struct {
			Action Action `json:"action"`
		}
		if err = json.Unmarshal([]byte(resp), &actionOnly); err == nil && actionOnly.Action.Name != "" {
			// 如果成功解析到action，构建完整的toolPrompt
			toolPrompt = tools.ToolPrompt{
				Action: actionOnly.Action,
				// 设置默认值
				Question:    "No question provided",
				Thought:     "No thought provided",
				Observation: "",
				FinalAnswer: "",
			}
			logger.Debug("成功解析action部分",
				zap.String("tool", toolPrompt.Action.Name),
				zap.String("input", toolPrompt.Action.Input),
			)
		} else {
			// 如果action解析也失败，尝试直接解析为Action
			if err = json.Unmarshal([]byte(resp), &action); err == nil && action.Name != "" {
				toolPrompt = tools.ToolPrompt{
					Action: action,
					// 设置默认值
					Question:    "No question provided",
					Thought:     "No thought provided",
					Observation: "",
					FinalAnswer: "",
				}
				logger.Debug("成功解析为Action",
					zap.String("tool", toolPrompt.Action.Name),
					zap.String("input", toolPrompt.Action.Input),
				)
			} else {
				// 停止解析工具提示计时
				parseDuration := perfStats.StopTimer("assistant_parse_tool_prompt")
				logger.Debug("解析工具提示失败",
					zap.Duration("duration", parseDuration),
					zap.Error(err),
					zap.String("response", resp),
				)

				if verbose {
					logger.Warn("无法解析工具提示，假定为最终答案",
						zap.Error(err),
						zap.String("response", resp),
					)
				}
				return resp, chatHistory, nil
			}
		}
	}

	// 停止解析工具提示计时
	parseDuration := perfStats.StopTimer("assistant_parse_tool_prompt")
	logger.Debug("解析工具提示成功",
		zap.Duration("duration", parseDuration),
		zap.String("tool", toolPrompt.Action.Name),
		zap.String("input", toolPrompt.Action.Input),
	)

	iterations := 0
	if maxIterations <= 0 {
		maxIterations = defaultMaxIterations
	}
	for {
		iterations++
		// 记录每次迭代的思考过程
		if verbose {
			logger.Debug("LLM思考过程",
				zap.Int("iteration", iterations),
				zap.String("thought", toolPrompt.Thought),
				zap.String("question", toolPrompt.Question),
				zap.Any("action", toolPrompt.Action),
				zap.String("observation", toolPrompt.Observation),
			)
		}

		if iterations > maxIterations {
			logger.Warn("达到最大迭代次数",
				zap.Int("maxIterations", maxIterations),
			)
			return toolPrompt.FinalAnswer, chatHistory, nil
		}

		if toolPrompt.FinalAnswer != "" && !isTemplateValue(toolPrompt.FinalAnswer) && toolPrompt.Observation != "" {
			logger.Info("获得最终答案",
				zap.String("finalAnswer", toolPrompt.FinalAnswer),
			)
			return toolPrompt.FinalAnswer, chatHistory, nil
		}

		if toolPrompt.Action.Name != "" {
			var observation string
			logger.Debug("执行工具",
				zap.String("tool", toolPrompt.Action.Name),
				zap.String("input", toolPrompt.Action.Input),
			)

			if verbose {
				logger.Debug("执行工具迭代",
					zap.Int("iteration", iterations),
					zap.String("tool", toolPrompt.Action.Name),
				)
				logger.Debug("工具输入",
					zap.String("tool", toolPrompt.Action.Name),
					zap.String("input", toolPrompt.Action.Input),
				)
			}

			// 开始工具执行计时
			perfStats.StartTimer("assistant_tool_" + toolPrompt.Action.Name)

			if toolFunc, ok := tools.CopilotTools[toolPrompt.Action.Name]; ok {
				ret, err := toolFunc(toolPrompt.Action.Input)
				observation = strings.TrimSpace(ret)

				// 停止工具执行计时
				toolDuration := perfStats.StopTimer("assistant_tool_" + toolPrompt.Action.Name)

				if err != nil {
					logger.Error("工具执行失败",
						zap.String("tool", toolPrompt.Action.Name),
						zap.Error(err),
						zap.Duration("duration", toolDuration),
					)
					observation = fmt.Sprintf("Tool %s failed with error: %s. Please try using a different tool or refine the input parameters.", toolPrompt.Action.Name, ret)

					// 将错误信息添加到对话历史中
					toolPrompt.Observation = observation
					assistantMessage, _ := json.Marshal(toolPrompt)
					chatHistory = append(chatHistory, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleUser,
						Content: string(assistantMessage),
					})

					// 继续下一轮迭代
					continue
				} else {
					logger.Debug("工具执行成功",
						zap.String("tool", toolPrompt.Action.Name),
						zap.String("observation", observation),
						zap.Duration("duration", toolDuration),
					)
				}
			}
			//else {
			//	// 停止工具执行计时（工具不可用的情况）
			//	toolDuration := perfStats.StopTimer("assistant_tool_" + toolPrompt.Action.Name)
			//
			//	logger.Warn("工具不可用",
			//		zap.String("tool", toolPrompt.Action.Name),
			//		zap.Duration("duration", toolDuration),
			//	)
			//	observation = fmt.Sprintf("Tool %s is not available. Considering switch to other supported tools.", toolPrompt.Action.Name)
			//}

			if verbose {
				logger.Debug("工具执行结果",
					zap.String("observation", observation),
				)
			}

			// 开始消息构建计时
			perfStats.StartTimer("assistant_construct_message")

			// Constrict the prompt to the max tokens allowed by the model.
			// This is required because the tool may have generated a long output.
			observation = llms.ConstrictPrompt(observation, model, 1024)
			toolPrompt.Observation = observation
			assistantMessage, _ := json.Marshal(toolPrompt)
			chatHistory = append(chatHistory, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: string(assistantMessage),
			})
			// Constrict the chat history to the max tokens allowed by the model.
			// This is required because the chat history may have grown too large.
			//chatHistory = llms.ConstrictMessages(chatHistory, model, maxTokens)

			// 停止消息构建计时
			constructDuration := perfStats.StopTimer("assistant_construct_message")
			logger.Debug("消息构建完成",
				zap.Duration("duration", constructDuration),
			)

			// 开始中间对话计时
			perfStats.StartTimer("assistant_intermediate_chat")

			resp, err := client.Chat(model, maxTokens, chatHistory)

			// 停止中间对话计时
			intermediateChatDuration := perfStats.StopTimer("assistant_intermediate_chat")
			logger.Debug("中间对话完成",
				zap.Duration("duration", intermediateChatDuration),
			)

			if err != nil {
				logger.Error("对话完成失败",
					zap.Error(err),
				)
				return "", chatHistory, fmt.Errorf("chat completion error: %v", err)
			}

			chatHistory = append(chatHistory, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: string(resp),
			})
			if verbose {
				logger.Debug("LLM 中间响应",
					zap.String("response", resp),
				)
			}

			// 开始解析中间响应计时
			perfStats.StartTimer("assistant_parse_intermediate")

			// extract the tool prompt from the LLM response.
			if err = json.Unmarshal([]byte(resp), &toolPrompt); err != nil {
				// 停止解析中间响应计时
				parseIntermediateDuration := perfStats.StopTimer("assistant_parse_intermediate")
				logger.Debug("解析中间响应失败",
					zap.Duration("duration", parseIntermediateDuration),
					zap.Error(err),
				)

				if verbose {
					logger.Warn("无法从 LLM 解析工具，总结最终答案",
						zap.Error(err),
					)
				}

				chatHistory = append(chatHistory, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: "Summarize all the chat history and respond to original question with final answer",
				})

				// 开始总结对话计时
				perfStats.StartTimer("assistant_summarize")

				resp, err = client.Chat(model, maxTokens, chatHistory)

				// 停止总结对话计时
				summarizeDuration := perfStats.StopTimer("assistant_summarize")
				logger.Debug("总结对话完成",
					zap.Duration("duration", summarizeDuration),
				)

				if err != nil {
					logger.Error("总结对话失败",
						zap.Error(err),
					)
					return "", chatHistory, fmt.Errorf("chat completion error: %v", err)
				}

				logger.Info("完成总结",
					zap.String("summary", resp),
				)

				// 尝试从响应中提取final_answer并处理格式
				// 这里处理LLM返回的JSON响应，确保只返回final_answer部分
				var finalResponse map[string]interface{}
				if err := json.Unmarshal([]byte(resp), &finalResponse); err == nil {
					if finalAnswer, ok := finalResponse["final_answer"].(string); ok && finalAnswer != "" {
						logger.Info("成功提取final_answer",
							zap.String("final_answer", finalAnswer),
						)
						if err == nil {
							return string(resp), chatHistory, nil
						}
						// 如果JSON序列化失败，直接返回原始的final_answer
						return finalAnswer, chatHistory, nil
					}
				}
				return resp, chatHistory, nil
			} else {
				// 停止解析中间响应计时
				parseIntermediateDuration := perfStats.StopTimer("assistant_parse_intermediate")
				logger.Debug("解析中间响应成功",
					zap.Duration("duration", parseIntermediateDuration),
				)
				if toolPrompt.FinalAnswer != "" && toolPrompt.Observation != "" {
					logger.Info("获得最终答案",
						zap.String("finalAnswer", toolPrompt.FinalAnswer),
					)
					return toolPrompt.FinalAnswer, chatHistory, nil
				}
			}
		}
	}
}

// isTemplateValue 检查字符串是否为模板值或占位符
// 参数：
//   - value: 要检查的字符串
//
// 返回：
//   - bool: 如果是模板值或占位符则返回true，否则返回false
func isTemplateValue(value string) bool {
	// 检查常见的模板值模式
	templatePatterns := []string{
		"<最终答案",
		"<final_answer",
		"<Final answer",
		"<最终回答",
		"<回答",
		"<答案",
		"使用 Markdown 格式",
		"使用Markdown格式",
		"换行符用 \\n 表示",
		"换行符用\\n表示",
	}

	// 如果值很短，可能是占位符
	if len(value) < 10 {
		return true
	}

	// 检查是否包含模板模式
	for _, pattern := range templatePatterns {
		if strings.Contains(value, pattern) {
			return true
		}
	}

	// 检查是否只包含指导性文本而没有实际内容
	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		return true
	}

	return false
}
