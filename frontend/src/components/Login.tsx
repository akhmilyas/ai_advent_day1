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
  const [isRegister, setIsRegister] = useState(false);

  // Login form state
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  // Register form state
  const [regUsername, setRegUsername] = useState('');
  const [regEmail, setRegEmail] = useState('');
  const [regPassword, setRegPassword] = useState('');
  const [regPasswordConfirm, setRegPasswordConfirm] = useState('');

  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await AuthService.login({ username, password });
      onLogin();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Login failed';
      setError(errorMessage || 'Invalid credentials');
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // Validation
    if (!regUsername.trim()) {
      setError('Username is required');
      return;
    }

    if (!regEmail.trim()) {
      setError('Email is required');
      return;
    }

    if (regPassword.length < 6) {
      setError('Password must be at least 6 characters');
      return;
    }

    if (regPassword !== regPasswordConfirm) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);

    try {
      await AuthService.register({
        username: regUsername,
        email: regEmail,
        password: regPassword,
      });
      onLogin();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Registration failed';
      if (errorMessage.includes('409') || errorMessage.includes('already exists')) {
        setError('Username already exists');
      } else {
        setError(errorMessage || 'Registration failed');
      }
    } finally {
      setLoading(false);
    }
  };

  const toggleMode = () => {
    setIsRegister(!isRegister);
    setError('');
    setUsername('');
    setPassword('');
    setRegUsername('');
    setRegEmail('');
    setRegPassword('');
    setRegPasswordConfirm('');
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
    authBox: {
      backgroundColor: colors.surface,
      padding: '40px',
      borderRadius: '8px',
      boxShadow: `0 2px 10px ${colors.shadow}`,
      width: '100%',
      maxWidth: '450px',
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
    buttonDisabled: {
      opacity: 0.6,
      cursor: 'not-allowed',
    },
    error: {
      color: '#ff6b6b',
      fontSize: '14px',
      margin: '0',
      padding: '10px',
      backgroundColor: 'rgba(255, 107, 107, 0.1)',
      borderRadius: '4px',
      textAlign: 'center' as const,
    },
    toggleContainer: {
      textAlign: 'center' as const,
      marginTop: '20px',
      fontSize: '14px',
      color: colors.textSecondary,
      transition: 'color 0.3s ease',
    },
    toggleLink: {
      color: colors.buttonPrimary,
      cursor: 'pointer',
      textDecoration: 'underline',
      fontWeight: 'bold' as const,
      marginLeft: '5px',
    },
    hint: {
      textAlign: 'center' as const,
      marginTop: '15px',
      fontSize: '13px',
      color: colors.textSecondary,
      transition: 'color 0.3s ease',
    },
  };

  return (
    <div style={styles.container}>
      <div style={styles.authBox}>
        {!isRegister ? (
          <>
            <h2 style={styles.title}>AI Chat Login</h2>
            <form onSubmit={handleLogin} style={styles.form}>
              <input
                type="text"
                placeholder="Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                style={styles.input}
                disabled={loading}
                required
              />
              <input
                type="password"
                placeholder="Password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                style={styles.input}
                disabled={loading}
                required
              />
              {error && <p style={styles.error}>{error}</p>}
              <button
                type="submit"
                style={{ ...styles.button, ...(loading ? styles.buttonDisabled : {}) }}
                disabled={loading}
              >
                {loading ? 'Logging in...' : 'Login'}
              </button>
            </form>
            <p style={styles.hint}>Default credentials: demo / demo123</p>
            <p style={styles.toggleContainer}>
              Don't have an account?
              <span style={styles.toggleLink} onClick={toggleMode}>
                Register here
              </span>
            </p>
          </>
        ) : (
          <>
            <h2 style={styles.title}>Create Account</h2>
            <form onSubmit={handleRegister} style={styles.form}>
              <input
                type="text"
                placeholder="Username"
                value={regUsername}
                onChange={(e) => setRegUsername(e.target.value)}
                style={styles.input}
                disabled={loading}
                required
              />
              <input
                type="email"
                placeholder="Email (optional)"
                value={regEmail}
                onChange={(e) => setRegEmail(e.target.value)}
                style={styles.input}
                disabled={loading}
              />
              <input
                type="password"
                placeholder="Password (min 6 characters)"
                value={regPassword}
                onChange={(e) => setRegPassword(e.target.value)}
                style={styles.input}
                disabled={loading}
                required
              />
              <input
                type="password"
                placeholder="Confirm Password"
                value={regPasswordConfirm}
                onChange={(e) => setRegPasswordConfirm(e.target.value)}
                style={styles.input}
                disabled={loading}
                required
              />
              {error && <p style={styles.error}>{error}</p>}
              <button
                type="submit"
                style={{ ...styles.button, ...(loading ? styles.buttonDisabled : {}) }}
                disabled={loading}
              >
                {loading ? 'Creating account...' : 'Register'}
              </button>
            </form>
            <p style={styles.toggleContainer}>
              Already have an account?
              <span style={styles.toggleLink} onClick={toggleMode}>
                Login here
              </span>
            </p>
          </>
        )}
      </div>
    </div>
  );
};
