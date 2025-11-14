-- Remove provider column from messages
ALTER TABLE messages
DROP COLUMN IF EXISTS provider;
