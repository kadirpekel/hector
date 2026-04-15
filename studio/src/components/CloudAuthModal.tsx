/**
 * CloudAuthModal
 *
 * Fly token-based sign-in for Hector Cloud.
 * Users create a Fly.io API token and paste it here.
 * The cloud validates it and returns a short-lived cloud JWT.
 */

import { useState } from 'react';
import { createPortal } from 'react-dom';
import { X, Loader2, CheckCircle, AlertCircle } from 'lucide-react';
import { useCloudAuthStore } from '../store/cloudAuthStore';
import { useCloudStore } from '../store/cloudStore';

interface CloudAuthModalProps {
    isOpen: boolean;
    onClose: () => void;
    onAuthenticated: () => void;
    onSkip?: () => void;
    title?: string;
    description?: string;
}

export function CloudAuthModal({ isOpen, onClose, onAuthenticated, onSkip, title, description }: CloudAuthModalProps) {
    const { loginWithToken, error, clearError, isAuthenticated, logout } = useCloudAuthStore();
    const cloudDisconnect = useCloudStore((s) => s.disconnect);
    const [flyToken, setFlyToken] = useState('');
    const [appName, setAppName] = useState('');
    const [loading, setLoading] = useState(false);

    const handleSubmit = async () => {
        if (!flyToken.trim() || !appName.trim()) return;
        setLoading(true);
        await loginWithToken(flyToken.trim(), appName.trim());
        setLoading(false);
        if (useCloudAuthStore.getState().isAuthenticated) {
            setTimeout(() => onAuthenticated(), 600);
        }
    };

    if (!isOpen) return null;

    return createPortal(
        <>
            {/* Backdrop */}
            <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50" onClick={onClose} />

            {/* Modal */}
            <div className="fixed inset-0 z-50 flex items-center justify-center p-4" onClick={onClose}>
                <div className="bg-gray-900 border border-white/10 rounded-2xl shadow-2xl overflow-hidden w-full max-w-md" onClick={(e) => e.stopPropagation()}>
                    {/* Header */}
                    <div className="flex items-center justify-between p-4 border-b border-white/10">
                        <div className="flex items-center gap-3">
                            <div className="p-2 bg-violet-500/20 rounded-lg">
                                <svg className="text-violet-400 w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
                                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 14H9V8h2v8zm4 0h-2V8h2v8z"/>
                                </svg>
                            </div>
                            <div>
                                <h2 className="text-lg font-semibold text-white">{title || 'Connect to Cloud'}</h2>
                                <p className="text-xs text-gray-400">{description || 'Sign in with your Fly.io API token'}</p>
                            </div>
                        </div>
                        <button onClick={onClose} className="p-1 text-gray-400 hover:text-white hover:bg-white/10 rounded transition-colors">
                            <X size={20} />
                        </button>
                    </div>

                    {/* Content */}
                    <div className="p-6">
                        {isAuthenticated === true ? (
                            <div className="text-center py-8">
                                <CheckCircle size={48} className="mx-auto text-green-500 mb-4" />
                                <h3 className="text-lg font-medium text-white mb-2">Authenticated!</h3>
                                <p className="text-sm text-gray-400">Connecting to Hector Cloud...</p>
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <p className="text-sm text-gray-400">
                                    Hector Cloud deploys Hector into <span className="text-white font-medium">your own</span> Fly.io app.
                                    Create the app and a scoped deploy token - we never need your personal account token.
                                </p>

                                <div className="rounded-lg bg-black/40 border border-white/10 p-3 space-y-1.5">
                                    <p className="text-xs text-gray-400 font-medium">Run these two commands once:</p>
                                    <code className="block text-xs text-violet-300 font-mono">flyctl apps create my-hector --org personal</code>
                                    <code className="block text-xs text-violet-300 font-mono">flyctl tokens create deploy -a my-hector</code>
                                    <p className="text-xs text-gray-500 pt-1">Use any app name you like - paste it and the token below.</p>
                                </div>

                                {error && (
                                    <div className="flex flex-col gap-2 text-sm text-red-400 bg-red-500/10 p-3 rounded-lg">
                                        <div className="flex items-center gap-2">
                                            <AlertCircle size={16} className="flex-shrink-0" />
                                            <span className="flex-1">{error}</span>
                                            <button onClick={clearError} className="text-red-400 hover:text-red-300">
                                                <X size={14} />
                                            </button>
                                        </div>
                                        {isAuthenticated !== null && (
                                            <button
                                                onClick={() => {
                                                    cloudDisconnect();
                                                    logout();
                                                    clearError();
                                                    setFlyToken('');
                                                    setAppName('');
                                                }}
                                                className="text-xs text-gray-400 hover:text-white underline underline-offset-2 text-left"
                                            >
                                                Clear stored credentials and try again
                                            </button>
                                        )}
                                    </div>
                                )}

                                <input
                                    type="text"
                                    placeholder="App name (e.g. my-hector)"
                                    value={appName}
                                    onChange={(e) => { setAppName(e.target.value); clearError(); }}
                                    className="w-full px-3 py-2.5 rounded-lg text-sm bg-black/40 border border-white/10 text-white placeholder-gray-600 focus:outline-none focus:border-violet-500/50"
                                    autoFocus
                                />

                                <input
                                    type="password"
                                    placeholder="Deploy token (FlyV1 fm2_...)"
                                    value={flyToken}
                                    onChange={(e) => { setFlyToken(e.target.value); clearError(); }}
                                    onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
                                    className="w-full px-3 py-2.5 rounded-lg text-sm bg-black/40 border border-white/10 text-white placeholder-gray-600 focus:outline-none focus:border-violet-500/50"
                                />

                                <button
                                    onClick={handleSubmit}
                                    disabled={loading || !flyToken.trim() || !appName.trim()}
                                    className="w-full py-3 rounded-lg font-medium transition-colors flex items-center justify-center gap-2 bg-violet-600 hover:bg-violet-500 disabled:opacity-50 disabled:cursor-not-allowed text-white"
                                >
                                    {loading ? <Loader2 size={16} className="animate-spin" /> : null}
                                    {loading ? 'Verifying...' : 'Connect'}
                                </button>

                                {onSkip && (
                                    <button
                                        onClick={() => { onSkip(); onClose(); }}
                                        className="w-full text-center text-xs text-gray-500 hover:text-gray-400 transition-colors py-2"
                                    >
                                        Skip for now (self-hosted only)
                                    </button>
                                )}
                            </div>
                        )}
                    </div>
                </div>
            </div>
        </>,
        document.body
    );
}

