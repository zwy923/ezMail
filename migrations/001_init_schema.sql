-- ==========================================================
-- 000_full_schema.sql (Phase 2 Ready)
-- Includes: users, emails, metadata, tasks, notifications,
--           notification logs, failed events
-- ==========================================================


-- ==============================
-- Users
-- ==============================
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- ==============================
-- Email status enum
-- ==============================
DO $$ BEGIN
    CREATE TYPE email_status AS ENUM ('received', 'classified');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;


-- ==============================
-- Raw incoming emails
-- ==============================
CREATE TABLE IF NOT EXISTS emails_raw (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    raw_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    status email_status NOT NULL DEFAULT 'received',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- ==============================
-- Agent Metadata (categories + summary + priority)
-- ==============================
CREATE TABLE IF NOT EXISTS emails_metadata (
    email_id INT PRIMARY KEY REFERENCES emails_raw(id) ON DELETE CASCADE,

    categories TEXT[] NOT NULL,         -- ["WORK","ACTION_REQUIRED"]
    priority TEXT NOT NULL,             -- LOW / MEDIUM / HIGH
    summary TEXT NOT NULL,              -- short 1-3 sentence summary

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- ==============================
-- Tasks (Agent-created tasks)
-- ==============================
CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,

    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,

    title TEXT NOT NULL,
    due_date DATE,
    status TEXT NOT NULL DEFAULT 'pending',    -- pending / done

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_user ON tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_tasks_email ON tasks(email_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);


-- ==============================
-- Notification Inbox (user notifications)
-- ==============================
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,

    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,

    channel TEXT NOT NULL,         -- EMAIL / PUSH / SMS
    message TEXT NOT NULL,         -- "You have an urgent email"
    is_read BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_email ON notifications(email_id);
CREATE INDEX IF NOT EXISTS idx_notifications_is_read ON notifications(is_read);


-- ==============================
-- Notification Log (audit logs)
-- ==============================
CREATE TABLE IF NOT EXISTS notifications_log (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,

    message TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_log_user ON notifications_log(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_log_email ON notifications_log(email_id);


-- ==============================
-- Failed Events (MQ failures)
-- ==============================
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

CREATE INDEX IF NOT EXISTS idx_failed_events_status ON failed_events(status);
CREATE INDEX IF NOT EXISTS idx_failed_events_email ON failed_events(email_id);
CREATE INDEX IF NOT EXISTS idx_failed_events_pending_retry
    ON failed_events(status, retry_count) WHERE status = 'pending';


-- ==============================
-- Extra useful indexes
-- ==============================
CREATE INDEX IF NOT EXISTS idx_emails_raw_user ON emails_raw(user_id);
CREATE INDEX IF NOT EXISTS idx_emails_raw_status ON emails_raw(status);