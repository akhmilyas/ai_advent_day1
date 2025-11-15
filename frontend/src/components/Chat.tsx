import React, { useState, useRef, useEffect, useMemo } from 'react';
import { ChatService } from '../services/chat';
import { AuthService } from '../services/auth';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';
import { SettingsModal } from './SettingsModal';
import { Sidebar } from './Sidebar';
import { ChatHeader } from './Chat/ChatHeader';
import { ChatMessages } from './Chat/ChatMessages';
import { ChatInput } from './Chat/ChatInput';
import { useChatStore, useSettingsStore, type ResponseFormat } from '../stores';

interface ChatProps {
  onLogout: () => void;
}

export const Chat: React.FC<ChatProps> = ({ onLogout }) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  // Zustand stores
  const {
    messages,
    conversationId,
    conversationTitle,
    conversationFormat,
    conversationSchema,
    isLoading,
    summaries,
    setMessages,
    addMessage,
    updateLastMessage,
    updateLastMessageMetadata,
    setConversationId,
    setConversationTitle,
    setConversationFormat,
    setConversationSchema,
    setLoading,
    setSummaries,
    addSummary,
    reset
  } = useChatStore();

  const {
    selectedModel,
    temperature,
    systemPrompt,
    responseFormat,
    responseSchema,
    provider,
    useWarAndPeace,
    warAndPeacePercent,
    settingsOpen,
    setModel,
    setSettingsOpen
  } = useSettingsStore();

  // Local state
  const [input, setInput] = useState('');
  const [summarizing, setSummarizing] = useState(false);

  const chatService = useMemo(() => new ChatService(), []);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<{ refreshConversations: () => Promise<void> }>(null);

  // Load default model if no model is selected
  useEffect(() => {
    if (!selectedModel) {
      const fetchDefaultModel = async () => {
        try {
          const models = await chatService.getAvailableModels();
          if (models.length > 0) {
            setModel(models[0].id);
          }
        } catch (error) {
          console.error('[Chat] Failed to fetch default model:', error);
        }
      };
      fetchDefaultModel();
    }
  }, [selectedModel, chatService, setModel]);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSelectConversation = async (convId: string, title: string) => {
    try {
      // Get conversation details to retrieve format and schema
      const conversations = await chatService.getConversations();
      const conversation = conversations.find(c => c.id === convId);

      const convMessages = await chatService.getConversationMessages(convId);
      setConversationId(convId);
      setConversationTitle(title);

      // Set conversation format and schema from the database
      if (conversation) {
        setConversationFormat((conversation.response_format || 'text') as ResponseFormat);
        setConversationSchema(conversation.response_schema || '');

        // Load all summaries for this conversation from backend
        try {
          const loadedSummaries = await chatService.getConversationSummaries(convId);
          setSummaries(loadedSummaries);
        } catch (error) {
          console.error('Error loading summaries:', error);
          setSummaries([]);
        }
      }

      setMessages(
        convMessages.map((msg) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          model: msg.model,
          temperature: msg.temperature,
          promptTokens: msg.prompt_tokens,
          completionTokens: msg.completion_tokens,
          totalTokens: msg.total_tokens,
          totalCost: msg.total_cost,
          latency: msg.latency,
          generationTime: msg.generation_time,
        }))
      );
    } catch (error) {
      console.error('Error loading conversation:', error);
      alert('Failed to load conversation');
    }
  };

  const handleNewConversation = () => {
    reset();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    const userMessage = input.trim();
    setInput('');
    setLoading(true);

    // Optimistically add user message to UI
    addMessage({ role: 'user', content: userMessage });

    // Add empty assistant message that will be filled in via streaming
    addMessage({ role: 'assistant', content: '' });

    try {
      // Stream the response and update the assistant message
      await chatService.streamMessage(
        userMessage,
        (chunk) => {
          // Update the last message (assistant) with the new chunk
          updateLastMessage(chunk);
        },
        (convId) => {
          // Set conversation ID when received from server
          setConversationId(convId);
          // For new conversations, set the format and schema that was used
          if (!conversationId) {
            setConversationFormat(responseFormat);
            setConversationSchema(responseSchema);
            // Refresh sidebar to show new conversation
            if (sidebarRef.current) {
              sidebarRef.current.refreshConversations();
            }
          }
        },
        conversationId || undefined,
        (modelName) => {
          // Update the last message with model metadata
          updateLastMessageMetadata({ model: modelName });
        },
        systemPrompt,
        // Only send format/schema for new conversations
        conversationId ? undefined : responseFormat,
        conversationId ? undefined : responseSchema,
        selectedModel || undefined,
        temperature,
        (temp) => {
          // Update the last message with temperature metadata
          updateLastMessageMetadata({ temperature: temp });
        },
        (usage) => {
          // Update the last message with usage metadata
          updateLastMessageMetadata({
            promptTokens: usage.prompt_tokens,
            completionTokens: usage.completion_tokens,
            totalTokens: usage.total_tokens,
            totalCost: usage.total_cost,
            latency: usage.latency,
            generationTime: usage.generation_time
          });
        },
        provider,
        useWarAndPeace,
        warAndPeacePercent
      );
      setLoading(false);
    } catch (error) {
      console.error('Error sending message:', error);
      const updated = [...messages];
      if (updated[updated.length - 1].role === 'assistant') {
        updated[updated.length - 1] = {
          ...updated[updated.length - 1],
          content: 'Error: Failed to get response',
        };
      }
      setMessages(updated);
      setLoading(false);
    }
  };

  const handleSummarize = async () => {
    if (!conversationId || summarizing || messages.length === 0) return;

    setSummarizing(true);
    try {
      const result = await chatService.summarizeConversation(conversationId, selectedModel, temperature);

      // Add the new summary to the list
      if (result.summarized_up_to_message_id && result.summary) {
        addSummary({
          upToMessageId: result.summarized_up_to_message_id,
          content: result.summary
        });
      }

      // Reload messages from server to get the IDs
      const convMessages = await chatService.getConversationMessages(conversationId);
      setMessages(
        convMessages.map((msg) => ({
          id: msg.id,
          role: msg.role,
          content: msg.content,
          model: msg.model,
          temperature: msg.temperature,
          promptTokens: msg.prompt_tokens,
          completionTokens: msg.completion_tokens,
          totalTokens: msg.total_tokens,
          totalCost: msg.total_cost,
          latency: msg.latency,
          generationTime: msg.generation_time,
        }))
      );

      // Also refresh the conversation list in sidebar to update the summary info
      if (sidebarRef.current) {
        await sidebarRef.current.refreshConversations();
      }
      alert('Conversation summarized successfully!');
    } catch (error) {
      console.error('Error summarizing conversation:', error);
      alert('Failed to summarize conversation');
    } finally {
      setSummarizing(false);
    }
  };

  const handleLogout = () => {
    AuthService.logout();
    onLogout();
  };

  return (
    <div style={styles.appContainer}>
      <Sidebar
        ref={sidebarRef}
        onSelectConversation={handleSelectConversation}
        onNewConversation={handleNewConversation}
        currentConversationId={conversationId}
      />
      <div style={{ ...styles.container, backgroundColor: colors.background }}>
        <ChatHeader
          conversationTitle={conversationTitle}
          model={selectedModel}
          conversationFormat={conversationFormat}
          showSummarizeButton={conversationId !== undefined && messages.length > 0}
          summarizing={summarizing}
          onSummarize={handleSummarize}
          onOpenSettings={() => setSettingsOpen(true)}
          onLogout={handleLogout}
        />

        <ChatMessages
          messages={messages}
          summaries={summaries}
          conversationFormat={conversationFormat}
          messagesEndRef={messagesEndRef}
        />

        <ChatInput
          value={input}
          onChange={setInput}
          onSubmit={handleSubmit}
          loading={isLoading}
        />

        <SettingsModal
          conversationFormat={conversationFormat}
          conversationSchema={conversationSchema}
          isExistingConversation={conversationId !== undefined}
        />
      </div>
    </div>
  );
};

const styles = {
  appContainer: {
    display: 'flex',
    height: '100vh',
    width: '100%',
  },
  container: {
    display: 'flex',
    flexDirection: 'column' as const,
    flex: 1,
    transition: 'background-color 0.3s ease',
  },
};
