import { clsx } from 'clsx';
import { Loader2, Lock, WifiOff } from 'lucide-react';
import { useServersStore } from '../store/serversStore';
import type { ServerStatus } from '../types';

interface CoverOverlayProps {
    onLoginClick: () => void;
    onRetryClick: () => void;
}

export function CoverOverlay({ onLoginClick, onRetryClick }: CoverOverlayProps) {
    const activeServer = useServersStore((s) => s.getActiveServer());

    if (!activeServer || activeServer.status === 'authenticated') {
        return null; // No overlay needed
    }

    const overlayContent: Record<Exclude<ServerStatus, 'authenticated'>, {
        icon: typeof Loader2;
        title: string;
        subtitle: string;
        action?: { label: string; onClick: () => void };
        animate?: boolean;
    }> = {
        added: {
            icon: Loader2,
            title: 'Connecting...',
            subtitle: `Discovering ${activeServer.config.name}`,
            animate: true,
        },
        checking: {
            icon: Loader2,
            title: 'Starting...',
            subtitle: `Starting server ${activeServer.config.name}`,
            animate: true,
        },
        auth_required: {
            icon: Lock,
            title: 'Login Required',
            subtitle: 'Authenticate to access this server',
            action: { label: 'Login', onClick: onLoginClick },
        },
        disconnected: {
            icon: WifiOff,
            title: 'Connection Lost',
            subtitle: activeServer.lastError || 'Attempting to reconnect...',
            action: { label: 'Retry', onClick: onRetryClick },
        },
        error: {
            icon: WifiOff,
            title: 'Connection Failed',
            subtitle: activeServer.lastError || 'Server encountered an error',
            action: { label: 'Retry', onClick: onRetryClick },
        },
        unreachable: {
            icon: WifiOff,
            title: 'Server Unavailable',
            subtitle: activeServer.lastError || 'Cannot reach the server',
            action: { label: 'Retry', onClick: onRetryClick },
        },
        stopping: {
            icon: Loader2,
            title: 'Stopping...',
            subtitle: 'Stopping server...',
            animate: true,
        },
        stopped: {
            icon: WifiOff,
            title: 'Stopped',
            subtitle: 'The server is stopped',
            action: { label: 'Start', onClick: onRetryClick }, // Retry acts as Start/Connect
        },
    };

    const content = overlayContent[activeServer.status as Exclude<ServerStatus, 'authenticated'>];
    if (!content) return null;

    const Icon = content.icon;

    return (
        <div className="absolute inset-0 z-30 flex items-center justify-center backdrop-blur-sm bg-black/60">
            <div className="text-center space-y-4">
                <Icon
                    size={48}
                    className={clsx(
                        'mx-auto text-gray-400',
                        content.animate && 'animate-spin'
                    )}
                />
                <div>
                    <h2 className="text-xl font-semibold text-white">{content.title}</h2>
                    <p className="text-gray-400 mt-1">{content.subtitle}</p>
                </div>
                {content.action && (
                    <button
                        onClick={content.action.onClick}
                        className={clsx(
                            'px-6 py-2 rounded-lg font-medium transition-all duration-200',
                            'bg-hector-red hover:bg-hector-red/90 text-white',
                            'hover:shadow-lg hover:shadow-red-500/20',
                            'active:scale-95 cursor-pointer'
                        )}
                    >
                        {content.action.label}
                    </button>
                )}
            </div>
        </div>
    );
}
