import { useEffect } from "react";
import { StudioMode } from "./components/ConfigBuilder/StudioMode";
import { useStore } from "./store/useStore";

function App() {
  // Use selectors for better performance - only subscribe to specific state slices
  const createSession = useStore((state) => state.createSession);
  const sessions = useStore((state) => state.sessions);
  const currentSessionId = useStore((state) => state.currentSessionId);

  useEffect(() => {
    // Create a new session if none exists
    if (!currentSessionId && Object.keys(sessions).length === 0) {
      createSession();
    }
  }, [currentSessionId, sessions, createSession]);

  // Studio Mode is the unified interface (config editor + chat)
  return <StudioMode />;
}

export default App;
