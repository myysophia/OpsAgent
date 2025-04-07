package audit

import (
	"time"

	"github.com/spf13/viper"
)

// LoadConfigFromViper 从 Viper 配置中加载审计日志配置
func LoadConfigFromViper(v *viper.Viper) *Config {
	if v == nil {
		return nil
	}

	// 检查是否启用审计日志
	if !v.GetBool("audit.enabled") {
		return &Config{Enabled: false}
	}

	// 加载数据库配置
	dbConfig := DatabaseConfig{
		Driver:   v.GetString("audit.database.driver"),
		Host:     v.GetString("audit.database.host"),
		Port:     v.GetInt("audit.database.port"),
		User:     v.GetString("audit.database.user"),
		Password: v.GetString("audit.database.password"),
		DBName:   v.GetString("audit.database.dbname"),
		SSLMode:  v.GetString("audit.database.sslmode"),
	}

	// 加载连接池配置
	dbConfig.MaxOpenConns = v.GetInt("audit.database.max_open_conns")
	dbConfig.MaxIdleConns = v.GetInt("audit.database.max_idle_conns")

	// 解析连接最大生命周期
	connMaxLifetimeStr := v.GetString("audit.database.conn_max_lifetime")
	if connMaxLifetimeStr != "" {
		if duration, err := time.ParseDuration(connMaxLifetimeStr); err == nil {
			dbConfig.ConnMaxLifetime = duration
		} else {
			// 默认为1小时
			dbConfig.ConnMaxLifetime = 1 * time.Hour
		}
	} else {
		// 默认为1小时
		dbConfig.ConnMaxLifetime = 1 * time.Hour
	}

	// 加载异步处理配置
	asyncConfig := AsyncConfig{
		Workers:         v.GetInt("audit.async.workers"),
		QueueSize:       v.GetInt("audit.async.queue_size"),
		BatchSize:       v.GetInt("audit.async.batch_size"),
		BatchIntervalMs: v.GetInt("audit.async.batch_interval_ms"),
	}

	// 加载数据保留策略
	retentionConfig := RetentionConfig{
		Days:        v.GetInt("audit.retention.days"),
		AutoCleanup: v.GetBool("audit.retention.auto_cleanup"),
		CleanupTime: v.GetString("audit.retention.cleanup_time"),
	}

	// 创建并返回完整配置
	return &Config{
		Enabled:   true,
		Database:  dbConfig,
		Async:     asyncConfig,
		Retention: retentionConfig,
	}
}
