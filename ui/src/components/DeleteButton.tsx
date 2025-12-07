import React, { useState, useRef, useEffect } from "react";
import { Trash2 } from "lucide-react";
import { cn } from "../lib/utils";

interface DeleteButtonProps {
  onDelete: () => void;
  sessionTitle: string;
}

export const DeleteButton: React.FC<DeleteButtonProps> = ({
  onDelete,
  sessionTitle,
}) => {
  const [showTooltip, setShowTooltip] = useState(false);
  const timeoutRef = useRef<number | null>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (showTooltip) {
      // Second click - confirm delete
      onDelete();
      setShowTooltip(false);
    } else {
      // First click - show tooltip
      setShowTooltip(true);
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      timeoutRef.current = setTimeout(() => {
        setShowTooltip(false);
      }, 3000); // Hide after 3 seconds
    }
  };

  const handleMouseLeave = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    setTimeout(() => {
      setShowTooltip(false);
    }, 200);
  };

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        onClick={handleClick}
        onMouseLeave={handleMouseLeave}
        className={cn(
          "opacity-0 group-hover:opacity-100 p-1 transition-opacity",
          showTooltip ? "text-red-400" : "hover:text-red-400",
        )}
        title={showTooltip ? "Click again to confirm" : "Delete conversation"}
      >
        <Trash2 size={14} />
      </button>

      {showTooltip && (
        <div className="absolute right-0 top-full mt-1 z-50 bg-gray-900 border border-white/20 rounded-lg shadow-lg p-2 min-w-[200px]">
          <div className="text-xs text-gray-300 mb-1">
            Delete "{sessionTitle}"?
          </div>
          <div className="text-xs text-gray-500">Click again to confirm</div>
        </div>
      )}
    </div>
  );
};
