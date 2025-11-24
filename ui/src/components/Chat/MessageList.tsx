import React from 'react';
import { Loader2 } from 'lucide-react';
import { useStore } from '../../store/useStore';
import type { Message } from '../../types';
import { MessageItem } from './MessageItem';

interface MessageListProps {
    messages: Message[];
}

export const MessageList: React.FC<MessageListProps> = ({ messages }) => {
    const { isGenerating } = useStore();
    const lastMessage = messages[messages.length - 1];
    // Show indicator if generating AND (last message is user OR (last message is agent but empty/waiting))
    const showIndicator = isGenerating && (!lastMessage || lastMessage.role === 'user' || (lastMessage.role === 'agent' && !lastMessage.text && (!lastMessage.widgets || lastMessage.widgets.length === 0)));

    return (
        <div className="flex flex-col gap-6 max-w-[760px] mx-auto w-full">
            {messages.map((message, index) => (
                <MessageItem
                    key={message.id}
                    message={message}
                    messageIndex={index}
                    isLastMessage={index === messages.length - 1}
                />
            ))}

            {showIndicator && (
                <div className="flex items-center gap-2 text-gray-500 text-sm animate-pulse px-4">
                    <Loader2 size={16} className="animate-spin" />
                    <span>Thinking...</span>
                </div>
            )}
        </div>
    );
};
