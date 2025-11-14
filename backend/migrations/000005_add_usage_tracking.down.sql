-- Remove usage tracking columns from messages
ALTER TABLE messages
DROP COLUMN IF EXISTS generation_time,
DROP COLUMN IF EXISTS latency,
DROP COLUMN IF EXISTS total_cost,
DROP COLUMN IF EXISTS total_tokens,
DROP COLUMN IF EXISTS completion_tokens,
DROP COLUMN IF EXISTS prompt_tokens,
DROP COLUMN IF EXISTS generation_id;
