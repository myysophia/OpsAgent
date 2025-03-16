package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// 全局日志实例
	globalLogger *zap.Logger
	// 确保只初始化一次
	loggerOnce sync.Once
	// 默认日志目录
	defaultLogDir = "logs"
	// 当前日志文件名
	currentLogFile string
	// 上次日志轮转时间
	lastRotateDate time.Time
	// 日志轮转锁
	rotateMutex sync.Mutex
)

// LogConfig 日志配置
type LogConfig struct {
	// 日志级别
	Level zapcore.Level
	// 日志目录
	LogDir string
	// 日志文件名
	Filename string
	// 单个日志文件最大大小，单位MB
	MaxSize int
	// 保留的旧日志文件最大数量
	MaxBackups int
	// 保留的日志文件最大天数
	MaxAge int
	// 是否压缩旧日志文件
	Compress bool
	// 是否在控制台输出
	ConsoleOutput bool
	// 是否使用彩色日志
	ColoredOutput bool
}

// DefaultLogConfig 返回默认日志配置
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:         zapcore.DebugLevel,
		LogDir:        defaultLogDir,
		// Go 的时间格式化语法使用特定的参考时间：2006-01-02 15:04:05
		// 其中 20060102 表示 YYYYMMDD 格式的日期
		Filename:      "kube-copilot-20060102.log", // 使用 Go 的时间格式化语法，按天拆分
		MaxSize:       10,                          // 10MB
		MaxBackups:    10,
		MaxAge:        7, // 7天
		Compress:      true,
		ConsoleOutput: true,
		ColoredOutput: true,
	}
}

// 检查是否需要轮转日志文件
func checkRotateLogger(config *LogConfig) {
	rotateMutex.Lock()
	defer rotateMutex.Unlock()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 如果是首次调用或日期变了，需要轮转日志文件
	if lastRotateDate.IsZero() || today.After(lastRotateDate) {
		// 格式化新的文件名
		newFilename := now.Format(config.Filename)
		
		// 如果是首次调用或文件名变了，需要重新初始化日志
		if currentLogFile == "" || newFilename != currentLogFile {
			// 关闭旧的日志
			if globalLogger != nil {
				globalLogger.Sync()
			}
			
			// 重置全局日志实例，以便下次调用 GetLogger 时重新初始化
			globalLogger = nil
			loggerOnce = sync.Once{}
			
			// 更新当前日志文件名和轮转时间
			currentLogFile = newFilename
			lastRotateDate = today
		}
	}
}

// InitLogger 初始化日志系统
func InitLogger(config *LogConfig) (*zap.Logger, error) {
	var err error
	loggerOnce.Do(func() {
		// 确保日志目录存在
		if err = os.MkdirAll(config.LogDir, 0755); err != nil {
			err = fmt.Errorf("创建日志目录失败: %v", err)
			return
		}

		// 获取当前日期，格式化文件名
		now := time.Now()
		filename := now.Format(config.Filename)
		
		// 更新当前日志文件名和轮转时间
		currentLogFile = filename
		lastRotateDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		// 创建 lumberjack 日志切割器
		lumberjackLogger := &lumberjack.Logger{
			Filename:   filepath.Join(config.LogDir, filename),
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}

		// 设置编码器配置
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// 如果启用彩色输出，使用彩色级别编码器
		if config.ColoredOutput {
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}

		// 创建核心
		var cores []zapcore.Core

		// 文件输出
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(lumberjackLogger),
			config.Level,
		)
		cores = append(cores, fileCore)

		// 如果启用控制台输出
		if config.ConsoleOutput {
			consoleCore := zapcore.NewCore(
				zapcore.NewConsoleEncoder(encoderConfig),
				zapcore.AddSync(os.Stdout),
				config.Level,
			)
			cores = append(cores, consoleCore)
		}

		// 合并所有核心
		core := zapcore.NewTee(cores...)

		// 创建日志记录器
		globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	})

	return globalLogger, err
}

// GetLogger 获取全局日志记录器
func GetLogger() *zap.Logger {
	// 检查是否需要轮转日志文件
	checkRotateLogger(DefaultLogConfig())
	
	if globalLogger == nil {
		// 如果尚未初始化，使用默认配置初始化
		logger, err := InitLogger(DefaultLogConfig())
		if err != nil {
			// 如果初始化失败，使用标准输出的开发配置
			config := zap.NewDevelopmentConfig()
			logger, _ = config.Build()
			logger.Error("初始化日志系统失败，使用默认开发配置", zap.Error(err))
		}
		return logger
	}
	return globalLogger
}

// Debug 输出调试级别日志
func Debug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// Info 输出信息级别日志
func Info(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// Warn 输出警告级别日志
func Warn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// Error 输出错误级别日志
func Error(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// Fatal 输出致命错误日志并退出程序
func Fatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// With 创建带有额外字段的日志记录器
func With(fields ...zap.Field) *zap.Logger {
	return GetLogger().With(fields...)
}

// Sync 同步日志缓冲区到输出
func Sync() error {
	if globalLogger != nil {
		return globalLogger.Sync()
	}
	return nil
}
