import { CheckCircle, XCircle, ScrollText, Settings, LogOut } from 'lucide-react';
import { useStore } from '../store/useStore';
import { useServersStore } from '../store/serversStore';

interface StatusBarProps {
  onOpenLogDrawer: () => void;
  onOpenSettings: () => void;
  onLogout: () => void;
}

export function StatusBar({ onOpenLogDrawer, onOpenSettings, onLogout }: StatusBarProps) {
  const activeServer = useServersStore((s) => s.getActiveServer());
  const isServerStudioEnabled = useStore((s) => s.isServerStudioEnabled);
  const isValidYaml = useStore((s) => s.studioIsValidYaml);
  const validationError = useStore((s) => s.studioValidationError);
  const viewMode = useStore((s) => s.studioViewMode);
  const designView = useStore((s) => s.studioDesignView);

  const isAuthenticated = activeServer?.status === 'authenticated';
  const isStudioMode = isAuthenticated && isServerStudioEnabled;

  return (
    <div className="flex-shrink-0 h-7 bg-black/80 border-t border-white/10 flex items-center justify-between px-3 text-[11px] select-none">
      {/* Left: Validation (studio only) */}
      <div className="flex items-center gap-3 min-w-0">
        {isStudioMode && (
          isValidYaml ? (
            <div className="flex items-center gap-1.5 text-green-400/80">
              <CheckCircle size={11} />
              <span>Valid</span>
            </div>
          ) : (
            <div className="flex items-center gap-1.5 text-red-400/80" title={validationError}>
              <XCircle size={11} />
              <span className="truncate max-w-[200px]">Invalid: {validationError}</span>
            </div>
          )
        )}
      </div>

      {/* Center: Mode (studio only) */}
      <div className="absolute left-1/2 -translate-x-1/2 text-gray-600">
        {isStudioMode && (
          <>
            <span className="uppercase tracking-wider">{viewMode}</span>
            {(viewMode === 'design' || viewMode === 'split') && designView && (
              <span className="text-gray-700"> / {designView}</span>
            )}
          </>
        )}
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-0.5">
        {isAuthenticated && (
          <button
            onClick={onOpenLogDrawer}
            className="flex items-center gap-1.5 px-2 py-0.5 text-gray-500 hover:text-white hover:bg-white/5 rounded transition-colors"
            title="Logs"
          >
            <ScrollText size={12} />
            <span>Logs</span>
          </button>
        )}

        <button
          onClick={onOpenSettings}
          className="flex items-center gap-1.5 px-2 py-0.5 text-gray-500 hover:text-white hover:bg-white/5 rounded transition-colors"
          title="Settings"
        >
          <Settings size={12} />
          <span>Settings</span>
        </button>

        {isAuthenticated && (
          <button
            onClick={onLogout}
            className="flex items-center gap-1.5 px-2 py-0.5 text-gray-500 hover:text-red-400 hover:bg-white/5 rounded transition-colors"
            title="Logout"
          >
            <LogOut size={12} />
            <span>Logout</span>
          </button>
        )}
      </div>
    </div>
  );
}
