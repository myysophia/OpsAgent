package audit

import "time"

// Config 审计日志配置
type Config struct {
	// 是否启用审计日志
	Enabled bool `yaml:"enabled"`

	// 数据库配置
	Database DatabaseConfig `yaml:"database"`

	// 异步处理配置
	Async AsyncConfig `yaml:"async"`

	// 数据保留策略
	Retention RetentionConfig `yaml:"retention"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// AsyncConfig 异步处理配置
type AsyncConfig struct {
	// 工作协程数量
	Workers int `yaml:"workers"`
	// 日志队列大小
	QueueSize int `yaml:"queue_size"`
	// 批处理大小
	BatchSize int `yaml:"batch_size"`
	// 批处理间隔(毫秒)
	BatchIntervalMs int `yaml:"batch_interval_ms"`
}

// RetentionConfig 数据保留策略
type RetentionConfig struct {
	// 数据保留天数，0表示永久保留
	Days int `yaml:"days"`
	// 是否启用自动清理
	AutoCleanup bool `yaml:"auto_cleanup"`
	// 清理时间，格式为"小时:分钟"
	CleanupTime string `yaml:"cleanup_time"`
}
