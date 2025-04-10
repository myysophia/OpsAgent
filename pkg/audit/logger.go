package audit

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/myysophia/OpsAgent/pkg/utils"
)

// LogEntry 表示一条待记录的日志条目
type LogEntry struct {
	SessionID         uuid.UUID
	InteractionID     uuid.UUID
	UserID            string
	ClientIP          string
	UserAgent         string
	Question          string
	ModelName         string
	Provider          string
	BaseURL           string
	Cluster           string
	Thought           string
	FinalAnswer       string
	Status            string
	ToolCalls         []ToolCall
	PerfMetrics       map[string]time.Duration
	TotalDuration     time.Duration
	AssistantDuration time.Duration
	ParseDuration     time.Duration
}

// ToolCall 表示一次工具调用
type ToolCall struct {
	Name        string
	Input       string
	Observation string
	SequenceNum int
	Duration    time.Duration
}

// AuditLogger 审计日志记录器
type AuditLogger struct {
	db            *sql.DB
	logChan       chan LogEntry
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	logger        *zap.Logger
	workerCount   int
	batchSize     int           // 批处理大小
	batchInterval time.Duration // 批处理间隔
	config        *Config       // 审计日志配置
}

// NewAuditLogger 创建新的审计日志记录器
func NewAuditLogger(config *Config) (*AuditLogger, error) {
	// 检查是否启用审计日志
	if config == nil || !config.Enabled {
		return nil, nil
	}

	// 构建数据库连接字符串
	dbConfig := config.Database

	// 构建连接字符串
	var connStr string
	if dbConfig.Driver == "postgres" {
		// 尝试三种不同的连接字符串格式

		// 1. 键值对格式
		connStr1 := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.User,
			dbConfig.Password, // 直接使用原始密码
			dbConfig.DBName,
			dbConfig.SSLMode,
		)

		// 2. URI 格式，使用原始密码
		connStr2 := fmt.Sprintf(
			"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
			dbConfig.User,
			dbConfig.Password,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.DBName,
			dbConfig.SSLMode,
		)

		// 3. URI 格式，对密码进行 URL 编码
		encodedPassword := url.QueryEscape(dbConfig.Password)
		connStr3 := fmt.Sprintf(
			"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
			dbConfig.User,
			encodedPassword,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.DBName,
			dbConfig.SSLMode,
		)

		// 选择使用第一种格式
		connStr = connStr1

		// 记录所有连接字符串格式（仅用于调试）
		tmpLogger := utils.GetLogger().Named("audit")
		tmpLogger.Debug("连接字符串格式",
			zap.String("format1", connStr1),
			zap.String("format2", connStr2),
			zap.String("format3", connStr3),
		)
	} else {
		// 其他数据库的连接字符串格式 - 对密码进行 URL 编码
		encodedPassword := url.QueryEscape(dbConfig.Password)
		connStr = fmt.Sprintf(
			"%s://%s:%s@%s:%d/%s?sslmode=%s",
			dbConfig.Driver,
			dbConfig.User,
			encodedPassword,
			dbConfig.Host,
			dbConfig.Port,
			dbConfig.DBName,
			dbConfig.SSLMode,
		)
	}

	// 获取日志记录器
	logger := utils.GetLogger().Named("audit")

	// 记录连接信息（不包含密码）
	logger.Debug("尝试连接数据库",
		zap.String("driver", dbConfig.Driver),
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("user", dbConfig.User),
		zap.String("dbname", dbConfig.DBName),
		zap.String("sslmode", dbConfig.SSLMode),
		zap.String("conn_str", connStr), // 注意：这会显示密码，仅在调试时使用
	)

	// 尝试使用不同的连接字符串格式连接数据库
	var db *sql.DB
	var err error
	var lastErr error

	if dbConfig.Driver == "postgres" {
		// 尝试使用第一种格式
		logger.Debug("尝试使用第一种连接字符串格式")
		db, err = sql.Open(dbConfig.Driver, connStr)
		if err != nil {
			lastErr = err
			logger.Warn("使用第一种格式打开数据库连接失败",
				zap.Error(err),
				zap.String("format", "1"),
			)

			// 尝试使用第二种格式
			connStr = fmt.Sprintf(
				"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
				dbConfig.User,
				dbConfig.Password,
				dbConfig.Host,
				dbConfig.Port,
				dbConfig.DBName,
				dbConfig.SSLMode,
			)
			logger.Debug("尝试使用第二种连接字符串格式")
			db, err = sql.Open(dbConfig.Driver, connStr)
			if err != nil {
				lastErr = err
				logger.Warn("使用第二种格式打开数据库连接失败",
					zap.Error(err),
					zap.String("format", "2"),
				)

				// 尝试使用第三种格式
				encodedPassword := url.QueryEscape(dbConfig.Password)
				connStr = fmt.Sprintf(
					"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
					dbConfig.User,
					encodedPassword,
					dbConfig.Host,
					dbConfig.Port,
					dbConfig.DBName,
					"require", // 尝试使用 SSL 连接
				)
				logger.Debug("尝试使用第三种连接字符串格式")
				db, err = sql.Open(dbConfig.Driver, connStr)
				if err != nil {
					lastErr = err
					logger.Warn("使用第三种格式打开数据库连接失败",
						zap.Error(err),
						zap.String("format", "3"),
					)

					// 尝试使用第四种格式，不包含 sslmode 参数
					connStr = fmt.Sprintf(
						"postgresql://%s:%s@%s:%d/%s",
						dbConfig.User,
						encodedPassword,
						dbConfig.Host,
						dbConfig.Port,
						dbConfig.DBName,
					)
					logger.Debug("尝试使用第四种连接字符串格式")
					db, err = sql.Open(dbConfig.Driver, connStr)
					if err != nil {
						lastErr = err
						logger.Warn("使用第四种格式打开数据库连接失败",
							zap.Error(err),
							zap.String("format", "4"),
						)

						// 尝试使用第五种格式，Supabase 特定格式
						connStr = fmt.Sprintf(
							"postgres://%s:%s@%s:%d/%s",
							dbConfig.User,
							encodedPassword,
							dbConfig.Host,
							dbConfig.Port,
							dbConfig.DBName,
						)
						logger.Debug("尝试使用第五种连接字符串格式")
						db, err = sql.Open(dbConfig.Driver, connStr)
						if err != nil {
							lastErr = err
							logger.Warn("使用第五种格式打开数据库连接失败",
								zap.Error(err),
								zap.String("format", "5"),
							)
						}
					}
				}
			}
		}
	} else {
		// 其他数据库直接使用原始连接字符串
		db, err = sql.Open(dbConfig.Driver, connStr)
		lastErr = err
	}

	// 如果所有尝试都失败，返回错误
	if err != nil {
		logger.Error("所有连接字符串格式都无法打开数据库连接",
			zap.Error(lastErr),
			zap.String("driver", dbConfig.Driver),
			zap.String("host", dbConfig.Host),
			zap.Int("port", dbConfig.Port),
			zap.String("user", dbConfig.User),
			zap.String("dbname", dbConfig.DBName),
		)
		return nil, fmt.Errorf("打开数据库连接失败: %w", lastErr)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(dbConfig.MaxOpenConns)
	db.SetMaxIdleConns(dbConfig.MaxIdleConns)
	db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

	// 测试连接
	logger.Debug("测试数据库连接")
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer pingCancel()

	if err := db.PingContext(pingCtx); err != nil {
		logger.Error("数据库连接测试失败",
			zap.Error(err),
			zap.String("driver", dbConfig.Driver),
			zap.String("host", dbConfig.Host),
			zap.Int("port", dbConfig.Port),
			zap.String("user", dbConfig.User),
			zap.String("dbname", dbConfig.DBName),
			zap.String("conn_str", connStr), // 注意：这会显示密码，仅在调试时使用
		)

		// 尝试获取更详细的错误信息
		var row *sql.Row
		var version string

		// 尝试执行一个简单的查询
		row = db.QueryRowContext(pingCtx, "SELECT version()")
		err2 := row.Scan(&version)
		if err2 != nil {
			logger.Error("执行简单查询失败",
				zap.Error(err2),
			)
		}

		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	logger.Info("数据库连接成功",
		zap.String("driver", dbConfig.Driver),
		zap.String("host", dbConfig.Host),
		zap.Int("port", dbConfig.Port),
		zap.String("dbname", dbConfig.DBName),
	)

	ctx, cancel := context.WithCancel(context.Background())

	al := &AuditLogger{
		db:            db,
		logChan:       make(chan LogEntry, config.Async.QueueSize),
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
		workerCount:   config.Async.Workers,
		batchSize:     config.Async.BatchSize,
		batchInterval: time.Duration(config.Async.BatchIntervalMs) * time.Millisecond,
		config:        config,
	}

	// 启动工作协程
	for i := 0; i < al.workerCount; i++ {
		al.wg.Add(1)
		go al.worker(i)
	}

	// 如果启用了自动清理，启动清理协程
	if config.Retention.AutoCleanup && config.Retention.Days > 0 {
		al.wg.Add(1)
		go al.cleanupWorker()
	}

	logger.Info("审计日志系统初始化完成",
		zap.Int("workers", al.workerCount),
		zap.Int("queue_size", config.Async.QueueSize),
		zap.Int("batch_size", al.batchSize),
		zap.Duration("batch_interval", al.batchInterval),
	)

	return al, nil
}

// LogInteraction 记录一次交互
func (al *AuditLogger) LogInteraction(entry LogEntry) {
	// 如果审计日志记录器为空（未启用），直接返回
	if al == nil {
		logger := utils.GetLogger().Named("audit-logger")
		logger.Warn("审计日志记录器为空，无法记录日志")
		return
	}

	// 输出详细的日志信息
	al.logger.Debug("开始记录交互日志",
		zap.String("interaction_id", entry.InteractionID.String()),
		zap.String("session_id", entry.SessionID.String()),
		zap.String("user_id", entry.UserID),
		zap.String("question", entry.Question),
		zap.String("model_name", entry.ModelName),
		zap.String("thought", entry.Thought),
		zap.Int("tool_calls_count", len(entry.ToolCalls)),
		zap.Int("perf_metrics_count", len(entry.PerfMetrics)),
	)

	// 输出工具调用信息
	for i, tool := range entry.ToolCalls {
		al.logger.Debug(fmt.Sprintf("工具调用 %d", i),
			zap.String("name", tool.Name),
			zap.String("input", tool.Input),
			zap.Int("sequence_num", tool.SequenceNum),
			zap.Duration("duration", tool.Duration),
		)
	}

	// 输出性能指标信息
	for name, duration := range entry.PerfMetrics {
		al.logger.Debug(fmt.Sprintf("性能指标: %s", name),
			zap.Duration("duration", duration),
		)
	}

	select {
	case al.logChan <- entry:
		// 成功加入队列
		al.logger.Debug("审计日志条目已加入队列",
			zap.String("interaction_id", entry.InteractionID.String()),
			zap.String("question", entry.Question),
		)
	default:
		// 队列已满，记录错误但不阻塞主流程
		al.logger.Error("审计日志队列已满，丢弃日志条目",
			zap.String("question", entry.Question),
			zap.String("model", entry.ModelName),
		)
	}
}

// worker 工作协程，处理日志队列
func (al *AuditLogger) worker(id int) {
	defer al.wg.Done()

	al.logger.Info("审计日志工作协程启动", zap.Int("worker_id", id))

	// 创建批处理定时器
	ticker := time.NewTicker(al.batchInterval)
	defer ticker.Stop()

	// 创建批处理缓冲区
	batch := make([]LogEntry, 0, al.batchSize)

	// 批量处理函数
	processBatch := func() {
		if len(batch) == 0 {
			return
		}

		al.logger.Debug("开始处理批次",
			zap.Int("worker_id", id),
			zap.Int("batch_size", len(batch)),
		)

		// 开始事务
		tx, err := al.db.BeginTx(al.ctx, nil)
		if err != nil {
			al.logger.Error("开始事务失败", zap.Error(err))
			return
		}
		defer tx.Rollback()

		// 处理批次中的每一条日志
		for _, entry := range batch {
			if err := al.saveToDatabase(entry, tx); err != nil {
				al.logger.Error("保存审计日志失败",
					zap.Error(err),
					zap.String("interaction_id", entry.InteractionID.String()),
				)
				return // 如果有错误，回滚整个批次
			}
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			al.logger.Error("提交事务失败", zap.Error(err))
			return
		} else {
			al.logger.Info("批量处理成功",
				zap.Int("worker_id", id),
				zap.Int("batch_size", len(batch)),
			)
		}

		al.logger.Debug("批量处理完成",
			zap.Int("worker_id", id),
			zap.Int("batch_size", len(batch)),
		)

		// 清空批次
		batch = batch[:0]
	}

	for {
		select {
		case entry := <-al.logChan:
			// 输出日志条目信息
			al.logger.Debug("从队列中获取到审计日志条目",
				zap.Int("worker_id", id),
				zap.String("interaction_id", entry.InteractionID.String()),
				zap.String("model_name", entry.ModelName),
				zap.String("provider", entry.Provider),
				zap.String("base_url", entry.BaseURL),
				zap.String("thought", entry.Thought),
				zap.Int("thought_length", len(entry.Thought)),
				zap.Int("tool_calls_count", len(entry.ToolCalls)),
				zap.Int("perf_metrics_count", len(entry.PerfMetrics)),
			)

			// 添加到批次
			batch = append(batch, entry)

			// 如果批次已满，立即处理
			if len(batch) >= al.batchSize {
				processBatch()
			}

		case <-ticker.C:
			// 定时处理批次，即使批次未满
			processBatch()

		case <-al.ctx.Done():
			// 处理剩余的批次
			processBatch()
			al.logger.Info("审计日志工作协程退出", zap.Int("worker_id", id))
			return
		}
	}
}

// saveToDatabase 将日志保存到数据库
func (al *AuditLogger) saveToDatabase(entry LogEntry, tx *sql.Tx) error {
	al.logger.Debug("开始保存审计日志到数据库",
		zap.String("interaction_id", entry.InteractionID.String()),
		zap.String("session_id", entry.SessionID.String()),
		zap.String("thought", entry.Thought),
		zap.Int("tool_calls_count", len(entry.ToolCalls)),
		zap.Int("perf_metrics_count", len(entry.PerfMetrics)),
	)

	// 如果没有提供事务，创建新的事务
	var err error
	var manageTx bool

	if tx == nil {
		tx, err = al.db.BeginTx(al.ctx, nil)
		if err != nil {
			return fmt.Errorf("开始事务失败: %w", err)
		}
		defer tx.Rollback()
		manageTx = true
	}

	// 1. 检查并插入会话记录
	var sessionExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM opsagent.sessions WHERE session_id = $1)", entry.SessionID).Scan(&sessionExists)
	if err != nil {
		al.logger.Error("检查会话记录失败",
			zap.Error(err),
			zap.String("session_id", entry.SessionID.String()),
		)
		return fmt.Errorf("检查会话记录失败: %w", err)
	}

	if !sessionExists {
		al.logger.Debug("会话记录不存在，准备插入",
			zap.String("session_id", entry.SessionID.String()),
		)

		_, err = tx.Exec(
			"INSERT INTO opsagent.sessions (session_id, user_id, client_ip, user_agent) VALUES ($1, $2, $3, $4)",
			entry.SessionID, entry.UserID, entry.ClientIP, entry.UserAgent,
		)
		if err != nil {
			al.logger.Error("插入会话记录失败",
				zap.Error(err),
				zap.String("session_id", entry.SessionID.String()),
			)
			return fmt.Errorf("插入会话记录失败: %w", err)
		}
	}

	// 2. 插入交互记录
	al.logger.Debug("准备插入交互记录",
		zap.String("interaction_id", entry.InteractionID.String()),
		zap.String("question", entry.Question),
		zap.String("model_name", entry.ModelName),
		zap.String("provider", entry.Provider),
		zap.String("base_url", entry.BaseURL),
		zap.String("cluster", entry.Cluster),
		zap.String("status", entry.Status),
		zap.Duration("total_duration", entry.TotalDuration),
		zap.Duration("assistant_duration", entry.AssistantDuration),
		zap.Duration("parse_duration", entry.ParseDuration),
	)

	// 构建 SQL 语句
	sql := `INSERT INTO opsagent.interactions
        (interaction_id, session_id, question, model_name, provider, base_url, cluster, final_answer, status,
        total_duration_ms, assistant_duration_ms, parse_duration_ms)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	// 准备参数
	params := []interface{}{
		entry.InteractionID,
		entry.SessionID,
		entry.Question,
		entry.ModelName,
		entry.Provider,
		entry.BaseURL,
		entry.Cluster,
		entry.FinalAnswer,
		entry.Status,
		int(entry.TotalDuration.Milliseconds()),
		int(entry.AssistantDuration.Milliseconds()),
		int(entry.ParseDuration.Milliseconds()),
	}

	// 输出参数值
	al.logger.Debug("插入交互记录参数",
		zap.String("sql", sql),
		zap.Any("params", params),
	)

	_, err = tx.Exec(sql, params...)
	if err != nil {
		al.logger.Error("插入交互记录失败",
			zap.Error(err),
			zap.String("interaction_id", entry.InteractionID.String()),
		)
		return fmt.Errorf("插入交互记录失败: %w", err)
	}

	// 3. 插入思考过程
	al.logger.Debug("检查思考过程",
		zap.String("interaction_id", entry.InteractionID.String()),
		zap.String("thought", entry.Thought),
		zap.Bool("is_empty", entry.Thought == ""),
		zap.Int("thought_length", len(entry.Thought)),
	)

	if entry.Thought != "" {
		al.logger.Debug("准备插入思考过程",
			zap.String("interaction_id", entry.InteractionID.String()),
			zap.String("thought", entry.Thought),
		)

		_, err = tx.Exec(
			"INSERT INTO opsagent.thoughts (interaction_id, thought) VALUES ($1, $2)",
			entry.InteractionID, entry.Thought,
		)
		if err != nil {
			al.logger.Error("插入思考过程失败",
				zap.Error(err),
				zap.String("interaction_id", entry.InteractionID.String()),
			)
			return fmt.Errorf("插入思考过程失败: %w", err)
		} else {
			al.logger.Info("插入思考过程成功",
				zap.String("interaction_id", entry.InteractionID.String()),
				zap.Int("thought_length", len(entry.Thought)),
			)
		}
	} else {
		al.logger.Warn("思考过程为空，跳过插入",
			zap.String("interaction_id", entry.InteractionID.String()),
		)
	}

	// 4. 插入工具调用记录
	al.logger.Debug("检查工具调用记录",
		zap.String("interaction_id", entry.InteractionID.String()),
		zap.Int("tool_calls_count", len(entry.ToolCalls)),
		zap.Bool("is_empty", len(entry.ToolCalls) == 0),
	)

	if len(entry.ToolCalls) == 0 {
		al.logger.Warn("工具调用记录为空，跳过插入",
			zap.String("interaction_id", entry.InteractionID.String()),
		)
	} else {
		al.logger.Debug("准备插入工具调用记录",
			zap.String("interaction_id", entry.InteractionID.String()),
			zap.Int("tool_calls_count", len(entry.ToolCalls)),
		)

		for i, tool := range entry.ToolCalls {
			al.logger.Debug(fmt.Sprintf("准备插入第 %d 个工具调用", i),
				zap.String("name", tool.Name),
				zap.String("input", tool.Input),
				zap.String("observation", tool.Observation),
				zap.Int("sequence_num", tool.SequenceNum),
				zap.Duration("duration", tool.Duration),
			)

			_, err = tx.Exec(
				`INSERT INTO opsagent.tool_calls
                (interaction_id, tool_name, tool_input, tool_observation, sequence_number, duration_ms)
                VALUES ($1, $2, $3, $4, $5, $6)`,
				entry.InteractionID, tool.Name, tool.Input, tool.Observation, tool.SequenceNum,
				int(tool.Duration.Milliseconds()),
			)
			if err != nil {
				al.logger.Error("插入工具调用记录失败",
					zap.Error(err),
					zap.String("interaction_id", entry.InteractionID.String()),
					zap.String("tool_name", tool.Name),
				)
				return fmt.Errorf("插入工具调用记录失败: %w", err)
			} else {
				al.logger.Info(fmt.Sprintf("插入第 %d 个工具调用成功", i),
					zap.String("interaction_id", entry.InteractionID.String()),
					zap.String("tool_name", tool.Name),
				)
			}
		}
	}

	// 5. 插入性能指标
	al.logger.Debug("检查性能指标",
		zap.String("interaction_id", entry.InteractionID.String()),
		zap.Int("perf_metrics_count", len(entry.PerfMetrics)),
		zap.Bool("is_empty", len(entry.PerfMetrics) == 0),
	)

	if len(entry.PerfMetrics) == 0 {
		al.logger.Warn("性能指标为空，跳过插入",
			zap.String("interaction_id", entry.InteractionID.String()),
		)
	} else {
		al.logger.Debug("准备插入性能指标",
			zap.String("interaction_id", entry.InteractionID.String()),
			zap.Int("perf_metrics_count", len(entry.PerfMetrics)),
		)

		// 输出所有性能指标
		var metricNames []string
		for name := range entry.PerfMetrics {
			metricNames = append(metricNames, name)
		}
		al.logger.Debug("所有性能指标名称",
			zap.Strings("metric_names", metricNames),
		)

		for name, duration := range entry.PerfMetrics {
			al.logger.Debug(fmt.Sprintf("准备插入性能指标: %s", name),
				zap.Duration("duration", duration),
				zap.Int("duration_ms", int(duration.Milliseconds())),
			)

			_, err = tx.Exec(
				"INSERT INTO opsagent.performance_metrics (interaction_id, metric_name, duration_ms) VALUES ($1, $2, $3)",
				entry.InteractionID, name, int(duration.Milliseconds()),
			)
			if err != nil {
				al.logger.Error("插入性能指标失败",
					zap.Error(err),
					zap.String("interaction_id", entry.InteractionID.String()),
					zap.String("metric_name", name),
				)
				return fmt.Errorf("插入性能指标失败: %w", err)
			} else {
				al.logger.Info(fmt.Sprintf("插入性能指标成功: %s", name),
					zap.String("interaction_id", entry.InteractionID.String()),
					zap.Duration("duration", duration),
				)
			}
		}
	}

	// 如果是自己管理的事务，提交事务
	if manageTx {
		return tx.Commit()
	}

	return nil
}

// cleanupWorker 清理过期数据的工作协程
func (al *AuditLogger) cleanupWorker() {
	defer al.wg.Done()

	al.logger.Info("审计日志清理协程启动",
		zap.Int("retention_days", al.config.Retention.Days),
		zap.String("cleanup_time", al.config.Retention.CleanupTime),
	)

	// 解析清理时间
	var hour, minute int
	_, err := fmt.Sscanf(al.config.Retention.CleanupTime, "%d:%d", &hour, &minute)
	if err != nil {
		al.logger.Error("解析清理时间失败，使用默认值 03:00",
			zap.Error(err),
			zap.String("cleanup_time", al.config.Retention.CleanupTime),
		)
		hour, minute = 3, 0
	}

	// 计算下一次清理时间
	now := time.Now()
	nextCleanup := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if nextCleanup.Before(now) {
		nextCleanup = nextCleanup.Add(24 * time.Hour)
	}

	// 创建定时器
	timer := time.NewTimer(nextCleanup.Sub(now))
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// 执行清理操作
			al.cleanup()

			// 计算下一次清理时间
			now = time.Now()
			nextCleanup = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
			if nextCleanup.Before(now) {
				nextCleanup = nextCleanup.Add(24 * time.Hour)
			}
			timer.Reset(nextCleanup.Sub(now))

		case <-al.ctx.Done():
			al.logger.Info("审计日志清理协程退出")
			return
		}
	}
}

// cleanup 清理过期数据
func (al *AuditLogger) cleanup() {
	al.logger.Info("开始清理过期审计日志数据")

	// 计算截止日期
	cutoffDate := time.Now().AddDate(0, 0, -al.config.Retention.Days)

	// 开始事务
	tx, err := al.db.BeginTx(context.Background(), nil)
	if err != nil {
		al.logger.Error("开始清理事务失败", zap.Error(err))
		return
	}
	defer tx.Rollback()

	// 查询过期的交互ID
	rows, err := tx.Query("SELECT interaction_id FROM opsagent.interactions WHERE created_at < $1", cutoffDate)
	if err != nil {
		al.logger.Error("查询过期交互记录失败", zap.Error(err))
		return
	}
	defer rows.Close()

	// 收集过期的交互ID
	var interactionIDs []interface{}
	var interactionID uuid.UUID
	for rows.Next() {
		if err := rows.Scan(&interactionID); err != nil {
			al.logger.Error("扫描交互ID失败", zap.Error(err))
			return
		}
		interactionIDs = append(interactionIDs, interactionID)
	}

	if len(interactionIDs) == 0 {
		al.logger.Info("没有过期的审计日志数据需要清理")
		return
	}

	// 构建占位符
	placeholders := make([]string, len(interactionIDs))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	// 删除相关记录
	tables := []string{"performance_metrics", "tool_calls", "thoughts", "interactions"}
	for _, table := range tables {
		query := fmt.Sprintf("DELETE FROM opsagent.%s WHERE interaction_id IN (%s)", table, placeholders)
		_, err := tx.Exec(query, interactionIDs...)
		if err != nil {
			al.logger.Error("删除过期记录失败",
				zap.Error(err),
				zap.String("table", table),
			)
			return
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		al.logger.Error("提交清理事务失败", zap.Error(err))
		return
	}

	al.logger.Info("清理过期审计日志数据完成",
		zap.Int("cleaned_records", len(interactionIDs)),
	)
}

// Close 关闭日志记录器
func (al *AuditLogger) Close() error {
	// 如果日志记录器为空（未启用）
	if al == nil {
		return nil
	}

	al.logger.Info("正在关闭审计日志记录器")

	// 发送关闭信号
	al.cancel()

	// 等待所有工作协程完成
	al.wg.Wait()

	// 处理剩余的日志条目
	close(al.logChan)

	// 批量处理剩余的日志
	batch := make([]LogEntry, 0, al.batchSize)
	for entry := range al.logChan {
		batch = append(batch, entry)

		// 如果批次已满，处理批次
		if len(batch) >= al.batchSize {
			al.processFinalBatch(batch)
			batch = batch[:0]
		}
	}

	// 处理最后一个不完整的批次
	if len(batch) > 0 {
		al.processFinalBatch(batch)
	}

	// 关闭数据库连接
	err := al.db.Close()
	if err != nil {
		return fmt.Errorf("关闭数据库连接失败: %w", err)
	}

	al.logger.Info("审计日志记录器已关闭")
	return nil
}

// processFinalBatch 处理最终批次
func (al *AuditLogger) processFinalBatch(batch []LogEntry) {
	// 开始事务
	tx, err := al.db.BeginTx(context.Background(), nil)
	if err != nil {
		al.logger.Error("关闭时开始事务失败", zap.Error(err))
		return
	}
	defer tx.Rollback()

	// 处理批次中的每一条日志
	for _, entry := range batch {
		if err := al.saveToDatabase(entry, tx); err != nil {
			al.logger.Error("关闭时保存审计日志失败",
				zap.Error(err),
				zap.String("interaction_id", entry.InteractionID.String()),
			)
			return // 如果有错误，回滚整个批次
		}
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		al.logger.Error("关闭时提交事务失败", zap.Error(err))
	}
}
