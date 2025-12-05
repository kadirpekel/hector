import React from "react";
import { X, Globe, Code, Radio, Minimize2 } from "lucide-react";
import { useStore } from "../store/useStore";

export const ConfigPanel: React.FC = () => {
  const {
    configVisible,
    setConfigVisible,
    endpointUrl,
    setEndpointUrl,
    protocol,
    setProtocol,
    streamingEnabled,
    setStreamingEnabled,
    minimalMode,
    setMinimalMode,
  } = useStore();

  if (!configVisible) return null;

  return (
    <div className="bg-hector-darker border-b border-white/10 p-4 animate-in slide-in-from-top duration-200">
      <div className="max-w-4xl mx-auto">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-gray-300 flex items-center gap-2">
            <Radio size={16} />
            Configuration
          </h3>
          <button
            onClick={() => setConfigVisible(false)}
            className="p-1 hover:bg-white/10 rounded transition-colors text-gray-400 hover:text-white"
            title="Close config"
          >
            <X size={16} />
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
          {/* Endpoint URL */}
          <div>
            <label className="block text-xs text-gray-400 mb-1.5 flex items-center gap-1.5">
              <Globe size={12} />
              Endpoint URL
            </label>
            <input
              type="text"
              value={endpointUrl}
              onChange={(e) => setEndpointUrl(e.target.value)}
              placeholder="http://localhost:8080"
              className="w-full px-3 py-2 bg-black/50 border border-white/20 rounded-lg text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-hector-green focus:border-hector-green"
            />
          </div>

          {/* Protocol */}
          <div>
            <label className="block text-xs text-gray-400 mb-1.5 flex items-center gap-1.5">
              <Code size={12} />
              Protocol
            </label>
            <select
              value={protocol}
              onChange={(e) =>
                setProtocol(e.target.value as "jsonrpc" | "rest")
              }
              className="w-full px-3 py-2 bg-black/50 border border-white/20 rounded-lg text-sm text-gray-200 focus:outline-none focus:ring-2 focus:ring-hector-green focus:border-hector-green"
            >
              <option value="jsonrpc">JSON-RPC 2.0</option>
              <option value="rest">REST (HTTP+JSON)</option>
            </select>
          </div>

          {/* Minimal Mode */}
          <div>
            <label className="block text-xs text-gray-400 mb-1.5 flex items-center gap-1.5">
              <Minimize2 size={12} />
              UI Mode
            </label>
            <div className="flex items-center gap-3 pt-2">
              <label className="flex items-center gap-2 text-sm text-gray-300 cursor-pointer">
                <input
                  type="checkbox"
                  checked={minimalMode}
                  onChange={(e) => setMinimalMode(e.target.checked)}
                  className="w-4 h-4 rounded border-white/20 bg-black/50 text-hector-green focus:ring-hector-green focus:ring-offset-0"
                />
                <span>Minimal mode</span>
              </label>
            </div>
          </div>
        </div>

        {/* Streaming Toggle */}
        <div className="flex items-center gap-2 pt-2 border-t border-white/10">
          <input
            type="checkbox"
            id="streaming"
            checked={streamingEnabled}
            onChange={(e) => setStreamingEnabled(e.target.checked)}
            className="w-4 h-4 rounded border-white/20 bg-black/50 text-hector-green focus:ring-hector-green focus:ring-offset-0"
          />
          <label
            htmlFor="streaming"
            className="text-sm text-gray-300 cursor-pointer"
          >
            Enable streaming responses
          </label>
        </div>

        <div className="mt-3 text-xs text-gray-500">
          <p>
            Changes take effect immediately. Endpoint URL is used for all API
            requests.
          </p>
        </div>
      </div>
    </div>
  );
};
