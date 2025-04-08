package tools

import (
	"go.uber.org/zap"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

var (
	logger *zap.Logger
	// 当前使用的 Kubernetes 上下文，由 handlers 包设置
	currentKubeContext string
)

func init() {
	// 使用新的日志工具包获取日志记录器
	logger = utils.GetLogger()
}

// SetCurrentKubeContext 设置当前使用的 Kubernetes 上下文
func SetCurrentKubeContext(context string) {
	currentKubeContext = context
	logger.Info("Set current Kubernetes context", zap.String("context", context))
}

// Tool 是一个接受输入并返回输出的函数类型
type Tool func(input string) (string, error)

// function call ，可以理解这里是hook点，可以在这里添加自己的工具
var CopilotTools = map[string]Tool{
	"search":  GoogleSearch,
	"python":  PythonREPL,
	"trivy":   Trivy,
	"kubectl": Kubectl,
	"jq":      JQ,
}

// ToolPrompt 定义了与 LLM 交互的 JSON 格式
type ToolPrompt struct {
	Question string   `json:"question"` // 用户输入的问题
	Thought  string   `json:"thought"`  // AI 的思考过程
	Action   struct { // 需要执行的动作
		Name  string `json:"name"`  // 工具名称
		Input string `json:"input"` // 工具输入
	} `json:"action"`
	Observation string `json:"observation"`  // 工具执行结果
	FinalAnswer string `json:"final_answer"` // 最终答案
}
