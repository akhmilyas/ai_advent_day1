import { AuthService } from './auth';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export interface ChatMessage {
  message: string;
}

export interface Message {
  role: 'user' | 'assistant';
  content: string;
}

export interface Conversation {
  id: string;
  title: string;
  response_format: string;
  response_schema: string;
  created_at: string;
  updated_at: string;
}

export interface UsageInfo {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  total_cost?: number;
}

export interface ConversationMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  model?: string;
  temperature?: number;
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
  total_cost?: number;
  created_at: string;
}

export interface Model {
  id: string;
  name: string;
  provider: string;
  tier: string;
}

export type OnChunkCallback = (chunk: string) => void;
export type OnConversationCallback = (conversationId: string) => void;
export type OnModelCallback = (model: string) => void;
export type OnTemperatureCallback = (temperature: number) => void;
export type OnUsageCallback = (usage: UsageInfo) => void;

export class ChatService {
  async streamMessage(
    message: string,
    onChunk: OnChunkCallback,
    onConversation?: OnConversationCallback,
    conversationId?: string,
    onModel?: OnModelCallback,
    systemPrompt?: string,
    responseFormat?: string,
    responseSchema?: string,
    model?: string,
    temperature?: number,
    onTemperature?: OnTemperatureCallback,
    onUsage?: OnUsageCallback
  ): Promise<void> {
    const payload: any = { message };
    if (conversationId) {
      payload.conversation_id = conversationId;
    }
    if (systemPrompt) {
      payload.system_prompt = systemPrompt;
    }
    if (responseFormat) {
      payload.response_format = responseFormat;
    }
    if (responseSchema) {
      payload.response_schema = responseSchema;
    }
    if (model) {
      payload.model = model;
    }
    if (temperature !== undefined) {
      payload.temperature = temperature;
    }

    const response = await fetch(`${API_URL}/api/chat/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...AuthService.getAuthHeader(),
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error('Failed to send message');
    }

    if (!response.body) {
      throw new Error('Response body is null');
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();

    try {
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value);
        const lines = chunk.split('\n');

        for (const line of lines) {
          // Parse SSE format: "data: content"
          if (line.startsWith('data: ')) {
            const content = line.slice(6);

            // Check for conversation ID metadata
            if (content.startsWith('CONV_ID:')) {
              const convId = content.slice(8);
              if (convId && onConversation) {
                onConversation(convId);
              }
            }
            // Check for model metadata
            else if (content.startsWith('MODEL:')) {
              const model = content.slice(6);
              if (model && onModel) {
                onModel(model);
              }
            }
            // Check for temperature metadata
            else if (content.startsWith('TEMPERATURE:')) {
              const temp = parseFloat(content.slice(12));
              if (!isNaN(temp) && onTemperature) {
                onTemperature(temp);
              }
            }
            // Check for usage metadata
            else if (content.startsWith('USAGE:')) {
              try {
                const usageJson = content.slice(6);
                const usage: UsageInfo = JSON.parse(usageJson);
                if (onUsage) {
                  onUsage(usage);
                }
              } catch (e) {
                console.error('Error parsing usage data:', e);
              }
            }
            // Skip [DONE] and empty events
            else if (content && content !== '[DONE]') {
              // Unescape newlines from SSE format
              const unescapedContent = content.replace(/\\n/g, '\n');
              onChunk(unescapedContent);
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }

  async getConversations(): Promise<Conversation[]> {
    const response = await fetch(`${API_URL}/api/conversations`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...AuthService.getAuthHeader(),
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch conversations');
    }

    const data = await response.json();
    return data.conversations || [];
  }

  async getConversationMessages(conversationId: string): Promise<ConversationMessage[]> {
    const response = await fetch(`${API_URL}/api/conversations/${conversationId}/messages`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...AuthService.getAuthHeader(),
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch conversation messages');
    }

    const data = await response.json();
    return data.messages || [];
  }

  async deleteConversation(conversationId: string): Promise<void> {
    const response = await fetch(`${API_URL}/api/conversations/${conversationId}`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        ...AuthService.getAuthHeader(),
      },
    });

    if (!response.ok) {
      throw new Error('Failed to delete conversation');
    }
  }

  async getAvailableModels(): Promise<Model[]> {
    const response = await fetch(`${API_URL}/api/models`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });

    if (!response.ok) {
      throw new Error('Failed to fetch available models');
    }

    const data = await response.json();
    return data.models || [];
  }
}
