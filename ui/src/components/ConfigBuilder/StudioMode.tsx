import React, { useState, useEffect } from 'react';
import Editor from '@monaco-editor/react';
import * as yaml from 'js-yaml';
import { CheckCircle, XCircle, Download, Rocket, Code } from 'lucide-react';
import { cn } from '../../lib/utils';

import { CanvasMode } from './CanvasMode';
import { FloatingChatButton } from './FloatingChatButton';
import { ChatWidget } from './ChatWidget';
import { SettingsModal } from './SettingsModal';

type ChatState = 'closed' | 'popup' | 'maximized';

export const StudioMode: React.FC = () => {
  const [yamlContent, setYamlContent] = useState<string>('');
  const [isValidYaml, setIsValidYaml] = useState(true);
  const [validationError, setValidationError] = useState<string>('');
  const [lastValidYaml, setLastValidYaml] = useState<string>('');
  const [viewMode, setViewMode] = useState<'split' | 'canvas' | 'yaml'>('split');
  const [editorTheme, setEditorTheme] = useState<'vs-dark' | 'vs-light' | 'hc-black'>('hc-black');
  const [loading, setLoading] = useState(true);
  const [deploying, setDeploying] = useState(false);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [chatState, setChatState] = useState<ChatState>('closed');
  const [isPinned, setIsPinned] = useState(false);
  const [justDeployed, setJustDeployed] = useState(false);
  const [showSettings, setShowSettings] = useState(false);

  // Load initial config
  useEffect(() => {
    const loadConfig = async () => {
      try {
        const response = await fetch('/api/config');
        if (response.ok) {
          const text = await response.text();
          setYamlContent(text);
          setLastValidYaml(text);
          validateYaml(text);
        }
      } catch (error) {
        console.error('Failed to load config:', error);
        setMessage({ type: 'error', text: 'Failed to load configuration' });
      } finally {
        setLoading(false);
      }
    };

    loadConfig();
  }, []);

  const validateYaml = (content: string) => {
    try {
      yaml.load(content);
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

  const handleYamlChange = (value: string | undefined) => {
    const newValue = value || '';
    setYamlContent(newValue);
    validateYaml(newValue);
  };

  const handleDeploy = async () => {
    if (!isValidYaml) {
      setMessage({ type: 'error', text: 'Cannot deploy invalid YAML' });
      return;
    }

    setDeploying(true);
    setMessage(null);

    try {
      const response = await fetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/yaml' },
        body: yamlContent,
      });

      const result = await response.json();

      if (response.ok) {
        setMessage({ type: 'success', text: '✅ Config deployed successfully!' });
        setJustDeployed(true);
        setTimeout(() => setMessage(null), 3000);
        setTimeout(() => setJustDeployed(false), 10000);
      } else {
        setMessage({ type: 'error', text: result.error || 'Deploy failed' });
      }
    } catch (err) {
      const error = err as Error;
      setMessage({ type: 'error', text: `Deploy error: ${error.message}` });
    } finally {
      setDeploying(false);
    }
  };

  const handleDownload = () => {
    const blob = new Blob([yamlContent], { type: 'application/yaml' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'hector-config.yaml';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  const handleFormatYAML = () => {
    try {
      const parsed = yaml.load(yamlContent);
      const formatted = yaml.dump(parsed, { indent: 2, lineWidth: -1, noRefs: true });
      setYamlContent(formatted);
      validateYaml(formatted);
      setMessage({ type: 'success', text: 'YAML formatted successfully' });
      setTimeout(() => setMessage(null), 2000);
    } catch (e) {
      const error = e as Error;
      setMessage({ type: 'error', text: `Format error: ${error.message}` });
    }
  };

  useEffect(() => {
    if (chatState === 'closed' || isPinned || chatState === 'maximized') return;

    const handleClickOutside = (e: MouseEvent) => {
      const target = e.target as HTMLElement;
      if (!target.closest('.chat-widget') && !target.closest('.floating-chat-button')) {
        setChatState('closed');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [chatState, isPinned]);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-screen bg-gradient-to-br from-hector-darker to-black">
        <div className="text-white">Loading configuration...</div>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-screen bg-gradient-to-br from-hector-darker to-black text-white">
      {/* Header */}
      <div className="flex-none flex items-center justify-between px-6 py-3 border-b border-white/10 bg-black/20">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <Code size={20} className="text-hector-green" />
            <h1 className="text-xl font-bold">Studio Mode</h1>
          </div>
          
          <div className="flex items-center gap-2">
            {isValidYaml ? (
              <div className="flex items-center gap-1.5 px-3 py-1.5 bg-green-500/20 border border-green-500/50 rounded-lg text-green-300 text-sm">
                <CheckCircle size={16} />
                <span>Valid</span>
              </div>
            ) : (
              <div className="flex items-center gap-1.5 px-3 py-1.5 bg-red-500/20 border border-red-500/50 rounded-lg text-red-300 text-sm" title={validationError}>
                <XCircle size={16} />
                <span>Invalid</span>
              </div>
            )}
          </div>
        </div>

        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1 bg-white/5 rounded-lg p-1">
            <button
              onClick={() => setViewMode('canvas')}
              className={cn(
                'px-3 py-1 rounded text-xs font-medium transition-colors',
                viewMode === 'canvas' ? 'bg-hector-green text-white' : 'text-gray-400 hover:text-white'
              )}
            >
              Canvas
            </button>
            <button
              onClick={() => setViewMode('split')}
              className={cn(
                'px-3 py-1 rounded text-xs font-medium transition-colors',
                viewMode === 'split' ? 'bg-hector-green text-white' : 'text-gray-400 hover:text-white'
              )}
            >
              Split
            </button>
            <button
              onClick={() => setViewMode('yaml')}
              className={cn(
                'px-3 py-1 rounded text-xs font-medium transition-colors',
                viewMode === 'yaml' ? 'bg-hector-green text-white' : 'text-gray-400 hover:text-white'
              )}
            >
              YAML
            </button>
          </div>
          
          <button
            onClick={handleFormatYAML}
            className="px-4 py-2 bg-white/5 hover:bg-white/10 rounded-lg transition-colors text-sm"
          >
            Format
          </button>
          <button
            onClick={handleDownload}
            className="flex items-center gap-2 px-4 py-2 bg-white/5 hover:bg-white/10 rounded-lg transition-colors text-sm"
          >
            <Download size={16} />
            Download
          </button>
          <button
            onClick={handleDeploy}
            disabled={!isValidYaml || deploying}
            className="flex items-center gap-2 px-4 py-2 bg-hector-green hover:bg-hector-green/90 disabled:bg-gray-600 disabled:cursor-not-allowed rounded-lg transition-colors text-sm font-medium"
          >
            <Rocket size={16} />
            {deploying ? 'Deploying...' : 'Deploy'}
          </button>
        </div>
      </div>

      {message && (
        <div className={`flex-none px-6 py-3 ${message.type === 'success' ? 'bg-green-500/20 text-green-300' : 'bg-red-500/20 text-red-300'}`}>
          {message.text}
        </div>
      )}

      <div className="flex-1 flex overflow-hidden">
        {viewMode === 'canvas' && (
          <div className="flex-1">
            <CanvasMode yamlContent={lastValidYaml} />
          </div>
        )}

        {viewMode === 'split' && (
          <>
            <div className="flex-1 flex flex-col">
              <div className="flex-none px-4 py-2 bg-black/20 border-b border-white/10 flex items-center justify-between">
                <h2 className="text-sm font-medium text-gray-400">Visual Preview</h2>
                {!isValidYaml && <span className="text-xs text-orange-400">Showing last valid state</span>}
              </div>
              <div className="flex-1 overflow-hidden">
                <CanvasMode yamlContent={lastValidYaml} />
              </div>
            </div>

            <div className="flex-1 flex flex-col border-l border-white/10">
              <div className="flex-none px-4 py-2 bg-black/20 border-b border-white/10 flex items-center justify-between">
                <h2 className="text-sm font-medium text-gray-400">YAML Editor</h2>
                <select
                  value={editorTheme}
                  onChange={(e) => setEditorTheme(e.target.value as typeof editorTheme)}
                  className="px-2 py-1 text-xs bg-white/5 border border-white/10 rounded hover:bg-white/10 transition-colors"
                >
                  <option value="hc-black">High Contrast</option>
                  <option value="vs-dark">Dark</option>
                  <option value="vs-light">Light</option>
                </select>
              </div>
              <div className="flex-1 overflow-hidden">
                <Editor
                  height="100%"
                  language="yaml"
                  value={yamlContent}
                  onChange={handleYamlChange}
                  theme={editorTheme}
                  options={{
                    minimap: { enabled: false },
                    fontSize: 14,
                    lineNumbers: 'on',
                    wordWrap: 'on',
                    tabSize: 2,
                    insertSpaces: true,
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                  }}
                />
              </div>
            </div>
          </>
        )}

        {viewMode === 'yaml' && (
          <div className="flex-1 flex flex-col">
            <div className="flex-none px-4 py-2 bg-black/20 border-b border-white/10 flex items-center justify-between">
              <h2 className="text-sm font-medium text-gray-400">YAML Editor</h2>
              <select
                value={editorTheme}
                onChange={(e) => setEditorTheme(e.target.value as typeof editorTheme)}
                className="px-2 py-1 text-xs bg-white/5 border border-white/10 rounded hover:bg-white/10 transition-colors"
              >
                <option value="hc-black">High Contrast</option>
                <option value="vs-dark">Dark</option>
                <option value="vs-light">Light</option>
              </select>
            </div>
            <div className="flex-1 overflow-hidden">
              <Editor
                height="100%"
                language="yaml"
                value={yamlContent}
                onChange={handleYamlChange}
                theme={editorTheme}
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  lineNumbers: 'on',
                  wordWrap: 'on',
                  tabSize: 2,
                  insertSpaces: true,
                  scrollBeyondLastLine: false,
                  automaticLayout: true,
                }}
              />
            </div>
          </div>
        )}
      </div>

      <FloatingChatButton
        onClick={() => setChatState(chatState === 'closed' ? 'popup' : 'closed')}
        isOpen={chatState !== 'closed'}
        showBadge={justDeployed}
      />

      {chatState !== 'closed' && (
        <ChatWidget
          state={chatState === 'popup' ? 'popup' : chatState === 'maximized' ? 'maximized' : 'expanded'}
          onStateChange={(newState) => {
            if (newState === 'closed') setChatState('closed');
            else if (newState === 'popup') setChatState('popup');
            else if (newState === 'maximized') setChatState('maximized');
            else setChatState('maximized');
          }}
          isPinned={isPinned}
          onPinChange={setIsPinned}
        />
      )}

      {showSettings && <SettingsModal isOpen={showSettings} onClose={() => setShowSettings(false)} />}
    </div>
  );
};
