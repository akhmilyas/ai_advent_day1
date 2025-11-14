-- Add response_format and response_schema columns to conversations
ALTER TABLE conversations
ADD COLUMN response_format VARCHAR(10) DEFAULT 'text',
ADD COLUMN response_schema TEXT;
