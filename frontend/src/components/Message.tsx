import React from 'react';
import { getTheme } from '../themes';
import { ResponseFormat } from './SettingsModal';
import { JsonMessage } from './Message/JsonMessage';
import { XmlMessage } from './Message/XmlMessage';
import { TextMessage } from './Message/TextMessage';
import { MessageMeta } from './Message/MessageMeta';

interface MessageProps {
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
  conversationFormat: ResponseFormat | undefined;
  colors: ReturnType<typeof getTheme>;
}

export const Message: React.FC<MessageProps> = ({
  role,
  content,
  model,
  temperature,
  promptTokens,
  completionTokens,
  totalTokens,
  totalCost,
  latency,
  generationTime,
  conversationFormat,
  colors
}) => {
  const styles = getStyles(colors);

  return (
    <div
      style={{
        ...styles.message,
        ...(role === 'user'
          ? {
              ...styles.userMessage,
              backgroundColor: colors.userMessageBg,
              color: colors.userMessageText,
            }
          : {
              ...styles.assistantMessage,
              backgroundColor: colors.assistantMessageBg,
              borderColor: colors.assistantMessageBorder,
              color: colors.assistantMessageText,
            }),
      }}
    >
      <div style={{ ...styles.messageRole, opacity: 0.7 }}>
        {role === 'user' ? 'You' : 'AI'}
        {role === 'assistant' && (model || temperature !== undefined) && (
          <span style={{ fontSize: '11px', marginLeft: '8px', opacity: 0.6 }}>
            ({model && <>{model}</>}{model && temperature !== undefined && ', '}{temperature !== undefined && <>temp: {temperature.toFixed(2)}</>})
          </span>
        )}
      </div>
      <div style={role === 'assistant' ? styles.assistantContent : styles.messageContent}>
        {role === 'assistant' ? (
          conversationFormat === 'json' ? (
            <JsonMessage content={content} colors={colors} />
          ) : conversationFormat === 'xml' ? (
            <XmlMessage content={content} colors={colors} />
          ) : (
            <TextMessage content={content} />
          )
        ) : (
          content
        )}
      </div>
      {role === 'assistant' && (
        <MessageMeta
          promptTokens={promptTokens}
          completionTokens={completionTokens}
          totalTokens={totalTokens}
          totalCost={totalCost}
          latency={latency}
          generationTime={generationTime}
          colors={colors}
        />
      )}
    </div>
  );
};

const getStyles = (colors: ReturnType<typeof getTheme>) => ({
  message: {
    padding: '12px 16px',
    borderRadius: '8px',
    maxWidth: '70%',
    transition: 'background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease',
  },
  userMessage: {
    alignSelf: 'flex-end',
    border: 'none',
  },
  assistantMessage: {
    alignSelf: 'flex-start',
    border: '1px solid',
  },
  messageRole: {
    fontSize: '12px',
    fontWeight: 'bold' as const,
    marginBottom: '4px',
  },
  messageContent: {
    fontSize: '16px',
    lineHeight: '1.5',
    whiteSpace: 'pre-wrap' as const,
  },
  assistantContent: {
    fontSize: '16px',
    lineHeight: '1.6',
  },
});
