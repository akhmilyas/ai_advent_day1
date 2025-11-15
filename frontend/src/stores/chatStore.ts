import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { ResponseFormat } from './settingsStore';

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

interface ChatState {
  // State
  messages: ChatMessage[];
  conversationId: string | undefined;
  conversationTitle: string;
  conversationFormat: ResponseFormat | undefined;
  conversationSchema: string;
  isLoading: boolean;
  summaries: Array<{ upToMessageId: string; content: string }>;

  // Actions
  addMessage: (message: ChatMessage) => void;
  updateLastMessage: (content: string) => void;
  updateLastMessageMetadata: (metadata: Partial<ChatMessage>) => void;
  setMessages: (messages: ChatMessage[]) => void;
  setConversationId: (id: string | undefined) => void;
  setConversationTitle: (title: string) => void;
  setConversationFormat: (format: ResponseFormat | undefined) => void;
  setConversationSchema: (schema: string) => void;
  setLoading: (loading: boolean) => void;
  setSummaries: (summaries: Array<{ upToMessageId: string; content: string }>) => void;
  addSummary: (summary: { upToMessageId: string; content: string }) => void;
  reset: () => void;
}

export const useChatStore = create<ChatState>()(
  devtools(
    (set) => ({
      // Initial state
      messages: [],
      conversationId: undefined,
      conversationTitle: '',
      conversationFormat: undefined,
      conversationSchema: '',
      isLoading: false,
      summaries: [],

      // Actions
      addMessage: (message) =>
        set((state) => ({
          messages: [...state.messages, message]
        }), false, 'addMessage'),

      updateLastMessage: (content) =>
        set((state) => ({
          messages: state.messages.map((msg, idx) =>
            idx === state.messages.length - 1
              ? { ...msg, content: msg.content + content }
              : msg
          )
        }), false, 'updateLastMessage'),

      updateLastMessageMetadata: (metadata) =>
        set((state) => ({
          messages: state.messages.map((msg, idx) =>
            idx === state.messages.length - 1
              ? { ...msg, ...metadata }
              : msg
          )
        }), false, 'updateLastMessageMetadata'),

      setMessages: (messages) => set({ messages }, false, 'setMessages'),

      setConversationId: (id) => set({ conversationId: id }, false, 'setConversationId'),

      setConversationTitle: (title) => set({ conversationTitle: title }, false, 'setConversationTitle'),

      setConversationFormat: (format) => set({ conversationFormat: format }, false, 'setConversationFormat'),

      setConversationSchema: (schema) => set({ conversationSchema: schema }, false, 'setConversationSchema'),

      setLoading: (loading) => set({ isLoading: loading }, false, 'setLoading'),

      setSummaries: (summaries) => set({ summaries }, false, 'setSummaries'),

      addSummary: (summary) =>
        set((state) => ({
          summaries: [...state.summaries, summary]
        }), false, 'addSummary'),

      reset: () => set({
        messages: [],
        conversationId: undefined,
        conversationTitle: '',
        conversationFormat: undefined,
        conversationSchema: '',
        isLoading: false,
        summaries: []
      }, false, 'reset')
    }),
    { name: 'chat-store' }
  )
);
