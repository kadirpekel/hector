import { ServerSettingsModal } from './ServerSettingsModal';
import { CloudAuthModal } from './CloudAuthModal';
import { CloudProgressModal } from './CloudProgressModal';
import { useState, useEffect } from 'react';

import { Plus, Server, LogIn, LogOut, Check, ChevronDown, Loader2, Settings, X, Cloud } from 'lucide-react';
import { useServersStore } from '../store/serversStore';
import { useCloudAuthStore } from '../store/cloudAuthStore';
import { useCloudStore } from '../store/cloudStore';
import { useStore } from '../store/useStore';
import { CLOUD_ENABLED } from '../lib/cloudEnabled';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { Input } from './ui/input';
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
    const [showAddForm, setShowAddForm] = useState(false);
    const [settingsModalServerId, setSettingsModalServerId] = useState<string | null>(null);
    const [showCloudAuth, setShowCloudAuth] = useState(false);
    const [newName, setNewName] = useState('');
    const [newUrl, setNewUrl] = useState('');
    const [newAdminKey, setNewAdminKey] = useState('');
    const [isOpen, setIsOpen] = useState(false);

    const servers = useServersStore((s) => s.servers);
    const activeServerId = useServersStore((s) => s.activeServerId);
    const selectServer = useServersStore((s) => s.selectServer);
    const removeServer = useServersStore((s) => s.removeServer);
    const activeServer = useServersStore((s) => s.getActiveServer());
    const serverList = Object.values(servers);

    const isAuthenticated = useCloudAuthStore((s) => s.isAuthenticated);
    const cloudStatus = useCloudStore((s) => s.status);
    const cloudConnect = useCloudStore((s) => s.connect);
    const cloudLogout = useCloudAuthStore((s) => s.logout);
    const cloudDisconnect = useCloudStore((s) => s.disconnect);

    // RAG Indexing Status
    const ragStatus = useStore((s) => s.ragStatus);

    // Listen for programmatic open requests from welcome screen
    useEffect(() => {
        const handleOpenSelector = () => {
            setIsOpen(true);
            setShowAddForm(true);
        };

        window.addEventListener('open-server-selector', handleOpenSelector);
        return () => {
            window.removeEventListener('open-server-selector', handleOpenSelector);
        };
    }, []);

    const handleCloudConnect = async () => {
        if (!CLOUD_ENABLED) return;
        setIsOpen(false);
        if (!isAuthenticated) {
            setShowCloudAuth(true);
            return;
        }
        await cloudConnect();
    };

    const handleAddServer = async (e: React.MouseEvent) => {
        e.preventDefault();
        e.stopPropagation();
        if (!newName.trim() || !newUrl.trim()) return;

        try {
            const url = newUrl.trim().replace(/\/+$/, '');
            const id = crypto.randomUUID();
            const adminKey = newAdminKey.trim() || undefined;

            useServersStore.getState().addServer({ id, name: newName.trim(), url, adminKey });

            // Probe the server to determine its status
            try {
                const resp = await fetch(`${url}/health`, { signal: AbortSignal.timeout(5000) });
                if (resp.ok) {
                    if (adminKey) {
                        // Verify admin key
                        const authResp = await fetch(`${url}/admin/apps`, {
                            headers: { 'Authorization': `Bearer ${adminKey}` },
                            signal: AbortSignal.timeout(5000),
                        });
                        if (authResp.ok) {
                            useServersStore.getState().setServerStatus(id, 'authenticated');
                        } else {
                            useServersStore.getState().setServerStatus(id, 'auth_required');
                        }
                    } else {
                        useServersStore.getState().setServerStatus(id, 'auth_required');
                    }
                } else if (resp.status === 401 || resp.status === 403) {
                    useServersStore.getState().setServerStatus(id, 'auth_required');
                } else {
                    useServersStore.getState().setServerStatus(id, 'unreachable');
                }
            } catch {
                useServersStore.getState().setServerStatus(id, 'unreachable');
            }

            setNewName('');
            setNewUrl('');
            setNewAdminKey('');
            setShowAddForm(false);
        } catch (error) {
            console.error('Failed to add server:', error);
        }
    };

    const handleRemoveServer = (id: string, e: React.MouseEvent) => {
        e.stopPropagation();
        if (!confirm('Remove this server from the list?')) return;
        removeServer(id);
    };

    const handleOpenSettings = (id: string, e: React.MouseEvent) => {
        e.stopPropagation();
        setIsOpen(false);
        setSettingsModalServerId(id);
    };

    const handleSelectServer = (id: string) => {
        // Check if RAG indexing is in progress and warn user
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

        // Clear RAG status when switching
        useStore.getState().setRagStatus(null);
        // Clear chat context when switching servers
        useStore.getState().createSession();

        // Probe server if status is unknown
        const server = useServersStore.getState().servers[id];
        if (server && (server.status === 'added' || !server.status)) {
            const url = server.config.url;
            fetch(`${url}/health`, { signal: AbortSignal.timeout(5000) })
                .then((resp) => {
                    if (resp.ok) {
                        useServersStore.getState().setServerStatus(id, server.config.adminKey ? 'authenticated' : 'auth_required');
                    } else {
                        useServersStore.getState().setServerStatus(id, resp.status === 401 || resp.status === 403 ? 'auth_required' : 'unreachable');
                    }
                })
                .catch(() => {
                    useServersStore.getState().setServerStatus(id, 'unreachable');
                });
        }
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

                                    <div className="flex items-center gap-1">
                                        {activeServerId === server.config.id && <Check size={14} className="text-green-500" />}
                                        {server.status === 'auth_required' && (
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                className="h-6 w-6 hover:bg-gray-700"
                                                onClick={(e) => { e.stopPropagation(); onLoginRequest(server.config.id); }}
                                            >
                                                <LogIn size={12} />
                                            </Button>
                                        )}
                                        {server.status === 'authenticated' && (
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                className="h-6 w-6 hover:bg-gray-700"
                                                onClick={(e) => { e.stopPropagation(); onLogoutRequest(server.config.id); }}
                                            >
                                                <LogOut size={12} />
                                            </Button>
                                        )}
                                        {/* Settings */}
                                        <TooltipProvider>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        className="h-6 w-6 hover:bg-gray-700"
                                                        onClick={(e) => handleOpenSettings(server.config.id, e)}
                                                    >
                                                        <Settings size={12} />
                                                    </Button>
                                                </TooltipTrigger>
                                                <TooltipContent>
                                                    <p>Settings</p>
                                                </TooltipContent>
                                            </Tooltip>
                                        </TooltipProvider>
                                        {/* Remove */}
                                        <TooltipProvider>
                                            <Tooltip>
                                                <TooltipTrigger asChild>
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        className="h-6 w-6 hover:bg-gray-700"
                                                        onClick={(e) => handleRemoveServer(server.config.id, e)}
                                                    >
                                                        <X size={12} />
                                                    </Button>
                                                </TooltipTrigger>
                                                <TooltipContent>
                                                    <p>Remove</p>
                                                </TooltipContent>
                                            </Tooltip>
                                        </TooltipProvider>
                                    </div>
                                </DropdownMenuItem>
                            ))
                        )}
                    </div>
                    <DropdownMenuSeparator className="bg-gray-800" />
                    {/* Cloud Connect - only shown when VITE_CLOUD_ENABLED=true */}
                    {CLOUD_ENABLED && (
                        <>
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
                            {isAuthenticated && (
                                <DropdownMenuItem
                                    onSelect={() => {
                                        cloudDisconnect();
                                        cloudLogout();
                                    }}
                                    className="cursor-pointer focus:bg-gray-800 text-red-400 focus:text-red-300"
                                >
                                    <X size={14} className="mr-2" />
                                    <span>Forget Cloud Credentials</span>
                                </DropdownMenuItem>
                            )}
                            <DropdownMenuSeparator className="bg-gray-800" />
                        </>
                    )}
                    {showAddForm ? (
                        <div className="p-2 space-y-2">
                            <Input
                                placeholder="Server Name"
                                value={newName}
                                onChange={(e) => setNewName(e.target.value)}
                                className="h-7 text-xs bg-black/40 border-gray-700"
                                onClick={(e) => e.stopPropagation()}
                                onKeyDown={(e) => e.key === 'Tab' && e.stopPropagation()}
                            />
                            <Input
                                placeholder="URL"
                                value={newUrl}
                                onChange={(e) => setNewUrl(e.target.value)}
                                className="h-7 text-xs bg-black/40 border-gray-700"
                                onClick={(e) => e.stopPropagation()}
                                onKeyDown={(e) => e.key === 'Tab' && e.stopPropagation()}
                            />
                            <Input
                                placeholder="Admin Key (optional)"
                                type="password"
                                value={newAdminKey}
                                onChange={(e) => setNewAdminKey(e.target.value)}
                                className="h-7 text-xs bg-black/40 border-gray-700"
                                onClick={(e) => e.stopPropagation()}
                                onKeyDown={(e) => e.key === 'Tab' && e.stopPropagation()}
                            />
                            <div className="flex gap-2">
                                <Button size="sm" variant="default" className="h-7 text-xs w-full bg-hector-green hover:bg-hector-green/80 text-white" onClick={handleAddServer}>
                                    Add
                                </Button>
                                <Button size="sm" variant="ghost" className="h-7 text-xs w-full hover:bg-gray-800" onClick={(e) => { e.stopPropagation(); setShowAddForm(false); }}>
                                    Cancel
                                </Button>
                            </div>
                        </div>
                    ) : (
                        <DropdownMenuItem onSelect={(e) => { e.preventDefault(); setShowAddForm(true); }} className="cursor-pointer focus:bg-gray-800 focus:text-white">
                            <Plus size={14} className="mr-2" />
                            <span>Add Server</span>
                        </DropdownMenuItem>
                    )}
                </DropdownMenuContent>
            </DropdownMenu>
            {settingsModalServerId && (
                <ServerSettingsModal
                    isOpen={!!settingsModalServerId}
                    onClose={() => setSettingsModalServerId(null)}
                    serverId={settingsModalServerId}
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
            <CloudProgressModal />
        </>
    );
}
