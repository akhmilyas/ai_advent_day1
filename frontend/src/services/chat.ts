import { AuthService } from './auth';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export interface ChatMessage {
  message: string;
}

export interface ChatResponse {
  response: string;
  error?: string;
}

export interface Message {
  role: 'user' | 'assistant';
  content: string;
}

export type OnChunkCallback = (chunk: string) => void;

export class ChatService {
  async sendMessage(message: string): Promise<ChatResponse> {
    const response = await fetch(`${API_URL}/api/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...AuthService.getAuthHeader(),
      },
      body: JSON.stringify({ message }),
    });

    if (!response.ok) {
      throw new Error('Failed to send message');
    }

    const data: ChatResponse = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }

    return data;
  }

  async streamMessage(message: string, onChunk: OnChunkCallback): Promise<void> {
    const response = await fetch(`${API_URL}/api/chat/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...AuthService.getAuthHeader(),
      },
      body: JSON.stringify({ message }),
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

            // Skip [DONE] and empty events
            if (content && content !== '[DONE]') {
              onChunk(content);
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }
  }
}
