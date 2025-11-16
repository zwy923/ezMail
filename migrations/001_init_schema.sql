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
    status VARCHAR(50) NOT NULL DEFAULT 'received', -- 'received' or 'classified'
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Classification metadata
CREATE TABLE IF NOT EXISTS emails_metadata (
    id SERIAL PRIMARY KEY,
    email_id INT NOT NULL REFERENCES emails_raw(id) ON DELETE CASCADE,
    category VARCHAR(255) NOT NULL,
    confidence FLOAT NOT NULL DEFAULT 1.0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Useful index for querying classifications
CREATE INDEX IF NOT EXISTS idx_emails_raw_user ON emails_raw(user_id);
CREATE INDEX IF NOT EXISTS idx_emails_metadata_email ON emails_metadata(email_id);
