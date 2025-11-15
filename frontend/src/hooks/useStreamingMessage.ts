import { useState, useCallback } from 'react';
import { ChatService, OnChunkCallback, OnConversationCallback, OnModelCallback, OnTemperatureCallback, OnUsageCallback } from '../services/chat';

export interface StreamingOptions {
  conversationId?: string;
  systemPrompt?: string;
  responseFormat?: string;
  responseSchema?: string;
  model?: string;
  temperature?: number;
  provider?: string;
  useWarAndPeace?: boolean;
  warAndPeacePercent?: number;
}

/**
 * Custom hook for handling streaming message operations
 * Manages streaming state and provides a clean API for sending messages
 */
export function useStreamingMessage(chatService: ChatService) {
  const [isStreaming, setIsStreaming] = useState(false);

  const streamMessage = useCallback(async (
    message: string,
    onChunk: OnChunkCallback,
    onConversation?: OnConversationCallback,
    onModel?: OnModelCallback,
    onTemperature?: OnTemperatureCallback,
    onUsage?: OnUsageCallback,
    options?: StreamingOptions
  ) => {
    setIsStreaming(true);
    try {
      await chatService.streamMessage(
        message,
        onChunk,
        onConversation,
        options?.conversationId,
        onModel,
        options?.systemPrompt,
        options?.responseFormat,
        options?.responseSchema,
        options?.model,
        options?.temperature,
        onTemperature,
        onUsage,
        options?.provider,
        options?.useWarAndPeace,
        options?.warAndPeacePercent
      );
    } finally {
      setIsStreaming(false);
    }
  }, [chatService]);

  return { streamMessage, isStreaming };
}
