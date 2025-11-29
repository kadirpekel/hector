import React, { useState } from 'react';
import { CheckSquare, Square, Hourglass, XSquare } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { TodoItem } from '../../types';

interface TodoListProps {
    todos: TodoItem[];
}

const VISIBLE_ITEMS = 4; // Show 4 items initially (offset for long lists)

export const TodoList: React.FC<TodoListProps> = ({ todos }) => {
    const [showAll, setShowAll] = useState(false);
    
    if (!todos || todos.length === 0) {
        return null;
    }

    const visibleTodos = showAll ? todos : todos.slice(0, VISIBLE_ITEMS);
    const hasMore = todos.length > VISIBLE_ITEMS;

    const getStatusIcon = (status: TodoItem['status']) => {
        switch (status) {
            case 'completed':
                return <CheckSquare size={14} className="text-green-400 shrink-0" />;
            case 'in_progress':
                return <Hourglass size={14} className="text-yellow-400 shrink-0 animate-pulse" />;
            case 'canceled':
                return <XSquare size={14} className="text-red-400 shrink-0" />;
            default:
                return <Square size={14} className="text-gray-400 shrink-0" />;
        }
    };

    return (
        <div className="ml-4 mt-1 mb-2 border-l-2 border-purple-500/30 pl-3 py-1">
            <div className="text-xs font-medium text-purple-300/70 mb-1.5">Tasks</div>
            <div className="space-y-1">
                {visibleTodos.map((todo, index) => (
                    <div
                        key={todo.id || index}
                        className={cn(
                            "flex items-start gap-2 py-0.5",
                            todo.status === 'completed' && "opacity-50"
                        )}
                    >
                        <div className="mt-0.5 shrink-0 flex items-center gap-1.5">
                            <span className="text-xs text-gray-500 font-mono w-4 text-right">
                                {index + 1}.
                            </span>
                            {getStatusIcon(todo.status)}
                        </div>
                        <div className="flex-1 min-w-0">
                            <div className={cn(
                                "text-xs leading-relaxed",
                                todo.status === 'completed' ? "line-through text-gray-500" : "text-gray-200"
                            )}>
                                {todo.content}
                            </div>
                        </div>
                    </div>
                ))}
                {hasMore && !showAll && (
                    <button
                        onClick={() => setShowAll(true)}
                        className="text-xs text-purple-400/70 hover:text-purple-400 ml-6 mt-1 transition-colors"
                    >
                        +{todos.length - VISIBLE_ITEMS} more
                    </button>
                )}
                {hasMore && showAll && (
                    <button
                        onClick={() => setShowAll(false)}
                        className="text-xs text-purple-400/70 hover:text-purple-400 ml-6 mt-1 transition-colors"
                    >
                        Show less
                    </button>
                )}
            </div>
        </div>
    );
};

