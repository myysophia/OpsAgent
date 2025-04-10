package middleware

import (
	"encoding/json"
	"fmt"
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

		// 获取日志记录器
		logger = utils.GetLogger().Named("audit-middleware")

		// 从上下文中提取数据
		if v, exists := c.Get("audit_data"); exists {
			logger.Debug("从上下文中获取到 audit_data",
				zap.Any("audit_data_type", fmt.Sprintf("%T", v)),
			)

			if data, ok := v.(map[string]interface{}); ok {
				// 检查关键字段是否存在
				var keys []string
				for k := range data {
					keys = append(keys, k)
				}
				logger.Debug("完整的 audit_data 内容",
					zap.Strings("keys", keys),
				)

				// 直接检查关键字段
				logger.Debug("检查关键字段",
					zap.Bool("has_model_name", data["model_name"] != nil),
					zap.Bool("has_provider", data["provider"] != nil),
					zap.Bool("has_base_url", data["base_url"] != nil),
					zap.Bool("has_thought", data["thought"] != nil),
					zap.Bool("has_tool_calls", data["tool_calls"] != nil),
					zap.Bool("has_perf_metrics", data["perf_metrics"] != nil),
				)

				// 检查关键字段的值
				if modelName, ok := data["model_name"].(string); ok {
					logger.Debug("检查 model_name 值", zap.String("model_name", modelName))
				}
				if provider, ok := data["provider"].(string); ok {
					logger.Debug("检查 provider 值", zap.String("provider", provider))
				}
				if baseURL, ok := data["base_url"].(string); ok {
					logger.Debug("检查 base_url 值", zap.String("base_url", baseURL))
				}
				if thought, ok := data["thought"].(string); ok {
					logger.Debug("检查 thought 值",
						zap.String("thought", thought),
						zap.Int("thought_length", len(thought)),
					)
				}
				if toolCalls, ok := data["tool_calls"].([]interface{}); ok {
					logger.Debug("检查 tool_calls 值",
						zap.Int("tool_calls_count", len(toolCalls)),
						zap.Any("tool_calls_type", fmt.Sprintf("%T", toolCalls)),
					)
				}
				if perfMetrics, ok := data["perf_metrics"].(map[string]interface{}); ok {
					logger.Debug("检查 perf_metrics 值",
						zap.Int("perf_metrics_count", len(perfMetrics)),
						zap.Any("perf_metrics_type", fmt.Sprintf("%T", perfMetrics)),
					)
				}
				if q, ok := data["question"].(string); ok {
					question = q
				}
				// 处理 model_name 字段
				if m, exists := data["model_name"]; exists {
					logger.Debug("获取到 model_name 字段",
						zap.Any("model_name_type", fmt.Sprintf("%T", m)),
						zap.Any("model_name_value", m),
					)
					if mStr, ok := m.(string); ok {
						modelName = mStr
						logger.Debug("设置 model_name", zap.String("model_name", modelName))
					}
				} else {
					logger.Debug("没有获取到 model_name 字段")
				}

				// 处理 provider 字段
				if p, exists := data["provider"]; exists {
					logger.Debug("获取到 provider 字段",
						zap.Any("provider_type", fmt.Sprintf("%T", p)),
						zap.Any("provider_value", p),
					)
					if pStr, ok := p.(string); ok {
						provider = pStr
						logger.Debug("设置 provider", zap.String("provider", provider))
					}
				} else {
					logger.Debug("没有获取到 provider 字段")
				}

				// 处理 base_url 字段
				if b, exists := data["base_url"]; exists {
					logger.Debug("获取到 base_url 字段",
						zap.Any("base_url_type", fmt.Sprintf("%T", b)),
						zap.Any("base_url_value", b),
					)
					if bStr, ok := b.(string); ok {
						baseURL = bStr
						logger.Debug("设置 base_url", zap.String("base_url", baseURL))
					}
				} else {
					logger.Debug("没有获取到 base_url 字段")
				}
				if cl, ok := data["cluster"].(string); ok {
					cluster = cl
				}
				// 处理 thought 字段
				if t, exists := data["thought"]; exists {
					logger.Debug("获取到 thought 字段",
						zap.Any("thought_type", fmt.Sprintf("%T", t)),
						zap.Any("thought_value", t),
					)
					if tStr, ok := t.(string); ok {
						thought = tStr
					}
				} else {
					logger.Debug("没有获取到 thought 字段")
				}
				if f, ok := data["final_answer"].(string); ok {
					finalAnswer = f
				}
				if s, ok := data["status"].(string); ok {
					status = s
				}
				// 处理 tool_calls 字段
				if tc, exists := data["tool_calls"]; exists {
					logger.Debug("获取到 tool_calls 字段",
						zap.Any("tool_calls_type", fmt.Sprintf("%T", tc)),
						zap.Any("tool_calls_value", tc),
					)

					// 尝试直接转换
					if tcArray, ok := tc.([]audit.ToolCall); ok {
						logger.Debug("直接转换 tool_calls 成功",
							zap.Int("tool_calls_count", len(tcArray)),
						)
						toolCalls = tcArray
					} else if tcArray, ok := tc.([]interface{}); ok {
						logger.Debug("将 tool_calls 从接口数组转换",
							zap.Int("tool_calls_count", len(tcArray)),
						)
						// 如果是接口数组，需要手动转换
						for i, item := range tcArray {
							logger.Debug(fmt.Sprintf("处理第 %d 个 tool_call", i),
								zap.Any("item_type", fmt.Sprintf("%T", item)),
								zap.Any("item_value", item),
							)

							if tcMap, ok := item.(map[string]interface{}); ok {
								var toolCall audit.ToolCall

								// 输出 tcMap 的所有键
								var keys []string
								for k := range tcMap {
									keys = append(keys, k)
								}
								logger.Debug(fmt.Sprintf("第 %d 个 tool_call 的键", i),
									zap.Strings("keys", keys),
								)

								// 尝试大写和小写的键
								if name, ok := tcMap["Name"].(string); ok {
									toolCall.Name = name
								} else if name, ok := tcMap["name"].(string); ok {
									toolCall.Name = name
								}

								if input, ok := tcMap["Input"].(string); ok {
									toolCall.Input = input
								} else if input, ok := tcMap["input"].(string); ok {
									toolCall.Input = input
								}

								if observation, ok := tcMap["Observation"].(string); ok {
									toolCall.Observation = observation
								} else if observation, ok := tcMap["observation"].(string); ok {
									toolCall.Observation = observation
								}

								if seqNum, ok := tcMap["SequenceNum"].(float64); ok {
									toolCall.SequenceNum = int(seqNum)
								} else if seqNum, ok := tcMap["sequenceNum"].(float64); ok {
									toolCall.SequenceNum = int(seqNum)
								} else if seqNum, ok := tcMap["sequence_num"].(float64); ok {
									toolCall.SequenceNum = int(seqNum)
								} else if seqNum, ok := tcMap["SequenceNum"].(int); ok {
									toolCall.SequenceNum = seqNum
								} else if seqNum, ok := tcMap["sequenceNum"].(int); ok {
									toolCall.SequenceNum = seqNum
								} else if seqNum, ok := tcMap["sequence_num"].(int); ok {
									toolCall.SequenceNum = seqNum
								}

								if duration, ok := tcMap["Duration"].(float64); ok {
									toolCall.Duration = time.Duration(duration) * time.Millisecond
								} else if duration, ok := tcMap["duration"].(float64); ok {
									toolCall.Duration = time.Duration(duration) * time.Millisecond
								} else if duration, ok := tcMap["Duration"].(int64); ok {
									toolCall.Duration = time.Duration(duration) * time.Millisecond
								} else if duration, ok := tcMap["duration"].(int64); ok {
									toolCall.Duration = time.Duration(duration) * time.Millisecond
								} else if duration, ok := tcMap["Duration"].(int); ok {
									toolCall.Duration = time.Duration(duration) * time.Millisecond
								} else if duration, ok := tcMap["duration"].(int); ok {
									toolCall.Duration = time.Duration(duration) * time.Millisecond
								}

								logger.Debug(fmt.Sprintf("添加第 %d 个 tool_call", i),
									zap.String("name", toolCall.Name),
									zap.String("input", toolCall.Input),
									zap.String("observation", toolCall.Observation),
									zap.Int("sequence_num", toolCall.SequenceNum),
									zap.Duration("duration", toolCall.Duration),
								)

								toolCalls = append(toolCalls, toolCall)
							}
						}
					}
				} else {
					logger.Debug("没有获取到 tool_calls 字段")
				}
				// 处理 perf_metrics 字段
				if pm, exists := data["perf_metrics"]; exists {
					logger.Debug("获取到 perf_metrics 字段",
						zap.Any("perf_metrics_type", fmt.Sprintf("%T", pm)),
						zap.Any("perf_metrics_value", pm),
					)

					// 尝试直接转换
					if pmMap, ok := pm.(map[string]time.Duration); ok {
						logger.Debug("直接转换 perf_metrics 成功",
							zap.Int("perf_metrics_count", len(pmMap)),
						)
						perfMetrics = pmMap
					} else if pmMap, ok := pm.(map[string]interface{}); ok {
						logger.Debug("将 perf_metrics 从接口映射转换",
							zap.Int("perf_metrics_count", len(pmMap)),
						)

						// 输出 pmMap 的所有键和值类型
						for key, value := range pmMap {
							logger.Debug(fmt.Sprintf("perf_metrics 键值对: %s", key),
								zap.Any("value_type", fmt.Sprintf("%T", value)),
								zap.Any("value", value),
							)
						}

						// 如果是接口映射，需要手动转换
						perfMetrics = make(map[string]time.Duration)
						for key, value := range pmMap {
							if duration, ok := value.(float64); ok {
								perfMetrics[key] = time.Duration(duration) * time.Millisecond
								logger.Debug(fmt.Sprintf("添加 perf_metric: %s (float64)", key),
									zap.Duration("duration", perfMetrics[key]),
								)
							} else if duration, ok := value.(int64); ok {
								perfMetrics[key] = time.Duration(duration) * time.Millisecond
								logger.Debug(fmt.Sprintf("添加 perf_metric: %s (int64)", key),
									zap.Duration("duration", perfMetrics[key]),
								)
							} else if duration, ok := value.(int); ok {
								perfMetrics[key] = time.Duration(duration) * time.Millisecond
								logger.Debug(fmt.Sprintf("添加 perf_metric: %s (int)", key),
									zap.Duration("duration", perfMetrics[key]),
								)
							} else if durationStr, ok := value.(string); ok {
								// 尝试将字符串转换为数字
								if d, err := time.ParseDuration(durationStr); err == nil {
									perfMetrics[key] = d
									logger.Debug(fmt.Sprintf("添加 perf_metric: %s (string)", key),
										zap.Duration("duration", perfMetrics[key]),
									)
								}
							}
						}

						logger.Debug("转换后的 perf_metrics",
							zap.Int("perf_metrics_count", len(perfMetrics)),
						)
					}
				} else {
					logger.Debug("没有获取到 perf_metrics 字段")
				}
				// 处理 assistant_duration 字段
				if ad, exists := data["assistant_duration"]; exists {
					logger.Debug("获取到 assistant_duration 字段",
						zap.Any("assistant_duration_type", fmt.Sprintf("%T", ad)),
						zap.Any("assistant_duration_value", ad),
					)

					// 尝试直接转换
					if duration, ok := ad.(time.Duration); ok {
						logger.Debug("直接转换 assistant_duration 成功")
						assistantDuration = duration
					} else if duration, ok := ad.(float64); ok {
						logger.Debug("将 assistant_duration 从 float64 转换")
						assistantDuration = time.Duration(duration) * time.Millisecond
					} else if duration, ok := ad.(int64); ok {
						logger.Debug("将 assistant_duration 从 int64 转换")
						assistantDuration = time.Duration(duration) * time.Millisecond
					} else if duration, ok := ad.(int); ok {
						logger.Debug("将 assistant_duration 从 int 转换")
						assistantDuration = time.Duration(duration) * time.Millisecond
					} else if durationStr, ok := ad.(string); ok {
						// 尝试将字符串转换为数字
						if d, err := time.ParseDuration(durationStr); err == nil {
							logger.Debug("将 assistant_duration 从字符串转换")
							assistantDuration = d
						}
					}
				} else {
					logger.Debug("没有获取到 assistant_duration 字段")
				}

				// 处理 parse_duration 字段
				if pd, exists := data["parse_duration"]; exists {
					logger.Debug("获取到 parse_duration 字段",
						zap.Any("parse_duration_type", fmt.Sprintf("%T", pd)),
						zap.Any("parse_duration_value", pd),
					)

					// 尝试直接转换
					if duration, ok := pd.(time.Duration); ok {
						logger.Debug("直接转换 parse_duration 成功")
						parseDuration = duration
					} else if duration, ok := pd.(float64); ok {
						logger.Debug("将 parse_duration 从 float64 转换")
						parseDuration = time.Duration(duration) * time.Millisecond
					} else if duration, ok := pd.(int64); ok {
						logger.Debug("将 parse_duration 从 int64 转换")
						parseDuration = time.Duration(duration) * time.Millisecond
					} else if duration, ok := pd.(int); ok {
						logger.Debug("将 parse_duration 从 int 转换")
						parseDuration = time.Duration(duration) * time.Millisecond
					} else if durationStr, ok := pd.(string); ok {
						// 尝试将字符串转换为数字
						if d, err := time.ParseDuration(durationStr); err == nil {
							logger.Debug("将 parse_duration 从字符串转换")
							parseDuration = d
						}
					}
				} else {
					logger.Debug("没有获取到 parse_duration 字段")
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

		// 输出调试日志
		logger.Debug("审计日志数据",
			zap.String("thought", thought),
			zap.Int("tool_calls_count", len(toolCalls)),
			zap.Int("perf_metrics_count", len(perfMetrics)),
			zap.Duration("assistant_duration", assistantDuration),
			zap.Duration("parse_duration", parseDuration),
		)

		// 输出调试日志
		logger.Debug("准备创建日志条目",
			zap.String("session_id", sessionID.String()),
			zap.String("interaction_id", interactionID.String()),
			zap.String("question", question),
			zap.String("model_name", modelName),
			zap.String("provider", provider),
			zap.String("base_url", baseURL),
			zap.String("cluster", cluster),
			zap.String("thought", thought),
			zap.Int("thought_length", len(thought)),
			zap.String("final_answer", finalAnswer),
			zap.String("status", status),
			zap.Int("tool_calls_count", len(toolCalls)),
			zap.Int("perf_metrics_count", len(perfMetrics)),
		)

		// 输出工具调用详情
		for i, tool := range toolCalls {
			logger.Debug(fmt.Sprintf("工具调用详情 %d", i),
				zap.String("name", tool.Name),
				zap.String("input", tool.Input),
				zap.String("observation", tool.Observation),
				zap.Int("sequence_num", tool.SequenceNum),
				zap.Duration("duration", tool.Duration),
			)
		}

		// 输出性能指标详情
		var metricNames []string
		for name := range perfMetrics {
			metricNames = append(metricNames, name)
		}
		logger.Debug("所有性能指标名称",
			zap.Strings("metric_names", metricNames),
		)

		for name, duration := range perfMetrics {
			logger.Debug(fmt.Sprintf("性能指标详情: %s", name),
				zap.Duration("duration", duration),
				zap.Int("duration_ms", int(duration.Milliseconds())),
			)
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

		// 输出完整的审计数据
		logger.Info("准备记录审计日志",
			zap.String("interaction_id", interactionID.String()),
			zap.String("session_id", sessionID.String()),
			zap.String("model_name", modelName),
			zap.String("provider", provider),
			zap.String("base_url", baseURL),
			zap.String("thought", thought),
			zap.Int("thought_length", len(thought)),
			zap.Int("tool_calls_count", len(toolCalls)),
			zap.Int("perf_metrics_count", len(perfMetrics)),
		)

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
