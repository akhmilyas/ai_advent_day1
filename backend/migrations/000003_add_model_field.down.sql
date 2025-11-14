-- Remove model column from messages
ALTER TABLE messages
DROP COLUMN IF EXISTS model;
