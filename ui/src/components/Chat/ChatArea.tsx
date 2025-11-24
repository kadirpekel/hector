import React, { useEffect, useRef } from 'react';
import { MessageList } from './MessageList';
import { InputArea } from './InputArea';
import { useStore } from '../../store/useStore';

export const ChatArea: React.FC = () => {
    const { currentSessionId, sessions } = useStore();
    const session = currentSessionId ? sessions[currentSessionId] : null;
    const messagesEndRef = useRef<HTMLDivElement>(null);

    const scrollToBottom = () => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    };

    useEffect(() => {
        scrollToBottom();
    }, [session?.messages.length, session?.messages[session?.messages.length - 1]?.text]);

    if (!session) {
        return (
            <div className="flex-1 flex items-center justify-center text-gray-500">
                Select or create a chat to begin
            </div>
        );
    }

    return (
        <div className="flex flex-col h-full w-full relative">
            <div className="flex-1 overflow-y-auto custom-scrollbar p-4 md:p-6 pb-32">
                <MessageList messages={session.messages} />
                <div ref={messagesEndRef} />
            </div>

            <div className="sticky bottom-0 left-0 right-0 p-4 bg-gradient-to-t from-black via-black/90 to-transparent pt-10 z-10">
                <div className="max-w-[760px] mx-auto w-full">
                    <InputArea />
                </div>
            </div>
        </div>
    );
};
