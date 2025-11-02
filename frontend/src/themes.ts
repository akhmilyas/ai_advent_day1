export interface ThemeColors {
  // Background colors
  background: string;
  surface: string;
  surfaceAlt: string;

  // Text colors
  text: string;
  textSecondary: string;

  // Border colors
  border: string;
  borderLight: string;

  // Component colors
  header: string;
  input: string;
  placeholder: string;

  // Message colors
  userMessageBg: string;
  userMessageText: string;
  assistantMessageBg: string;
  assistantMessageBorder: string;
  assistantMessageText: string;

  // Button colors
  buttonPrimary: string;
  buttonPrimaryText: string;
  buttonPrimaryDisabled: string;
  buttonDanger: string;
  buttonDangerText: string;

  // Empty state
  emptyStateText: string;

  // Shadows
  shadow: string;
}

export const lightTheme: ThemeColors = {
  background: '#f5f5f5',
  surface: '#ffffff',
  surfaceAlt: '#f9f9f9',

  text: '#333333',
  textSecondary: '#666666',

  border: '#dddddd',
  borderLight: '#eeeeee',

  header: '#ffffff',
  input: '#ffffff',
  placeholder: '#999999',

  userMessageBg: '#007bff',
  userMessageText: '#ffffff',
  assistantMessageBg: '#ffffff',
  assistantMessageBorder: '#dddddd',
  assistantMessageText: '#333333',

  buttonPrimary: '#007bff',
  buttonPrimaryText: '#ffffff',
  buttonPrimaryDisabled: '#cccccc',
  buttonDanger: '#dc3545',
  buttonDangerText: '#ffffff',

  emptyStateText: '#999999',

  shadow: 'rgba(0, 0, 0, 0.1)',
};

export const darkTheme: ThemeColors = {
  background: '#1a1a1a',
  surface: '#2d2d2d',
  surfaceAlt: '#3a3a3a',

  text: '#e0e0e0',
  textSecondary: '#b0b0b0',

  border: '#4a4a4a',
  borderLight: '#3a3a3a',

  header: '#2d2d2d',
  input: '#3a3a3a',
  placeholder: '#888888',

  userMessageBg: '#0056cc',
  userMessageText: '#ffffff',
  assistantMessageBg: '#3a3a3a',
  assistantMessageBorder: '#4a4a4a',
  assistantMessageText: '#e0e0e0',

  buttonPrimary: '#007bff',
  buttonPrimaryText: '#ffffff',
  buttonPrimaryDisabled: '#555555',
  buttonDanger: '#dc3545',
  buttonDangerText: '#ffffff',

  emptyStateText: '#888888',

  shadow: 'rgba(0, 0, 0, 0.5)',
};

export const getTheme = (isDark: boolean): ThemeColors => {
  return isDark ? darkTheme : lightTheme;
};
