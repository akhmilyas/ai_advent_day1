-- Add model column to messages
ALTER TABLE messages
ADD COLUMN model VARCHAR(255);
