import { useEffect } from 'react';
import { Layout } from './components/Layout';
import { ChatArea } from './components/Chat/ChatArea';
import { useStore } from './store/useStore';

function App() {
  const { createSession, sessions, currentSessionId } = useStore();

  useEffect(() => {
    // Create a new session if none exists
    if (!currentSessionId && Object.keys(sessions).length === 0) {
      createSession();
    }
  }, [currentSessionId, sessions, createSession]);

  return (
    <Layout>
      <ChatArea />
    </Layout>
  );
}

export default App;
