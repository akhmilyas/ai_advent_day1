import { useState, useCallback } from 'react';
import { AuthService, LoginCredentials, RegisterCredentials } from '../services/auth';

/**
 * Custom hook for managing authentication state
 * Provides login, register, logout, and authentication status
 */
export function useAuth() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(
    AuthService.isAuthenticated()
  );
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const login = useCallback(async (credentials: LoginCredentials) => {
    setLoading(true);
    setError(null);
    try {
      await AuthService.login(credentials);
      setIsAuthenticated(true);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Login failed';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const register = useCallback(async (credentials: RegisterCredentials) => {
    setLoading(true);
    setError(null);
    try {
      await AuthService.register(credentials);
      setIsAuthenticated(true);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Registration failed';
      setError(errorMessage);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  const logout = useCallback(() => {
    AuthService.logout();
    setIsAuthenticated(false);
  }, []);

  return {
    isAuthenticated,
    loading,
    error,
    login,
    register,
    logout
  };
}
