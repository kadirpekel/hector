import React, { useState, useRef, useEffect } from "react";
import { Paperclip, Send, X } from "lucide-react";
import { StreamParser } from "../../lib/stream-parser";
import { useStore } from "../../store/useStore";
import type { Attachment } from "../../types";
import { cn } from "../../lib/utils";
import { TIMING, UI } from "../../lib/constants";
import { handleError } from "../../lib/error-handler";
import { generateId, generateShortId } from "../../lib/id-generator";

export const InputArea: React.FC = () => {
  // Use selectors for better performance - only subscribe to specific state slices
  const currentSessionId = useStore((state) => state.currentSessionId);
  const addMessage = useStore((state) => state.addMessage);
  const selectedAgent = useStore((state) => state.selectedAgent);
  const isGenerating = useStore((state) => state.isGenerating);
  const setIsGenerating = useStore((state) => state.setIsGenerating);
  const setActiveStreamParser = useStore((state) => state.setActiveStreamParser);
  const cancelGeneration = useStore((state) => state.cancelGeneration);
  const supportedFileTypes = useStore((state) => state.supportedFileTypes);
  const sessions = useStore((state) => state.sessions);
  const updateSessionTitle = useStore((state) => state.updateSessionTitle);
  const [input, setInput] = useState("");
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Auto-resize textarea
  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height =
        Math.min(textareaRef.current.scrollHeight, UI.MAX_TEXTAREA_HEIGHT) +
        "px";
    }
  }, [input]);

  // Track previous generation state
  const prevIsGenerating = useRef(isGenerating);
  const prevSessionId = useRef(currentSessionId);

  // Auto-focus when session changes
  useEffect(() => {
    if (
      currentSessionId &&
      currentSessionId !== prevSessionId.current &&
      selectedAgent
    ) {
      prevSessionId.current = currentSessionId;
      const timer = setTimeout(() => {
        textareaRef.current?.focus();
      }, TIMING.AUTO_FOCUS_DELAY);
      return () => clearTimeout(timer);
    }
  }, [currentSessionId, selectedAgent]);

  // Focus when generation stops (user input now expected)
  useEffect(() => {
    if (
      prevIsGenerating.current &&
      !isGenerating &&
      selectedAgent &&
      currentSessionId
    ) {
      // Generation just completed - focus input for next user input
      const timer = setTimeout(() => {
        textareaRef.current?.focus();
      }, TIMING.POST_GENERATION_FOCUS_DELAY);
      return () => clearTimeout(timer);
    }
    prevIsGenerating.current = isGenerating;
  }, [isGenerating, selectedAgent, currentSessionId]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      const files = Array.from(e.target.files);
      const newAttachments: Attachment[] = [];

      for (const file of files) {
        // Check if file type is supported
        // Check exact match first, then check if it matches a pattern (e.g., image/*)
        const isSupported = supportedFileTypes.some((type) => {
          if (type.includes("*")) {
            // Handle wildcard patterns like "image/*"
            const baseType = type.split("/")[0];
            return file.type.startsWith(baseType + "/");
          }
          return file.type === type;
        });

        if (!isSupported) {
          continue;
        }

        const reader = new FileReader();
        const base64Promise = new Promise<string>((resolve) => {
          reader.onload = (e) => {
            const result = e.target?.result as string;
            // Remove data URI prefix
            resolve(result.split(",")[1]);
          };
          reader.readAsDataURL(file);
        });

        const previewPromise = new Promise<string>((resolve) => {
          const r = new FileReader();
          r.onload = (e) => resolve(e.target?.result as string);
          r.readAsDataURL(file);
        });

        const [base64, preview] = await Promise.all([
          base64Promise,
          previewPromise,
        ]);

        newAttachments.push({
          id: generateId(),
          file,
          preview,
          base64,
          mediaType: file.type,
        });
      }

      setAttachments([...attachments, ...newAttachments]);
      // Reset input
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  };

  const removeAttachment = (id: string) => {
    setAttachments(attachments.filter((a) => a.id !== id));
  };

  const handleSend = async () => {
    if (
      (!input.trim() && attachments.length === 0) ||
      isGenerating ||
      !currentSessionId
    )
      return;

    const messageText = input.trim();
    const messageAttachments = [...attachments];

    // Clear input immediately
    setInput("");
    setAttachments([]);
    if (textareaRef.current) textareaRef.current.style.height = "auto";

    // Add user message to UI
    const userMessageId = generateId();
    addMessage(currentSessionId, {
      id: userMessageId,
      role: "user",
      text: messageText,
      metadata: {
        images: messageAttachments,
      },
      widgets: [],
      time: new Date().toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      }),
    });

    // Update session title from first user message
    const session = sessions[currentSessionId];
    if (session && session.title === "New conversation" && messageText) {
      const title =
        messageText.length > UI.MAX_TITLE_LENGTH
          ? messageText.substring(0, UI.MAX_TITLE_LENGTH) + "..."
          : messageText;
      updateSessionTitle(currentSessionId, title);
    }

    // Create agent message placeholder
    const agentMessageId = generateId();
    addMessage(currentSessionId, {
      id: agentMessageId,
      role: "agent",
      text: "",
      metadata: {},
      widgets: [],
      time: new Date().toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      }),
    });

    setIsGenerating(true);

    try {
      const parser = new StreamParser(currentSessionId, agentMessageId);
      setActiveStreamParser(parser);

      // Prepare parts - A2A spec requires "kind" field
      const parts: Array<{
        kind: string;
        text?: string;
        file?: { bytes: string; mimeType: string; name: string };
      }> = [];
      if (messageText) parts.push({ kind: "text", text: messageText });

      for (const att of messageAttachments) {
        parts.push({
          kind: "file",
          file: {
            bytes: att.base64,
            mimeType: att.mediaType,
            name: att.file.name,
          },
        });
      }

      // A2A spec: params.message with contextId for multi-turn
      // Note: taskId is server-generated, NOT sent by client
      const requestBody = {
        jsonrpc: "2.0",
        method: "message/stream",
        params: {
          message: {
            contextId: sessions[currentSessionId].contextId,
            role: "user",
            parts: parts,
          },
        },
        id: generateShortId(),
      };

      if (!selectedAgent) throw new Error("No agent selected");
      // A2A spec: POST to agent's URL - streaming is determined by method name
      await parser.stream(selectedAgent.url, requestBody);
    } catch (error: unknown) {
      if (error instanceof Error && error.name !== "AbortError") {
        handleError(error, "Failed to send message");
        setIsGenerating(false);
      }
    } finally {
      // Note: Don't set isGenerating(false) here - StreamParser.stream() handles it
      // This prevents prematurely showing the send button during HITL approval flow
      setActiveStreamParser(null);
    }
  };

  return (
    <div className="relative bg-black/60 backdrop-blur-xl border border-white/10 rounded-xl shadow-2xl overflow-hidden transition-all focus-within:ring-1 focus-within:ring-hector-green/50 focus-within:border-hector-green/50">
      {/* Attachments Preview */}
      {attachments.length > 0 && (
        <div className="flex gap-2 p-3 overflow-x-auto custom-scrollbar border-b border-white/5 bg-white/5">
          {attachments.map((att) => (
            <div key={att.id} className="relative group flex-shrink-0">
              <img
                src={att.preview}
                alt="attachment"
                className="h-16 w-16 object-cover rounded-lg border border-white/10"
              />
              <button
                onClick={() => removeAttachment(att.id)}
                className="absolute -top-1 -right-1 bg-red-500 text-white rounded-full p-0.5 opacity-0 group-hover:opacity-100 transition-opacity shadow-lg"
              >
                <X size={12} />
              </button>
            </div>
          ))}
        </div>
      )}

      <div className="flex items-end gap-2 p-3">
        {/* File Input */}
        <input
          type="file"
          ref={fileInputRef}
          onChange={handleFileSelect}
          accept={supportedFileTypes.join(",")}
          multiple
          className="hidden"
        />
        <button
          onClick={() => fileInputRef.current?.click()}
          className="p-2 text-gray-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors flex-shrink-0"
          title="Attach image"
        >
          <Paperclip size={20} />
        </button>

        {/* Text Input */}
        <textarea
          ref={textareaRef}
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={
            selectedAgent
              ? `Message ${selectedAgent.name}...`
              : "Select an agent to start..."
          }
          disabled={!selectedAgent}
          rows={1}
          className="flex-1 bg-transparent border-none focus:ring-0 focus:outline-none resize-none py-2 px-1 text-sm text-gray-100 placeholder-gray-500 max-h-[200px] custom-scrollbar"
        />

        {/* Send/Cancel Button */}
        <button
          onClick={isGenerating ? cancelGeneration : handleSend}
          disabled={
            (!input.trim() && attachments.length === 0 && !isGenerating) ||
            !selectedAgent
          }
          className={cn(
            "p-2 rounded-lg transition-all flex-shrink-0 flex items-center justify-center",
            isGenerating
              ? "bg-red-600 text-white hover:bg-red-700 shadow-lg shadow-red-600/20"
              : (input.trim() || attachments.length > 0) && selectedAgent
                ? "bg-hector-green text-white hover:bg-[#0d9668] shadow-lg shadow-hector-green/20"
                : "bg-white/5 text-gray-500 cursor-not-allowed",
          )}
          title={isGenerating ? "Cancel generation" : "Send message"}
        >
          {isGenerating ? <X size={20} /> : <Send size={20} />}
        </button>
      </div>
    </div>
  );
};
