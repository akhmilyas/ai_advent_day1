-- Remove temperature column from messages
ALTER TABLE messages
DROP COLUMN IF EXISTS temperature;
