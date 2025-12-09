import React, { useEffect } from "react";
import { CheckCircle, X } from "lucide-react";
import { useStore } from "../store/useStore";
import { cn } from "../lib/utils";

export const SuccessDisplay: React.FC = () => {
  const successMessage = useStore((state) => state.successMessage);
  const setSuccessMessage = useStore((state) => state.setSuccessMessage);

  useEffect(() => {
    if (successMessage) {
      const timer = setTimeout(() => {
        setSuccessMessage(null);
      }, 3500);
      return () => clearTimeout(timer);
    }
  }, [successMessage, setSuccessMessage]);

  if (!successMessage) return null;

  return (
    <div
      className={cn(
        "fixed top-4 right-4 z-50",
        "flex items-center gap-2 px-3 py-2 max-w-sm",
        "bg-black/80 border border-green-500/30 rounded-lg",
        "shadow-lg backdrop-blur-md",
        "animate-in slide-in-from-right-2 fade-in duration-200"
      )}
    >
      <CheckCircle size={14} className="text-green-400 flex-shrink-0" />
      <p className="text-xs text-green-50 font-medium flex-1">{successMessage}</p>
      <button
        onClick={() => setSuccessMessage(null)}
        className="p-0.5 hover:bg-white/10 rounded transition-colors text-green-400/60 hover:text-green-300 flex-shrink-0"
        title="Dismiss"
      >
        <X size={12} />
      </button>
    </div>
  );
};
