import React, { useState, useEffect, useImperativeHandle } from 'react';
import { ChatService, Conversation } from '../services/chat';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';

interface SidebarProps {
  onSelectConversation: (conversationId: string, title: string) => void;
  onNewConversation: () => void;
  currentConversationId?: string;
  onRefreshConversations?: (callback: () => Promise<void>) => void;
}

export const Sidebar = React.forwardRef<
  { refreshConversations: () => Promise<void> },
  SidebarProps
>(
  (
    {
      onSelectConversation,
      onNewConversation,
      currentConversationId,
      onRefreshConversations,
    },
    ref
  ) => {
    const { theme } = useTheme();
    const colors = getTheme(theme === 'dark');
    const [conversations, setConversations] = useState<Conversation[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const chatService = React.useMemo(() => new ChatService(), []);

    useEffect(() => {
      loadConversations();
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    useImperativeHandle(ref, () => ({
      refreshConversations: loadConversations,
    }));

    const loadConversations = async () => {
      try {
        setLoading(true);
        setError(null);
        const convs = await chatService.getConversations();
        setConversations(convs);
      } catch (err) {
        console.error('Error loading conversations:', err);
        setError('Failed to load conversations');
      } finally {
        setLoading(false);
      }
    };

    const styles = getStyles(colors);

    return (
      <div style={styles.sidebar}>
        <div style={styles.header}>
          <h3 style={styles.title}>Conversations</h3>
          <button
            onClick={onNewConversation}
            style={styles.newButton}
            title="Start a new conversation"
          >
            +
          </button>
        </div>

        <div style={styles.content}>
          {loading && <p style={styles.message}>Loading...</p>}
          {error && <p style={{ ...styles.message, color: colors.buttonDanger }}>Error: {error}</p>}
          {!loading && conversations.length === 0 && (
            <p style={styles.message}>No conversations yet</p>
          )}

          {!loading &&
            conversations.map((conv) => (
              <ConversationItem
                key={conv.id}
                conversation={conv}
                isActive={currentConversationId === conv.id}
                onSelect={() => onSelectConversation(conv.id, conv.title)}
                onDelete={async () => {
                  try {
                    await chatService.deleteConversation(conv.id);
                    // If deleted conversation is currently selected, clear it
                    if (currentConversationId === conv.id) {
                      onNewConversation();
                    }
                    // Refresh the list
                    await loadConversations();
                  } catch (error) {
                    console.error('Error deleting conversation:', error);
                    alert('Failed to delete conversation');
                  }
                }}
                colors={colors}
              />
            ))}
        </div>
      </div>
    );
  }
);

interface ConversationItemProps {
  conversation: Conversation;
  isActive: boolean;
  onSelect: () => void;
  onDelete: () => void;
  colors: ReturnType<typeof getTheme>;
}

const ConversationItem: React.FC<ConversationItemProps> = ({
  conversation,
  isActive,
  onSelect,
  onDelete,
  colors,
}) => {
  const [showDeleteConfirm, setShowDeleteConfirm] = React.useState(false);
  const styles = getConversationItemStyles(colors);

  const handleDelete = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (showDeleteConfirm) {
      onDelete();
      setShowDeleteConfirm(false);
    } else {
      setShowDeleteConfirm(true);
    }
  };

  if (showDeleteConfirm) {
    return (
      <div style={styles.confirmContainer}>
        <p style={styles.confirmText}>Delete "{conversation.title}"?</p>
        <div style={styles.confirmButtons}>
          <button
            onClick={() => setShowDeleteConfirm(false)}
            style={styles.confirmCancel}
          >
            Keep
          </button>
          <button onClick={handleDelete} style={styles.confirmDelete}>
            Delete
          </button>
        </div>
      </div>
    );
  }

  return (
    <button
      onClick={onSelect}
      style={{
        ...styles.conversationItem,
        ...(isActive ? styles.conversationItemActive : {}),
      }}
      onMouseEnter={(e) => {
        const deleteBtn = e.currentTarget.querySelector('[data-delete-btn]') as HTMLElement;
        if (deleteBtn) {
          deleteBtn.style.opacity = '1';
        }
      }}
      onMouseLeave={(e) => {
        const deleteBtn = e.currentTarget.querySelector('[data-delete-btn]') as HTMLElement;
        if (deleteBtn && !showDeleteConfirm) {
          deleteBtn.style.opacity = '0';
        }
      }}
    >
      <div style={styles.itemContent}>
        <div style={styles.conversationTitle}>{conversation.title}</div>
        <div style={styles.conversationDate}>
          {new Date(conversation.updated_at).toLocaleDateString()}
        </div>
      </div>
      <button
        data-delete-btn
        onClick={handleDelete}
        style={styles.deleteButton}
        title="Delete conversation"
      >
        🗑️
      </button>
    </button>
  );
};

const getConversationItemStyles = (colors: ReturnType<typeof getTheme>) => ({
  conversationItem: {
    width: '100%',
    padding: '12px 16px',
    backgroundColor: 'transparent',
    border: 'none',
    textAlign: 'left' as const,
    cursor: 'pointer',
    transition: 'background-color 0.2s ease',
    borderLeft: `3px solid transparent`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    position: 'relative' as const,
  },
  conversationItemActive: {
    backgroundColor: colors.surfaceAlt,
    borderLeftColor: colors.buttonPrimary,
  },
  itemContent: {
    flex: 1,
    minWidth: 0,
  },
  conversationTitle: {
    color: colors.text,
    fontSize: '14px',
    fontWeight: '500' as const,
    marginBottom: '4px',
    overflow: 'hidden' as const,
    textOverflow: 'ellipsis' as const,
    whiteSpace: 'nowrap' as const,
  },
  conversationDate: {
    color: colors.textSecondary,
    fontSize: '12px',
  },
  deleteButton: {
    background: 'none',
    border: 'none',
    fontSize: '16px',
    cursor: 'pointer',
    padding: '4px 8px',
    opacity: 0,
    transition: 'opacity 0.2s ease',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    marginLeft: '8px',
    flexShrink: 0,
  },
  confirmContainer: {
    padding: '12px 16px',
    backgroundColor: colors.surfaceAlt,
    borderLeft: `3px solid ${colors.buttonDanger}`,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: '8px',
    borderRadius: '0 4px 4px 0',
  },
  confirmText: {
    margin: 0,
    fontSize: '13px',
    color: colors.text,
    flex: 1,
    overflow: 'hidden' as const,
    textOverflow: 'ellipsis' as const,
    whiteSpace: 'nowrap' as const,
  },
  confirmButtons: {
    display: 'flex',
    gap: '6px',
    flexShrink: 0,
  },
  confirmCancel: {
    padding: '4px 10px',
    fontSize: '12px',
    border: `1px solid ${colors.border}`,
    backgroundColor: colors.surface,
    color: colors.text,
    borderRadius: '3px',
    cursor: 'pointer',
    transition: 'background-color 0.2s ease',
  },
  confirmDelete: {
    padding: '4px 10px',
    fontSize: '12px',
    border: 'none',
    backgroundColor: colors.buttonDanger,
    color: colors.buttonDangerText,
    borderRadius: '3px',
    cursor: 'pointer',
    transition: 'opacity 0.2s ease',
  },
});

const getStyles = (colors: ReturnType<typeof getTheme>) => ({
  sidebar: {
    width: '300px',
    backgroundColor: colors.surface,
    borderRight: `1px solid ${colors.border}`,
    display: 'flex',
    flexDirection: 'column' as const,
    height: '100vh',
    transition: 'background-color 0.3s ease, border-color 0.3s ease',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    padding: '20px',
    borderBottom: `1px solid ${colors.border}`,
  },
  title: {
    margin: 0,
    fontSize: '18px',
    fontWeight: 'bold' as const,
    color: colors.text,
  },
  newButton: {
    width: '32px',
    height: '32px',
    borderRadius: '50%',
    border: `1px solid ${colors.border}`,
    backgroundColor: colors.buttonPrimary,
    color: colors.buttonPrimaryText,
    fontSize: '20px',
    cursor: 'pointer',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    transition: 'opacity 0.3s ease',
  },
  content: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '10px 0',
  },
  message: {
    padding: '16px',
    color: colors.textSecondary,
    fontSize: '14px',
    textAlign: 'center' as const,
  },
});
