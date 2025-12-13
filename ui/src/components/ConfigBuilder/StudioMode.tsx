import React, { useState, useEffect, useRef, useCallback } from 'react';
import * as yaml from 'js-yaml';
import { CheckCircle, XCircle, Rocket, Settings, Download } from 'lucide-react';
import { cn } from '../../lib/utils';
import { yamlToGraph } from '../../lib/canvas-converter';
import { CanvasMode } from './CanvasMode';
import { ChatWidget } from './ChatWidget';
import { SettingsModal } from './SettingsModal';
import { ConfigEditor } from './ConfigEditor';
import { InfrastructureSidebar } from './InfrastructureSidebar';
import { useStore } from '../../store/useStore';
import { api } from '../../services/api';
import { configureYamlSchema } from '../../lib/monaco';
import { EDITOR } from '../../lib/constants';

export const StudioMode: React.FC = () => {
  const [yamlContent, setYamlContent] = useState<string>('');
  const [isValidYaml, setIsValidYaml] = useState(true);
  const [validationError, setValidationError] = useState<string>('');
  const [lastValidYaml, setLastValidYaml] = useState<string>('');
  const [editorTheme, setEditorTheme] = useState<'vs-dark' | 'vs-light' | 'hc-black'>('hc-black');
  const [loading, setLoading] = useState(true);
  const [deploying, setDeploying] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(true);
  const [nodes, setNodes] = useState<any[]>([]);
  const editorRef = useRef<any>(null);
  const latestYamlContentRef = useRef<string>('');
  const debounceTimerRef = useRef<number | null>(null);

  // Studio Mode State
  const [isStudioModeEnabled, setIsStudioModeEnabled] = useState(true);

  // Check Server Mode
  useEffect(() => {
    const checkMode = async () => {
      try {
        const res = await fetch('/health');
        if (res.ok) {
          const data = await res.json();
          // If studio_mode is explicitly false, disable it in UI
          if (data.studio_mode === false) {
            setIsStudioModeEnabled(false);
            setViewMode('chat');
            setSidebarCollapsed(true);
          }
        }
      } catch (e) {
        console.error("Failed to check server mode:", e);
      }
    };
    checkMode();
  }, []);

  // 'design' = Left Pane Full (Toggle Editor/Canvas)
  // 'split' = Left Pane (Resizable) + Right Pane (Chat)
  // 'chat' = Right Pane Full
  const [viewMode, setViewMode] = useState<'design' | 'split' | 'chat'>('split');
  // Content of the Left Pane (in Design/Split modes)
  const [designView, setDesignView] = useState<'editor' | 'canvas'>('editor');

  // Resizable Panel Logic (Must be before conditional returns)
  const [rightPanelWidth, setRightPanelWidth] = useState(500);
  const isDraggingRef = useRef(false);

  // Auto-switch to Canvas when message sent (to show flow)
  const handleMessageSent = useCallback(() => {
    // Clear selection so the user can see the flow execution without distraction
    useStore.getState().setSelectedNodeId(null);

    // If in Split mode and looking at Editor, switch to Canvas to see the flow
    if (viewMode === 'split' && designView === 'editor') {
      setDesignView('canvas');
    }
  }, [viewMode, designView]);

  const startResizing = useCallback(() => {
    isDraggingRef.current = true;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none'; // Prevent text selection
  }, []);

  const stopResizing = useCallback(() => {
    isDraggingRef.current = false;
    document.body.style.cursor = 'default';
    document.body.style.userSelect = '';
  }, []);

  const resize = useCallback((e: MouseEvent) => {
    if (isDraggingRef.current) {
      const newWidth = window.innerWidth - e.clientX;
      // Constraints: Min 300px, Max 800px (or 50% screen)
      if (newWidth > 300 && newWidth < window.innerWidth * 0.7) {
        setRightPanelWidth(newWidth);
      }
    }
  }, []);

  useEffect(() => {
    window.addEventListener('mousemove', resize);
    window.addEventListener('mouseup', stopResizing);
    return () => {
      window.removeEventListener('mousemove', resize);
      window.removeEventListener('mouseup', stopResizing);
    };
  }, [resize, stopResizing]);

  const handleEditorChange = useCallback((value: string | undefined) => {
    const val = value || '';
    latestYamlContentRef.current = val;

    // Clear existing timer
    if (debounceTimerRef.current !== null) {
      clearTimeout(debounceTimerRef.current);
    }

    // Set new timer - single unified debounce
    debounceTimerRef.current = window.setTimeout(() => {
      setYamlContent(val);
      validateYaml(val);
      try {
        const { nodes: newNodes } = yamlToGraph(val);
        setNodes(newNodes);
      } catch { }
    }, EDITOR.DEBOUNCE_DELAY_MS);
  }, []);

  // Sync editor content when switching back to editor
  useEffect(() => {
    if (((viewMode === 'design' || viewMode === 'split') && designView === 'editor') && editorRef.current) {
      editorRef.current.setValue(latestYamlContentRef.current);
    }
  }, [viewMode, designView]);

  const handleEditorMount = React.useCallback((editor: any) => {
    editorRef.current = editor;
  }, []);

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current !== null) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  // Fetch and configure schema
  useEffect(() => {
    // Only fetch schema if in studio mode
    if (!isStudioModeEnabled) return;

    const initSchema = async () => {
      try {
        const schema = await api.fetchSchema();
        configureYamlSchema(schema);
        useStore.getState().setSchema(schema);
      } catch (e) {
        console.error("Failed to fetch schema:", e);
      }
    };
    initSchema();
  }, [isStudioModeEnabled]);

  // Update schema with dynamic enums (agents) when YAML changes
  useEffect(() => {
    const timer = setTimeout(() => {
      try {
        const parsed = yaml.load(yamlContent) as any;
        const agents = parsed?.agents ? Object.keys(parsed.agents) : [];
        if (agents.length > 0) {
          // Clone schema to avoid mutation issues (if any)
          const currentSchema = useStore.getState().schema;
          if (currentSchema) {
            // Deep clone simple strategy
            const newSchema = JSON.parse(JSON.stringify(currentSchema));

            // Inject dynamic enums for known fields that reference agents
            // patchEnum helper...
            const patchEnum = (defName: string, propName: string) => {
              if (newSchema.definitions?.[defName]?.properties?.[propName]) {
                if (newSchema.definitions[defName].properties[propName].type === 'array') {
                  newSchema.definitions[defName].properties[propName].items = {
                    type: "string",
                    enum: agents
                  };
                } else {
                  newSchema.definitions[defName].properties[propName].enum = agents;
                }
              }
            };
            patchEnum("AgentConfig", "sub_agents");
            configureYamlSchema(newSchema);
          }
        }
      } catch (e) {
        // Invalid YAML, ignore
      }
    }, 1000); // Debounce
    return () => clearTimeout(timer);
  }, [yamlContent]);

  // Load initial config
  useEffect(() => {
    // Don't try loading config if not in studio mode (it will fail/is irrelevant)
    // We can rely on the checkMode effect to set isStudioModeEnabled to false eventually
    // But initially it is true.
    // If we are truly not in studio mode, the /api/config might fail 403.
    // We should handle that.

    const loadConfig = async () => {
      try {
        const response = await fetch('/api/config');

        // If 403 Forbidden, we are likely not in studio mode
        if (response.status === 403) {
          return;
        }

        if (response.ok) {
          const text = await response.text();
          latestYamlContentRef.current = text;
          setYamlContent(text);
          setLastValidYaml(text);
          validateYaml(text);
          if (editorRef.current) {
            editorRef.current.setValue(text);
          }
        }
      } catch (error) {
        console.error('Failed to load config:', error);
      } finally {
        setLoading(false);
      }
    };

    loadConfig();
  }, []);

  const validateYaml = (content: string) => {
    try {
      const parsed = yaml.load(content) as any;

      // Basic Schema Validation
      if (!parsed || typeof parsed !== 'object') throw new Error('Root must be an object');
      if (!parsed.agents) throw new Error('Missing "agents" section');

      // Check agents
      Object.entries(parsed.agents).forEach(([id, agent]: [string, any]) => {
        if (!agent.llm && !agent.type) throw new Error(`Agent "${id}" missing "llm" or "type"`);
        if (agent.type && ['sequential', 'parallel', 'loop'].includes(agent.type)) {
          if (!agent.sub_agents || !Array.isArray(agent.sub_agents)) {
            throw new Error(`Workflow "${id}" missing "sub_agents" array`);
          }
        }
      });

      setIsValidYaml(true);
      setValidationError('');
      setLastValidYaml(content);
      return true;
    } catch (e) {
      const error = e as Error;
      setIsValidYaml(false);
      setValidationError(error.message);
      return false;
    }
  };

  const handleDeploy = async () => {
    const contentToDeploy = latestYamlContentRef.current;

    if (!validateYaml(contentToDeploy)) {
      useStore.getState().setError('Cannot deploy invalid YAML configuration. Please fix validation errors first.');
      return;
    }

    setDeploying(true);

    try {
      const response = await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/yaml' },
        body: contentToDeploy,
      });

      const result = await response.json();

      if (response.ok) {
        useStore.getState().setSuccessMessage('Configuration deployed successfully! Agents are reloading...');

        // Poll for agents with retry logic instead of blind timeout
        const reloadAgentsWithRetry = async (maxRetries = 5, delayMs = 500) => {
          for (let attempt = 0; attempt < maxRetries; attempt++) {
            try {
              // Wait before attempting (exponential backoff)
              await new Promise(resolve => setTimeout(resolve, delayMs * Math.pow(1.5, attempt)));

              await useStore.getState().reloadAgents();
              console.log(`✅ Agents reloaded after deploy (attempt ${attempt + 1})`);
              return; // Success!
            } catch (e) {
              console.warn(`Failed to reload agents (attempt ${attempt + 1}/${maxRetries}):`, e);
              if (attempt === maxRetries - 1) {
                // Final attempt failed
                console.error('Failed to reload agents after all retries:', e);
                useStore.getState().setError('Failed to reload agents after deployment. Please refresh the page.');
              }
            }
          }
        };

        // Start reload in background (don't await to avoid blocking UI)
        reloadAgentsWithRetry();
      } else {
        useStore.getState().setError(`Deploy failed: ${result.error || result.message || 'Unknown error'}`);
      }
    } catch (err) {
      const error = err as Error;
      useStore.getState().setError(`Deploy error: ${error.message}`);
    } finally {
      setDeploying(false);
    }
  };

  const handleDownload = () => {
    const blob = new Blob([latestYamlContentRef.current], { type: 'text/yaml' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'config.yaml';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const handleFormatYAML = () => {
    try {
      const current = latestYamlContentRef.current;
      const parsed = yaml.load(current);
      const formatted = yaml.dump(parsed, { indent: 2, lineWidth: -1, noRefs: true });

      latestYamlContentRef.current = formatted;
      setYamlContent(formatted);
      validateYaml(formatted);

      if (editorRef.current) {
        editorRef.current.setValue(formatted);
      }
    } catch (e) {
      const error = e as Error;
      console.error(`Format error: ${error.message}`);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-hector-darker to-black">
        <div className="text-white">Loading configuration...</div>
      </div>
    );
  }

  // Determine visibility
  const showLeft = (viewMode === 'design' || viewMode === 'split') && isStudioModeEnabled;
  const showRight = viewMode === 'chat' || viewMode === 'split';

  return (
    <div className="flex flex-col h-screen bg-gradient-to-br from-hector-darker to-black text-white">
      {/* Header */}
      <div className="h-10 bg-black/40 border-b border-white/10 grid grid-cols-3 px-4 items-center">
        {/* Left: Logo */}
        <div className="flex items-center gap-3">
          <div className="w-3 h-3 rounded-full bg-hector-green shadow-[0_0_10px_rgba(16,185,129,0.5)]" />
          <span className="font-bold tracking-wider text-sm">HECTOR {isStudioModeEnabled ? "STUDIO" : "CHAT"}</span>
        </div>

        {/* Center: View Modes Toggle - HIDDEN IN CHAT-ONLY MODE */}
        <div className="flex justify-center">
          {isStudioModeEnabled && (
            <div className="flex items-center bg-white/5 rounded-lg p-0.5">
              <button
                onClick={() => setViewMode('design')}
                className={cn("px-4 py-1 text-xs font-medium rounded-md transition-all", viewMode === 'design' ? "bg-white/10 text-white shadow-sm" : "text-gray-400 hover:text-white")}
              >
                Design
              </button>
              <button
                onClick={() => setViewMode('split')}
                className={cn("px-4 py-1 text-xs font-medium rounded-md transition-all", viewMode === 'split' ? "bg-white/10 text-white shadow-sm" : "text-gray-400 hover:text-white")}
              >
                Split
              </button>
              <button
                onClick={() => setViewMode('chat')}
                className={cn("px-4 py-1 text-xs font-medium rounded-md transition-all", viewMode === 'chat' ? "bg-white/10 text-white shadow-sm" : "text-gray-400 hover:text-white")}
              >
                Chat
              </button>
            </div>
          )}
        </div>

        {/* Right: Download / Deploy - HIDDEN IN CHAT-ONLY MODE */}
        <div className="flex items-center justify-end gap-3">
          {isStudioModeEnabled && (
            <>
              <button
                onClick={handleDownload}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-white/5 hover:bg-white/10 text-xs text-gray-300 transition-colors"
                title="Download YAML"
              >
                <Download size={12} />
                <span className="font-medium">Download</span>
              </button>
              <button
                onClick={handleDeploy}
                disabled={!isValidYaml || deploying}
                className={cn(
                  "flex items-center gap-1.5 px-3 py-1.5 rounded transition-colors text-xs font-medium",
                  !isValidYaml || deploying ? "bg-white/5 text-gray-500" : "bg-hector-green text-white hover:bg-hector-green/90"
                )}
              >
                <Rocket size={12} />
                <span>{deploying ? 'Deploying...' : 'Deploy'}</span>
              </button>
            </>
          )}
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Sidebar - Always visible on far left (Infrastructure) - HIDDEN IN CHAT-ONLY MODE */}
        {showLeft && (
          <InfrastructureSidebar
            yamlContent={yamlContent}
            nodes={nodes}
            collapsed={sidebarCollapsed}
            onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
          />
        )}

        {/* Left Pane (Design View) */}
        {showLeft && (
          <div className="flex-1 flex flex-col min-w-0 bg-hector-darker/50">
            {/* Left Pane Header (Design Tools) */}
            <div className="h-9 border-b border-white/10 flex items-center justify-between px-3 bg-black/10">
              {/* Design View Toggle */}
              <div className="flex items-center gap-2">
                <span className="text-[10px] uppercase font-bold text-gray-500 tracking-wider">View</span>
                <div className="flex items-center bg-white/5 rounded-lg p-0.5">
                  <button
                    onClick={() => setDesignView('editor')}
                    className={cn("px-2 py-0.5 text-xs font-medium rounded transition-all flex items-center gap-1.5", designView === 'editor' ? "bg-hector-green text-white" : "text-gray-400 hover:text-white")}
                  >
                    <Settings size={10} />
                    <span>Code</span>
                  </button>
                  <button
                    onClick={() => setDesignView('canvas')}
                    className={cn("px-2 py-0.5 text-xs font-medium rounded transition-all flex items-center gap-1.5", designView === 'canvas' ? "bg-indigo-500 text-white" : "text-gray-400 hover:text-white")}
                  >
                    <Rocket size={10} className="rotate-45" />
                    <span>Canvas</span>
                  </button>
                </div>
              </div>

              {/* Right Side: Format Button */}
              <button
                onClick={handleFormatYAML}
                className="flex items-center gap-1.5 text-gray-400 hover:text-white transition-colors text-xs font-medium px-2 py-1 rounded hover:bg-white/5"
                title="Format YAML"
              >
                <span>Format</span>
              </button>
            </div>

            <div className="flex-1 overflow-hidden relative">
              {designView === 'canvas' ? (
                <>
                  <CanvasMode yamlContent={lastValidYaml} />
                  {!isValidYaml && (
                    <div className="absolute top-4 right-4 bg-orange-500/10 text-orange-400 px-2 py-1 rounded text-xs border border-orange-500/20 backdrop-blur-sm z-50">
                      Showing last valid state
                    </div>
                  )}
                </>
              ) : (
                <div
                  className="w-full h-full"
                  onKeyDown={(e) => {
                    e.nativeEvent.stopImmediatePropagation();
                    e.stopPropagation();
                  }}
                >
                  <ConfigEditor
                    initialValue={latestYamlContentRef.current}
                    onChange={handleEditorChange}
                    onMount={handleEditorMount}
                    theme={editorTheme}
                  />
                </div>
              )}
            </div>
          </div>
        )}

        {/* Resize Handle (Only in Split Mode when both are visible) */}
        {viewMode === 'split' && isStudioModeEnabled && (
          <div
            className="w-1 cursor-col-resize hover:bg-hector-green/50 hover:shadow-[0_0_10px_rgba(16,185,129,0.5)] transition-colors active:bg-hector-green z-50 flex items-center justify-center group relative"
            onMouseDown={startResizing}
          >
            {/* Visual handle indicator */}
            <div className="h-8 w-1 bg-white/10 rounded-full group-hover:bg-white/40 transition-colors" />
            {/* Invisible wider area for easier grabbing */}
            <div className="absolute inset-y-0 -left-2 -right-2 bg-transparent z-10" />
          </div>
        )}

        {/* Right Pane (Chat View) */}
        {showRight && (
          <div
            className={cn("flex flex-col border-l border-white/10 bg-black/20 flex-shrink-0", viewMode === 'chat' ? "flex-1 border-l-0" : "")}
            style={{ width: viewMode === 'chat' ? '100%' : rightPanelWidth }}
          >
            {/* Pane Content */}
            <div className="flex-1 overflow-hidden relative">
              <ChatWidget
                state={viewMode === 'chat' ? 'maximized' : 'pane'}
                onStateChange={() => { }}
                isPinned={true}
                onPinChange={() => { }}
                onMessageSent={handleMessageSent}
                hideControls={!isStudioModeEnabled}
              />
            </div>
          </div>
        )}
      </div>

      {/* Footer (IDE Status Bar) - Refined */}
      {isStudioModeEnabled && (
        <div className="h-8 bg-black/80 border-t border-white/10 flex items-center justify-between px-4 text-xs select-none">

          {/* Left: Validation Status */}
          <div className="flex items-center gap-4">
            {isValidYaml ? (
              <div className="flex items-center gap-1.5 text-green-400">
                <CheckCircle size={12} />
                <span>Valid Configuration</span>
              </div>
            ) : (
              <div className="flex items-center gap-1.5 text-red-400" title={validationError}>
                <XCircle size={12} />
                <span>Invalid Configuration: {validationError}</span>
              </div>
            )}
          </div>

          {/* Right: Settings */}
          <div className="flex items-center gap-4">
            {/* Mode Indicator */}
            <span className="text-gray-500">
              Mode: <span className="text-gray-300 font-medium">{viewMode.toUpperCase()}</span>
              {showLeft && <span> / {designView.toUpperCase()}</span>}
            </span>

            <div className="w-px h-3 bg-white/10"></div>

            <button onClick={() => setShowSettings(true)} className="flex items-center gap-1.5 text-gray-400 hover:text-white transition-colors">
              <Settings size={12} />
              <span>Settings</span>
            </button>
          </div>
        </div>
      )}

      {showSettings && (
        <SettingsModal
          isOpen={showSettings}
          onClose={() => setShowSettings(false)}
          editorTheme={editorTheme}
          onThemeChange={setEditorTheme}
        />
      )}
    </div>
  );
};
