import { useStore } from '../store/useStore';
import type { Message } from '../types';

export class StreamParser {
    private sessionId: string;
    private messageId: string;
    private abortController: AbortController;

    constructor(sessionId: string, messageId: string) {
        this.sessionId = sessionId;
        this.messageId = messageId;
        this.abortController = new AbortController();
    }

    public abort() {
        this.abortController.abort();
    }

    private getBaseUrl(): string {
        const state = useStore.getState();
        return state.endpointUrl || window.location.origin;
    }

    public async stream(url: string, requestBody: any) {
        const { updateMessage, addMessage, setError, setIsGenerating } = useStore.getState();
        
        // Update base URL if needed (handle relative URLs)
        const baseUrl = this.getBaseUrl();
        const fullUrl = url.startsWith('http') ? url : `${baseUrl}${url}`;

        try {
            const response = await fetch(fullUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(requestBody),
                signal: this.abortController.signal,
            });

            if (!response.ok) {
                const errorText = await response.text().catch(() => 'Unknown error');
                throw new Error(`HTTP ${response.status}: ${errorText.substring(0, 100)}`);
            }
            if (!response.body) throw new Error('No response body');

            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';

            while (true) {
                const { done, value } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        try {
                            const data = JSON.parse(line.substring(6));
                            this.handleData(data);
                        } catch (e) {
                            console.error('Error parsing SSE data:', e);
                        }
                    }
                }
            }
        } catch (error: any) {
            setIsGenerating(false);
            if (error.name === 'AbortError') {
                console.log('Stream aborted');
                updateMessage(this.sessionId, this.messageId, { cancelled: true });
            } else {
                console.error('Stream error:', error);
                const errorMessage = error.message || 'An unknown error occurred';
                setError(errorMessage);
                addMessage(this.sessionId, {
                    id: Math.random().toString(36).substring(7),
                    role: 'system',
                    text: `Error: ${errorMessage}`,
                    metadata: {},
                    toolCalls: [],
                    thinkingBlocks: [],
                    widgets: [],
                    time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
                });
            }
        }
    }

    private handleData(data: any) {
        const { setSessionTaskId, sessions } = useStore.getState();
        const session = sessions[this.sessionId];
        if (!session) return;

        const message = session.messages.find(m => m.id === this.messageId);
        if (!message) return;

        const result = data.result || data;

        // Handle Task Status Updates
        if (result.statusUpdate) {
            const statusUpdate = result.statusUpdate;

            // Update Task ID if present
            if (statusUpdate.taskId) {
                setSessionTaskId(this.sessionId, statusUpdate.taskId);
            }

            // Check for INPUT_REQUIRED state
            if (statusUpdate.status?.state === 'TASK_STATE_INPUT_REQUIRED') {
                // We might want to trigger some UI state here, but the approval widget 
                // usually comes in the message update, so we just ensure taskId is set for resuming
            }

            // Check if statusUpdate contains a message update
            if (statusUpdate.status?.update) {
                this.processMessageUpdate(message, statusUpdate.status.update);
            }
        }
        // Handle Direct Message Updates
        else if (result.message) {
            if (result.message.taskId) {
                setSessionTaskId(this.sessionId, result.message.taskId);
            }
            this.processMessageUpdate(message, result.message);
        }
        // Handle Parts directly
        else if (result.parts) {
            this.processMessageUpdate(message, result);
        }
    }

    private processMessageUpdate(currentMessage: Message, update: any) {
        const { updateMessage } = useStore.getState();
        const parts = update.parts || [];

        // Process parts sequentially to maintain correct order
        // This is the proper way to handle content ordering
        const widgetMap = new Map<string, any>();
        const contentOrder: string[] = currentMessage.metadata?.contentOrder ? [...currentMessage.metadata.contentOrder] : [];
        
        // Initialize with existing widgets
        let maxOrder = 0;
        currentMessage.widgets.forEach((w, idx) => {
            widgetMap.set(w.id, { ...w, _order: idx });
            maxOrder = Math.max(maxOrder, idx);
        });
        
        // Track text segments - we'll accumulate text but also track where widgets appear
        let accumulatedText = currentMessage.text;
        
        // Process parts in order - this is critical for correct ordering
        parts.forEach((part: any, partIndex: number) => {
            // Check if this is a widget part
            const isThinking = part.metadata?.event_type === 'thinking' || part.metadata?.block_type === 'thinking';
            const isToolCall = part.metadata?.event_type === 'tool_call' && !part.metadata?.hasOwnProperty('is_error');
            const isApproval = part.data?.data?.interaction_type === 'tool_approval';
            const isToolResult = part.metadata?.event_type === 'tool_call' && part.metadata?.hasOwnProperty('is_error');
            
            if (isThinking || isToolCall || isApproval) {
                // Widget appears - track its order
                if (isThinking) {
                    const thinkingId = part.metadata?.block_id || `thinking-${partIndex}`;
                    if (!widgetMap.has(thinkingId)) {
                        widgetMap.set(thinkingId, {
                            id: thinkingId,
                            type: 'thinking',
                            data: { type: part.metadata?.thinking_type || 'default' },
                            status: 'active',
                            content: part.text || part.data?.data?.text || '',
                            isExpanded: false,
                            _order: maxOrder++
                        });
                        // Add to content order if not already there
                        if (!contentOrder.includes(thinkingId)) {
                            contentOrder.push(thinkingId);
                        }
                    } else {
                        // Update existing thinking block
                        const existing = widgetMap.get(thinkingId);
                        widgetMap.set(thinkingId, {
                            ...existing,
                            content: (existing.content || '') + (part.text || part.data?.data?.text || ''),
                            status: 'active'
                        });
                    }
                } else if (isToolCall) {
                    const toolId = part.metadata?.tool_call_id || part.data?.data?.id || `tool-${partIndex}`;
                    if (!widgetMap.has(toolId)) {
                        widgetMap.set(toolId, {
                            id: toolId,
                            type: 'tool',
                            data: {
                                name: part.metadata?.tool_name || part.data?.data?.name || 'unknown',
                                args: part.data?.data?.arguments || {}
                            },
                            status: 'working',
                            content: '',
                            isExpanded: false,
                            _order: maxOrder++
                        });
                        if (!contentOrder.includes(toolId)) {
                            contentOrder.push(toolId);
                        }
                    }
                } else if (isApproval) {
                    const approvalId = part.data?.data?.approval_id || `approval-${partIndex}`;
                    if (!widgetMap.has(approvalId)) {
                        widgetMap.set(approvalId, {
                            id: approvalId,
                            type: 'approval',
                            data: {
                                toolName: part.data?.data?.tool_name,
                                toolInput: part.data?.data?.tool_input,
                                options: part.data?.data?.options
                            },
                            status: 'pending',
                            isExpanded: true,
                            _order: maxOrder++
                        });
                        if (!contentOrder.includes(approvalId)) {
                            contentOrder.push(approvalId);
                        }
                    }
                }
            } else if (isToolResult) {
                // Tool result - update existing tool widget
                const toolId = part.metadata?.tool_call_id || part.data?.data?.tool_call_id;
                if (toolId && widgetMap.has(toolId)) {
                    const existing = widgetMap.get(toolId);
                    const isDenied = part.metadata?.is_error && 
                        (part.data?.data?.content?.includes('TOOL_EXECUTION_DENIED') || 
                         part.data?.data?.content?.includes('user denied'));
                    
                    widgetMap.set(toolId, {
                        ...existing,
                        status: isDenied ? 'failed' : (part.metadata?.is_error ? 'failed' : 'success'),
                        content: part.data?.data?.content || existing.content || ''
                    });
                    
                    // If tool was denied, update any related approval widget
                    if (isDenied) {
                        widgetMap.forEach((widget, widgetId) => {
                            if (widget.type === 'approval' && 
                                widget.data?.toolName === existing.data?.name &&
                                widget.status === 'pending') {
                                widgetMap.set(widgetId, {
                                    ...widget,
                                    status: 'decided',
                                    decision: 'deny'
                                });
                            }
                        });
                    }
                    
                    // Check for image generation
                    if (existing.data.name === 'generate_image' && part.metadata?.url) {
                        const imageId = `img-${toolId}`;
                        if (!widgetMap.has(imageId)) {
                            widgetMap.set(imageId, {
                                id: imageId,
                                type: 'image',
                                data: {
                                    url: part.metadata.url,
                                    revised_prompt: part.metadata.revised_prompt
                                },
                                status: 'success',
                                isExpanded: true,
                                _order: existing._order + 0.5
                            });
                            // Insert image right after tool in content order
                            const toolIndex = contentOrder.indexOf(toolId);
                            if (toolIndex !== -1 && !contentOrder.includes(imageId)) {
                                contentOrder.splice(toolIndex + 1, 0, imageId);
                            }
                        }
                    }
                }
            } else if (part.text && !isThinking && !isToolCall && !isApproval) {
                // Regular text part - accumulate it
                const textContent = part.text || '';
                
                // Filter out verbose backend messages that are redundant with widgets
                // These patterns match common approval-related backend messages
                const approvalPatterns = [
                    /^✅\s*Approved:/,
                    /^🚫\s*Denied:/,
                    /SUCCESS:\s*Approved:/i,
                    /DENIED:\s*Denied:/i,
                    /Command Execution Request/i,
                    /This will execute on the server/i,
                    /🔐 Tool Approval Required/i,
                    /Please respond with:\s*approve or deny/i,
                    /Tool:\s*\w+\s*Input:/i, // Matches "Tool: X Input:" pattern
                ];
                
                const shouldFilter = approvalPatterns.some(pattern => textContent.match(pattern));
                
                if (textContent && !shouldFilter) {
                    accumulatedText += textContent;
                }
            }
        });

        // Mark completed thinking blocks when text appears
        if (accumulatedText.length > currentMessage.text.length) {
            widgetMap.forEach((widget, id) => {
                if (widget.type === 'thinking' && widget.status === 'active') {
                    widgetMap.set(id, { ...widget, status: 'completed' });
                }
            });
        }
        
        // Mark thinking as completed when tool calls start
        const hasNewToolCalls = parts.some((p: any) => 
            p.metadata?.event_type === 'tool_call' && !p.metadata?.hasOwnProperty('is_error')
        );
        if (hasNewToolCalls) {
            widgetMap.forEach((widget, id) => {
                if (widget.type === 'thinking' && widget.status === 'active') {
                    widgetMap.set(id, { ...widget, status: 'completed' });
                }
            });
        }

        // Sort widgets by their order in contentOrder array, maintaining insertion order
        const orderedWidgets: any[] = [];
        const unorderedWidgets: any[] = [];
        
        // First, add widgets in contentOrder
        contentOrder.forEach(widgetId => {
            const widget = widgetMap.get(widgetId);
            if (widget) {
                const { _order, ...cleanWidget } = widget;
                orderedWidgets.push(cleanWidget);
            }
        });
        
        // Then add any widgets not in contentOrder (shouldn't happen, but safety)
        widgetMap.forEach((widget, id) => {
            if (!contentOrder.includes(id)) {
                const { _order, ...cleanWidget } = widget;
                unorderedWidgets.push(cleanWidget);
            }
        });
        
        const newWidgets = [...orderedWidgets, ...unorderedWidgets];

        updateMessage(this.sessionId, this.messageId, {
            text: accumulatedText,
            widgets: newWidgets,
            metadata: {
                ...currentMessage.metadata,
                contentOrder: contentOrder.length > 0 ? contentOrder : undefined
            }
        });
    }

}
