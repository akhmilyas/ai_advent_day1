import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { getTheme } from '../../themes';
import { ThemeToggle } from './ThemeToggle';
import { SettingsButton } from './SettingsButton';
import { SummarizeButton } from './SummarizeButton';
import { LogoutButton } from './LogoutButton';

interface ChatHeaderProps {
  conversationTitle: string;
  model: string;
  conversationFormat: string | undefined;
  showSummarizeButton: boolean;
  summarizing: boolean;
  onSummarize: () => void;
  onOpenSettings: () => void;
  onLogout: () => void;
}

export const ChatHeader: React.FC<ChatHeaderProps> = ({
  conversationTitle,
  model,
  conversationFormat,
  showSummarizeButton,
  summarizing,
  onSummarize,
  onOpenSettings,
  onLogout,
}) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  return (
    <div style={{
      display: 'flex',
      justifyContent: 'space-between',
      alignItems: 'center',
      padding: '20px',
      borderBottom: '1px solid',
      transition: 'background-color 0.3s ease, border-color 0.3s ease',
      backgroundColor: colors.header,
      borderBottomColor: colors.border,
    }}>
      <div>
        <h2 style={{
          margin: 0,
          transition: 'color 0.3s ease',
          color: colors.text,
        }}>
          {conversationTitle || 'AI Chat'}
        </h2>
        <div style={{
          display: 'flex',
          gap: '16px',
          alignItems: 'center',
          flexWrap: 'wrap' as const,
        }}>
          {model && (
            <p style={{
              margin: '4px 0 0 0',
              fontSize: '12px',
              opacity: 0.7,
              transition: 'color 0.3s ease',
              color: colors.textSecondary,
            }}>
              {model}
            </p>
          )}
          {conversationFormat && conversationFormat !== 'text' && (
            <p style={{
              margin: '4px 0 0 0',
              fontSize: '12px',
              opacity: 0.8,
              transition: 'color 0.3s ease',
              color: colors.textSecondary,
            }}>
              Format: <strong style={{ color: colors.text }}>{conversationFormat.toUpperCase()}</strong>
            </p>
          )}
        </div>
      </div>
      <div style={{
        display: 'flex',
        gap: '10px',
        alignItems: 'center',
      }}>
        {showSummarizeButton && (
          <SummarizeButton onClick={onSummarize} disabled={summarizing} />
        )}
        <ThemeToggle />
        <SettingsButton onClick={onOpenSettings} />
        <LogoutButton onClick={onLogout} />
      </div>
    </div>
  );
};
