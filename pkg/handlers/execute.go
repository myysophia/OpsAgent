package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"

	"github.com/myysophia/OpsAgent/pkg/assistants"
	"github.com/myysophia/OpsAgent/pkg/audit"
	"github.com/myysophia/OpsAgent/pkg/tools"
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
	Thought     string `json:"thought"`
}

const executeSystemPrompt_cn = `您是Kubernetes和云原生网络的技术专家，您的任务是遵循链式思维方法，确保彻底性和准确性，同时遵守约束。
有一个集群上下文对照表，用于帮助用户查询不同集群的资源。对照表如下：
| 集群上下文名称 (context) | namespace | 用户说明 |
| --- | --- | --- |
| eks-au | au | 澳洲/澳洲节点/au/AU/Au |
| ask-cn | cn | 中国/中国节点/cn/Cn/CN |
| ask-eu | eu | 欧洲/欧洲节点/法兰克福/EU |
| ask-uat | uat | uat环境/测试环境/uat |
| eks-in | in | 印度/印度节点/IN/In/in |
| eks-us | us | 美国/美国节点/us/Us/US |
| eks-ems-eu-new | ems-eu | 储能欧洲/EMS EU集群/ems eu/储能EU |
| cce-ems-plus-2 | ems-plus-mapai | 储能中国节点/ems cn 集群 |
| ems-uat-new-1 | ems-uat | 储能uat节点/储能测试环境/ems uat环境 |
注意
1. 当用户询问集群相关问题时，请优先根据此表提供准确的集群上下文名称和命名空间映射，添加到要执行的tool中,tool中必须要指定--context和--namespace参数。
2. 当用户的问题中包含多个context时，调用tools时应该逐个context查询进行输出。
	例如用户问题: 储能中国节点和ems uat环境的pod镜像版本、cpu/memory request、cpu/memory limit?
    对应的tool: 'kubectl --kubeconfig=./config --context=cce-ems-plus-2 --namespace=ems-plus-mapai get pods -o custom-columns='NAME:.metadata.name,IMAGE:.spec.containers[*].image,CPU_REQ:.spec.containers[*].resources.requests.cpu,MEM_REQ:.spec.containers[*].resources.requests.memory,CPU_LIMIT:.spec.containers[*].resources.limits.cpu,MEM_LIMIT:.spec.containers[*].resources.limits.memory' --no-headers && kubectl --kubeconfig=./config --context=ems-uat-new-1 --namespace=ems-uat get pods -o custom-columns='NAME:.metadata.name,IMAGE:.spec.containers[*].image,CPU_REQ:.spec.containers[*].resources.requests.cpu,MEM_REQ:.spec.containers[*].resources.requests.memory,CPU_LIMIT:.spec.containers[*].resources.limits.cpu,MEM_LIMIT:.spec.containers[*].resources.limits.memory' --no-headers'


有一个服务名称对照表，用于帮助用户查询不同服务的关键字和资源名称。对照表如下：
| 中文名称       | 英文关键字/别名                            | Kubernetes 资源名称                   |
|------------|-------------------------------------|---------------------------------------|
| 账户         | account                             | vnnox-middle-account-manage           |
| 设备网关       | device-gateway/device gateway       | vnnox-middle-device-gateway           |
| 设备管理平台     | device-management/device management | vnnox-middle-device-management        |
| 内部网关       | 网关/gateway-service/device service   | vnnox-middle-gateway-service          |
| 储能后端服务     | 储能后端/energy-cloud/energy cloud      | energy-cloud-service                  |
| 工商储前端      | ems front                           | ems-front                             |
| 家储后端服务     | 户储后端/energy management              | energy-management-service             |
| 储能兼容服务     | 兼容服务/energy compatibility           | energy-compatibility                  |
| 储能数字孪生       | digital twin                        | digital-twin-platform                 |
| 储能低代码后端      | lowcode service/低代码 后端              | cloud-lowcode-service                 |
| 音量微服务/音量服务/音量      | volume              | vnnox-volume                 |
| 亮度微服务/亮度服务/亮度      | brightness/bright              |     vnnox-brightness             |
| 储能低代码后端      | lowcode service/低代码 后端              | cloud-lowcode-service                 |
| 储能低代码后端      | lowcode service/低代码 后端              | cloud-lowcode-service                 |
| mysql数据库   | mysql/vnnox mysql                   | vnnox-mysql                           |
| redis数据库   | redis/vnnox redis                   | vnnox-redis                           |
| mongo数据库   | mongo/vnnox mongo                   | vnnox-mongo                           |
| kong proxy | external gateway                    | kong-proxy                            |
| iotdb dn   | iotdb-datanode/iotdb datanode       | iotdb-datanode                        |

当用户询问服务相关问题时，请优先根据此表提供准确的资源名称或关键字映射,如果没有匹配到，请使用模糊匹配。
使用此表时，请遵循以下原则：
- 如果用户询问某个服务如何连接，优先使用查询 LoadBalancer/ExternalName 类型的svc。
可用工具：
- kubectl：用于执行 Kubernetes 命令。必须使用正确语法（例如 'kubectl get pods' 而非 'kubectl get pod'），避免使用 -o json/yaml 全量输出。
- python：用于复杂逻辑或调用 Kubernetes Python SDK。输入：Python 脚本，输出：通过 print(...) 返回。
- trivy：用于扫描镜像漏洞。输入：镜像名称，输出：漏洞报告。
- jq：用于处理 JSON 数据。输入：有效的 jq 表达式，始终使用 'test()' 进行名称匹配。
- iotdbtools: 当用户要求把文件直接传到oss时，调用这个工具将pod中的文件直接上传到oss,iotdbtools位于当前目录
	这是一个iotdbtools的例子: "./iotdbtools backup --config ./config --namespace ems-eu --pods vnnox-middle-oauth-f88d9c78c-247tf  --datadir /app/logs --containers vnnox-middle-oauth --verbose 2",调用工具是动态将用户会给你namespace、pod名称、数据目录和容器名称传入,使用./iotdbtools backup 时--config使用./config
您采取的步骤如下：
1. 问题识别：清楚定义问题，描述目标。
2. 诊断命令：根据问题选择工具
3. 输出解释：分析工具输出，描述结果。如果输出为空，必须明确告知用户未找到相关信息。
4. 故障排除策略：根据输出制定策略。
5. 可行解决方案：提出解决方案，确保命令准确。

严格约束：
- 避免使用 -o json/yaml 全量输出，优先使用 jsonpath 、--go-template、 custom-columns 进行查询,注意用户输入都是模糊的,筛选时需要模糊匹配。
- 使用 --no-headers 选项减少不必要的输出。
- jq 表达式中，名称匹配必须使用 'test()'，避免使用 '=='。
- 命令参数涉及特殊字符（如 []、()、"）时，优先使用单引号 ' 包裹，避免 Shell 解析错误。
- 避免在 zsh 中使用未转义的双引号（如 \"），防止触发模式匹配。
- 当使用awk时使用单引号（如 '{print $1}'），避免双引号转义导致语法错误。
- 当用户问题中包含"镜像版本、版本号、分支"时，优先使用kubectl get pods -o custom-columns='NAME:.metadata.name,IMAGE:.spec.containers[*].image' | grep '用户问题中的服务名称'。
- 当用户问题中包含"域名、访问地址"时，优先查询ingress 资源进行匹配。
- kubectl命令不指定namespace时，优先使用默认的namespace查询
- 不要使用--field-selector spec.nodeName=xxx进行资源筛选查询，总是认为用户的问题是模糊的。
- 使用kubectl命令不要使用 -l 对label进行资源筛选查询
- 当调用tools是kubectl 时,请务必使用kubectl --kubeconfig=./config 来避免没有权限的问题.
极其重要：输出格式约束
您的**完整**响应**必须**是一个**单一、有效**的 JSON 对象。
**绝对不能**在 JSON 对象之前或之后包含任何额外的文本、解释、问候语或注释。
**绝对不能**使用 Markdown 代码块将整个 JSON 对象包裹起来。

重要提示：始终使用以下 JSON 格式返回响应：
{
  "question": "<用户的输入问题>",
  "thought": "<您的分析和思考过程>",
  "action": {
    "name": "<工具名称>",
    "input": "<工具输入>"
  },
  "observation": "",
  "final_answer": "<最终答案。只有在完成所有流程且无需采取任何行动后才能确定。此字段的*值*可以使用 Markdown 格式（如果需要格式化输出，例如列表或代码片段），但整个响应仍然必须是纯粹的 JSON 对象。>"
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
6. 当用户问题中出现"删除、重启、delete、patch、drop"等关键字时，必须委婉拒绝用户没有权限执行这些操作。
7. 当用户提问"你是谁？你可以干什么的时候？你可以做什么？"时，请委婉告诉用户你可以干什么？
当结果为空时，应该这样处理：
1. 首先尝试使用更宽松的查询,但是总应该避免全量输出(-ojson/yaml)，例如使用 jsonpath 或 custom-columns 来获取特定字段。
2. 如果仍然为空，在 final_answer 中提供：
   - 当前查询条件说明
   - 可能的原因（如命名空间问题、权限问题等）
   - 建议的解决方案
   - 是否需要用户提供更多信息
目标：
在 Kubernetes 和云原生网络领域内识别问题根本原因，提供清晰、可行的解决方案，同时保持诊断和故障排除的运营约束。`

const (
	defaultMaxIterations = 5
)

// 存储当前使用的 Kubernetes 上下文名称
var currentKubeContext string

// getContextFromRAG 调用RAG接口获取适合的Kubernetes上下文
func getContextFromRAG(query string) error {
	// 获取 logger
	logger := utils.GetLogger()
	// 获取性能统计工具
	perfStats := utils.GetPerfStats()
	// 开始整体执行计时
	defer perfStats.TraceFunc("getContextFromRAG_total")()

	// 构建请求体
	reqBody := map[string]interface{}{
		"query":      query,
		"parameters": map[string]interface{}{},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 发送请求
	resp, err := http.Post("http://localhost:8080/api/v1/switch-context", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	var result struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if result.Code != 200 {
		return fmt.Errorf("get context from RAG failed: %s", result.Message)
	}

	// 输出原始响应信息以便调试
	logger.Debug("Raw response data",
		zap.Any("data", result.Data),
		zap.String("data_type", fmt.Sprintf("%T", result.Data)),
	)

	// 定义有效的context列表
	validContexts := map[string]bool{
		"eks-au":         true,
		"ask-cn":         true,
		"ask-eu":         true,
		"ask-ua":         true,
		"eks-in":         true,
		"eks-us":         true,
		"eks-ems-eu-new": true,
		"cce-ems-plus-2": true,
		"ems-uat-new-1":  true,
	}

	// 尝试解析响应中的上下文信息
	if dataMap, ok := result.Data.(map[string]interface{}); ok {
		logger.Debug("Response data is a map", zap.Any("data_map", dataMap))

		// 尝试直接从数据中提取 context_name
		if contextName, ok := dataMap["context_name"].(string); ok && contextName != "" {
			// 验证context是否在有效列表中
			if !validContexts[contextName] {
				return fmt.Errorf("invalid context: %s, please specify a valid context from: eks-au, ask-cn, ask-eu, ask-ua, eks-in, eks-us, eks-ems-eu-new, cce-ems-plus-2, ems-uat-new-1", contextName)
			}
			// 直接使用 context_name
			currentKubeContext = contextName
			tools.SetCurrentKubeContext(contextName)
			logger.Info("Set Kubernetes context directly", zap.String("context", contextName))
		} else {
			// 尝试从旧格式中提取上下文信息
			if contextSwitch, ok := dataMap["context_switch"].(map[string]interface{}); ok {
				logger.Debug("Found context_switch in data", zap.Any("context_switch", contextSwitch))

				// 尝试从 command 中提取上下文名称
				if cmd, ok := contextSwitch["command"].(string); ok {
					logger.Debug("Found command in context_switch", zap.String("command", cmd))

					// 直接使用命令作为上下文名称
					contextName := cmd

					if contextName != "" {
						// 验证context是否在有效列表中
						if !validContexts[contextName] {
							return fmt.Errorf("invalid context: %s, please specify a valid context from: eks-au, ask-cn, ask-eu, ask-ua, eks-in, eks-us, eks-ems-eu-new, cce-ems-plus-2, ems-uat-new-1", contextName)
						}
						// 设置当前使用的上下文
						currentKubeContext = contextName
						// 设置 tools 包中的上下文变量
						tools.SetCurrentKubeContext(contextName)
						logger.Info("Set Kubernetes context from command",
							zap.String("context", contextName),
							zap.String("command", cmd),
						)
					}
				}
			}
		}
	} else if dataStr, ok := result.Data.(string); ok {
		// 如果 Data 是字符串，尝试解析为 JSON
		var contextData struct {
			KubecontextKnowledge []struct {
				ContextName      string `json:"context_name"`
				RegionIdentifier string `json:"region_identifier"`
			} `json:"kubecontext_knowledge"`
		}

		if err := json.Unmarshal([]byte(dataStr), &contextData); err != nil {
			logger.Warn("Failed to parse kubecontext_knowledge data",
				zap.Error(err),
				zap.String("data", dataStr),
			)
		} else if len(contextData.KubecontextKnowledge) > 0 {
			// 成功解析到了上下文知识
			context := contextData.KubecontextKnowledge[0]

			// 验证context是否在有效列表中
			if !validContexts[context.ContextName] {
				return fmt.Errorf("invalid context: %s, please specify a valid context from: eks-au, ask-cn, ask-eu, ask-ua, eks-in, eks-us, eks-ems-eu-new, cce-ems-plus-2, ems-uat-new-1", context.ContextName)
			}

			// 设置当前使用的上下文
			currentKubeContext = context.ContextName
			// 设置 tools 包中的上下文变量
			tools.SetCurrentKubeContext(context.ContextName)

			logger.Info("Set Kubernetes context from knowledge",
				zap.String("context", context.ContextName),
				zap.String("region", context.RegionIdentifier),
			)
		}
	}

	logger.Debug("Context information retrieved successfully",
		zap.String("current_context", currentKubeContext),
	)
	return nil
}

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

	// 获取适合的Kubernetes上下文
	// logger.Info("开始获取Kubernetes上下文", zap.String("args", req.Args))
	//err := getContextFromRAG(req.Args)
	//if err != nil {
	//	logger.Error("RAG Flow 服务异常!", zap.Error(err))
	//	c.JSON(http.StatusInternalServerError, gin.H{
	//		"error":  fmt.Sprintf("RAG Flow 服务异常! %v", err),
	//		"status": "error",
	//		"message": fmt.Sprintf("无法获取你要查询的集群是哪一个？您可以这样问：\n\n"+
	//			"中国节点%s\n"+
	//			"储能中国节点%s\n"+
	//			".......\n"+
	//			"您是想查询哪个节点的信息呢？",
	//			req.Args, req.Args),
	//	})
	//	return
	//}
	//
	//// 检查上下文是否设置成功
	//logger.Info("获取Kubernetes上下文成功",
	//	zap.String("current_context", currentKubeContext),
	//)
	//
	//// 如果上下文为空，设置一个默认值
	//if currentKubeContext == "" {
	//	currentKubeContext = "ask-cn" // 设置一个默认值
	//	tools.SetCurrentKubeContext(currentKubeContext)
	//	logger.Warn("上下文为空，设置默认值", zap.String("default_context", currentKubeContext))
	//}

	// 确定使用的模型
	executeModel := req.CurrentModel
	if executeModel == "" {
		executeModel = "gpt-4o"
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
	response, chatHistory, err := assistants.AssistantWithConfig(executeModel, messages, 8192, true, true, defaultMaxIterations, apiKey, req.BaseUrl)

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

	// 获取日志记录器
	logger = utils.GetLogger().Named("execute-handler")

	// 提取工具使用历史
	var toolsHistory []ToolHistory
	var auditToolCalls []audit.ToolCall

	logger.Debug("开始提取工具使用历史",
		zap.Int("chat_history_length", len(chatHistory)),
	)

	for i := 0; i < len(chatHistory); i++ {
		if chatHistory[i].Role == openai.ChatMessageRoleUser && i > 0 {
			logger.Debug(fmt.Sprintf("处理第 %d 条聊天记录", i),
				zap.String("role", string(chatHistory[i].Role)),
				zap.String("content", chatHistory[i].Content),
			)

			var toolPrompt map[string]interface{}
			if err := json.Unmarshal([]byte(chatHistory[i].Content), &toolPrompt); err == nil {
				logger.Debug("解析工具提示成功",
					zap.Any("tool_prompt", toolPrompt),
				)

				if action, ok := toolPrompt["action"].(map[string]interface{}); ok {
					logger.Debug("获取到动作",
						zap.Any("action", action),
					)

					name, _ := action["name"].(string)
					input, _ := action["input"].(string)
					observation, _ := toolPrompt["observation"].(string)
					thought, _ := toolPrompt["thought"].(string)

					if name != "" && input != "" {
						logger.Debug("添加工具调用",
							zap.String("name", name),
							zap.String("input", input),
							zap.String("observation", observation),
							zap.String("thought", thought),
						)

						// 添加到工具历史
						toolsHistory = append(toolsHistory, ToolHistory{
							Name:        name,
							Input:       input,
							Observation: observation,
							Thought:     thought,
						})

						// 添加到审计工具调用
						logger.Debug("添加工具调用到审计记录",
							zap.String("name", name),
							zap.String("input", input),
							zap.String("observation", observation),
							zap.String("thought", thought),
							zap.Int("sequence_num", len(auditToolCalls)),
						)

						auditToolCalls = append(auditToolCalls, audit.ToolCall{
							Name:        name,
							Input:       input,
							Observation: observation,
							SequenceNum: len(auditToolCalls),
							Duration:    0, // 暂时无法获取工具调用耗时
						})
					}
				}
			}
		}
	}

	logger.Debug("工具使用历史提取完成",
		zap.Int("tools_history_count", len(toolsHistory)),
		zap.Int("audit_tool_calls_count", len(auditToolCalls)),
	)

	// 确保工具调用不为空
	if len(auditToolCalls) == 0 {
		// 添加一个虚拟的工具调用，确保表不为空
		auditToolCalls = append(auditToolCalls, audit.ToolCall{
			Name:        "no-tool-used",
			Input:       "No tools were used in this interaction",
			Observation: "No observation available",
			SequenceNum: 0,
			Duration:    0,
		})
		logger.Warn("工具调用为空，添加虚拟工具调用",
			zap.Int("audit_tool_calls_count", len(auditToolCalls)),
		)
	}

	// 输出所有工具调用详情
	for i, tool := range auditToolCalls {
		logger.Debug(fmt.Sprintf("工具调用详情 %d", i),
			zap.String("name", tool.Name),
			zap.String("input", tool.Input),
			zap.String("observation", tool.Observation),
			zap.Int("sequence_num", tool.SequenceNum),
			zap.Duration("duration", tool.Duration),
		)
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

			// 将审计数据存入上下文
			c.Set("audit_data", responseData)

			// 记录响应内容
			logger.Debug("返回响应内容",
				zap.Any("response_data", responseData),
				zap.Bool("show_thought", showThought),
				zap.String("thought", thought),
			)

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

			// 将审计数据存入上下文
			c.Set("audit_data", responseData)

			// 记录响应内容
			logger.Debug("返回响应内容",
				zap.Any("response_data", responseData),
				zap.Bool("show_thought", showThought),
				zap.String("thought", aiResp.Thought),
			)

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

				// 将审计数据存入上下文
				c.Set("audit_data", responseData)

				// 记录响应内容
				logger.Debug("返回响应内容",
					zap.Any("response_data", responseData),
					zap.Bool("show_thought", showThought),
					zap.String("thought", thought),
				)

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

		// 将审计数据存入上下文
		c.Set("audit_data", responseData)

		// 记录响应内容
		logger.Debug("返回响应内容",
			zap.Any("response_data", responseData),
			zap.Bool("show_thought", showThought),
			zap.String("thought", aiResp.Thought),
		)

		c.JSON(http.StatusOK, responseData)
		return
	}

	parseDuration := perfStats.StopTimer("execute_response_parse")
	logger.Debug("响应解析完成（标准格式）",
		zap.Duration("duration", parseDuration),
	)

	// 获取性能指标
	perfMetrics := perfStats.GetTimers()
	_ = perfMetrics // 使用_来避免未使用变量的警告

	// 获取会话和交互ID
	var sessionID, interactionID uuid.UUID
	if v, exists := c.Get("session_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			sessionID = id
		}
	}
	if sessionID == uuid.Nil {
		sessionID = uuid.New()
	}
	if v, exists := c.Get("interaction_id"); exists {
		if id, ok := v.(uuid.UUID); ok {
			interactionID = id
		}
	}
	if interactionID == uuid.Nil {
		interactionID = uuid.New()
	}

	if aiResp.FinalAnswer != "" {
		responseData := gin.H{
			"message": aiResp.FinalAnswer,
			"status":  "success",
		}

		// 根据showThought配置决定是否返回工具历史
		if showThought {
			responseData["tools_history"] = toolsHistory
		}

		// 记录响应内容
		logger.Debug("返回响应内容",
			zap.Any("response_data", responseData),
			zap.Bool("show_thought", showThought),
		)

		c.JSON(http.StatusOK, responseData)
	} else {
		responseData := gin.H{
			"message": "指令正在执行中，请稍候...",
			"status":  "processing",
		}

		// 根据showThought配置决定是否返回工具历史
		if showThought {
			responseData["tools_history"] = toolsHistory
		}

		// 记录响应内容
		logger.Debug("返回响应内容",
			zap.Any("response_data", responseData),
			zap.Bool("show_thought", showThought),
		)

		c.JSON(http.StatusOK, responseData)
	}
}
