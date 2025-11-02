import React, { useState, useEffect, useRef } from 'react';
import { ChatService } from '../services/chat';
import { AuthService } from '../services/auth';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';

interface Message {
  role: 'user' | 'assistant';
  content: string;
}

interface ChatProps {
  onLogout: () => void;
}

export const Chat: React.FC<ChatProps> = ({ onLogout }) => {
  const { theme, toggleTheme } = useTheme();
  const colors = getTheme(theme === 'dark');
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [streamingContent, setStreamingContent] = useState('');
  const chatService = useRef(new ChatService());
  const messagesEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Connect to WebSocket on mount
    chatService.current.connectStream(
      (chunk) => {
        setStreamingContent((prev) => prev + chunk);
      },
      (error) => {
        console.error('Stream error:', error);
        setLoading(false);
        setStreamingContent('');
      },
      () => {
        setMessages((prev) => [
          ...prev,
          { role: 'assistant', content: streamingContent },
        ]);
        setStreamingContent('');
        setLoading(false);
      }
    );

    return () => {
      chatService.current.disconnectStream();
    };
  }, [streamingContent]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, streamingContent]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || loading) return;

    const userMessage = input.trim();
    setInput('');
    setMessages((prev) => [...prev, { role: 'user', content: userMessage }]);
    setLoading(true);
    setStreamingContent('');

    try {
      if (chatService.current.isConnected()) {
        chatService.current.sendStreamMessage(userMessage);
      } else {
        // Fallback to REST API if WebSocket is not connected
        const response = await chatService.current.sendMessage(userMessage);
        setMessages((prev) => [
          ...prev,
          { role: 'assistant', content: response },
        ]);
        setLoading(false);
      }
    } catch (error) {
      console.error('Error sending message:', error);
      setMessages((prev) => [
        ...prev,
        { role: 'assistant', content: 'Error: Failed to get response' },
      ]);
      setLoading(false);
    }
  };

  const handleLogout = () => {
    chatService.current.disconnectStream();
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
            <div style={styles.messageContent}>{msg.content}</div>
          </div>
        ))}

        {streamingContent && (
          <div
            style={{
              ...styles.message,
              ...styles.assistantMessage,
              backgroundColor: colors.assistantMessageBg,
              borderColor: colors.assistantMessageBorder,
              color: colors.assistantMessageText,
            }}
          >
            <div style={{ ...styles.messageRole, opacity: 0.7 }}>AI</div>
            <div style={styles.messageContent}>
              {streamingContent}
              <span style={styles.cursor}>▊</span>
            </div>
          </div>
        )}

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
  cursor: {
    animation: 'blink 1s infinite',
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
