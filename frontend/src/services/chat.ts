import { AuthService } from './auth';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export interface ChatMessage {
  message: string;
}

export interface ChatResponse {
  response: string;
  error?: string;
  history?: Message[];
}

export interface Message {
  role: 'user' | 'assistant';
  content: string;
}

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
}
