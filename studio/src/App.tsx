import { useRef, useState, useEffect } from "react";
import { StudioMode } from "./components/ConfigBuilder/StudioMode";
import { ErrorDisplay } from "./components/ErrorDisplay";
import { SuccessDisplay } from "./components/SuccessDisplay";
import { useStore } from "./store/useStore";
import { useServersStore } from "./store/serversStore";
import { useAppsStore } from "./store/appsStore";
import { useServersInit } from "./lib/hooks/useServersInit";
import { useHealthPolling } from "./lib/hooks/useHealthPolling";
import { AppHeader } from "./components/AppHeader";
import { CoverOverlay } from "./components/CoverOverlay";
import { LoginModal } from "./components/LoginModal";
import { LogDrawer } from "./components/LogDrawer";
import { DEFAULT_SUPPORTED_FILE_TYPES } from "./lib/constants";
import { SessionXRayModal } from "./components/SessionXRayModal";
import { CloudAuthModal } from "./components/CloudAuthModal";
import { useCloudAuthStore } from "./store/cloudAuthStore";
import { useCloudStore } from './store/cloudStore';
import { HOST_SERVER_ID } from "./lib/embedded";
import { probeServer } from "./lib/probeServer";
import { WelcomeScreen } from "./components/WelcomeScreen";
import { HostConnectingScreen } from "./components/HostConnectingScreen";

function App() {
  // Initialize servers from persisted state (probe health)
  useServersInit();

  // Poll active server health every 10s
  useHealthPolling();

  const loadAgents = useStore((state) => state.loadAgents);
  const activeServer = useServersStore((s) => s.getActiveServer());

  // Modal states
  const [loginServerId, setLoginServerId] = useState<string | null>(null);
  const [showLoginModal, setShowLoginModal] = useState(false);
  const [showCloudAuth, setShowCloudAuth] = useState(false);
  const isCloudAuthenticated = useCloudAuthStore((s) => s.isAuthenticated);
  const cloudStatus = useCloudStore((s) => s.status);
  const cloudConnect = useCloudStore((s) => s.connect);
  const [showLogDrawer, setShowLogDrawer] = useState(false);
  const xRaySessionId = useStore((state) => state.xRaySessionId);
  const setXRaySessionId = useStore((state) => state.setXRaySessionId);

  // Listen for open-log-drawer events
  useEffect(() => {
    const handleOpenLogDrawer = () => setShowLogDrawer(true);
    const handleAuthExpired = () => {
      const activeServerId = useServersStore.getState().activeServerId;
      if (activeServerId) {
        // Clear stale app tokens so fresh ones are fetched on re-login
        useAppsStore.getState().clearServerTokens(activeServerId);
        useServersStore.getState().setServerStatus(activeServerId, 'auth_required', 'Session expired');
      }
    };

    window.addEventListener('open-log-drawer', handleOpenLogDrawer);
    window.addEventListener('auth-expired', handleAuthExpired);
    return () => {
      window.removeEventListener('open-log-drawer', handleOpenLogDrawer);
      window.removeEventListener('auth-expired', handleAuthExpired);
    };
  }, []);

  // Track active server ID to only reset session on actual switch
  const activeServerIdRef = useRef<string | null>(null);

  // When active server changes, load agents and reset state
  useEffect(() => {
    if (activeServer?.status === 'authenticated') {
      if (activeServerIdRef.current !== activeServer.config.id) {
        const { cancelGeneration } = useStore.getState();
        cancelGeneration();

        useStore.getState().setAvailableAgents([]);
        useStore.getState().setAgentCard(null);
        useStore.getState().setSupportedFileTypes([...DEFAULT_SUPPORTED_FILE_TYPES]);
        useStore.getState().setSchema(null);
        useStore.setState({ agentsLoaded: false });

        (async () => {
          await Promise.all([
            loadAgents(),
            useAppsStore.getState().loadApps(activeServer.config.id)
          ]);

          const selectedAgent = useStore.getState().selectedAgent;
          const agentName = selectedAgent?.name;
          const activeAppId = useAppsStore.getState().getActiveAppId(activeServer.config.id);

          const sessions = useStore.getState().sessions;
          const effectiveAppId = activeAppId || 'default';
          const serverAppSessions = Object.values(sessions)
            .filter(s => {
              if (s.serverId !== activeServer.config.id) return false;
              if (s.appId !== effectiveAppId) return false;
              // Match agent (allow unassigned sessions)
              if (agentName && s.agentName && s.agentName !== agentName) return false;
              return true;
            })
            .sort((a, b) => new Date(b.created).getTime() - new Date(a.created).getTime());

          if (serverAppSessions.length > 0) {
            useStore.getState().selectSession(serverAppSessions[0].id);
            useStore.getState().loadSessionEvents(serverAppSessions[0].id);
          } else {
            useStore.getState().createSession();
          }

          // Sync sessions from backend
          useStore.getState().syncSessions();
        })();

        activeServerIdRef.current = activeServer.config.id;
      } else {
        useStore.getState().setAvailableAgents([]);
        useStore.setState({ agentsLoaded: false });
        loadAgents();
        useAppsStore.getState().loadApps(activeServer.config.id);
      }

      // Check studio mode
      const checkStudioMode = async () => {
        try {
          const res = await fetch(`${activeServer.config.url}/health`);
          if (res.ok) {
            const data = await res.json();
            const studioEnabled = !!(data.admin && data.admin.enabled);
            useStore.getState().setIsServerStudioEnabled(studioEnabled);
            if (!studioEnabled) {
              useStore.getState().setStudioViewMode('chat');
            }
          }
        } catch (e) {
          console.error('Failed to check server studio mode:', e);
        }
      };
      checkStudioMode();
    }
  }, [activeServer?.config.id, activeServer?.status, activeServer?.config.url, loadAgents]);

  const handleLoginRequest = (serverId: string) => {
    setLoginServerId(serverId);
    setShowLoginModal(true);
  };

  const handleLogoutRequest = async (_serverId: string) => {
    // In web mode, "logout" just clears the admin key and sets status to auth_required
    const servers = useServersStore.getState().servers;
    const server = servers[_serverId];
    if (!server) return;
    useServersStore.getState().updateServerConfig(_serverId, { adminKey: undefined });
    useServersStore.getState().setServerStatus(_serverId, 'auth_required');
  };

  const handleLoginSuccess = () => {
    setShowLoginModal(false);
  };

  const handleRetryConnection = async () => {
    if (!activeServer) return;
    probeServer(activeServer.config.id, activeServer.config.url);
  };

  const loginServer = loginServerId ? useServersStore.getState().servers[loginServerId] : null;

  return (
    <div className="flex flex-col w-screen h-screen bg-black text-white overflow-hidden font-sans">
      <AppHeader
        onLoginRequest={handleLoginRequest}
        onLogoutRequest={handleLogoutRequest}
        onOpenLogDrawer={() => setShowLogDrawer(true)}
      />

      <main className="flex-1 min-h-0 relative overflow-hidden flex flex-col">
        <CoverOverlay
          onLoginClick={() => activeServer && handleLoginRequest(activeServer.config.id)}
          onRetryClick={handleRetryConnection}
        />

        {activeServer?.status === 'authenticated' ? (
          <StudioMode />
        ) : activeServer?.config.id === HOST_SERVER_ID ? (
          <HostConnectingScreen server={activeServer} onRetry={handleRetryConnection} />
        ) : !activeServer ? (
          <WelcomeScreen
            isCloudAuthenticated={isCloudAuthenticated}
            cloudStatus={cloudStatus}
            onConnectCloud={() => {
              if (isCloudAuthenticated) {
                cloudConnect();
              } else {
                setShowCloudAuth(true);
              }
            }}
          />
        ) : (
          <div className="flex-1 bg-gray-900/20" />
        )}
      </main>

      {loginServer && (
        <LoginModal
          server={loginServer.config}
          isOpen={showLoginModal}
          onClose={() => setShowLoginModal(false)}
          onLoginSuccess={handleLoginSuccess}
        />
      )}
      <CloudAuthModal
        isOpen={showCloudAuth}
        onClose={() => setShowCloudAuth(false)}
        onAuthenticated={() => {
          setShowCloudAuth(false);
          cloudConnect();
        }}
        onSkip={() => setShowCloudAuth(false)}
      />
      <ErrorDisplay />
      <SuccessDisplay />
      <LogDrawer
        isOpen={showLogDrawer}
        onClose={() => setShowLogDrawer(false)}
      />

      {xRaySessionId && (
        <SessionXRayModal
          sessionId={xRaySessionId}
          onClose={() => setXRaySessionId(null)}
        />
      )}
    </div>
  );
}

export default App;
