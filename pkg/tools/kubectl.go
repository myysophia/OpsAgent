package tools

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

// executeShellCommand 执行shell命令并返回输出
// 参数：
//   - command: 要执行的shell命令
//
// 返回：
//   - string: 命令执行的输出
//   - error: 执行过程中的错误
func executeShellCommand(command string) (string, error) {
	// 获取性能统计工具
	perfStats := utils.GetPerfStats()
	// 开始shell命令执行计时
	defer perfStats.TraceFunc("shell_command_execute")()

	logger.Debug("执行shell命令",
		zap.String("command", command),
	)

	// 使用bash执行命令
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("shell命令执行失败",
			zap.String("command", command),
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return string(output), err
	}

	logger.Debug("shell命令执行成功",
		zap.String("command", command),
		zap.String("output", string(output)),
	)
	return string(output), nil
}

// Kubectl 执行kubectl命令并返回输出
// 功能特性：
// 1. 自动添加kubectl前缀（如果缺少）
// 2. 处理命令执行错误并提供详细日志
// 3. 智能判断命令类型并选择合适的执行方式
// 参数：
//   - command: kubectl命令（可以包含或不包含"kubectl"前缀）
//
// 返回：
//   - string: 命令执行的输出
//   - error: 执行过程中的错误
func Kubectl(command string) (string, error) {
	// 获取性能统计工具
	perfStats := utils.GetPerfStats()
	// 开始kubectl命令执行计时
	defer perfStats.TraceFunc("kubectl_command")()

	// 记录开始时间
	startTime := time.Now()

	logger.Debug("执行kubectl命令",
		zap.String("command", command),
	)

	// 确保命令以kubectl开头
	if !strings.HasPrefix(command, "kubectl") {
		command = "kubectl " + command
	}

	// 添加kubeconfig参数
	if !strings.Contains(command, "--kubeconfig") {
		command = strings.Replace(command, "kubectl", "kubectl --kubeconfig=./config", 1)
	}

	// 如果有上下文信息，并且命令中没有指定 --context 参数，则添加上下文参数
	if currentKubeContext != "" && !strings.Contains(command, "--context") {
		// 将 --context 参数添加在 kubectl 命令之后
		parts := strings.SplitN(command, " ", 2)
		if len(parts) == 2 {
			// 如果命令包含空格，将 --context 参数添加在 kubectl 之后
			command = parts[0] + " --context=\"" + currentKubeContext + "\" " + parts[1]
		} else {
			// 如果命令只有 kubectl，直接添加 --context 参数
			command = command + " --context=\"" + currentKubeContext + "\""
		}
		logger.Debug("Added context to kubectl command",
			zap.String("context", currentKubeContext),
			zap.String("command", command),
		)
	}

	// 执行命令
	output, err := executeShellCommand(command)

	// 记录执行时间
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("kubectl命令执行失败",
			zap.String("command", command),
			zap.Error(err),
			zap.String("output", output),
			zap.Duration("duration", duration),
		)

		// 记录失败的命令性能
		perfStats.RecordMetric("kubectl_command_failed", duration)

		// 如果输出包含特定错误信息，提供更友好的错误提示
		if strings.Contains(output, "not found") {
			return output, err
		}
		if strings.Contains(output, "forbidden") || strings.Contains(output, "Forbidden") {
			return output, err
		}
		if strings.Contains(output, "Unable to connect to the server") {
			return output, err
		}

		return output, err
	}

	logger.Debug("kubectl 命令执行成功",
		zap.String("command", command),
		zap.Duration("duration", duration),
	)

	// 记录成功的命令性能
	perfStats.RecordMetric("kubectl_command_success", duration)

	// 根据命令类型记录更详细的性能指标
	if strings.Contains(command, "get") {
		perfStats.RecordMetric("kubectl_get", duration)
	} else if strings.Contains(command, "describe") {
		perfStats.RecordMetric("kubectl_describe", duration)
	} else if strings.Contains(command, "logs") {
		perfStats.RecordMetric("kubectl_logs", duration)
	} else if strings.Contains(command, "exec") {
		perfStats.RecordMetric("kubectl_exec", duration)
	} else if strings.Contains(command, "apply") {
		perfStats.RecordMetric("kubectl_apply", duration)
	} else if strings.Contains(command, "delete") {
		perfStats.RecordMetric("kubectl_delete", duration)
	}

	// 过滤掉无关的错误信息
	output = filterKubectlOutput(output)

	return output, nil
}

// filterKubectlOutput 过滤kubectl输出中的无关错误信息
// 参数：
//   - output: 原始输出内容
//
// 返回：
//   - string: 过滤后的输出内容
func filterKubectlOutput(output string) string {
	// 按行分割输出
	lines := strings.Split(output, "\n")
	var filteredLines []string

	// 需要过滤的错误信息模式
	errorPatterns := []string{
		"metrics.k8s.io/v1beta1: the server is currently unable to handle the request",
		"external.metrics.k8s.io/v1beta1: the server is currently unable to handle the request",
		"memcache.go",                    // 过滤掉所有包含memcache.go的行
		"couldn't get resource list for", // 已有的过滤条件
	}

	// 遍历每一行，过滤掉匹配模式的行
	for _, line := range lines {
		shouldKeep := true

		for _, pattern := range errorPatterns {
			if strings.Contains(line, pattern) {
				shouldKeep = false
				break
			}
		}

		// 过滤掉常见的k8s错误日志格式（如E0307开头的错误）
		if len(line) > 0 && line[0] == 'E' && len(line) > 5 {
			// 匹配类似E0307这样的错误日志前缀
			if _, err := strconv.Atoi(line[1:5]); err == nil {
				shouldKeep = false
			}
		}

		if shouldKeep {
			filteredLines = append(filteredLines, line)
		}
	}

	// 将过滤后的行重新连接为字符串
	filteredOutput := strings.Join(filteredLines, "\n")

	// 如果过滤后内容与原内容不同，记录日志
	if filteredOutput != output {
		logger.Debug("过滤了kubectl输出中的错误信息",
			zap.String("original_length", fmt.Sprintf("%d", len(output))),
			zap.String("filtered_length", fmt.Sprintf("%d", len(filteredOutput))),
		)
	}

	return filteredOutput
}
