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

	"github.com/fatih/color"
	"github.com/myysophia/OpsAgent/pkg/kubernetes"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"github.com/myysophia/OpsAgent/pkg/workflows"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// 分析命令的配置参数
var analysisName string      // 资源名称
var analysisNamespace string // 命名空间
var analysisResource string  // 资源类型
// --model gpt-4o --verbose analyze velero-588d776b7b-tpzrg velero pod
func init() {
	// 初始化命令行参数
	analyzeCmd.PersistentFlags().StringVarP(&analysisName, "name", "", "", "Resource name")
	analyzeCmd.PersistentFlags().StringVarP(&analysisNamespace, "namespace", "n", "default", "Resource namespace")
	analyzeCmd.PersistentFlags().StringVarP(&analysisResource, "resource", "r", "pod", "Resource type")
	analyzeCmd.MarkFlagRequired("name")
}

// analyzeCmd 实现 Kubernetes 资源分析功能
// 支持分析 Pod、Service 等资源的配置问题
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze issues for a given resource",
	Run: func(cmd *cobra.Command, args []string) {
		// 获取日志记录器
		logger := utils.GetLogger()

		if analysisName == "" && len(args) > 0 {
			analysisName = args[0]
		}
		if analysisName == "" {
			logger.Error("未提供资源名称")
			utils.Error("请提供一个资源名称")
			return
		}

		logger.Info("开始分析资源",
			zap.String("resource", analysisResource),
			zap.String("namespace", analysisNamespace),
			zap.String("name", analysisName),
		)
		utils.Info(fmt.Sprintf("正在分析 %s %s/%s", analysisResource, analysisNamespace, analysisName))

		manifests, err := kubernetes.GetYaml(analysisResource, analysisName, analysisNamespace)
		if err != nil {
			logger.Error("获取资源清单失败",
				zap.Error(err),
			)
			color.Red(err.Error())
			return
		}

		response, err := workflows.AnalysisFlow(model, manifests, verbose)
		if err != nil {
			logger.Error("分析资源失败",
				zap.Error(err),
			)
			color.Red(err.Error())
			return
		}

		utils.RenderMarkdown(response)
	},
}
