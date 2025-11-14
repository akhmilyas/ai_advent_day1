-- Remove response_format and response_schema columns from conversations
ALTER TABLE conversations
DROP COLUMN IF EXISTS response_schema,
DROP COLUMN IF EXISTS response_format;
