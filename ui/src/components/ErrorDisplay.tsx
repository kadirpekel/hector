import React, { useEffect } from 'react';
import { AlertCircle, X } from 'lucide-react';
import { useStore } from '../store/useStore';
import { cn } from '../lib/utils';
import { TIMING } from '../lib/constants';

export const ErrorDisplay: React.FC = () => {
    const { error, setError } = useStore();

    useEffect(() => {
        if (error) {
            const timer = setTimeout(() => {
                setError(null);
            }, TIMING.ERROR_AUTO_DISMISS);

            return () => clearTimeout(timer);
        }
    }, [error, setError]);

    if (!error) return null;

    return (
        <div
            className={cn(
                "fixed top-4 right-4 z-50 max-w-md bg-red-900/90 border border-red-500/50 rounded-lg shadow-lg p-4 animate-in slide-in-from-right duration-200",
                "backdrop-blur-sm"
            )}
        >
            <div className="flex items-start gap-3">
                <AlertCircle size={20} className="text-red-400 flex-shrink-0 mt-0.5" />
                <div className="flex-1 min-w-0">
                    <h4 className="text-sm font-semibold text-red-200 mb-1">Error</h4>
                    <p className="text-sm text-red-100 break-words">{error}</p>
                </div>
                <button
                    onClick={() => setError(null)}
                    className="p-1 hover:bg-red-800/50 rounded transition-colors text-red-300 hover:text-red-100 flex-shrink-0"
                    title="Dismiss"
                >
                    <X size={16} />
                </button>
            </div>
        </div>
    );
};

