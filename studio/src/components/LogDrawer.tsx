import React, { useRef, useEffect, useState } from "react";
import { X, Copy, Trash2, Pause, Play, Minimize2 } from "lucide-react";

interface LogDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

export const LogDrawer: React.FC<LogDrawerProps> = ({ isOpen, onClose }) => {
  const [logs, setLogs] = useState<string[]>([]);
  const [paused, setPaused] = useState(false);
  const logContainerRef = useRef<HTMLDivElement>(null);
  const pausedRef = useRef(false);
  pausedRef.current = paused;

  useEffect(() => {
    if (!paused && logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
    }
  }, [logs, paused]);

  const copyLogs = () => {
    navigator.clipboard.writeText(logs.join('\n'));
  };

  const clearLogs = () => {
    setLogs([]);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed bottom-0 left-0 right-0 z-40 bg-zinc-900 border-t border-white/10 transition-all"
      style={{ height: '250px' }}
    >
      <div className="flex items-center justify-between px-4 py-2 border-b border-white/10">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-gray-300">Logs</span>
          <span className="text-xs text-gray-500">{logs.length} entries</span>
        </div>
        <div className="flex items-center gap-1">
          <button onClick={() => setPaused(!paused)} className="p-1.5 hover:bg-white/10 rounded transition-colors" title={paused ? "Resume" : "Pause"}>
            {paused ? <Play size={14} className="text-gray-400" /> : <Pause size={14} className="text-gray-400" />}
          </button>
          <button onClick={copyLogs} className="p-1.5 hover:bg-white/10 rounded transition-colors" title="Copy">
            <Copy size={14} className="text-gray-400" />
          </button>
          <button onClick={clearLogs} className="p-1.5 hover:bg-white/10 rounded transition-colors" title="Clear">
            <Trash2 size={14} className="text-gray-400" />
          </button>
          <button onClick={onClose} className="p-1.5 hover:bg-white/10 rounded transition-colors" title="Close">
            <X size={14} className="text-gray-400" />
          </button>
        </div>
      </div>
      <div ref={logContainerRef} className="overflow-y-auto p-3 font-mono text-xs text-gray-400 leading-5" style={{ height: 'calc(100% - 40px)' }}>
        {logs.length === 0 ? (
          <div className="text-gray-600 text-center mt-8">No logs available. Log streaming is not yet supported in web mode.</div>
        ) : (
          logs.map((line, i) => (
            <div key={i} className="whitespace-pre-wrap break-all hover:bg-white/5">{line}</div>
          ))
        )}
      </div>
    </div>
  );
};
