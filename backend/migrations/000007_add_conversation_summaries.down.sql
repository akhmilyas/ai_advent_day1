-- Remove active_summary_id column from conversations table
ALTER TABLE conversations
DROP COLUMN IF EXISTS active_summary_id;

-- Drop conversation_summaries table
DROP TABLE IF EXISTS conversation_summaries CASCADE;
