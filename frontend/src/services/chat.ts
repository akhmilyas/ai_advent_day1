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
  created_at: string;
  updated_at: string;
}

export interface ConversationMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

export type OnChunkCallback = (chunk: string) => void;
export type OnConversationCallback = (conversationId: string) => void;
export type OnModelCallback = (model: string) => void;

export class ChatService {
  async streamMessage(
    message: string,
    onChunk: OnChunkCallback,
    onConversation?: OnConversationCallback,
    conversationId?: string,
    onModel?: OnModelCallback,
    systemPrompt?: string
  ): Promise<void> {
    const payload: any = { message };
    if (conversationId) {
      payload.conversation_id = conversationId;
    }
    if (systemPrompt) {
      payload.system_prompt = systemPrompt;
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
}
