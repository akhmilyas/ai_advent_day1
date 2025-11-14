import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import { getTheme } from '../../themes';

interface LogoutButtonProps {
  onClick: () => void;
}

export const LogoutButton: React.FC<LogoutButtonProps> = ({ onClick }) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');

  return (
    <button
      onClick={onClick}
      style={{
        padding: '8px 16px',
        border: 'none',
        borderRadius: '4px',
        cursor: 'pointer',
        transition: 'background-color 0.3s ease',
        backgroundColor: colors.buttonDanger,
        color: colors.buttonDangerText,
      }}
    >
      Logout
    </button>
  );
};
