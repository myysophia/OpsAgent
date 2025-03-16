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
	"os/exec"
	"strings"
	"go.uber.org/zap"
)

// Trivy runs trivy against the image and returns the output
func Trivy(image string) (string, error) {
	logger.Debug("准备执行 Trivy 扫描",
		zap.String("raw_image", image),
	)

	image = strings.TrimSpace(image)
	if strings.HasPrefix(image, "image ") {
		image = strings.TrimPrefix(image, "image ")
	}

	logger.Debug("构建命令",
		zap.String("image", image),
	)

	cmd := exec.Command("trivy", "image", image, "--scanners", "vuln")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Trivy 扫描失败",
			zap.String("image", image),
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return strings.TrimSpace(string(output)), err
	}

	logger.Info("Trivy 扫描完成",
		zap.String("image", image),
		zap.String("output", string(output)),
	)
	return strings.TrimSpace(string(output)), nil
}
