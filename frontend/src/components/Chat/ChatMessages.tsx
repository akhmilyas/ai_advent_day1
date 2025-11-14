import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { getTheme } from '../../themes';
import { Message } from '../Message';
import { ResponseFormat } from '../SettingsModal';

interface ChatMessage {
  id?: string;
  role: 'user' | 'assistant';
  content: string;
  model?: string;
  temperature?: number;
  promptTokens?: number;
  completionTokens?: number;
  totalTokens?: number;
  totalCost?: number;
  latency?: number;
  generationTime?: number;
}

interface Summary {
  upToMessageId: string;
  content: string;
}

interface ChatMessagesProps {
  messages: ChatMessage[];
  summaries: Summary[];
  conversationFormat: ResponseFormat | null;
  messagesEndRef: React.RefObject<HTMLDivElement>;
}

export const ChatMessages: React.FC<ChatMessagesProps> = ({
  messages,
  summaries,
  conversationFormat,
  messagesEndRef,
}) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  return (
    <div style={{
      flex: 1,
      overflowY: 'auto' as const,
      padding: '20px',
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '16px',
      transition: 'background-color 0.3s ease',
      backgroundColor: colors.background,
    }}>
      {messages.length === 0 && (
        <div style={{
          textAlign: 'center' as const,
          marginTop: '100px',
          transition: 'color 0.3s ease',
          color: colors.emptyStateText,
        }}>
          <p>Start a conversation by typing a message below</p>
        </div>
      )}

      {messages.map((msg, idx) => {
        // Find if this message is the end of a summary
        const summaryForThisMessage = summaries.find(s => s.upToMessageId === msg.id);

        return (
          <React.Fragment key={idx}>
            <Message
              role={msg.role}
              content={msg.content}
              model={'model' in msg ? msg.model : undefined}
              temperature={'temperature' in msg ? msg.temperature : undefined}
              promptTokens={'promptTokens' in msg ? msg.promptTokens : undefined}
              completionTokens={'completionTokens' in msg ? msg.completionTokens : undefined}
              totalTokens={'totalTokens' in msg ? msg.totalTokens : undefined}
              totalCost={'totalCost' in msg ? msg.totalCost : undefined}
              latency={'latency' in msg ? msg.latency : undefined}
              generationTime={'generationTime' in msg ? msg.generationTime : undefined}
              conversationFormat={conversationFormat}
              colors={colors}
            />
            {/* Show summary divider after the last summarized message */}
            {summaryForThisMessage && (
              <div
                style={{
                  margin: '20px 0',
                  padding: '15px',
                  backgroundColor: colors.surface,
                  borderRadius: '8px',
                  border: `2px dashed ${colors.border}`,
                  color: colors.textSecondary,
                }}
              >
                <details>
                  <summary style={{ cursor: 'pointer', fontStyle: 'italic', userSelect: 'none' }}>
                    📋 Messages above have been summarized (click to view)
                  </summary>
                  <div style={{
                    marginTop: '10px',
                    padding: '10px',
                    backgroundColor: colors.background,
                    borderRadius: '4px',
                    whiteSpace: 'pre-wrap',
                    fontSize: '0.9em',
                    color: colors.text,
                  }}>
                    {summaryForThisMessage.content || 'No summary content available'}
                  </div>
                </details>
              </div>
            )}
          </React.Fragment>
        );
      })}

      <div ref={messagesEndRef} />
    </div>
  );
};
