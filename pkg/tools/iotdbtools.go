package tools

import (
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// IotDBTools 处理IoTDB相关的工具操作
func IotDBTools(input string) (string, error) {
	logger.Info("Executing IotDBTools", zap.String("input", input))

	// 解析输入参数
	args := strings.Fields(input)
	if len(args) < 2 {
		return "", fmt.Errorf("invalid input format. Usage: iotdbtools <command> [options]")
	}

	// 构建命令
	cmd := exec.Command("iotdbtools", args[1:]...)

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to execute iotdbtools",
			zap.Error(err),
			zap.String("output", string(output)))
		return "", fmt.Errorf("failed to execute iotdbtools: %v, output: %s", err, string(output))
	}

	return string(output), nil
}
