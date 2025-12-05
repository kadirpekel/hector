import React from "react";
import { MessageCircle } from "lucide-react";

interface FloatingChatButtonProps {
  onClick: () => void;
  isOpen: boolean;
  showBadge?: boolean;
  style?: React.CSSProperties;
}

export const FloatingChatButton: React.FC<FloatingChatButtonProps> = ({
  onClick,
  isOpen,
  showBadge = false,
  style = {},
}) => {
  return (
    <div className="fixed bottom-6 right-6 z-40" style={style}>
      {/* Simple animated rings with longer intervals */}
      <style>
        {`
          @keyframes ai-ring {
            0% { 
              transform: scale(1); 
              opacity: 0.5; 
            }
            100% { 
              transform: scale(1.8); 
              opacity: 0; 
            }
          }
          .ai-ring-1 {
            animation: ai-ring 4s ease-out infinite;
          }
          .ai-ring-2 {
            animation: ai-ring 4s ease-out infinite;
            animation-delay: 2s;
          }
        `}
      </style>

      <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
        {/* First ring */}
        <div className="ai-ring-1 absolute w-20 h-20 rounded-full bg-gradient-to-r from-blue-500/60 to-purple-500/60" />
        {/* Second ring - offset by 2s */}
        <div className="ai-ring-2 absolute w-20 h-20 rounded-full bg-gradient-to-r from-purple-500/60 to-blue-500/60" />
      </div>

      {/* Main button - larger */}
      <button
        onClick={onClick}
        className={`floating-chat-button relative w-20 h-20 rounded-full bg-gradient-to-br from-blue-500 via-blue-600 to-purple-600 text-white shadow-2xl hover:scale-110 active:scale-95 transition-all duration-300 flex items-center justify-center ${
          isOpen ? "scale-95 opacity-90" : ""
        }`}
        style={{
          boxShadow:
            "0 12px 40px rgba(59, 130, 246, 0.6), 0 0 20px rgba(147, 51, 234, 0.3)",
        }}
        title="Test your agent"
      >
        <MessageCircle size={32} strokeWidth={2} />

        {showBadge && (
          <div className="absolute -top-1 -right-1 w-6 h-6 bg-hector-green rounded-full flex items-center justify-center text-xs font-bold animate-bounce border-2 border-white shadow-lg">
            !
          </div>
        )}
      </button>
    </div>
  );
};
