package utils

import (
	"github.com/spf13/viper"
)

var config *viper.Viper

// GetConfig 获取配置实例
func GetConfig() *viper.Viper {
	if config == nil {
		config = viper.New()
		config.SetConfigName("config")
		config.SetConfigType("yaml")

		// 设置配置文件路径
		config.AddConfigPath("configs")
		config.AddConfigPath(".")

		// 读取配置文件
		if err := config.ReadInConfig(); err != nil {
			// 如果配置文件不存在，使用默认配置
			config.SetDefault("jwt.key", "your-secret-key-please-change-in-production")
			config.SetDefault("jwt.expire", "24h")
			config.SetDefault("server.port", 8080)
			config.SetDefault("server.host", "0.0.0.0")
			config.SetDefault("log.level", "info")
			config.SetDefault("log.format", "json")
			config.SetDefault("log.output", "stdout")
			config.SetDefault("perf.enabled", true)
			config.SetDefault("perf.reset_interval", "24h")
		}
	}
	return config
}

// InitConfig 初始化配置
func InitConfig() error {
	config = viper.New()
	config.SetConfigName("config")
	config.SetConfigType("yaml")

	// 设置配置文件路径
	config.AddConfigPath("configs")
	config.AddConfigPath(".")

	// 读取配置文件
	if err := config.ReadInConfig(); err != nil {
		return err
	}

	return nil
}
