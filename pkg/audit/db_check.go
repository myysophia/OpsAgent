package audit

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

// CheckDatabase 检查数据库连接和表结构
func CheckDatabase(config *Config) error {
	// 获取日志记录器
	logger := utils.GetLogger().Named("audit")

	// 如果配置为空或者未启用审计日志，直接返回
	if config == nil {
		logger.Warn("审计日志配置为空")
		return nil
	}

	if !config.Enabled {
		logger.Info("审计日志未启用")
		return nil
	}

	// 构建数据库连接字符串
	dbConfig := config.Database
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.User,
		dbConfig.Password,
		dbConfig.DBName,
		dbConfig.SSLMode,
	)

	// 获取日志记录器
	logger = utils.GetLogger().Named("audit")
	logger.Info("检查数据库连接",
		zap.String("driver", dbConfig.Driver),
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("user", dbConfig.User),
		zap.String("dbname", dbConfig.DBName),
	)

	// 连接数据库
	db, err := sql.Open(dbConfig.Driver, connStr)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		logger.Error("测试数据库连接失败",
			zap.Error(err),
			zap.String("driver", dbConfig.Driver),
			zap.String("host", dbConfig.Host),
			zap.Int("port", dbConfig.Port),
			zap.String("user", dbConfig.User),
			zap.String("dbname", dbConfig.DBName),
		)

		// 如果数据库不存在，尝试连接到 postgres 数据库并创建数据库
		if strings.Contains(err.Error(), "does not exist") {
			logger.Warn("数据库不存在，尝试创建数据库",
				zap.String("dbname", dbConfig.DBName),
			)

			// 连接到 postgres 数据库
			postgresConnStr := fmt.Sprintf(
				"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
				dbConfig.Host,
				dbConfig.Port,
				dbConfig.User,
				dbConfig.Password,
				dbConfig.SSLMode,
			)

			postgresDB, err := sql.Open(dbConfig.Driver, postgresConnStr)
			if err != nil {
				return fmt.Errorf("连接到 postgres 数据库失败: %w", err)
			}
			defer postgresDB.Close()

			// 测试连接
			if err := postgresDB.Ping(); err != nil {
				return fmt.Errorf("测试 postgres 数据库连接失败: %w", err)
			}

			// 创建数据库
			_, err = postgresDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbConfig.DBName))
			if err != nil {
				return fmt.Errorf("创建数据库失败: %w", err)
			}

			logger.Info("数据库创建成功",
				zap.String("dbname", dbConfig.DBName),
			)

			// 重新连接到新创建的数据库
			db, err = sql.Open(dbConfig.Driver, connStr)
			if err != nil {
				return fmt.Errorf("连接到新创建的数据库失败: %w", err)
			}

			// 测试连接
			if err := db.Ping(); err != nil {
				return fmt.Errorf("测试新创建的数据库连接失败: %w", err)
			}

			// 创建表结构
			if err := createTables(db); err != nil {
				return fmt.Errorf("创建表结构失败: %w", err)
			}

			logger.Info("表结构创建成功")
		} else {
			// 其他错误
			return fmt.Errorf("测试数据库连接失败: %w", err)
		}
	}

	// 检查表是否存在
	tables := []string{"sessions", "interactions", "thoughts", "tool_calls", "performance_metrics"}
	for _, table := range tables {
		var exists bool
		query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = '%s')", table)
		err := db.QueryRow(query).Scan(&exists)
		if err != nil {
			return fmt.Errorf("检查表 %s 是否存在失败: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("表 %s 不存在", table)
		}
		logger.Info(fmt.Sprintf("表 %s 存在", table))
	}

	// 检查表结构
	for _, table := range tables {
		var columns []string
		query := fmt.Sprintf("SELECT column_name FROM information_schema.columns WHERE table_name = '%s'", table)
		rows, err := db.Query(query)
		if err != nil {
			return fmt.Errorf("获取表 %s 的列失败: %w", table, err)
		}
		defer rows.Close()

		for rows.Next() {
			var column string
			if err := rows.Scan(&column); err != nil {
				return fmt.Errorf("扫描表 %s 的列失败: %w", table, err)
			}
			columns = append(columns, column)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("遍历表 %s 的列失败: %w", table, err)
		}

		logger.Info(fmt.Sprintf("表 %s 的列: %s", table, strings.Join(columns, ", ")))
	}

	logger.Info("数据库检查通过")
	return nil
}

// createTables 创建表结构
func createTables(db *sql.DB) error {
	// 获取日志记录器
	logger := utils.GetLogger().Named("audit")

	// 尝试从文件中读取 SQL 脚本
	sqlFilePath := "audit.sql"

	// 尝试不同的路径
	paths := []string{
		sqlFilePath,
		filepath.Join(".", sqlFilePath),
		filepath.Join("..", sqlFilePath),
		filepath.Join("../..", sqlFilePath),
		filepath.Join("configs", sqlFilePath),
		filepath.Join("../configs", sqlFilePath),
		filepath.Join("../../configs", sqlFilePath),
	}

	var sqlContent string
	var err error
	var foundPath string

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			// 文件存在
			content, err := ioutil.ReadFile(path)
			if err == nil {
				sqlContent = string(content)
				foundPath = path
				break
			}
		}
	}

	if sqlContent == "" {
		// 如果没有找到 SQL 脚本文件，使用内置的 SQL 脚本
		logger.Warn("没有找到 SQL 脚本文件，使用内置的 SQL 脚本")
		sqlContent = defaultSQLScript
	} else {
		logger.Info("使用 SQL 脚本文件",
			zap.String("path", foundPath),
		)
	}

	// 分割 SQL 脚本
	statements := strings.Split(sqlContent, ";")

	// 执行每个 SQL 语句
	for _, statement := range statements {
		// 去除空白字符
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		// 执行 SQL 语句
		_, err = db.Exec(statement)
		if err != nil {
			logger.Error("执行 SQL 语句失败",
				zap.Error(err),
				zap.String("statement", statement),
			)
			return fmt.Errorf("执行 SQL 语句失败: %w", err)
		}
	}

	return nil
}

// defaultSQLScript 默认的 SQL 脚本
const defaultSQLScript = `
-- 会话表 (sessions)
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    session_id UUID NOT NULL,
    user_id VARCHAR(100),
    client_ip VARCHAR(50),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id)
);

-- 问答记录表 (interactions)
CREATE TABLE interactions (
    id SERIAL PRIMARY KEY,
    interaction_id UUID NOT NULL,
    session_id UUID NOT NULL,
    question TEXT NOT NULL,
    model_name VARCHAR(100) NOT NULL,
    provider VARCHAR(100),
    base_url TEXT,
    cluster VARCHAR(100),
    final_answer TEXT,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    total_duration_ms INTEGER,
    assistant_duration_ms INTEGER,
    parse_duration_ms INTEGER,
    UNIQUE(interaction_id),
    FOREIGN KEY (session_id) REFERENCES sessions(session_id)
);

-- 思考过程表 (thoughts)
CREATE TABLE thoughts (
    id SERIAL PRIMARY KEY,
    interaction_id UUID NOT NULL,
    thought TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (interaction_id) REFERENCES interactions(interaction_id)
);

-- 工具调用表 (tool_calls)
CREATE TABLE tool_calls (
    id SERIAL PRIMARY KEY,
    interaction_id UUID NOT NULL,
    tool_name VARCHAR(100) NOT NULL,
    tool_input TEXT NOT NULL,
    tool_observation TEXT,
    sequence_number INTEGER NOT NULL,
    duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (interaction_id) REFERENCES interactions(interaction_id)
);

-- 性能指标表 (performance_metrics)
CREATE TABLE performance_metrics (
    id SERIAL PRIMARY KEY,
    interaction_id UUID NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    duration_ms INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (interaction_id) REFERENCES interactions(interaction_id)
);

-- 索引创建

-- 会话表索引
CREATE INDEX idx_sessions_created_at ON sessions(created_at);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- 问答记录表索引
CREATE INDEX idx_interactions_created_at ON interactions(created_at);
CREATE INDEX idx_interactions_session_id ON interactions(session_id);
CREATE INDEX idx_interactions_model_name ON interactions(model_name);
CREATE INDEX idx_interactions_status ON interactions(status);

-- 工具调用表索引
CREATE INDEX idx_tool_calls_interaction_id ON tool_calls(interaction_id);
CREATE INDEX idx_tool_calls_tool_name ON tool_calls(tool_name);

-- 性能指标表索引
CREATE INDEX idx_performance_metrics_interaction_id ON performance_metrics(interaction_id);
CREATE INDEX idx_performance_metrics_metric_name ON performance_metrics(metric_name);
`
