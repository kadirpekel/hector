import React, { useEffect } from "react";
import { AlertCircle, X } from "lucide-react";
import { useStore } from "../store/useStore";
import { cn } from "../lib/utils";

export const ErrorDisplay: React.FC = () => {
  const error = useStore((state) => state.error);
  const setError = useStore((state) => state.setError);

  useEffect(() => {
    if (error) {
      const timer = setTimeout(() => {
        setError(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [error, setError]);

  if (!error) return null;

  return (
    <div
      className={cn(
        "fixed top-4 right-4 z-50",
        "flex items-center gap-2 px-3 py-2 max-w-sm",
        "bg-black/80 border border-red-500/30 rounded-lg",
        "shadow-lg backdrop-blur-md",
        "animate-in slide-in-from-right-2 fade-in duration-200"
      )}
    >
      <AlertCircle size={14} className="text-red-400 flex-shrink-0" />
      <p className="text-xs text-red-50 font-medium flex-1 break-words">{error}</p>
      <button
        onClick={() => setError(null)}
        className="p-0.5 hover:bg-white/10 rounded transition-colors text-red-400/60 hover:text-red-300 flex-shrink-0"
        title="Dismiss"
      >
        <X size={12} />
      </button>
    </div>
  );
};
