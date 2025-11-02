import React, { useState } from 'react';
import { Login } from './components/Login';
import { Chat } from './components/Chat';
import { AuthService } from './services/auth';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(
    AuthService.isAuthenticated()
  );

  return (
    <div>
      {isAuthenticated ? (
        <Chat onLogout={() => setIsAuthenticated(false)} />
      ) : (
        <Login onLogin={() => setIsAuthenticated(true)} />
      )}
    </div>
  );
}

export default App;
