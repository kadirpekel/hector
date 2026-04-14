import { useState } from 'react';
import { Rocket, LayoutTemplate, Split, MessageSquare, ChevronRight } from 'lucide-react';
import { useStore } from '../store/useStore';
import { useServersStore } from '../store/serversStore';

import { ServerSelector } from './ServerSelector';
import { AppSelector } from './AppSelector';
import { cn } from '../lib/utils';
import { Button } from './ui/button';
import { api } from '../services/api';

import hectorIcon from '../assets/hector.png';

interface AppHeaderProps {
    onLoginRequest: (serverId: string) => void;
    onLogoutRequest: (serverId: string) => void;
}

export function AppHeader({ onLoginRequest, onLogoutRequest }: AppHeaderProps) {
    const activeServer = useServersStore((s) => s.getActiveServer());

    // Studio State
    const studioViewMode = useStore((s) => s.studioViewMode);
    const setStudioViewMode = useStore((s) => s.setStudioViewMode);
    const studioIsValidYaml = useStore((s) => s.studioIsValidYaml);
    const studioYamlContent = useStore((s) => s.studioYamlContent);
    const studioIsDeploying = useStore((s) => s.studioIsDeploying);
    const setStudioIsDeploying = useStore((s) => s.setStudioIsDeploying);
    const isServerStudioEnabled = useStore((s) => s.isServerStudioEnabled);

    const isStudioEnabled = activeServer?.status === 'authenticated' && isServerStudioEnabled;

    const handleDeploy = async () => {
        if (!isStudioEnabled || !studioIsValidYaml || studioIsDeploying) return;

        setStudioIsDeploying(true);
        try {
            const result = await api.saveConfig(studioYamlContent);
            await useStore.getState().reloadAgents();
            useStore.getState().setSuccessMessage(
                result.message || 'Configuration deployed and applied successfully!'
            );
        } catch (error) {
            useStore.getState().setError(`Deploy failed: ${(error as Error).message}`);
        } finally {
            setStudioIsDeploying(false);
        }
    };

    return (
        <header className="flex-shrink-0 h-14 bg-black/60 border-b border-white/5 flex items-center px-4 backdrop-blur-md z-50 justify-between gap-4">
            {/* Left: Branding & Context */}
            <div className="flex items-center gap-4 flex-shrink-0">
                <div className="flex items-center gap-3 select-none">
                    <img src={hectorIcon} alt="Hector" className="w-6 h-6 object-contain" />
                    <span className="font-bold tracking-wider text-xs text-white/90">STUDIO</span>
                </div>

                <div className="h-4 w-px bg-white/10" />

                <div className="flex items-center text-sm">
                    <ServerSelector
                        onLoginRequest={onLoginRequest}
                        onLogoutRequest={onLogoutRequest}
                        variant="ghost"
                        className="h-8 px-2 font-medium text-muted-foreground hover:text-foreground hover:bg-white/5 data-[state=open]:bg-white/5 justify-start w-auto sm:w-auto min-w-0 max-w-[200px]"
                    />
                    {activeServer?.status === 'authenticated' && (
                        <>
                            <ChevronRight className="text-muted-foreground mx-1" size={14} />
                            <AppSelector
                                variant="ghost"
                                className="h-8 px-2 font-medium text-foreground hover:bg-white/5 data-[state=open]:bg-white/5 justify-start w-auto sm:w-auto min-w-0 max-w-[200px]"
                            />
                        </>
                    )}
                </div>
            </div>

            {/* Center: Mode Switcher */}
            {isStudioEnabled && (
                <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 flex">
                    <div className="flex items-center bg-white/5 rounded-lg p-1 border border-white/5 shadow-inner">
                        <button
                            onClick={() => setStudioViewMode('design')}
                            className={cn(
                                "flex items-center gap-2 px-4 py-1.5 text-xs font-medium rounded-md transition-all",
                                studioViewMode === 'design'
                                    ? "bg-white/10 text-white shadow-sm ring-1 ring-white/5"
                                    : "text-muted-foreground hover:text-white hover:bg-white/5"
                            )}
                        >
                            <LayoutTemplate size={14} />
                            <span>Design</span>
                        </button>
                        <button
                            onClick={() => setStudioViewMode('split')}
                            className={cn(
                                "flex items-center gap-2 px-4 py-1.5 text-xs font-medium rounded-md transition-all",
                                studioViewMode === 'split'
                                    ? "bg-white/10 text-white shadow-sm ring-1 ring-white/5"
                                    : "text-muted-foreground hover:text-white hover:bg-white/5"
                            )}
                        >
                            <Split size={14} />
                            <span>Split</span>
                        </button>
                        <button
                            onClick={() => setStudioViewMode('chat')}
                            className={cn(
                                "flex items-center gap-2 px-4 py-1.5 text-xs font-medium rounded-md transition-all",
                                studioViewMode === 'chat'
                                    ? "bg-white/10 text-white shadow-sm ring-1 ring-white/5"
                                    : "text-muted-foreground hover:text-white hover:bg-white/5"
                            )}
                        >
                            <MessageSquare size={14} />
                            <span>Chat</span>
                        </button>
                    </div>
                </div>
            )}

            {/* Right: Actions */}
            <div className="flex items-center gap-3 flex-shrink-0">
                {isStudioEnabled && (
                    <Button
                        variant="default"
                        size="sm"
                        onClick={handleDeploy}
                        disabled={!studioIsValidYaml || studioIsDeploying}
                        className={cn(
                            "h-8 gap-2 font-medium transition-all text-xs px-4 shadow-[0_0_10px_rgba(16,185,129,0.1)]",
                            !studioIsValidYaml
                                ? "bg-red-500/20 text-red-400 hover:bg-red-500/30 border border-red-500/20"
                                : "bg-hector-green hover:bg-hector-green/90 text-white border border-transparent"
                        )}
                    >
                        <Rocket size={14} className={cn(studioIsDeploying && "animate-pulse")} />
                        <span>{studioIsDeploying ? 'Deploying...' : 'Deploy'}</span>
                    </Button>
                )}
            </div>
        </header>
    );
}
