# OpsAgent 问答日志审计系统设计

## 1. 概述

本文档描述了OpsAgent问答日志审计系统的设计方案，该系统将用户问题、模型类型、LLM思考过程、最终结果和性能指标等数据记录到PostgreSQL数据库中，以便后续分析和审计。系统采用异步处理方式，确保日志记录不会影响主要业务流程的性能。

## 2. 需求分析

### 2.1 功能需求

1. 记录完整的问答交互日志，包括：
   - 用户问题
   - 使用的模型类型
   - LLM的思考过程
   - 工具调用历史
   - 最终结果
   - 各阶段处理耗时

2. 异步处理日志记录操作，不影响主要业务流程
3. 提供查询和分析接口，支持按时间、用户、模型等维度查询
4. 支持数据导出功能

### 2.2 非功能需求

1. 性能：日志记录操作不应显著影响API响应时间
2. 可靠性：确保日志数据不丢失
3. 安全性：敏感信息脱敏处理
4. 可扩展性：支持未来可能的数据分析需求

## 3. 系统架构

### 3.1 整体架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  API服务    │────▶│  日志队列   │────▶│ 日志处理器  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │ PostgreSQL  │
                                        └─────────────┘
```

### 3.2 组件说明

1. **API服务**：现有的OpsAgent API服务，负责处理用户请求
2. **日志队列**：使用Go channel或消息队列存储待处理的日志
3. **日志处理器**：异步处理日志队列中的数据，写入数据库
4. **PostgreSQL**：存储日志数据的关系型数据库

## 4. 数据库设计

### 4.1 表结构设计

#### 4.1.1 会话表 (sessions)

```sql
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    session_id UUID NOT NULL,
    user_id VARCHAR(100),
    client_ip VARCHAR(50),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id)
);
```

#### 4.1.2 问答记录表 (interactions)

```sql
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
```

#### 4.1.3 思考过程表 (thoughts)

```sql
CREATE TABLE thoughts (
    id SERIAL PRIMARY KEY,
    interaction_id UUID NOT NULL,
    thought TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (interaction_id) REFERENCES interactions(interaction_id)
);
```

#### 4.1.4 工具调用表 (tool_calls)

```sql
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
```

#### 4.1.5 性能指标表 (performance_metrics)

```sql
CREATE TABLE performance_metrics (
    id SERIAL PRIMARY KEY,
    interaction_id UUID NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    duration_ms INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (interaction_id) REFERENCES interactions(interaction_id)
);
```

### 4.2 索引设计

```sql
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
```

## 5. 代码实现

### 5.1 日志记录器结构

```go
// pkg/audit/logger.go

package audit

import (
    "context"
    "database/sql"
    "github.com/google/uuid"
    "github.com/sashabaranov/go-openai"
    "go.uber.org/zap"
    "sync"
    "time"

    _ "github.com/lib/pq"
    "github.com/myysophia/OpsAgent/pkg/utils"
)

// LogEntry 表示一条待记录的日志条目
type LogEntry struct {
    SessionID       uuid.UUID
    InteractionID   uuid.UUID
    UserID          string
    ClientIP        string
    UserAgent       string
    Question        string
    ModelName       string
    Provider        string
    BaseURL         string
    Cluster         string
    Thought         string
    FinalAnswer     string
    Status          string
    ToolCalls       []ToolCall
    PerfMetrics     map[string]time.Duration
    TotalDuration   time.Duration
    AssistantDuration time.Duration
    ParseDuration   time.Duration
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
}

// NewAuditLogger 创建新的审计日志记录器
func NewAuditLogger(config *utils.Config) (*AuditLogger, error) {
    // 检查是否启用审计日志
    if !config.Audit.Enabled {
        return nil, nil
    }

    // 构建数据库连接字符串
    dbConfig := config.Audit.Database
    connStr := fmt.Sprintf(
        "%s://%s:%s@%s:%d/%s?sslmode=%s",
        dbConfig.Driver,
        dbConfig.User,
        dbConfig.Password,
        dbConfig.Host,
        dbConfig.Port,
        dbConfig.DBName,
        dbConfig.SSLMode,
    )

    db, err := sql.Open(dbConfig.Driver, connStr)
    if err != nil {
        return nil, err
    }

    // 设置连接池参数
    db.SetMaxOpenConns(dbConfig.MaxOpenConns)
    db.SetMaxIdleConns(dbConfig.MaxIdleConns)
    db.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

    // 测试连接
    if err := db.Ping(); err != nil {
        db.Close()
        return nil, err
    }

    ctx, cancel := context.WithCancel(context.Background())
    logger := utils.GetLogger().Named("audit")

    al := &AuditLogger{
        db:          db,
        logChan:     make(chan LogEntry, config.Audit.Async.QueueSize),
        ctx:         ctx,
        cancel:      cancel,
        logger:      logger,
        workerCount: config.Audit.Async.Workers,
        batchSize:   config.Audit.Async.BatchSize,
        batchInterval: time.Duration(config.Audit.Async.BatchIntervalMs) * time.Millisecond,
    }

    // 启动工作协程
    for i := 0; i < al.workerCount; i++ {
        al.wg.Add(1)
        go al.worker(i)
    }

    return al, nil
}

// LogInteraction 记录一次交互
func (al *AuditLogger) LogInteraction(entry LogEntry) {
    select {
    case al.logChan <- entry:
        // 成功加入队列
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
    // 如果没有提供事务，创建新的事务
    var err error
    var manageTx bool

    if tx == nil {
        tx, err = al.db.BeginTx(al.ctx, nil)
        if err != nil {
            return err
        }
        defer tx.Rollback()
        manageTx = true
    }

    // 1. 检查并插入会话记录
    var sessionExists bool
    err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM sessions WHERE session_id = $1)", entry.SessionID).Scan(&sessionExists)
    if err != nil {
        return err
    }

    if !sessionExists {
        _, err = tx.Exec(
            "INSERT INTO sessions (session_id, user_id, client_ip, user_agent) VALUES ($1, $2, $3, $4)",
            entry.SessionID, entry.UserID, entry.ClientIP, entry.UserAgent,
        )
        if err != nil {
            return err
        }
    }

    // 2. 插入交互记录
    _, err = tx.Exec(
        `INSERT INTO interactions
        (interaction_id, session_id, question, model_name, provider, base_url, cluster, final_answer, status,
        total_duration_ms, assistant_duration_ms, parse_duration_ms)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
        entry.InteractionID, entry.SessionID, entry.Question, entry.ModelName, entry.Provider, entry.BaseURL,
        entry.Cluster, entry.FinalAnswer, entry.Status,
        int(entry.TotalDuration.Milliseconds()),
        int(entry.AssistantDuration.Milliseconds()),
        int(entry.ParseDuration.Milliseconds()),
    )
    if err != nil {
        return err
    }

    // 3. 插入思考过程
    if entry.Thought != "" {
        _, err = tx.Exec(
            "INSERT INTO thoughts (interaction_id, thought) VALUES ($1, $2)",
            entry.InteractionID, entry.Thought,
        )
        if err != nil {
            return err
        }
    }

    // 4. 插入工具调用记录
    for _, tool := range entry.ToolCalls {
        _, err = tx.Exec(
            `INSERT INTO tool_calls
            (interaction_id, tool_name, tool_input, tool_observation, sequence_number, duration_ms)
            VALUES ($1, $2, $3, $4, $5, $6)`,
            entry.InteractionID, tool.Name, tool.Input, tool.Observation, tool.SequenceNum,
            int(tool.Duration.Milliseconds()),
        )
        if err != nil {
            return err
        }
    }

    // 5. 插入性能指标
    for name, duration := range entry.PerfMetrics {
        _, err = tx.Exec(
            "INSERT INTO performance_metrics (interaction_id, metric_name, duration_ms) VALUES ($1, $2, $3)",
            entry.InteractionID, name, int(duration.Milliseconds()),
        )
        if err != nil {
            return err
        }
    }

    // 如果是自己管理的事务，提交事务
    if manageTx {
        return tx.Commit()
    }

    return nil
}

// Close 关闭日志记录器
func (al *AuditLogger) Close() error {
    // 如果日志记录器为空（未启用）
    if al == nil {
        return nil
    }

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
    return al.db.Close()
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
```

### 5.2 中间件实现

```go
// pkg/middleware/audit.go

package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/myysophia/OpsAgent/pkg/audit"
    "github.com/myysophia/OpsAgent/pkg/utils"
    "go.uber.org/zap"
    "time"
)

// AuditMiddleware 审计中间件
func AuditMiddleware(auditLogger *audit.AuditLogger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 只对特定路径进行审计
        if c.Request.URL.Path != "/api/execute" && c.Request.URL.Path != "/api/diagnose" && c.Request.URL.Path != "/api/analyze" {
            c.Next()
            return
        }

        // 获取或创建会话ID
        var sessionID uuid.UUID
        sessionIDStr, err := c.Cookie("session_id")
        if err != nil || sessionIDStr == "" {
            sessionID = uuid.New()
            c.SetCookie("session_id", sessionID.String(), 86400, "/", "", false, true)
        } else {
            sessionID, err = uuid.Parse(sessionIDStr)
            if err != nil {
                sessionID = uuid.New()
                c.SetCookie("session_id", sessionID.String(), 86400, "/", "", false, true)
            }
        }

        // 创建交互ID
        interactionID := uuid.New()

        // 将ID注入上下文
        c.Set("session_id", sessionID)
        c.Set("interaction_id", interactionID)

        // 记录开始时间
        startTime := time.Now()

        // 处理请求
        c.Next()

        // 计算总耗时
        totalDuration := time.Since(startTime)

        // 从上下文中获取其他需要记录的信息
        var (
            question string
            modelName string
            provider string
            baseURL string
            cluster string
            thought string
            finalAnswer string
            status string
            toolCalls []audit.ToolCall
            perfMetrics map[string]time.Duration
            assistantDuration time.Duration
            parseDuration time.Duration
        )

        // 从上下文中提取数据
        if v, exists := c.Get("audit_data"); exists {
            if data, ok := v.(map[string]interface{}); ok {
                if q, ok := data["question"].(string); ok {
                    question = q
                }
                if m, ok := data["model_name"].(string); ok {
                    modelName = m
                }
                if p, ok := data["provider"].(string); ok {
                    provider = p
                }
                if b, ok := data["base_url"].(string); ok {
                    baseURL = b
                }
                if cl, ok := data["cluster"].(string); ok {
                    cluster = cl
                }
                if t, ok := data["thought"].(string); ok {
                    thought = t
                }
                if f, ok := data["final_answer"].(string); ok {
                    finalAnswer = f
                }
                if s, ok := data["status"].(string); ok {
                    status = s
                }
                if tc, ok := data["tool_calls"].([]audit.ToolCall); ok {
                    toolCalls = tc
                }
                if pm, ok := data["perf_metrics"].(map[string]time.Duration); ok {
                    perfMetrics = pm
                }
                if ad, ok := data["assistant_duration"].(time.Duration); ok {
                    assistantDuration = ad
                }
                if pd, ok := data["parse_duration"].(time.Duration); ok {
                    parseDuration = pd
                }
            }
        }

        // 创建日志条目
        entry := audit.LogEntry{
            SessionID:         sessionID,
            InteractionID:     interactionID,
            UserID:            c.GetString("username"),
            ClientIP:          c.ClientIP(),
            UserAgent:         c.Request.UserAgent(),
            Question:          question,
            ModelName:         modelName,
            Provider:          provider,
            BaseURL:           baseURL,
            Cluster:           cluster,
            Thought:           thought,
            FinalAnswer:       finalAnswer,
            Status:            status,
            ToolCalls:         toolCalls,
            PerfMetrics:       perfMetrics,
            TotalDuration:     totalDuration,
            AssistantDuration: assistantDuration,
            ParseDuration:     parseDuration,
        }

        // 异步记录日志
        auditLogger.LogInteraction(entry)
    }
}
```

### 5.3 修改Execute处理函数

```go
// pkg/handlers/execute.go (修改部分)

// Execute 处理执行请求
func Execute(c *gin.Context) {
    // 获取性能统计工具
    perfStats := utils.GetPerfStats()
    // 开始整体执行计时
    defer perfStats.TraceFunc("execute_total")()

    // 获取 logger
    logger := utils.GetLogger()

    // ... 现有代码 ...

    // 调用 AI 助手
    response, chatHistory, err := assistants.AssistantWithConfig(executeModel, messages, 8192, true, true, defaultMaxIterations, apiKey, req.BaseUrl)

    // 停止 AI 助手执行计时
    assistantDuration := perfStats.StopTimer("execute_assistant")
    logger.Info("AI助手执行完成",
        zap.Duration("duration", assistantDuration),
    )

    // ... 现有代码 ...

    // 解析 AI 响应
    var aiResp AIResponse
    err = json.Unmarshal([]byte(response), &aiResp)

    // ... 现有代码 ...

    // 收集审计数据
    toolCalls := make([]audit.ToolCall, 0, len(toolsHistory))
    for i, tool := range toolsHistory {
        toolCalls = append(toolCalls, audit.ToolCall{
            Name:        tool.Name,
            Input:       tool.Input,
            Observation: tool.Observation,
            SequenceNum: i,
            // 工具调用耗时可能需要在工具执行时记录
            Duration:    0,
        })
    }

    // 收集性能指标
    perfMetrics := perfStats.GetTimers()

    // 准备审计数据
    auditData := map[string]interface{}{
        "question":           req.Instructions,
        "model_name":         executeModel,
        "provider":           req.Provider,
        "base_url":           req.BaseUrl,
        "cluster":            req.Cluster,
        "thought":            aiResp.Thought,
        "final_answer":       aiResp.FinalAnswer,
        "status":             "success",
        "tool_calls":         toolCalls,
        "perf_metrics":       perfMetrics,
        "assistant_duration": assistantDuration,
        "parse_duration":     parseDuration,
    }

    // 将审计数据存入上下文
    c.Set("audit_data", auditData)

    // ... 现有代码 ...
}
```

### 5.4 主程序初始化

```go
// cmd/kube-copilot/server.go (修改部分)

// 添加导入
import (
    "github.com/myysophia/OpsAgent/pkg/audit"
    "github.com/myysophia/OpsAgent/pkg/utils"
    // ... 其他导入 ...
)

// serverCmd 修改
var serverCmd = &cobra.Command{
    Use:   "server",
    Short: "Start the API server",
    Run: func(cmd *cobra.Command, args []string) {
        // 加载配置文件
        config, err := utils.LoadConfig("configs/config.yaml")
        if err != nil {
            fmt.Printf("加载配置文件失败: %v\n", err)
            os.Exit(1)
        }

        // 初始化日志
        initLogger(config)
        defer logger.Sync()

        logger.Info("启动服务器",
            zap.Int("port", config.Server.Port),
            zap.Bool("show-thought", config.Server.ShowThought),
        )

        // ... 现有代码 ...

        // 初始化审计日志记录器
        auditLogger, err := audit.NewAuditLogger(config)
        if err != nil {
            logger.Fatal("初始化审计日志记录器失败",
                zap.Error(err),
            )
        }
        defer auditLogger.Close()

        // 使用pkg/api/router.go中的Router函数
        r := api.Router(auditLogger)

        // 设置监听地址
        addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
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
```

### 5.5 修改路由设置

```go
// pkg/api/router.go (修改部分)

// 修改导入
import (
    "github.com/myysophia/OpsAgent/pkg/audit"
    // ... 其他导入 ...
)

// 修改Router函数签名
func Router(auditLogger *audit.AuditLogger) *gin.Engine {
    // ... 现有代码 ...

    // 添加审计中间件
    r.Use(middleware.AuditMiddleware(auditLogger))

    // ... 现有代码 ...

    return r
}
```

## 6. 部署与配置

### 6.1 数据库配置

1. 安装PostgreSQL：
   ```bash
   # Ubuntu/Debian
   sudo apt-get update
   sudo apt-get install postgresql postgresql-contrib

   # CentOS/RHEL
   sudo yum install postgresql-server postgresql-contrib
   sudo postgresql-setup initdb
   sudo systemctl start postgresql
   sudo systemctl enable postgresql
   ```

2. 创建数据库和用户：
   ```sql
   CREATE DATABASE opsagent;
   CREATE USER opsagent WITH ENCRYPTED PASSWORD 'your_secure_password';
   GRANT ALL PRIVILEGES ON DATABASE opsagent TO opsagent;
   ```

3. 创建表结构：
   ```bash
   psql -U opsagent -d opsagent -f schema.sql
   ```

### 6.2 应用配置

审计日志系统的配置已集成到 `configs/config.yaml` 文件中，示例配置如下：

```yaml
# 审计日志配置
audit:
  # 是否启用审计日志
  enabled: true

  # 数据库配置
  database:
    driver: "postgres"
    host: "localhost"
    port: 5432
    user: "opsagent"
    password: "your_secure_password"
    dbname: "opsagent"
    sslmode: "disable"
    max_open_conns: 10
    max_idle_conns: 5
    conn_max_lifetime: 1h

  # 异步处理配置
  async:
    # 工作协程数量
    workers: 5
    # 日志队列大小
    queue_size: 1000
    # 批处理大小
    batch_size: 10
    # 批处理间隔(毫秒)
    batch_interval_ms: 1000

  # 数据保留策略
  retention:
    # 数据保留天数，0表示永久保留
    days: 90
    # 是否启用自动清理
    auto_cleanup: true
    # 清理时间，格式为"小时:分钟"
    cleanup_time: "03:00"
```

启动OpsAgent服务时，系统将自动读取配置文件中的审计日志配置：

```bash
./kube-copilot server
```

## 7. 监控与维护

### 7.1 监控指标

1. 日志队列长度
2. 日志处理延迟
3. 数据库写入错误率
4. 数据库大小增长率

### 7.2 维护计划

1. 定期备份数据库
2. 设置数据保留策略，定期归档或清理旧数据
3. 监控数据库性能，必要时优化索引或查询

### 7.3 故障恢复

1. 如果数据库连接失败，日志记录器应尝试重新连接
2. 如果日志队列已满，应记录警告并丢弃最旧的日志条目
3. 定期检查并修复数据库一致性问题

## 8. 扩展与未来改进

### 8.1 可能的扩展

1. 添加更多分析维度，如用户行为模式、常见问题类型等
2. 实现实时分析仪表板，展示关键指标
3. 集成机器学习模型，自动识别异常模式或优化建议

### 8.2 未来改进

1. 支持分布式部署，使用外部消息队列（如Kafka、RabbitMQ）替代内存通道
2. 实现数据分片和时序分区，提高大规模数据处理能力
3. 添加更细粒度的权限控制，支持多角色访问审计数据

## 9. 结论

本设计方案提供了一个完整的解决方案，用于将OpsAgent问答日志记录到PostgreSQL数据库中，并通过异步处理确保不影响主要业务流程。该方案考虑了性能、可靠性和可扩展性等关键因素，为后续的数据分析和审计提供了坚实基础。
