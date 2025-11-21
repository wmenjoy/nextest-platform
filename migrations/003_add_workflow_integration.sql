-- Migration: Add workflow integration support
-- Purpose: Enable bidirectional integration between test cases and workflows
-- Date: 2025-11-21
-- Related: testcase-workflow-integration.md, PRD.md (功能3), USER-STORIES.md (Story 2.15-2.20)

-- ========================================
-- Part 1: Extend test_cases table for workflow integration
-- ========================================

-- Add workflow integration columns to test_cases
-- Mode 1: WorkflowID - Reference to standalone workflow
-- Mode 2: WorkflowDef - Embedded workflow definition
ALTER TABLE test_cases ADD COLUMN workflow_id VARCHAR(255) DEFAULT NULL;
ALTER TABLE test_cases ADD COLUMN workflow_def TEXT DEFAULT NULL;

-- Create index on workflow_id for fast lookup of test cases referencing a workflow
CREATE INDEX idx_test_cases_workflow_id ON test_cases(workflow_id);

-- ========================================
-- Part 2: Create workflows table
-- ========================================

-- Main workflow definition table
CREATE TABLE IF NOT EXISTS workflows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(32),
    description TEXT,
    definition TEXT NOT NULL,            -- JSONB: Complete workflow definition (steps, variables, etc.)
    is_test_case BOOLEAN DEFAULT 0,      -- Flag: Is this workflow referenced by test cases?
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    created_by VARCHAR(64)
);

CREATE INDEX idx_workflows_workflow_id ON workflows(workflow_id);
CREATE INDEX idx_workflows_is_test_case ON workflows(is_test_case);
CREATE INDEX idx_workflows_deleted_at ON workflows(deleted_at);

-- ========================================
-- Part 3: Create workflow execution tracking tables
-- ========================================

-- Workflow execution records (top-level tracking)
CREATE TABLE IF NOT EXISTS workflow_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id VARCHAR(255) UNIQUE NOT NULL,
    workflow_id VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,         -- running, success, failed, cancelled
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration INTEGER,                    -- milliseconds
    context TEXT,                        -- JSONB: Execution context (variables, step results)
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (workflow_id) REFERENCES workflows(workflow_id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_runs_run_id ON workflow_runs(run_id);
CREATE INDEX idx_workflow_runs_workflow_id ON workflow_runs(workflow_id);
CREATE INDEX idx_workflow_runs_status ON workflow_runs(status);
CREATE INDEX idx_workflow_runs_start_time ON workflow_runs(start_time);

-- Step-level execution tracking (for real-time data flow monitoring)
CREATE TABLE IF NOT EXISTS workflow_step_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id VARCHAR(255) NOT NULL,
    step_id VARCHAR(255) NOT NULL,
    step_name VARCHAR(255),
    status VARCHAR(32) NOT NULL,         -- pending, running, success, failed, skipped
    start_time DATETIME,
    end_time DATETIME,
    duration INTEGER,                    -- milliseconds
    input_data TEXT,                     -- JSONB: Input data snapshot (for data flow visualization)
    output_data TEXT,                    -- JSONB: Output data snapshot (for data flow visualization)
    error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_step_executions_run_id ON workflow_step_executions(run_id);
CREATE INDEX idx_workflow_step_executions_step_id ON workflow_step_executions(step_id);

-- Step execution logs (structured logging)
CREATE TABLE IF NOT EXISTS workflow_step_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id VARCHAR(255) NOT NULL,
    step_id VARCHAR(255) NOT NULL,
    level VARCHAR(16) NOT NULL,          -- debug, info, warn, error
    message TEXT NOT NULL,
    timestamp DATETIME NOT NULL,

    FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_step_logs_run_id ON workflow_step_logs(run_id);
CREATE INDEX idx_workflow_step_logs_step_id ON workflow_step_logs(step_id);
CREATE INDEX idx_workflow_step_logs_timestamp ON workflow_step_logs(timestamp);

-- Variable change history (for debugging and auditing)
CREATE TABLE IF NOT EXISTS workflow_variable_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id VARCHAR(255) NOT NULL,
    step_id VARCHAR(255),                -- Which step triggered this change
    var_name VARCHAR(255) NOT NULL,
    old_value TEXT,                      -- JSONB: Previous value
    new_value TEXT,                      -- JSONB: New value
    change_type VARCHAR(16) NOT NULL,    -- create, update, delete
    timestamp DATETIME NOT NULL,

    FOREIGN KEY (run_id) REFERENCES workflow_runs(run_id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_variable_changes_run_id ON workflow_variable_changes(run_id);
CREATE INDEX idx_workflow_variable_changes_var_name ON workflow_variable_changes(var_name);
CREATE INDEX idx_workflow_variable_changes_timestamp ON workflow_variable_changes(timestamp);

-- ========================================
-- Part 4: Link test results to workflow runs
-- ========================================

-- Add workflow_run_id to test_results to link test execution with workflow execution
ALTER TABLE test_results ADD COLUMN workflow_run_id VARCHAR(255) DEFAULT NULL;

CREATE INDEX idx_test_results_workflow_run_id ON test_results(workflow_run_id);

-- ========================================
-- Migration Notes:
-- ========================================
-- 1. All new columns are nullable for backward compatibility
-- 2. Existing test cases remain fully functional without modification
-- 3. Three integration modes supported:
--    - Mode 1: test_cases.workflow_id references workflows.workflow_id
--    - Mode 2: test_cases.workflow_def contains embedded workflow definition
--    - Mode 3: workflows can reference test cases via TestCaseAction steps
-- 4. Real-time monitoring infrastructure includes:
--    - workflow_step_executions: Input/output data snapshots
--    - workflow_step_logs: Structured logging per step
--    - workflow_variable_changes: Variable mutation tracking
-- 5. Foreign keys ensure cascading deletes for data integrity
