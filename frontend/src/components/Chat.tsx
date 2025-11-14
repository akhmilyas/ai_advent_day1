import React, { useState, useRef, useEffect } from 'react';
import { ChatService } from '../services/chat';
import { AuthService } from '../services/auth';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';
import { SettingsModal, ResponseFormat, ProviderType } from './SettingsModal';
import { Sidebar } from './Sidebar';
import { ChatHeader } from './Chat/ChatHeader';
import { ChatMessages } from './Chat/ChatMessages';
import { ChatInput } from './Chat/ChatInput';

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

interface ChatProps {
  onLogout: () => void;
}

export const Chat: React.FC<ChatProps> = ({ onLogout }) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [conversationId, setConversationId] = useState<string | undefined>(undefined);
  const [conversationTitle, setConversationTitle] = useState<string>('');
  const [model, setModel] = useState<string>('');
  const [systemPrompt, setSystemPrompt] = useState<string>('');
  const [responseFormat, setResponseFormat] = useState<ResponseFormat>('text');
  const [responseSchema, setResponseSchema] = useState<string>('');
  const [conversationFormat, setConversationFormat] = useState<ResponseFormat | null>(null);
  const [conversationSchema, setConversationSchema] = useState<string>('');
  const [temperature, setTemperature] = useState<number>(0.7);
  const [provider, setProvider] = useState<ProviderType>('openrouter');
  const [useWarAndPeace, setUseWarAndPeace] = useState<boolean>(false);
  const [warAndPeacePercent, setWarAndPeacePercent] = useState<number>(100);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [summarizing, setSummarizing] = useState(false);
  const [summaries, setSummaries] = useState<Array<{ upToMessageId: string; content: string }>>([]);
  const chatService = useRef(new ChatService());
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const sidebarRef = useRef<{ refreshConversations: () => Promise<void> }>(null);

  // Load settings from localStorage on mount
  useEffect(() => {
    const savedPrompt = localStorage.getItem('systemPrompt');
    const savedFormat = localStorage.getItem('responseFormat');
    const savedSchema = localStorage.getItem('responseSchema');
    const savedModel = localStorage.getItem('selectedModel');
    const savedTemperature = localStorage.getItem('temperature');
    const savedProvider = localStorage.getItem('provider');
    const savedUseWarAndPeace = localStorage.getItem('useWarAndPeace');
    const savedWarAndPeacePercent = localStorage.getItem('warAndPeacePercent');

    if (savedPrompt) {
      setSystemPrompt(savedPrompt);
    }
    if (savedFormat && (savedFormat === 'text' || savedFormat === 'json' || savedFormat === 'xml')) {
      setResponseFormat(savedFormat as ResponseFormat);
    }
    if (savedSchema) {
      setResponseSchema(savedSchema);
    }
    if (savedModel) {
      setModel(savedModel);
    } else {
      // If no saved model, fetch and use the first model from config as default
      const fetchDefaultModel = async () => {
        try {
          const models = await chatService.current.getAvailableModels();
          if (models.length > 0) {
            setModel(models[0].id);
            localStorage.setItem('selectedModel', models[0].id);
          }
        } catch (error) {
          console.error('[Chat] Failed to fetch default model:', error);
        }
      };
      fetchDefaultModel();
    }
    if (savedTemperature) {
      const temp = parseFloat(savedTemperature);
      if (!isNaN(temp)) {
        setTemperature(temp);
      }
    }
    if (savedProvider && (savedProvider === 'openrouter' || savedProvider === 'genkit')) {
      setProvider(savedProvider as ProviderType);
    }
    if (savedUseWarAndPeace === 'true') {
      setUseWarAndPeace(true);
    }
    if (savedWarAndPeacePercent) {
      const percent = parseInt(savedWarAndPeacePercent);
      if (!isNaN(percent) && percent >= 0 && percent <= 100) {
        setWarAndPeacePercent(percent);
      }
    }
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  const handleSystemPromptChange = (prompt: string) => {
    setSystemPrompt(prompt);
    localStorage.setItem('systemPrompt', prompt);
  };

  const handleResponseFormatChange = (format: ResponseFormat) => {
    setResponseFormat(format);
    localStorage.setItem('responseFormat', format);
  };

  const handleResponseSchemaChange = (schema: string) => {
    setResponseSchema(schema);
    localStorage.setItem('responseSchema', schema);
  };

  const handleModelChange = (modelId: string) => {
    setModel(modelId);
    localStorage.setItem('selectedModel', modelId);
  };

  const handleTemperatureChange = (temp: number) => {
    setTemperature(temp);
    localStorage.setItem('temperature', temp.toString());
  };

  const handleProviderChange = (prov: ProviderType) => {
    setProvider(prov);
    localStorage.setItem('provider', prov);
  };

  const handleUseWarAndPeaceChange = (use: boolean) => {
    setUseWarAndPeace(use);
    localStorage.setItem('useWarAndPeace', use.toString());
  };

  const handleWarAndPeacePercentChange = (percent: number) => {
    setWarAndPeacePercent(percent);
    localStorage.setItem('warAndPeacePercent', percent.toString());
  };

  const handleSelectConversation = async (convId: string, title: string) => {
    try {
      // Get conversation details to retrieve format and schema
      const conversations = await chatService.current.getConversations();
      const conversation = conversations.find(c => c.id === convId);

      const convMessages = await chatService.current.getConversationMessages(convId);
      setConversationId(convId);
      setConversationTitle(title);

      // Set conversation format and schema from the database
      if (conversation) {
        setConversationFormat((conversation.response_format || 'text') as ResponseFormat);
        setConversationSchema(conversation.response_schema || '');

        // Load all summaries for this conversation from backend
        try {
          const loadedSummaries = await chatService.current.getConversationSummaries(convId);
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
    setConversationId(undefined);
    setConversationTitle('');
    setMessages([]);
    setModel('');
    setConversationFormat(null);
    setConversationSchema('');
    setSummaries([]);
  };

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
        conversationId,
        (modelName) => {
          // Set model when received from server
          setModel(modelName);
          // Update the last message (assistant) with the model
          setMessages((prev) => {
            const updated = [...prev];
            if (updated.length > 0 && updated[updated.length - 1].role === 'assistant') {
              updated[updated.length - 1] = {
                ...updated[updated.length - 1],
                model: modelName,
              };
            }
            return updated;
          });
        },
        systemPrompt,
        // Only send format/schema for new conversations
        conversationId ? undefined : responseFormat,
        conversationId ? undefined : responseSchema,
        model || undefined,
        temperature,
        (temp) => {
          // Update the last message (assistant) with the temperature
          setMessages((prev) => {
            const updated = [...prev];
            if (updated.length > 0 && updated[updated.length - 1].role === 'assistant') {
              updated[updated.length - 1] = {
                ...updated[updated.length - 1],
                temperature: temp,
              };
            }
            return updated;
          });
        },
        (usage) => {
          // Update the last message (assistant) with usage data
          setMessages((prev) => {
            const updated = [...prev];
            if (updated.length > 0 && updated[updated.length - 1].role === 'assistant') {
              updated[updated.length - 1] = {
                ...updated[updated.length - 1],
                promptTokens: usage.prompt_tokens,
                completionTokens: usage.completion_tokens,
                totalTokens: usage.total_tokens,
                totalCost: usage.total_cost,
                latency: usage.latency,
                generationTime: usage.generation_time,
              };
            }
            return updated;
          });
        },
        provider,
        useWarAndPeace,
        warAndPeacePercent
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

  const handleSummarize = async () => {
    if (!conversationId || summarizing || messages.length === 0) return;

    setSummarizing(true);
    try {
      const result = await chatService.current.summarizeConversation(conversationId, model, temperature);

      // Add the new summary to the list
      if (result.summarized_up_to_message_id && result.summary) {
        setSummaries(prev => [...prev, {
          upToMessageId: result.summarized_up_to_message_id,
          content: result.summary
        }]);
      }

      // Reload messages from server to get the IDs
      const convMessages = await chatService.current.getConversationMessages(conversationId);
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
          model={model}
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
          loading={loading}
        />

        <SettingsModal
          isOpen={settingsOpen}
          onClose={() => setSettingsOpen(false)}
          systemPrompt={systemPrompt}
          onSystemPromptChange={handleSystemPromptChange}
          responseFormat={responseFormat}
          onResponseFormatChange={handleResponseFormatChange}
          responseSchema={responseSchema}
          onResponseSchemaChange={handleResponseSchemaChange}
          conversationFormat={conversationFormat}
          conversationSchema={conversationSchema}
          isExistingConversation={conversationId !== undefined}
          selectedModel={model}
          onModelChange={handleModelChange}
          temperature={temperature}
          onTemperatureChange={handleTemperatureChange}
          provider={provider}
          onProviderChange={handleProviderChange}
          useWarAndPeace={useWarAndPeace}
          onUseWarAndPeaceChange={handleUseWarAndPeaceChange}
          warAndPeacePercent={warAndPeacePercent}
          onWarAndPeacePercentChange={handleWarAndPeacePercentChange}
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
