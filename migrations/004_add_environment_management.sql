-- Environment Management Migration
-- SQLite version
-- This migration adds environment management support for managing test environments
-- and their associated variables (Dev/Staging/Prod, etc.)

-- Environments table
-- Stores test environment configurations that can be activated for test execution
CREATE TABLE IF NOT EXISTS environments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    env_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT 0 NOT NULL,  -- SQLite uses 0/1 for boolean
    variables TEXT,                        -- JSON object for environment-level metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

-- Indexes for environments table
CREATE INDEX idx_environments_env_id ON environments(env_id);
CREATE INDEX idx_environments_is_active ON environments(is_active);
CREATE INDEX idx_environments_deleted_at ON environments(deleted_at);

-- Environment Variables table
-- Stores individual environment variables with type and secret support
CREATE TABLE IF NOT EXISTS environment_variables (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    env_id VARCHAR(50) NOT NULL,
    key VARCHAR(255) NOT NULL,
    value TEXT,
    value_type VARCHAR(50) DEFAULT 'string',  -- string, number, boolean, json
    is_secret BOOLEAN DEFAULT 0,              -- SQLite uses 0/1 for boolean
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (env_id) REFERENCES environments(env_id) ON DELETE CASCADE
);

-- Indexes for environment_variables table
CREATE INDEX idx_environment_variables_env_id ON environment_variables(env_id);
CREATE INDEX idx_environment_variables_key ON environment_variables(key);

-- Insert default environments with sample variables
-- Development Environment
INSERT INTO environments (env_id, name, description, is_active, variables)
VALUES (
    'dev',
    'Development',
    'Development environment for local testing',
    0,
    '{"color":"blue","priority":"low"}'
);

INSERT INTO environment_variables (env_id, key, value, value_type, is_secret)
VALUES
    ('dev', 'BASE_URL', 'http://localhost:3000', 'string', 0),
    ('dev', 'API_TIMEOUT', '5000', 'number', 0),
    ('dev', 'DEBUG_MODE', 'true', 'boolean', 0),
    ('dev', 'API_KEY', 'dev-secret-key-12345', 'string', 1);

-- Staging Environment
INSERT INTO environments (env_id, name, description, is_active, variables)
VALUES (
    'staging',
    'Staging',
    'Pre-production staging environment',
    0,
    '{"color":"yellow","priority":"medium"}'
);

INSERT INTO environment_variables (env_id, key, value, value_type, is_secret)
VALUES
    ('staging', 'BASE_URL', 'https://staging.example.com', 'string', 0),
    ('staging', 'API_TIMEOUT', '10000', 'number', 0),
    ('staging', 'DEBUG_MODE', 'false', 'boolean', 0),
    ('staging', 'API_KEY', 'staging-secret-key-67890', 'string', 1);

-- Production Environment
INSERT INTO environments (env_id, name, description, is_active, variables)
VALUES (
    'prod',
    'Production',
    'Production environment',
    1,
    '{"color":"red","priority":"high"}'
);

INSERT INTO environment_variables (env_id, key, value, value_type, is_secret)
VALUES
    ('prod', 'BASE_URL', 'https://api.example.com', 'string', 0),
    ('prod', 'API_TIMEOUT', '30000', 'number', 0),
    ('prod', 'DEBUG_MODE', 'false', 'boolean', 0),
    ('prod', 'API_KEY', 'prod-secret-key-abcdef', 'string', 1);

-- ============================================================
-- ROLLBACK INSTRUCTIONS
-- ============================================================
-- To rollback this migration, execute the following commands:
--
-- DROP INDEX IF EXISTS idx_environment_variables_key;
-- DROP INDEX IF EXISTS idx_environment_variables_env_id;
-- DROP TABLE IF EXISTS environment_variables;
-- DROP INDEX IF EXISTS idx_environments_deleted_at;
-- DROP INDEX IF EXISTS idx_environments_is_active;
-- DROP INDEX IF EXISTS idx_environments_env_id;
-- DROP TABLE IF EXISTS environments;
-- ============================================================
