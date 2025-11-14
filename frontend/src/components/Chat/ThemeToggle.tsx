import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { getTheme } from '../../themes';

export const ThemeToggle: React.FC = () => {
  const { theme, toggleTheme } = useTheme();
  const colors = getTheme(theme === 'dark');

  return (
    <button
      onClick={toggleTheme}
      style={{
        padding: '8px 12px',
        fontSize: '16px',
        border: '1px solid',
        borderRadius: '4px',
        cursor: 'pointer',
        transition: 'background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease',
        backgroundColor: colors.surface,
        color: colors.text,
        borderColor: colors.border,
      }}
      title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
    >
      {theme === 'light' ? '🌙' : '☀️'}
    </button>
  );
};
