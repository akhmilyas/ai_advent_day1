-- Add provider column to messages
ALTER TABLE messages
ADD COLUMN provider VARCHAR(50);
