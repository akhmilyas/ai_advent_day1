import React, { useState } from 'react';
import { AuthService } from '../services/auth';
import { useTheme } from '../contexts/ThemeContext';
import { getTheme } from '../themes';

interface LoginProps {
  onLogin: () => void;
}

export const Login: React.FC<LoginProps> = ({ onLogin }) => {
  const { theme } = useTheme();
  const colors = getTheme(theme === 'dark');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await AuthService.login({ username, password });
      onLogin();
    } catch (err) {
      setError('Login failed. Please check your credentials.');
    } finally {
      setLoading(false);
    }
  };

  const styles = {
    container: {
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      height: '100vh',
      backgroundColor: colors.background,
      transition: 'background-color 0.3s ease',
    },
    loginBox: {
      backgroundColor: colors.surface,
      padding: '40px',
      borderRadius: '8px',
      boxShadow: `0 2px 10px ${colors.shadow}`,
      width: '100%',
      maxWidth: '400px',
      transition: 'background-color 0.3s ease, box-shadow 0.3s ease',
    },
    title: {
      textAlign: 'center' as const,
      marginBottom: '30px',
      color: colors.text,
      transition: 'color 0.3s ease',
    },
    form: {
      display: 'flex',
      flexDirection: 'column' as const,
      gap: '15px',
    },
    input: {
      padding: '12px',
      fontSize: '16px',
      backgroundColor: colors.input,
      color: colors.text,
      border: `1px solid ${colors.border}`,
      borderRadius: '4px',
      transition: 'background-color 0.3s ease, color 0.3s ease, border-color 0.3s ease',
    },
    button: {
      padding: '12px',
      fontSize: '16px',
      backgroundColor: colors.buttonPrimary,
      color: colors.buttonPrimaryText,
      border: 'none',
      borderRadius: '4px',
      cursor: 'pointer',
      transition: 'background-color 0.3s ease',
    },
    error: {
      color: '#ff6b6b',
      fontSize: '14px',
      margin: '0',
    },
    hint: {
      textAlign: 'center' as const,
      marginTop: '20px',
      fontSize: '14px',
      color: colors.textSecondary,
      transition: 'color 0.3s ease',
    },
  };

  return (
    <div style={styles.container}>
      <div style={styles.loginBox}>
        <h2 style={styles.title}>AI Chat Login</h2>
        <form onSubmit={handleSubmit} style={styles.form}>
          <input
            type="text"
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            style={styles.input}
            disabled={loading}
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={styles.input}
            disabled={loading}
          />
          {error && <p style={styles.error}>{error}</p>}
          <button type="submit" style={styles.button} disabled={loading}>
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>
        <p style={styles.hint}>Default credentials: demo / demo123</p>
      </div>
    </div>
  );
};
