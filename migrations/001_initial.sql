-- 测试管理服务数据库初始化脚本
-- SQLite版本

-- 测试分组表
CREATE TABLE IF NOT EXISTS test_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id VARCHAR(255) UNIQUE NOT NULL,
    parent_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE INDEX idx_test_groups_group_id ON test_groups(group_id);
CREATE INDEX idx_test_groups_parent_id ON test_groups(parent_id);

-- 测试案例表
CREATE TABLE IF NOT EXISTS test_cases (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_id VARCHAR(255) UNIQUE NOT NULL,
    group_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,              -- http, command, integration, performance, etc.
    priority VARCHAR(10),                    -- P0, P1, P2
    status VARCHAR(50) DEFAULT 'active',     -- active, inactive, deprecated
    objective TEXT,
    timeout INTEGER DEFAULT 300,             -- seconds

    -- JSON配置字段
    preconditions TEXT,      -- JSON array
    steps TEXT,              -- JSON array
    http_config TEXT,        -- JSON object
    command_config TEXT,     -- JSON object
    integration_config TEXT, -- JSON object
    performance_config TEXT, -- JSON object
    database_config TEXT,    -- JSON object
    security_config TEXT,    -- JSON object
    grpc_config TEXT,        -- JSON object
    websocket_config TEXT,   -- JSON object
    e2e_config TEXT,         -- JSON object
    assertions TEXT,         -- JSON array
    tags TEXT,               -- JSON array
    custom_config TEXT,      -- JSON object for extensibility

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,

    FOREIGN KEY (group_id) REFERENCES test_groups(group_id) ON DELETE CASCADE
);

CREATE INDEX idx_test_cases_test_id ON test_cases(test_id);
CREATE INDEX idx_test_cases_group_id ON test_cases(group_id);
CREATE INDEX idx_test_cases_type ON test_cases(type);
CREATE INDEX idx_test_cases_priority ON test_cases(priority);
CREATE INDEX idx_test_cases_status ON test_cases(status);

-- 测试执行结果表
CREATE TABLE IF NOT EXISTS test_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_id VARCHAR(255) NOT NULL,
    run_id VARCHAR(255),                     -- 批量执行ID
    status VARCHAR(50) NOT NULL,             -- passed, failed, error, skipped
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration INTEGER,                        -- milliseconds
    error TEXT,
    failures TEXT,                           -- JSON array
    metrics TEXT,                            -- JSON object
    artifacts TEXT,                          -- JSON array
    logs TEXT,                               -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (test_id) REFERENCES test_cases(test_id) ON DELETE CASCADE
);

CREATE INDEX idx_test_results_test_id ON test_results(test_id);
CREATE INDEX idx_test_results_run_id ON test_results(run_id);
CREATE INDEX idx_test_results_status ON test_results(status);
CREATE INDEX idx_test_results_start_time ON test_results(start_time);

-- 测试执行批次表
CREATE TABLE IF NOT EXISTS test_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    total INTEGER DEFAULT 0,
    passed INTEGER DEFAULT 0,
    failed INTEGER DEFAULT 0,
    errors INTEGER DEFAULT 0,
    skipped INTEGER DEFAULT 0,
    start_time DATETIME,
    end_time DATETIME,
    duration INTEGER,                        -- milliseconds
    status VARCHAR(50) DEFAULT 'running',    -- running, completed, cancelled
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_test_runs_run_id ON test_runs(run_id);
CREATE INDEX idx_test_runs_status ON test_runs(status);
CREATE INDEX idx_test_runs_start_time ON test_runs(start_time);

-- 插入默认根分组
INSERT OR IGNORE INTO test_groups (group_id, name, description)
VALUES ('root', '根分组', '测试案例根分组');
