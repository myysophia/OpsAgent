package main

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/myysophia/OpsAgent/pkg/api"
	"github.com/myysophia/OpsAgent/pkg/utils"
)

var (
	// API server flags
	port        int
	jwtKey      string
	logger      *zap.Logger
	showThought bool

	// Execute flags (从 execute.go 同步)
	maxTokens     = 8192
	countTokens   = true
	verbose       = true
	maxIterations = 10
)

const (
	VERSION          = "v1.0.2"
	DEFAULT_USERNAME = "admin"
	DEFAULT_PASSWORD = "novastar"
)

// JWT claims structure
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// initLogger 初始化 Zap 日志配置
func initLogger() {
	// 使用新的日志工具包初始化日志
	logConfig := utils.DefaultLogConfig()
	// 设置日志级别为 Debug
	logConfig.Level = zapcore.DebugLevel

	var err error
	logger, err = utils.InitLogger(logConfig)
	if err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}

	// 初始化性能统计工具
	perfStats := utils.GetPerfStats()
	perfStats.SetLogger(logger)
	perfStats.SetEnableLogging(true)

	logger.Info("日志系统初始化完成",
		zap.String("log_dir", logConfig.LogDir),
		zap.String("log_file", logConfig.Filename),
		zap.Int("max_size_mb", logConfig.MaxSize),
		zap.Int("max_backups", logConfig.MaxBackups),
		zap.Int("max_age_days", logConfig.MaxAge),
	)
}

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the API server",
	Run: func(cmd *cobra.Command, args []string) {
		// 初始化日志
		initLogger()
		defer logger.Sync()

		logger.Info("启动服务器",
			zap.Int("port", port),
			zap.Bool("show-thought", showThought),
		)

		// 验证必要参数
		if jwtKey == "" {
			logger.Fatal("缺少必要参数: jwt-key")
		}

		// 设置全局变量
		utils.SetGlobalVar("jwtKey", []byte(jwtKey))
		utils.SetGlobalVar("showThought", showThought)
		utils.SetGlobalVar("logger", logger)

		// 使用pkg/api/router.go中的Router函数
		r := api.Router()

		addr := fmt.Sprintf(":%d", port)
		logger.Info("服务器开始监听",
			zap.String("address", addr),
		)

		if err := r.Run(addr); err != nil {
			logger.Fatal("服务器启动失败",
				zap.Error(err),
			)
		}
	},
}

func init() {
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run the server on")
	serverCmd.Flags().StringVar(&jwtKey, "jwt-key", "", "Key for signing JWT tokens")
	serverCmd.Flags().BoolVar(&showThought, "show-thought", false, "Whether to show LLM's thought process in API responses")
	serverCmd.MarkFlagRequired("jwt-key")
	rootCmd.AddCommand(serverCmd)
}
