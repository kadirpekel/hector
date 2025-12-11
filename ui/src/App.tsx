import { useEffect, useRef } from "react";
import { StudioMode } from "./components/ConfigBuilder/StudioMode";
import { ErrorDisplay } from "./components/ErrorDisplay";
import { SuccessDisplay } from "./components/SuccessDisplay";
import { useStore } from "./store/useStore";

function App() {
  console.log("RENDER: App (Top Level)");
  // Use selectors for better performance - only subscribe to specific state slices
  // Use selectors for better performance - only subscribe to specific state slices
  const loadAgents = useStore((state) => state.loadAgents);
  const createSession = useStore((state) => state.createSession);
  const currentSessionId = useStore((state) => state.currentSessionId);

  // Prevent double initialization on mount
  const initializedRef = useRef(false);

  useEffect(() => {
    // Run initialization only once
    if (initializedRef.current) return;
    initializedRef.current = true;

    // Load agents on mount
    loadAgents();

    // Create a new session if none exists
    // Read directly from state to avoid subscription causing re-renders
    const state = useStore.getState();
    if (!currentSessionId && Object.keys(state.sessions).length === 0) {
      createSession();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // Empty deps - run only on mount

  // Note: Auto-selection is now handled by:
  // 1. loadAgents() restoration from localStorage
  // 2. ChatWidget's initialSelectionDoneRef guard
  // Removed this effect to prevent race conditions and competing auto-selection logic

  // Studio Mode is the unified interface (config editor + chat)
  return (
    <>
      <StudioMode />
      <ErrorDisplay />
      <SuccessDisplay />
    </>
  );
}

export default App;
