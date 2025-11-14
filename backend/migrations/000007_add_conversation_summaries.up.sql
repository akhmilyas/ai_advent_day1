-- Create conversation_summaries table
CREATE TABLE conversation_summaries (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    summary_content TEXT NOT NULL,
    summarized_up_to_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_summaries_conversation_id ON conversation_summaries(conversation_id);

-- Add active_summary_id column to conversations table
ALTER TABLE conversations
ADD COLUMN active_summary_id UUID REFERENCES conversation_summaries(id) ON DELETE SET NULL;
