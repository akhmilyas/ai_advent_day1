import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { getTheme } from '../../themes';

interface SummarizeButtonProps {
  onClick: () => void;
  disabled: boolean;
}

export const SummarizeButton: React.FC<SummarizeButtonProps> = ({ onClick, disabled }) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  return (
    <button
      onClick={onClick}
      disabled={disabled}
      style={{
        padding: '8px 12px',
        fontSize: '16px',
        border: `1px solid ${colors.border}`,
        borderRadius: '4px',
        cursor: disabled ? 'wait' : 'pointer',
        transition: 'background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease',
        backgroundColor: disabled ? colors.border : colors.surface,
        color: colors.text,
        opacity: disabled ? 0.6 : 1,
      }}
      title={disabled ? 'Summarizing...' : 'Summarize conversation'}
    >
      {disabled ? '⏳' : '📝'}
    </button>
  );
};
