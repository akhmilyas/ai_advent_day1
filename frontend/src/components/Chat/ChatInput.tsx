import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { getTheme } from '../../themes';

interface ChatInputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  loading: boolean;
}

export const ChatInput: React.FC<ChatInputProps> = ({
  value,
  onChange,
  onSubmit,
  loading,
}) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Submit on Enter, allow new line on Shift+Enter
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (!loading && value.trim()) {
        onSubmit(e as any);
      }
    }
  };

  return (
    <form
      onSubmit={onSubmit}
      style={{
        display: 'flex',
        alignItems: 'flex-end',
        gap: '10px',
        padding: '20px',
        borderTop: '1px solid',
        boxShadow: 'none',
        transition: 'background-color 0.3s ease, border-color 0.3s ease',
        backgroundColor: colors.header,
        borderTopColor: colors.border,
      }}
    >
      <textarea
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Type your message... (Shift+Enter for new line)"
        style={{
          flex: 1,
          padding: '12px',
          fontSize: '16px',
          borderRadius: '4px',
          border: '1px solid',
          boxShadow: 'none !important',
          outline: 'none',
          minHeight: '44px',
          maxHeight: '200px',
          resize: 'vertical',
          fontFamily: 'inherit',
          lineHeight: '1.5',
          transition: 'background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease',
          backgroundColor: colors.input,
          color: colors.text,
          borderColor: colors.border,
        } as React.CSSProperties}
        disabled={loading}
        rows={1}
      />
      <button
        type="submit"
        style={{
          padding: '12px 24px',
          fontSize: '16px',
          border: 'none',
          borderRadius: '4px',
          cursor: loading || !value.trim() ? 'not-allowed' : 'pointer',
          transition: 'background-color 0.3s ease',
          backgroundColor: loading || !value.trim() ? colors.buttonPrimaryDisabled : colors.buttonPrimary,
          color: colors.buttonPrimaryText,
        }}
        disabled={loading || !value.trim()}
      >
        Send
      </button>
    </form>
  );
};
