-- ==========================================================
-- 001_init_schema.sql
-- Initial database schema for MyGoProject
-- Includes: users, emails, metadata, tasks, notifications,
--           notification logs, failed events
-- ==========================================================

-- ==============================
-- Custom Types
-- ==============================
DO $$ BEGIN
    CREATE TYPE email_status AS ENUM ('received', 'classified');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- ==============================
-- Core Tables
-- ==============================

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Raw incoming emails
CREATE TABLE IF NOT EXISTS emails_raw (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    raw_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status email_status NOT NULL DEFAULT 'received',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Agent Metadata (categories + summary + priority)
CREATE TABLE IF NOT EXISTS emails_metadata (
    email_id INT PRIMARY KEY REFERENCES emails_raw(id) ON DELETE CASCADE,
    categories TEXT[] NOT NULL,         -- ["WORK","ACTION_REQUIRED"]
    priority TEXT NOT NULL,             -- LOW / MEDIUM / HIGH
    summary TEXT NOT NULL,              -- short 1-3 sentence summary
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Habits (Recurring tasks)
CREATE TABLE IF NOT EXISTS habits (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    recurrence_pattern VARCHAR(100) NOT NULL, -- "weekly Wednesday", "daily", "monthly 1"
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Projects (AI-generated project plans)
CREATE TABLE IF NOT EXISTS projects (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    target_date DATE, -- Project deadline
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active / completed / cancelled
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Milestones (Project phases)
CREATE TABLE IF NOT EXISTS milestones (
    id SERIAL PRIMARY KEY,
    project_id INT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    phase_order INT NOT NULL, -- Order of phase (1, 2, 3, ...)
    target_date DATE, -- Milestone deadline
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending / in_progress / completed
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Tasks (Agent-created tasks, habit-generated tasks, and project tasks)
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT DEFAULT NULL REFERENCES emails_raw(id) ON DELETE CASCADE, -- NULL for habit/project-generated tasks
    habit_id INT DEFAULT NULL REFERENCES habits(id) ON DELETE CASCADE, -- NULL for one-time/project tasks
    project_id INT DEFAULT NULL REFERENCES projects(id) ON DELETE CASCADE, -- NULL for non-project tasks
    milestone_id INT DEFAULT NULL REFERENCES milestones(id) ON DELETE CASCADE, -- NULL for tasks not in a milestone
    title VARCHAR(255) NOT NULL,
    due_date DATE,
    priority VARCHAR(20) DEFAULT 'MEDIUM', -- LOW / MEDIUM / HIGH
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending / done / overdue
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Task Dependencies (Task prerequisites)
CREATE TABLE IF NOT EXISTS task_dependencies (
    id SERIAL PRIMARY KEY,
    task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    depends_on_task_id INT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT task_dependencies_no_self_reference CHECK (task_id != depends_on_task_id)
);

-- Notification Inbox (user notifications)
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,         -- EMAIL / PUSH / SMS
    message TEXT NOT NULL,         -- "You have an urgent email"
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Notification Log (audit logs)
CREATE TABLE IF NOT EXISTS notifications_log (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Failed Events (MQ failures)
CREATE TABLE IF NOT EXISTS failed_events (
    id SERIAL PRIMARY KEY,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    routing_key VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    error_message TEXT,
    retry_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',   -- pending / retried / failed
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ==============================
-- Indexes
-- ==============================

-- Users indexes (if needed in future)
-- Currently no indexes needed for users table

-- Emails indexes
CREATE INDEX IF NOT EXISTS idx_emails_raw_user ON emails_raw(user_id);
CREATE INDEX IF NOT EXISTS idx_emails_raw_status ON emails_raw(status);

-- Habits indexes
CREATE INDEX IF NOT EXISTS idx_habits_user ON habits(user_id);
CREATE INDEX IF NOT EXISTS idx_habits_active ON habits(is_active) WHERE is_active = TRUE;

-- Projects indexes
CREATE INDEX IF NOT EXISTS idx_projects_user ON projects(user_id);
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);

-- Milestones indexes
CREATE INDEX IF NOT EXISTS idx_milestones_project ON milestones(project_id);
CREATE INDEX IF NOT EXISTS idx_milestones_status ON milestones(status);
CREATE INDEX IF NOT EXISTS idx_milestones_order ON milestones(project_id, phase_order);

-- Tasks indexes
CREATE INDEX IF NOT EXISTS idx_tasks_user ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_habit ON tasks(habit_id);
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_milestone ON tasks(milestone_id);
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks(due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);

-- Task Dependencies indexes
CREATE INDEX IF NOT EXISTS idx_task_dependencies_task ON task_dependencies(task_id);
CREATE INDEX IF NOT EXISTS idx_task_dependencies_depends_on ON task_dependencies(depends_on_task_id);

-- Unique constraint: only one pending task per email_id + user_id combination
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_unique_pending_email_user
    ON tasks(email_id, user_id) WHERE status = 'pending' AND email_id IS NOT NULL;

-- Unique constraint: only one pending task per habit_id + due_date (幂等性：避免重复生成同一天的重复任务)
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_unique_pending_habit_date
    ON tasks(habit_id, due_date) WHERE status = 'pending' AND habit_id IS NOT NULL;

-- Notifications indexes
CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_email ON notifications(email_id);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read);

-- Notification logs indexes
CREATE INDEX IF NOT EXISTS idx_notifications_log_user ON notifications_log(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_log_email ON notifications_log(email_id);

-- Failed events indexes
CREATE INDEX IF NOT EXISTS idx_failed_events_status ON failed_events(status);
CREATE INDEX IF NOT EXISTS idx_failed_events_email ON failed_events(email_id);
CREATE INDEX IF NOT EXISTS idx_failed_events_pending_retry
    ON failed_events(status, retry_count) WHERE status = 'pending';
