-- Add status field to emails_metadata to track classification status
ALTER TABLE emails_metadata 
ADD COLUMN IF NOT EXISTS status VARCHAR(50) NOT NULL DEFAULT 'success';

-- Update existing records to have 'success' status
UPDATE emails_metadata SET status = 'success' WHERE status IS NULL;

-- Add index for status queries
CREATE INDEX IF NOT EXISTS idx_emails_metadata_status 
    ON emails_metadata(status);

