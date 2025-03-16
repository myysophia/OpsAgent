package main

import (
	"github.com/myysophia/OpsAgent/pkg/utils"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	//// global flags
	//model string
	//maxTokens     int
	//countTokens   bool
	//verbose       bool
	//maxIterations int

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:     "k8s-aiagent",
		Version: VERSION,
		Short:   "Kubernetes Copilot - An AI agent for Kubernetes",
	}
)

// init initializes the command line flags
func init() {
	rootCmd.PersistentFlags().StringVarP(&model, "model", "m", "qwen-max", "qwen model to use")
	rootCmd.PersistentFlags().IntVarP(&maxTokens, "max-tokens", "t", 8192, "Max tokens for the the model")
	rootCmd.PersistentFlags().BoolVarP(&countTokens, "count-tokens", "c", false, "Print tokens count")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().IntVarP(&maxIterations, "max-iterations", "x", 10, "Max iterations for the agent running")

	rootCmd.AddCommand(serverCmd)
}

func main() {
	// 初始化配置
	if err := utils.InitConfig(); err != nil {
		utils.Error("配置文件加载失败，使用默认配置", zap.Error(err))
	}

	// 初始化日志系统
	config := utils.GetConfig()
	logConfig := utils.DefaultLogConfig()

	// 设置日志级别
	level := config.GetString("log.level")
	switch level {
	case "debug":
		logConfig.Level = zapcore.DebugLevel
	case "info":
		logConfig.Level = zapcore.InfoLevel
	case "warn":
		logConfig.Level = zapcore.WarnLevel
	case "error":
		logConfig.Level = zapcore.ErrorLevel
	default:
		logConfig.Level = zapcore.InfoLevel
	}

	// 设置日志输出格式
	if config.GetString("log.format") == "json" {
		logConfig.ColoredOutput = false
	}

	// 设置日志输出位置
	if config.GetString("log.output") != "stdout" {
		logConfig.ConsoleOutput = false
		logConfig.LogDir = config.GetString("log.output")
	}

	// 初始化日志
	if _, err := utils.InitLogger(logConfig); err != nil {
		panic(err)
	}
	defer utils.Sync()

	if err := rootCmd.Execute(); err != nil {
		utils.Fatal("命令执行失败", zap.Error(err))
	}
}
