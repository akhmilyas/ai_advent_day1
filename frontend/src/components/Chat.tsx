import React, { useState, useRef, useEffect } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { ChatService, Message } from '../services/chat';
import { AuthService } from '../services/auth';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';

interface ChatProps {
  onLogout: () => void;
}

export const Chat: React.FC<ChatProps> = ({ onLogout }) => {
  const { theme, toggleTheme } = useTheme();
  const colors = getTheme(theme === 'dark');
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [conversationId, setConversationId] = useState<number | undefined>(undefined);
  const chatService = useRef(new ChatService());
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || loading) return;

    const userMessage = input.trim();
    setInput('');
    setLoading(true);

    // Optimistically add user message to UI
    setMessages((prev) => [...prev, { role: 'user', content: userMessage }]);

    // Add empty assistant message that will be filled in via streaming
    setMessages((prev) => [...prev, { role: 'assistant', content: '' }]);

    try {
      // Stream the response and update the assistant message
      await chatService.current.streamMessage(
        userMessage,
        (chunk) => {
          // Update the last message (assistant) with the new chunk
          setMessages((prev) => {
            const updated = [...prev];
            if (updated[updated.length - 1].role === 'assistant') {
              updated[updated.length - 1] = {
                ...updated[updated.length - 1],
                content: updated[updated.length - 1].content + chunk,
              };
            }
            return updated;
          });
        },
        (convId) => {
          // Set conversation ID when received from server
          setConversationId(convId);
        },
        conversationId
      );
      setLoading(false);
    } catch (error) {
      console.error('Error sending message:', error);
      setMessages((prev) => {
        const updated = [...prev];
        if (updated[updated.length - 1].role === 'assistant') {
          updated[updated.length - 1] = {
            ...updated[updated.length - 1],
            content: 'Error: Failed to get response',
          };
        }
        return updated;
      });
      setLoading(false);
    }
  };

  const handleLogout = () => {
    AuthService.logout();
    onLogout();
  };

  return (
    <div style={{ ...styles.container, backgroundColor: colors.background }}>
      <div style={{ ...styles.header, backgroundColor: colors.header, borderBottomColor: colors.border }}>
        <h2 style={{ ...styles.title, color: colors.text }}>AI Chat</h2>
        <div style={styles.buttonGroup}>
          <button
            onClick={toggleTheme}
            style={{
              ...styles.themeButton,
              backgroundColor: colors.surface,
              color: colors.text,
              border: `1px solid ${colors.border}`,
            }}
            title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
          >
            {theme === 'light' ? '🌙' : '☀️'}
          </button>
          <button
            onClick={handleLogout}
            style={{
              ...styles.logoutButton,
              backgroundColor: colors.buttonDanger,
              color: colors.buttonDangerText,
            }}
          >
            Logout
          </button>
        </div>
      </div>

      <div style={{ ...styles.messagesContainer, backgroundColor: colors.background }}>
        {messages.length === 0 && (
          <div style={{ ...styles.emptyState, color: colors.emptyStateText }}>
            <p>Start a conversation by typing a message below</p>
          </div>
        )}

        {messages.map((msg, idx) => (
          <div
            key={idx}
            style={{
              ...styles.message,
              ...(msg.role === 'user'
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
              {msg.role === 'user' ? 'You' : 'AI'}
            </div>
            <div style={msg.role === 'assistant' ? styles.assistantContent : styles.messageContent}>
              {msg.role === 'assistant' ? (
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
                  {msg.content}
                </ReactMarkdown>
              ) : (
                msg.content
              )}
            </div>
          </div>
        ))}

        <div ref={messagesEndRef} />
      </div>

      <form
        onSubmit={handleSubmit}
        style={{
          ...styles.inputContainer,
          backgroundColor: colors.header,
          borderTopColor: colors.border,
        }}
      >
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type your message..."
          style={{
            ...styles.input,
            backgroundColor: colors.input,
            color: colors.text,
            borderColor: colors.border,
          }}
          disabled={loading}
        />
        <button
          type="submit"
          style={{
            ...styles.sendButton,
            backgroundColor: colors.buttonPrimary,
            color: colors.buttonPrimaryText,
            ...(loading || !input.trim() ? { backgroundColor: colors.buttonPrimaryDisabled } : {}),
          }}
          disabled={loading || !input.trim()}
        >
          Send
        </button>
      </form>
    </div>
  );
};

const styles = {
  container: {
    display: 'flex',
    flexDirection: 'column' as const,
    height: '100vh',
    transition: 'background-color 0.3s ease',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '20px',
    borderBottom: '1px solid',
    transition: 'background-color 0.3s ease, border-color 0.3s ease',
  },
  title: {
    margin: 0,
    transition: 'color 0.3s ease',
  },
  buttonGroup: {
    display: 'flex',
    gap: '10px',
    alignItems: 'center',
  },
  themeButton: {
    padding: '8px 12px',
    fontSize: '16px',
    border: '1px solid',
    borderRadius: '4px',
    cursor: 'pointer',
    transition: 'background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease',
  },
  logoutButton: {
    padding: '8px 16px',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    transition: 'background-color 0.3s ease',
  },
  messagesContainer: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '20px',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '16px',
    transition: 'background-color 0.3s ease',
  },
  emptyState: {
    textAlign: 'center' as const,
    marginTop: '100px',
    transition: 'color 0.3s ease',
  },
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
  inputContainer: {
    display: 'flex',
    gap: '10px',
    padding: '20px',
    borderTop: '1px solid',
    transition: 'background-color 0.3s ease, border-color 0.3s ease',
  },
  input: {
    flex: 1,
    padding: '12px',
    fontSize: '16px',
    borderRadius: '4px',
    transition: 'background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease',
  },
  sendButton: {
    padding: '12px 24px',
    fontSize: '16px',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    transition: 'background-color 0.3s ease',
  },
  sendButtonDisabled: {
    cursor: 'not-allowed',
  },
};
