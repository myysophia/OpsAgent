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
	"bufio"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/myysophia/OpsAgent/pkg/kubernetes"
	"github.com/myysophia/OpsAgent/pkg/utils"
	"github.com/myysophia/OpsAgent/pkg/workflows"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var generatePrompt string

func init() {
	generateCmd.PersistentFlags().StringVarP(&generatePrompt, "prompt", "p", "", "Prompts to generate Kubernetes manifests")
	generateCmd.MarkFlagRequired("prompt")
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Kubernetes manifests",
	Run: func(cmd *cobra.Command, args []string) {
		// 获取日志记录器
		logger := utils.GetLogger()

		if generatePrompt == "" {
			logger.Error("未提供生成提示")
			color.Red("Please specify a prompt")
			return
		}

		logger.Info("开始生成 Kubernetes 清单",
			zap.String("prompt", generatePrompt),
			zap.String("model", model),
		)

		response, err := workflows.GeneratorFlow(model, generatePrompt, verbose)
		if err != nil {
			logger.Error("生成清单失败",
				zap.Error(err),
			)
			color.Red(err.Error())
			return
		}

		// Extract the yaml from the response
		yaml := response
		if strings.Contains(response, "```") {
			yaml = utils.ExtractYaml(response)
		}

		logger.Info("生成清单成功",
			zap.Int("yaml_length", len(yaml)),
		)

		utils.Info("生成的清单:")
		color.New(color.FgGreen).Printf("%s\n\n", yaml)

		// apply the yaml to kubernetes cluster
		color.New(color.FgRed).Printf("是否要将生成的清单应用到集群中？(y/n)")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			approve := scanner.Text()
			if strings.ToLower(approve) != "y" && strings.ToLower(approve) != "yes" {
				break
			}

			if err := kubernetes.ApplyYaml(yaml); err != nil {
				color.Red(err.Error())
				return
			}

			color.New(color.FgGreen).Printf("Applied the generated manifests to cluster successfully!")
			break
		}
	},
}
