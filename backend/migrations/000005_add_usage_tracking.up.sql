-- Add usage tracking columns to messages
ALTER TABLE messages
ADD COLUMN generation_id VARCHAR(255),
ADD COLUMN prompt_tokens INTEGER,
ADD COLUMN completion_tokens INTEGER,
ADD COLUMN total_tokens INTEGER,
ADD COLUMN total_cost REAL,
ADD COLUMN latency INTEGER,
ADD COLUMN generation_time INTEGER;
