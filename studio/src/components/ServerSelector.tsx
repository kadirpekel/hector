import { CloudAuthModal } from './CloudAuthModal';
import { CloudProgressModal } from './CloudProgressModal';
import { useState, useEffect } from 'react';

import { Server, Check, ChevronDown, Loader2, Cloud } from 'lucide-react';
import { useServersStore } from '../store/serversStore';
import { useCloudAuthStore } from '../store/cloudAuthStore';
import { useCloudStore } from '../store/cloudStore';
import { useStore } from '../store/useStore';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from './ui/dropdown-menu';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from './ui/tooltip';
import type { ServerState } from '../types';

interface ServerSelectorProps {
    onLoginRequest: (serverId: string) => void;
    onLogoutRequest: (serverId: string) => void;
    className?: string;
    variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link";
}

export function ServerSelector({ onLoginRequest, onLogoutRequest, className, variant = "outline" }: ServerSelectorProps) {
    const [showCloudAuth, setShowCloudAuth] = useState(false);
    const [isOpen, setIsOpen] = useState(false);

    const servers = useServersStore((s) => s.servers);
    const activeServerId = useServersStore((s) => s.activeServerId);
    const selectServer = useServersStore((s) => s.selectServer);
    const activeServer = useServersStore((s) => s.getActiveServer());
    const serverList = Object.values(servers);

    const isAuthenticated = useCloudAuthStore((s) => s.isAuthenticated);
    const cloudStatus = useCloudStore((s) => s.status);
    const cloudConnect = useCloudStore((s) => s.connect);
    const cloudConnect = useCloudStore((s) => s.connect);

    // RAG Indexing Status
    const ragStatus = useStore((s) => s.ragStatus);

    // Listen for programmatic open requests
    useEffect(() => {
        const handleOpenSelector = () => setIsOpen(true);
        window.addEventListener('open-server-selector', handleOpenSelector);
        return () => window.removeEventListener('open-server-selector', handleOpenSelector);
    }, []);

    const handleCloudConnect = async () => {
        setIsOpen(false);
        if (!isAuthenticated) {
            setShowCloudAuth(true);
            return;
        }
        await cloudConnect();
    };

    const handleSelectServer = (id: string) => {
        const ragStatus = useStore.getState().ragStatus;
        if (ragStatus?.isIndexing) {
            const confirmed = window.confirm(
                'Document indexing is currently in progress.\n\n' +
                'Switching servers will interrupt indexing. If checkpoints are not enabled, progress may be lost.\n\n' +
                'Are you sure you want to switch?'
            );
            if (!confirmed) return;
        }

        selectServer(id);
        setIsOpen(false);
        useStore.getState().setRagStatus(null);
        useStore.getState().createSession();
    };

    const getStatusColor = (status: ServerState['status']) => {
        switch (status) {
            case 'authenticated': return 'bg-green-500';
            case 'auth_required': return 'bg-yellow-500';
            case 'disconnected': return 'bg-red-500';
            case 'unreachable': return 'bg-red-500';
            default: return 'bg-blue-500';
        }
    };

    return (
        <>
            <DropdownMenu open={isOpen} onOpenChange={setIsOpen}>
                <DropdownMenuTrigger asChild>
                    <Button variant={variant} className={cn("min-w-0 w-40 sm:w-72 justify-between px-3", className)}>
                        <div className="flex items-center gap-2">
                            <Server size={14} />
                            <span className="truncate max-w-[280px]">
                                {activeServer?.config.name || 'Select Server'}
                            </span>
                        </div>
                        <div className="flex items-center gap-2">
                            {activeServer && (
                                ragStatus?.isIndexing ? (
                                    <TooltipProvider>
                                        <Tooltip>
                                            <TooltipTrigger asChild>
                                                <span className="flex items-center cursor-help">
                                                    <Loader2 size={12} className="animate-spin text-yellow-400" />
                                                </span>
                                            </TooltipTrigger>
                                            <TooltipContent>
                                                <div className="text-xs">
                                                    <div className="font-medium mb-1">Indexing Documents</div>
                                                    {Object.entries(ragStatus.stores).map(([name, store]) => (
                                                        <div key={name} className="text-gray-400">
                                                            {name}: {(store as any).indexed}/{(store as any).total}
                                                        </div>
                                                    ))}
                                                </div>
                                            </TooltipContent>
                                        </Tooltip>
                                    </TooltipProvider>
                                ) : (
                                    <div className={cn(
                                        "w-2 h-2 rounded-full",
                                        getStatusColor(activeServer.status),
                                        (activeServer.status === 'disconnected' || activeServer.status === 'unreachable') && 'animate-pulse'
                                    )} />
                                )
                            )}
                            <ChevronDown size={14} className="opacity-50" />
                        </div>
                    </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-[320px] bg-gray-900 border-gray-800 text-gray-300">
                    <DropdownMenuLabel>Servers</DropdownMenuLabel>
                    <DropdownMenuSeparator className="bg-gray-800" />
                    <div className="max-h-[200px] overflow-y-auto">
                        {serverList.length === 0 ? (
                            <div className="p-2 text-sm text-center text-gray-500">No servers configured</div>
                        ) : (
                            serverList.map((server) => (
                                <DropdownMenuItem
                                    key={server.config.id}
                                    onSelect={() => handleSelectServer(server.config.id)}
                                    className={cn(
                                        "flex items-center justify-between cursor-pointer focus:bg-gray-800 focus:text-white",
                                        activeServerId === server.config.id && "bg-gray-800/50"
                                    )}
                                >
                                    <div className="flex items-center gap-2 overflow-hidden">
                                        <div className={cn(
                                            "w-2 h-2 rounded-full flex-shrink-0",
                                            getStatusColor(server.status),
                                            (server.status === 'disconnected' || server.status === 'unreachable') && 'animate-pulse'
                                        )} />
                                        <div className="flex flex-col min-w-0">
                                            <span className="font-medium truncate text-xs">{server.config.name}</span>
                                            <span className="text-[10px] text-gray-500 truncate">{server.config.url}</span>
                                        </div>
                                    </div>

                                    <div className="flex items-center gap-1 shrink-0">
                                        {activeServerId === server.config.id && <Check size={14} className="text-green-500" />}
                                    </div>
                                </DropdownMenuItem>
                            ))
                        )}
                    </div>
                    <DropdownMenuSeparator className="bg-gray-800" />
                    {/* Cloud Connect */}
                    <DropdownMenuItem
                        onSelect={handleCloudConnect}
                        className="cursor-pointer focus:bg-gray-800 focus:text-white"
                        disabled={cloudStatus === 'working'}
                    >
                        {cloudStatus === 'working' ? (
                            <Loader2 size={14} className="mr-2 animate-spin" />
                        ) : (
                            <Cloud size={14} className="mr-2" />
                        )}
                        <span>
                            {isAuthenticated
                                ? (cloudStatus === 'connected' ? 'Reconnect to Cloud' : 'Connect to Cloud')
                                : 'Set Up Cloud'}
                        </span>
                    </DropdownMenuItem>
                </DropdownMenuContent>
            </DropdownMenu>
            <CloudAuthModal
                isOpen={showCloudAuth}
                onClose={() => setShowCloudAuth(false)}
                onAuthenticated={() => {
                    setShowCloudAuth(false);
                    cloudConnect();
                }}
                onSkip={() => setShowCloudAuth(false)}
            />
            <CloudProgressModal />
        </>
    );
}
