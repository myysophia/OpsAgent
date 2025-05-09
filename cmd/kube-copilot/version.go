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
package main

import (
	"fmt"

	"github.com/myysophia/OpsAgent/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
// VERSION is the version of kube-copilot.
// VERSION = "v0.1.8"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of kube-copilot",
	Run: func(cmd *cobra.Command, args []string) {
		// 获取日志记录器
		logger := utils.GetLogger()

		logger.Info("版本信息",
			zap.String("version", VERSION),
		)
		utils.Info(fmt.Sprintf("kube-copilot %s", VERSION))
	},
}
