import React, { useState } from 'react';
import { Login } from './components/Login';
import { Chat } from './components/Chat';
import { AuthService } from './services/auth';
import { ThemeProvider } from './contexts/ThemeContext';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(
    AuthService.isAuthenticated()
  );

  return (
    <ThemeProvider>
      <div>
        {isAuthenticated ? (
          <Chat onLogout={() => setIsAuthenticated(false)} />
        ) : (
          <Login onLogin={() => setIsAuthenticated(true)} />
        )}
      </div>
    </ThemeProvider>
  );
}

export default App;
