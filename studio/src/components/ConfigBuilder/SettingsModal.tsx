import React, { useState } from "react";
import { X, Zap, Palette } from "lucide-react";
import { useStore } from "../../store/useStore";
import { cn } from "../../lib/utils";

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  editorTheme: 'vs-dark' | 'vs-light' | 'hc-black';
  onThemeChange: (theme: 'vs-dark' | 'vs-light' | 'hc-black') => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({
  isOpen,
  onClose,
  editorTheme,
  onThemeChange,
}) => {
  const streamingEnabled = useStore((state) => state.streamingEnabled);
  const setStreamingEnabled = useStore((state) => state.setStreamingEnabled);

  if (!isOpen) return null;

  return (
    <>
      <div
        className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 animate-in fade-in duration-200"
        onClick={onClose}
      />

      <div className="fixed inset-0 flex items-center justify-center z-50 p-4 pointer-events-none">
        <div
          className="bg-gradient-to-br from-hector-darker to-black border border-white/20 rounded-2xl shadow-2xl w-full max-w-lg max-h-[90vh] flex flex-col pointer-events-auto animate-in zoom-in-95 duration-200"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <svg
                className="w-5 h-5 text-hector-green"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
              </svg>
              Settings
            </h2>
            <button
              onClick={onClose}
              className="p-1 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors"
            >
              <X size={20} />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-6 min-h-[300px]">
            <div className="space-y-6">
              {/* Streaming */}
              <div>
                <label className="flex items-center justify-between cursor-pointer group">
                  <div>
                    <div className="text-sm font-medium text-gray-300 group-hover:text-white transition-colors flex items-center gap-2">
                      <Zap size={16} className="text-yellow-400" />
                      Streaming Responses
                    </div>
                    <div className="text-xs text-gray-500 mt-0.5">
                      Enable real-time message streaming
                    </div>
                  </div>
                  <div className="relative">
                    <input
                      type="checkbox"
                      checked={streamingEnabled}
                      onChange={(e) => setStreamingEnabled(e.target.checked)}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-black/50 border border-white/20 rounded-full peer-checked:bg-hector-green peer-checked:border-hector-green transition-all"></div>
                    <div className="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-5"></div>
                  </div>
                </label>
              </div>

              {/* Editor Theme */}
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2 flex items-center gap-2">
                  <Palette size={16} className="text-purple-400" />
                  Editor Theme
                </label>
                <select
                  value={editorTheme}
                  onChange={(e) => onThemeChange(e.target.value as any)}
                  className="w-full px-4 py-2 bg-black/50 border border-white/20 rounded-lg text-sm text-white focus:outline-none focus:ring-2 focus:ring-hector-green focus:border-transparent appearance-none cursor-pointer"
                >
                  <option value="hc-black">High Contrast (Default)</option>
                  <option value="vs-dark">Dark Visual Studio</option>
                  <option value="vs-light">Light Visual Studio</option>
                </select>
                <p className="text-xs text-gray-500 mt-1">
                  Color scheme for the YAML editor
                </p>
              </div>

              {/* Version */}
              <div className="pt-4 border-t border-white/10">
                <div className="text-xs text-gray-500">
                  Hector Studio v{__APP_VERSION__}
                </div>
              </div>
            </div>
          </div>

          <div className="px-6 py-4 bg-black/20 border-t border-white/10 flex justify-end">
            <button
              onClick={onClose}
              className="px-6 py-2 bg-hector-green hover:bg-hector-green/80 text-white rounded-lg font-medium transition-colors"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    </>
  );
};
