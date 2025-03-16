package tools

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"os/exec"
	"strings"
	"time"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

// JQ 执行jq命令处理JSON数据
// 功能特性：
// 1. 支持复杂的jq表达式
// 2. 自动验证JSON数据格式
// 3. 处理管道操作
// 参数：
//   - input: 输入格式为 "JSON数据 | jq表达式"
//
// 返回：
//   - string: jq处理后的结果
//   - error: 处理过程中的错误
func JQ(input string) (string, error) {
	// 获取性能统计工具
	perfStats := utils.GetPerfStats()
	// 开始jq命令执行计时
	defer perfStats.TraceFunc("jq_command")()

	// 记录开始时间
	startTime := time.Now()

	logger.Debug("准备执行 jq 命令",
		zap.String("input", input),
	)

	// 解析输入，分离JSON数据和jq表达式
	parts := strings.Split(input, "|")
	if len(parts) != 2 {
		return "", fmt.Errorf("输入格式错误，应为: JSON数据 | jq表达式")
	}

	jsonData := strings.TrimSpace(parts[0])
	jqExpr := strings.TrimSpace(parts[1])

	// 开始JSON验证计时
	perfStats.StartTimer("jq_json_validation")

	// 验证JSON数据格式是否有效
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(jsonData), &jsonObj); err != nil {
		// 停止JSON验证计时
		validationDuration := perfStats.StopTimer("jq_json_validation")
		logger.Debug("JSON验证失败",
			zap.Error(err),
			zap.Duration("duration", validationDuration),
		)

		return "", fmt.Errorf("无效的JSON数据: %v", err)
	}

	// 停止JSON验证计时
	validationDuration := perfStats.StopTimer("jq_json_validation")
	logger.Debug("JSON验证成功",
		zap.Duration("duration", validationDuration),
	)

	// 开始jq执行计时
	perfStats.StartTimer("jq_execution")

	// 使用管道直接传递数据执行jq命令
	cmd := exec.Command("jq", jqExpr)
	cmd.Stdin = strings.NewReader(jsonData)

	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()

	// 停止jq执行计时
	executionDuration := perfStats.StopTimer("jq_execution")

	// 记录总执行时间
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("jq 命令执行失败",
			zap.Error(err),
			zap.String("output", string(output)),
			zap.Duration("execution_duration", executionDuration),
			zap.Duration("total_duration", duration),
		)

		// 记录失败的命令性能
		perfStats.RecordMetric("jq_command_failed", duration)

		return strings.TrimSpace(string(output)), err
	}

	logger.Debug("jq 命令执行成功",
		zap.String("output", string(output)),
		zap.Duration("execution_duration", executionDuration),
		zap.Duration("total_duration", duration),
	)

	// 记录成功的命令性能
	perfStats.RecordMetric("jq_command_success", duration)

	// 记录jq表达式的复杂度（基于表达式长度和特定操作符的数量）
	complexity := len(jqExpr)
	complexity += strings.Count(jqExpr, "|") * 2
	complexity += strings.Count(jqExpr, "select") * 5
	complexity += strings.Count(jqExpr, "map") * 3

	if complexity > 20 {
		perfStats.RecordMetric("jq_complex_query", duration)
	} else {
		perfStats.RecordMetric("jq_simple_query", duration)
	}

	return strings.TrimSpace(string(output)), nil
}

// processJSONWithJQ 智能处理JSON数据并提取特定字段
// 功能：
// 1. 自动构建jq查询表达式
// 2. 处理复杂的JSON结构
// 参数：
//   - jsonData: 原始JSON数据
//   - query: 要执行的jq查询
//
// 返回：
//   - string: 处理后的结果
//   - error: 处理过程中的错误
func processJSONWithJQ(jsonData string, query string) (string, error) {
	// 获取性能统计工具
	perfStats := utils.GetPerfStats()
	// 开始处理计时
	defer perfStats.TraceFunc("process_json_with_jq")()

	// 构建完整的jq命令输入
	input := fmt.Sprintf("%s | %s", jsonData, query)
	return JQ(input)
}
