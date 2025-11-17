-- ==========================================================
-- 001_init_schema.sql
-- Initial schema for user auth, emails, metadata, notifications & logs
-- ==========================================================

-- ==============================
-- Users table
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
-- Classification metadata
-- ==============================
CREATE TABLE IF NOT EXISTS emails_metadata (
    id SERIAL PRIMARY KEY,
    email_id INT NOT NULL UNIQUE REFERENCES emails_raw(id) ON DELETE CASCADE,
    category VARCHAR(255) NOT NULL,
    confidence FLOAT NOT NULL DEFAULT 1.0,
    status VARCHAR(50) NOT NULL DEFAULT 'success',  -- merged field
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- ==============================
-- Notification Log (audit logs)
-- ==============================
CREATE TABLE IF NOT EXISTS notifications_log (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,
    message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- ==============================
-- User Notifications (inbox)
-- ==============================
CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- ==============================
-- Useful indexes
-- ==============================

-- For emails_raw
CREATE INDEX IF NOT EXISTS idx_emails_raw_user
    ON emails_raw(user_id);

-- For emails_metadata
CREATE INDEX IF NOT EXISTS idx_emails_metadata_email
    ON emails_metadata(email_id);

CREATE INDEX IF NOT EXISTS idx_emails_metadata_status
    ON emails_metadata(status);

-- For notifications_log
CREATE INDEX IF NOT EXISTS idx_notifications_log_user
    ON notifications_log(user_id);

CREATE INDEX IF NOT EXISTS idx_notifications_log_email
    ON notifications_log(email_id);

-- For notifications
CREATE INDEX IF NOT EXISTS idx_notifications_user
    ON notifications(user_id);
