import React from "react";

export const StreamingIndicator: React.FC = () => {
  return (
    <div className="flex items-center gap-1 px-4 py-2">
      <div className="flex gap-1">
        <span
          className="w-2 h-2 rounded-full bg-hector-green animate-bounce"
          style={{ animationDelay: "0ms", animationDuration: "0.6s" }}
        />
        <span
          className="w-2 h-2 rounded-full bg-hector-green animate-bounce"
          style={{ animationDelay: "150ms", animationDuration: "0.6s" }}
        />
        <span
          className="w-2 h-2 rounded-full bg-hector-green animate-bounce"
          style={{ animationDelay: "300ms", animationDuration: "0.6s" }}
        />
      </div>
    </div>
  );
};
