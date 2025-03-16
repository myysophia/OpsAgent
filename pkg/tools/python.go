/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package tools

import (
	"fmt"
	"github.com/fatih/color"
	"os/exec"
	"strings"
	"go.uber.org/zap"
)

// PythonREPL runs the given Python script and returns the output.
func PythonREPL(script string) (string, error) {
	logger.Debug("准备执行 Python 脚本",
		zap.String("script", script),
	)

	escapedScript := strings.ReplaceAll(script, "\"", "\\\"")
	cmdStr := fmt.Sprintf("cd ~/k8s/python-cli && source k8s-env/bin/activate && python3 -c \"%s\"", escapedScript)
	cmd := exec.Command("bash", "-c", cmdStr)
	
	logger.Debug("构建命令",
		zap.String("command", cmdStr),
	)
	color.Cyan("Python scripts is: %s", cmdStr)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Python 脚本执行失败",
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return strings.TrimSpace(string(output)), err
	}

	logger.Debug("Python 脚本执行成功",
		zap.String("output", string(output)),
	)
	return strings.TrimSpace(string(output)), nil
}

// SwitchK8sEnv 切换到指定的 Kubernetes 环境
func SwitchK8sEnv(envName string) error {
	logger.Info("切换 Kubernetes 环境",
		zap.String("environment", envName),
	)

	cmd := exec.Command("k8s-env", "switch", envName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("环境切换失败",
			zap.String("environment", envName),
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return fmt.Errorf("failed to switch to %s: %s, output: %s", envName, err, output)
	}

	logger.Info("环境切换成功",
		zap.String("environment", envName),
		zap.String("output", string(output)),
	)
	fmt.Printf("Switched to %s: %s\n", envName, output)
	return nil
}
