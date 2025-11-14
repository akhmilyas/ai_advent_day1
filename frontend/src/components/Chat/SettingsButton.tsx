import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { getTheme } from '../../themes';

interface SettingsButtonProps {
  onClick: () => void;
}

export const SettingsButton: React.FC<SettingsButtonProps> = ({ onClick }) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  return (
    <button
      onClick={onClick}
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
      title="Settings"
    >
      ⚙️
    </button>
  );
};
