-- OpsAgent 问答日志审计系统数据库结构

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

-- 添加注释
COMMENT ON TABLE sessions IS '用户会话信息表';
COMMENT ON TABLE interactions IS '用户与LLM的问答交互记录表';
COMMENT ON TABLE thoughts IS 'LLM思考过程记录表';
COMMENT ON TABLE tool_calls IS '工具调用记录表';
COMMENT ON TABLE performance_metrics IS '性能指标记录表';

-- 添加字段注释
COMMENT ON COLUMN sessions.session_id IS '会话唯一标识符';
COMMENT ON COLUMN sessions.user_id IS '用户标识符';
COMMENT ON COLUMN sessions.client_ip IS '客户端IP地址';
COMMENT ON COLUMN sessions.user_agent IS '用户代理信息';
COMMENT ON COLUMN sessions.created_at IS '会话创建时间';

COMMENT ON COLUMN interactions.interaction_id IS '交互唯一标识符';
COMMENT ON COLUMN interactions.session_id IS '关联的会话ID';
COMMENT ON COLUMN interactions.question IS '用户提问内容';
COMMENT ON COLUMN interactions.model_name IS '使用的模型名称';
COMMENT ON COLUMN interactions.provider IS '模型提供商';
COMMENT ON COLUMN interactions.base_url IS 'API基础URL';
COMMENT ON COLUMN interactions.cluster IS '集群信息';
COMMENT ON COLUMN interactions.final_answer IS 'LLM最终回答';
COMMENT ON COLUMN interactions.status IS '交互状态';
COMMENT ON COLUMN interactions.total_duration_ms IS '总处理时间(毫秒)';
COMMENT ON COLUMN interactions.assistant_duration_ms IS 'LLM处理时间(毫秒)';
COMMENT ON COLUMN interactions.parse_duration_ms IS '解析处理时间(毫秒)';

COMMENT ON COLUMN thoughts.interaction_id IS '关联的交互ID';
COMMENT ON COLUMN thoughts.thought IS 'LLM思考过程内容';

COMMENT ON COLUMN tool_calls.interaction_id IS '关联的交互ID';
COMMENT ON COLUMN tool_calls.tool_name IS '工具名称';
COMMENT ON COLUMN tool_calls.tool_input IS '工具输入参数';
COMMENT ON COLUMN tool_calls.tool_observation IS '工具执行结果';
COMMENT ON COLUMN tool_calls.sequence_number IS '工具调用序号';
COMMENT ON COLUMN tool_calls.duration_ms IS '工具执行时间(毫秒)';

COMMENT ON COLUMN performance_metrics.interaction_id IS '关联的交互ID';
COMMENT ON COLUMN performance_metrics.metric_name IS '指标名称';
COMMENT ON COLUMN performance_metrics.duration_ms IS '指标值(毫秒)';
