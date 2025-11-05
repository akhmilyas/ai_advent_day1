import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { getTheme } from '../themes';
import { ResponseFormat } from './SettingsModal';

interface MessageProps {
  role: 'user' | 'assistant';
  content: string;
  conversationFormat: ResponseFormat | null;
  colors: ReturnType<typeof getTheme>;
}

// Helper function to render JSON as a table
const renderJsonAsTable = (jsonString: string, colors: ReturnType<typeof getTheme>) => {
  try {
    const parsed = JSON.parse(jsonString);
    const entries = Object.entries(parsed);

    return (
      <div>
        {/* Show original JSON */}
        <details style={{ marginBottom: '12px' }}>
          <summary style={{
            cursor: 'pointer',
            padding: '8px',
            backgroundColor: colors.surfaceAlt,
            borderRadius: '4px',
            fontFamily: 'monospace',
            fontSize: '13px',
            fontWeight: 'bold',
            color: colors.text,
          }}>
            View Raw JSON
          </summary>
          <pre style={{
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
            fontFamily: 'monospace',
            backgroundColor: colors.surfaceAlt,
            padding: '12px',
            borderRadius: '4px',
            marginTop: '8px',
            fontSize: '13px',
            overflow: 'auto',
          }}>
            {JSON.stringify(parsed, null, 2)}
          </pre>
        </details>

        {/* Show table view */}
        <table style={{
          width: '100%',
          borderCollapse: 'collapse',
          border: `1px solid ${colors.border}`,
        }}>
          <thead>
            <tr style={{ backgroundColor: colors.surfaceAlt }}>
              <th style={{
                padding: '10px',
                textAlign: 'left',
                fontWeight: 'bold',
                borderBottom: `2px solid ${colors.border}`,
                width: '30%',
              }}>
                Key
              </th>
              <th style={{
                padding: '10px',
                textAlign: 'left',
                fontWeight: 'bold',
                borderBottom: `2px solid ${colors.border}`,
              }}>
                Value
              </th>
            </tr>
          </thead>
          <tbody>
            {entries.map(([key, value], idx) => (
              <tr key={idx} style={{ borderBottom: `1px solid ${colors.border}` }}>
                <td style={{
                  padding: '10px',
                  fontFamily: 'monospace',
                  fontWeight: 'bold',
                  verticalAlign: 'top',
                }}>
                  {key}
                </td>
                <td style={{
                  padding: '10px',
                  fontFamily: 'monospace',
                  whiteSpace: 'pre-wrap',
                  wordBreak: 'break-word',
                }}>
                  {typeof value === 'object' ? JSON.stringify(value, null, 2) : String(value)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  } catch (e) {
    // If parsing fails, return raw content
    return <pre style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontFamily: 'monospace' }}>{jsonString}</pre>;
  }
};

export const Message: React.FC<MessageProps> = ({ role, content, conversationFormat, colors }) => {
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
      </div>
      <div style={role === 'assistant' ? styles.assistantContent : styles.messageContent}>
        {role === 'assistant' ? (
          conversationFormat === 'json' ? (
            renderJsonAsTable(content, colors)
          ) : (
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                h1: ({ children }) => <h1 style={{ marginTop: '16px', marginBottom: '12px', fontSize: '28px', fontWeight: 'bold' }}>{children}</h1>,
                h2: ({ children }) => <h2 style={{ marginTop: '14px', marginBottom: '10px', fontSize: '24px', fontWeight: 'bold' }}>{children}</h2>,
                h3: ({ children }) => <h3 style={{ marginTop: '12px', marginBottom: '8px', fontSize: '20px', fontWeight: 'bold' }}>{children}</h3>,
                p: ({ children }) => <p style={{ marginBottom: '12px' }}>{children}</p>,
                ul: ({ children }) => <ul style={{ marginLeft: '20px', marginBottom: '12px', paddingLeft: '20px' }}>{children}</ul>,
                ol: ({ children }) => <ol style={{ marginLeft: '20px', marginBottom: '12px', paddingLeft: '20px' }}>{children}</ol>,
                li: ({ children }) => <li style={{ marginBottom: '6px' }}>{children}</li>,
                code: ({ children }) => <code style={{ backgroundColor: 'rgba(0,0,0,0.2)', padding: '2px 6px', borderRadius: '3px', fontFamily: 'monospace', fontSize: '14px' }}>{children}</code>,
                pre: ({ children }) => <pre style={{ backgroundColor: 'rgba(0,0,0,0.3)', padding: '12px', borderRadius: '6px', overflow: 'auto', marginBottom: '12px', fontFamily: 'monospace' }}>{children}</pre>,
                blockquote: ({ children }) => <blockquote style={{ borderLeft: '4px solid', paddingLeft: '12px', marginLeft: '0', marginBottom: '12px', opacity: 0.8 }}>{children}</blockquote>,
                table: ({ children }) => <table style={{ borderCollapse: 'collapse', marginBottom: '12px', border: '1px solid rgba(255,255,255,0.2)', width: '100%' }}>{children}</table>,
                thead: ({ children }) => <thead style={{ backgroundColor: 'rgba(0,0,0,0.2)', borderBottom: '2px solid rgba(255,255,255,0.3)' }}>{children}</thead>,
                tbody: ({ children }) => <tbody>{children}</tbody>,
                tr: ({ children }) => <tr style={{ borderBottom: '1px solid rgba(255,255,255,0.1)' }}>{children}</tr>,
                th: ({ children }) => <th style={{ padding: '10px', textAlign: 'left', fontWeight: 'bold' }}>{children}</th>,
                td: ({ children }) => <td style={{ padding: '10px' }}>{children}</td>,
              }}
            >
              {content}
            </ReactMarkdown>
          )
        ) : (
          content
        )}
      </div>
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
