import React from 'react';
import { Menu } from 'lucide-react';
import { Sidebar } from './Sidebar';
import { ConfigPanel } from './ConfigPanel';
import { ErrorDisplay } from './ErrorDisplay';
import { useStore } from '../store/useStore';
import { cn } from '../lib/utils';

interface LayoutProps {
    children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
    // Use selectors for better performance - only subscribe to specific state slices
    const sidebarVisible = useStore((state) => state.sidebarVisible);
    const setSidebarVisible = useStore((state) => state.setSidebarVisible);
    const minimalMode = useStore((state) => state.minimalMode);

    return (
        <div className="flex h-screen w-full bg-gradient-to-br from-hector-darker to-black text-gray-100 overflow-hidden font-sans">
            {/* Error Display */}
            <ErrorDisplay />

            {/* Mobile Sidebar Overlay */}
            <div
                className={cn(
                    "fixed inset-0 bg-black/80 z-40 md:hidden transition-opacity duration-300",
                    sidebarVisible && !minimalMode ? "opacity-100 pointer-events-auto" : "opacity-0 pointer-events-none"
                )}
                onClick={() => setSidebarVisible(false)}
            />

            {/* Sidebar Container - Desktop: in flex layout, Mobile: fixed overlay */}
            {sidebarVisible && !minimalMode ? (
                <div className="relative z-50 h-full flex-shrink-0 transition-all duration-300">
                    <Sidebar />
                </div>
            ) : (
                <div className="fixed -translate-x-full w-64 z-50 h-full md:hidden transition-transform duration-300">
                    <Sidebar />
                </div>
            )}

            {/* Main Content */}
            <div className="flex-1 flex flex-col h-full min-w-0 relative overflow-hidden">
                {/* Config Panel */}
                <ConfigPanel />
                {/* Mobile Header / Sidebar Toggle */}
                <div className="md:hidden flex items-center p-4 border-b border-white/10 bg-black/20 backdrop-blur-sm z-30 flex-shrink-0">
                    <button
                        onClick={() => setSidebarVisible(true)}
                        className="p-2 -ml-2 hover:bg-white/10 rounded-lg transition-colors"
                    >
                        <Menu size={24} />
                    </button>
                    <span className="ml-3 font-semibold text-hector-green tracking-wide">HECTOR</span>
                </div>

                {/* Desktop Sidebar Toggle (Floating) */}
                {(!sidebarVisible || minimalMode) && (
                    <button
                        onClick={() => {
                            setSidebarVisible(true);
                            useStore.getState().setMinimalMode(false);
                        }}
                        className="hidden md:flex absolute top-4 left-4 z-30 p-2 bg-black/40 hover:bg-white/10 rounded-lg text-gray-400 hover:text-white transition-colors border border-white/5 backdrop-blur-sm"
                        title="Show Sidebar"
                    >
                        <Menu size={20} />
                    </button>
                )}

                {/* Content Area */}
                <main className="flex-1 relative overflow-hidden">
                    {children}
                </main>
            </div>
        </div>
    );
};
