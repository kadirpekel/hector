import React, { useState } from "react";
import { LogIn, AlertCircle, X, Shield } from "lucide-react";
import type { ServerConfig } from "../types";
import { useServersStore } from "../store/serversStore";

interface LoginModalProps {
    server: ServerConfig;
    isOpen: boolean;
    onClose: () => void;
    onLoginSuccess: () => void;
}

export const LoginModal: React.FC<LoginModalProps> = ({ server, isOpen, onClose, onLoginSuccess }) => {
    const [adminKey, setAdminKey] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleLogin = async () => {
        if (!adminKey.trim()) {
            setError("Please enter an admin key.");
            return;
        }

        setLoading(true);
        setError(null);

        try {
            // Verify the key works by calling /health with it
            const res = await fetch(`${server.url.replace(/\/$/, '')}/admin/apps`, {
                headers: { 'Authorization': `Bearer ${adminKey.trim()}` },
            });

            if (res.ok || res.status === 200) {
                // Key works - save it and mark server as authenticated
                useServersStore.getState().updateServerConfig(server.id, { adminKey: adminKey.trim() });
                useServersStore.getState().setServerStatus(server.id, 'authenticated');
                onLoginSuccess();
                onClose();
                setAdminKey("");
            } else if (res.status === 401 || res.status === 403) {
                setError("Invalid admin key. Please check and try again.");
            } else {
                setError(`Server returned ${res.status}. Check the server URL.`);
            }
        } catch {
            setError("Could not connect to server.");
        } finally {
            setLoading(false);
        }
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-md z-50 flex items-center justify-center p-4">
            <div className="bg-zinc-900 border border-white/10 rounded-xl max-w-md w-full shadow-2xl overflow-hidden">
                <div className="p-6 pb-0 flex justify-between items-start">
                    <div className="flex items-center gap-3">
                        <div className="p-3 bg-hector-green/10 rounded-xl text-hector-green">
                            <Shield size={24} />
                        </div>
                        <div>
                            <h3 className="text-xl font-bold">Authentication Required</h3>
                            <p className="text-sm text-gray-400">Connect to {server.name}</p>
                        </div>
                    </div>
                    <button onClick={onClose} className="p-1 hover:bg-white/10 rounded-full transition-colors">
                        <X size={20} className="text-gray-500 hover:text-white" />
                    </button>
                </div>

                <div className="p-6 space-y-4">
                    <div className="bg-white/5 rounded-lg p-4 text-sm text-gray-300 border border-white/5">
                        <p>Server <strong>{server.url}</strong> requires an admin key (auth-secret).</p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1.5">Admin Key</label>
                        <input
                            type="password"
                            value={adminKey}
                            onChange={(e) => setAdminKey(e.target.value)}
                            onKeyDown={(e) => e.key === 'Enter' && handleLogin()}
                            placeholder="Enter admin key..."
                            className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2.5 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-hector-green/50 focus:ring-1 focus:ring-hector-green/30"
                            autoFocus
                        />
                    </div>

                    {error && (
                        <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg flex items-start gap-2 text-red-400 text-sm">
                            <AlertCircle size={16} className="mt-0.5 flex-shrink-0" />
                            <p>{error}</p>
                        </div>
                    )}

                    <div className="flex justify-end gap-3 pt-2">
                        <button
                            onClick={onClose}
                            className="px-4 py-2.5 text-gray-400 hover:text-white hover:bg-white/5 rounded-lg transition-colors font-medium text-sm"
                        >
                            Cancel
                        </button>
                        <button
                            onClick={handleLogin}
                            disabled={loading || !adminKey.trim()}
                            className="px-6 py-2.5 bg-hector-green hover:bg-[#0d9668] text-white rounded-lg transition-all shadow-lg shadow-hector-green/20 font-medium text-sm flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {loading ? (
                                <>
                                    <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                                    Verifying...
                                </>
                            ) : (
                                <>
                                    <LogIn size={16} />
                                    Connect
                                </>
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};
