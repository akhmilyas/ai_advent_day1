import { useState, useCallback, useEffect, useMemo } from 'react';
import { ChatService, Conversation } from '../services/chat';

/**
 * Custom hook for managing conversations
 * Handles loading, deleting, and refreshing conversations
 */
export function useConversations() {
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const chatService = useMemo(() => new ChatService(), []);

  const loadConversations = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const convs = await chatService.getConversations();
      setConversations(convs);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to load conversations';
      setError(errorMessage);
      console.error('Error loading conversations:', err);
    } finally {
      setLoading(false);
    }
  }, [chatService]);

  const deleteConversation = useCallback(async (id: string) => {
    try {
      await chatService.deleteConversation(id);
      setConversations(prev => prev.filter(c => c.id !== id));
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to delete conversation';
      setError(errorMessage);
      console.error('Error deleting conversation:', err);
      throw err;
    }
  }, [chatService]);

  useEffect(() => {
    loadConversations();
  }, [loadConversations]);

  return {
    conversations,
    loading,
    error,
    refresh: loadConversations,
    deleteConversation
  };
}
