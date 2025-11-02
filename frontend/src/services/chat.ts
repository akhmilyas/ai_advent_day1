import { AuthService } from './auth';

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';
const WS_URL = process.env.REACT_APP_WS_URL || 'ws://localhost:8080';

export interface ChatMessage {
  message: string;
}

export interface ChatResponse {
  response: string;
  error?: string;
}

export interface WSMessage {
  type: 'start' | 'chunk' | 'end' | 'error';
  content: string;
}

export type StreamCallback = (chunk: string) => void;
export type ErrorCallback = (error: string) => void;
export type CompleteCallback = () => void;

export class ChatService {
  private ws: WebSocket | null = null;

  async sendMessage(message: string): Promise<string> {
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

    return data.response;
  }

  connectStream(
    onChunk: StreamCallback,
    onError: ErrorCallback,
    onComplete: CompleteCallback
  ): void {
    const token = AuthService.getToken();
    if (!token) {
      onError('Not authenticated');
      return;
    }

    this.ws = new WebSocket(`${WS_URL}/api/chat/stream`);

    this.ws.onopen = () => {
      console.log('WebSocket connected');
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WSMessage = JSON.parse(event.data);

        switch (message.type) {
          case 'start':
            // Message started
            break;
          case 'chunk':
            onChunk(message.content);
            break;
          case 'end':
            onComplete();
            break;
          case 'error':
            onError(message.content);
            break;
        }
      } catch (err) {
        console.error('Error parsing message:', err);
        onError('Error parsing message');
      }
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
      onError('WebSocket connection error');
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
    };

    // Send authorization after connection
    setTimeout(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        // The coder/websocket library handles auth via HTTP headers during upgrade
        // So we don't need to send auth separately
      }
    }, 100);
  }

  sendStreamMessage(message: string): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ message }));
    }
  }

  disconnectStream(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
}
