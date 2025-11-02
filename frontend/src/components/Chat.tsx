import React, { useState, useEffect, useRef } from 'react';
import { ChatService } from '../services/chat';
import { AuthService } from '../services/auth';

interface Message {
  role: 'user' | 'assistant';
  content: string;
}

interface ChatProps {
  onLogout: () => void;
}

export const Chat: React.FC<ChatProps> = ({ onLogout }) => {
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
    <div style={styles.container}>
      <div style={styles.header}>
        <h2 style={styles.title}>AI Chat</h2>
        <button onClick={handleLogout} style={styles.logoutButton}>
          Logout
        </button>
      </div>

      <div style={styles.messagesContainer}>
        {messages.length === 0 && (
          <div style={styles.emptyState}>
            <p>Start a conversation by typing a message below</p>
          </div>
        )}

        {messages.map((msg, idx) => (
          <div
            key={idx}
            style={{
              ...styles.message,
              ...(msg.role === 'user' ? styles.userMessage : styles.assistantMessage),
            }}
          >
            <div style={styles.messageRole}>
              {msg.role === 'user' ? 'You' : 'AI'}
            </div>
            <div style={styles.messageContent}>{msg.content}</div>
          </div>
        ))}

        {streamingContent && (
          <div style={{ ...styles.message, ...styles.assistantMessage }}>
            <div style={styles.messageRole}>AI</div>
            <div style={styles.messageContent}>
              {streamingContent}
              <span style={styles.cursor}>▊</span>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      <form onSubmit={handleSubmit} style={styles.inputContainer}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="Type your message..."
          style={styles.input}
          disabled={loading}
        />
        <button
          type="submit"
          style={{
            ...styles.sendButton,
            ...(loading || !input.trim() ? styles.sendButtonDisabled : {}),
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
    backgroundColor: '#f5f5f5',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '20px',
    backgroundColor: 'white',
    borderBottom: '1px solid #ddd',
  },
  title: {
    margin: 0,
    color: '#333',
  },
  logoutButton: {
    padding: '8px 16px',
    backgroundColor: '#dc3545',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
  },
  messagesContainer: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '20px',
    display: 'flex',
    flexDirection: 'column' as const,
    gap: '16px',
  },
  emptyState: {
    textAlign: 'center' as const,
    color: '#999',
    marginTop: '100px',
  },
  message: {
    padding: '12px 16px',
    borderRadius: '8px',
    maxWidth: '70%',
  },
  userMessage: {
    alignSelf: 'flex-end',
    backgroundColor: '#007bff',
    color: 'white',
  },
  assistantMessage: {
    alignSelf: 'flex-start',
    backgroundColor: 'white',
    border: '1px solid #ddd',
  },
  messageRole: {
    fontSize: '12px',
    fontWeight: 'bold' as const,
    marginBottom: '4px',
    opacity: 0.7,
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
    backgroundColor: 'white',
    borderTop: '1px solid #ddd',
  },
  input: {
    flex: 1,
    padding: '12px',
    fontSize: '16px',
    border: '1px solid #ddd',
    borderRadius: '4px',
  },
  sendButton: {
    padding: '12px 24px',
    fontSize: '16px',
    backgroundColor: '#007bff',
    color: 'white',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
  },
  sendButtonDisabled: {
    backgroundColor: '#ccc',
    cursor: 'not-allowed',
  },
};
