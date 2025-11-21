-- Migration: Add lifecycle hooks to test_cases and test_groups
-- Purpose: Support setup/teardown hooks for test lifecycle management
-- Date: 2025-01-20

-- Add lifecycle hook columns to test_cases table
ALTER TABLE test_cases ADD COLUMN setup_hooks TEXT;
ALTER TABLE test_cases ADD COLUMN teardown_hooks TEXT;

-- Add lifecycle hook columns to test_groups table
ALTER TABLE test_groups ADD COLUMN setup_hooks TEXT;
ALTER TABLE test_groups ADD COLUMN teardown_hooks TEXT;

-- Update existing records to have empty arrays for hooks (NULL -> '[]')
UPDATE test_cases SET setup_hooks = '[]' WHERE setup_hooks IS NULL;
UPDATE test_cases SET teardown_hooks = '[]' WHERE teardown_hooks IS NULL;
UPDATE test_groups SET setup_hooks = '[]' WHERE setup_hooks IS NULL;
UPDATE test_groups SET teardown_hooks = '[]' WHERE teardown_hooks IS NULL;
