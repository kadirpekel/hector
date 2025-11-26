import { useStore } from '../store/useStore';
import type { Message, Widget } from '../types';
import { getBaseUrl } from './api-utils';
import { handleError } from './error-handler';

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


    public async stream(url: string, requestBody: unknown) {
        const { updateMessage, setIsGenerating } = useStore.getState();
        
        // Update base URL if needed (handle relative URLs)
        const baseUrl = getBaseUrl();
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
                        } catch (parseError) {
                            // Silently skip malformed SSE data chunks
                            // Log only in development
                            if (import.meta.env.DEV) {
                                console.error('Error parsing SSE data:', parseError);
                            }
                        }
                    }
                }
            }
            
            // Stream completed successfully
            setIsGenerating(false);
        } catch (error: unknown) {
            setIsGenerating(false);
            if (error instanceof Error && error.name === 'AbortError') {
                // Stream was cancelled by user - update message state
                updateMessage(this.sessionId, this.messageId, { cancelled: true });
            } else {
                // Real error occurred - use error handler (will display via ErrorDisplay)
                handleError(error, 'Stream error');
            }
        }
    }

    private handleData(data: unknown) {
        const { setSessionTaskId, sessions } = useStore.getState();
        const session = sessions[this.sessionId];
        if (!session) return;

        const message = session.messages.find(m => m.id === this.messageId);
        if (!message) return;

        const result = (data as { result?: unknown }).result || data;

        const resultObj = result as {
            statusUpdate?: {
                taskId?: string;
                status?: {
                    state?: string;
                    update?: unknown;
                };
            };
            message?: {
                taskId?: string;
                [key: string]: unknown;
            };
            parts?: unknown[];
            [key: string]: unknown;
        };

        // Handle Task Status Updates
        if (resultObj.statusUpdate) {
            const statusUpdate = resultObj.statusUpdate;

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
        else if (resultObj.message) {
            if (resultObj.message.taskId) {
                setSessionTaskId(this.sessionId, resultObj.message.taskId);
            }
            this.processMessageUpdate(message, resultObj.message);
        }
        // Handle Parts directly
        else if (resultObj.parts) {
            this.processMessageUpdate(message, resultObj);
        }
    }

    private processMessageUpdate(currentMessage: Message, update: unknown) {
        const { updateMessage } = useStore.getState();
        const updateObj = update as { parts?: unknown[] };
        const parts = updateObj.parts || [];

        // Process parts sequentially to maintain correct order
        // This is the proper way to handle content ordering
        const widgetMap = new Map<string, Widget>();
        const contentOrder: string[] = currentMessage.metadata?.contentOrder ? [...currentMessage.metadata.contentOrder] : [];
        
        // Initialize with existing widgets
        currentMessage.widgets.forEach((w) => {
            widgetMap.set(w.id, w);
        });
        
        // Track text that appears before widgets vs after widgets
        let accumulatedText = currentMessage.text;
        let textBeforeWidgets = currentMessage.metadata?.textBeforeWidgets as string || '';
        let textAfterWidgets = currentMessage.metadata?.textAfterWidgets as string || '';
        
        // Process parts in order - this is critical for correct ordering
        parts.forEach((part: unknown, partIndex: number) => {
            const partObj = part as {
                metadata?: {
                    event_type?: string;
                    block_type?: string;
                    tool_call_id?: string;
                    tool_name?: string;
                    thinking_type?: string;
                    block_id?: string;
                    is_error?: boolean;
                    url?: string;
                    revised_prompt?: string;
                };
                data?: {
                    data?: {
                        interaction_type?: string;
                        approval_id?: string;
                        tool_name?: string;
                        tool_input?: unknown;
                        options?: string[];
                        id?: string;
                        name?: string;
                        arguments?: unknown;
                        text?: string;
                        content?: string;
                        tool_call_id?: string;
                    };
                };
                text?: string;
            };
            
            // Check if this is a widget part
            const metadata = partObj.metadata || {};
            const isThinking = metadata.event_type === 'thinking' || metadata.block_type === 'thinking';
            const isToolCall = metadata.event_type === 'tool_call' && !('is_error' in metadata);
            const isApproval = partObj.data?.data?.interaction_type === 'tool_approval';
            const isToolResult = metadata.event_type === 'tool_call' && ('is_error' in metadata);
            
            if (isThinking || isToolCall || isApproval) {
                // Widget appears - track its order
                if (isThinking) {
                    const thinkingId = metadata.block_id || `thinking-${partIndex}`;
                    if (!widgetMap.has(thinkingId)) {
                        widgetMap.set(thinkingId, {
                            id: thinkingId,
                            type: 'thinking',
                            data: { type: metadata.thinking_type || 'default' },
                            status: 'active',
                            content: partObj.text || partObj.data?.data?.text || '',
                            isExpanded: false
                        });
                        // Add to content order if not already there
                        if (!contentOrder.includes(thinkingId)) {
                            contentOrder.push(thinkingId);
                        }
                    } else {
                        // Update existing thinking block
                        const existing = widgetMap.get(thinkingId);
                        if (existing) {
                            widgetMap.set(thinkingId, {
                                ...existing,
                                content: (existing.content || '') + (partObj.text || partObj.data?.data?.text || ''),
                                status: 'active'
                            });
                        }
                    }
                } else if (isToolCall) {
                    const toolId = metadata.tool_call_id || partObj.data?.data?.id || `tool-${partIndex}`;
                    if (!widgetMap.has(toolId)) {
                        widgetMap.set(toolId, {
                            id: toolId,
                            type: 'tool',
                            data: {
                                name: metadata.tool_name || partObj.data?.data?.name || 'unknown',
                                args: partObj.data?.data?.arguments || {}
                            },
                            status: 'working',
                            content: '',
                            isExpanded: false
                        });
                        if (!contentOrder.includes(toolId)) {
                            contentOrder.push(toolId);
                        }
                    }
                } else if (isApproval) {
                    const approvalId = partObj.data?.data?.approval_id || `approval-${partIndex}`;
                    if (!widgetMap.has(approvalId)) {
                        widgetMap.set(approvalId, {
                            id: approvalId,
                            type: 'approval',
                            data: {
                                toolName: partObj.data?.data?.tool_name,
                                toolInput: partObj.data?.data?.tool_input,
                                options: partObj.data?.data?.options
                            },
                            status: 'pending',
                            isExpanded: true
                        });
                        if (!contentOrder.includes(approvalId)) {
                            contentOrder.push(approvalId);
                        }
                    }
                }
            } else if (isToolResult) {
                // Tool result - update existing tool widget
                const toolId = metadata.tool_call_id || partObj.data?.data?.tool_call_id;
                if (toolId && widgetMap.has(toolId)) {
                    const existing = widgetMap.get(toolId);
                    if (!existing) return;
                    
                    const content = partObj.data?.data?.content || '';
                    const isDenied = metadata.is_error && 
                        (content.includes('TOOL_EXECUTION_DENIED') || 
                         content.includes('user denied'));
                    
                    widgetMap.set(toolId, {
                        ...existing,
                        status: isDenied ? 'failed' : (metadata.is_error ? 'failed' : 'success'),
                        content: content || existing.content || ''
                    });
                    
                    // If tool was denied, update any related approval widget
                    if (isDenied) {
                        widgetMap.forEach((widget, widgetId) => {
                            if (widget.type === 'approval' && 
                                widget.data?.toolName === (existing.data as { name?: string })?.name &&
                                widget.status === 'pending') {
                                widgetMap.set(widgetId, {
                                    ...widget,
                                    status: 'decided',
                                    decision: 'deny'
                                });
                            }
                        });
                    }
                    
                    // Check for image generation - insert image widget right after the tool
                    if ((existing.data as { name?: string })?.name === 'generate_image' && metadata.url) {
                        const imageId = `img-${toolId}`;
                        if (!widgetMap.has(imageId)) {
                            widgetMap.set(imageId, {
                                id: imageId,
                                type: 'image',
                                data: {
                                    url: metadata.url,
                                    revised_prompt: metadata.revised_prompt
                                },
                                status: 'success',
                                isExpanded: true
                            });
                            // Insert image right after tool in content order (clean insertion, no hacky ordering)
                            const toolIndex = contentOrder.indexOf(toolId);
                            if (toolIndex !== -1 && !contentOrder.includes(imageId)) {
                                contentOrder.splice(toolIndex + 1, 0, imageId);
                            } else if (toolIndex === -1 && !contentOrder.includes(imageId)) {
                                // Tool not in order yet, add both
                                contentOrder.push(toolId);
                                contentOrder.push(imageId);
                            }
                        }
                    }
                }
            } else if (partObj.text && !isThinking && !isToolCall && !isApproval) {
                // Regular text part - accumulate it and track its position
                const textContent = partObj.text || '';
                
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
                    
                    // Track text position: before widgets or after widgets
                    // If contentOrder has widgets, text goes after; otherwise before
                    const hasWidgetsInOrder = contentOrder.some(id => widgetMap.has(id));
                    
                    if (hasWidgetsInOrder) {
                        // Text appears after widgets
                        textAfterWidgets += textContent;
                        // Add text marker to contentOrder if not already present
                        if (!contentOrder.includes('__text_after__')) {
                            contentOrder.push('__text_after__');
                        }
                    } else {
                        // Text appears before widgets
                        textBeforeWidgets += textContent;
                        // Add text marker to contentOrder if not already present
                        if (!contentOrder.includes('__text_before__')) {
                            contentOrder.unshift('__text_before__'); // Add at beginning
                        }
                    }
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
        const hasNewToolCalls = parts.some((p: unknown) => {
            const pObj = p as { metadata?: { event_type?: string; is_error?: boolean } };
            return pObj.metadata?.event_type === 'tool_call' && !('is_error' in (pObj.metadata || {}));
        });
        if (hasNewToolCalls) {
            widgetMap.forEach((widget, id) => {
                if (widget.type === 'thinking' && widget.status === 'active') {
                    widgetMap.set(id, { ...widget, status: 'completed' });
                }
            });
        }

        // Build final widget array in contentOrder sequence
        const orderedWidgets: Widget[] = [];
        const seenWidgetIds = new Set<string>();
        
        // First, add widgets in contentOrder (maintains stream order)
        contentOrder.forEach(widgetId => {
            const widget = widgetMap.get(widgetId);
            if (widget) {
                orderedWidgets.push(widget);
                seenWidgetIds.add(widgetId);
            }
        });
        
        // Then add any widgets not in contentOrder (shouldn't happen, but safety fallback)
        widgetMap.forEach((widget, id) => {
            if (!seenWidgetIds.has(id)) {
                orderedWidgets.push(widget);
            }
        });
        
        const newWidgets = orderedWidgets;

        updateMessage(this.sessionId, this.messageId, {
            text: accumulatedText,
            widgets: newWidgets,
            metadata: {
                ...currentMessage.metadata,
                contentOrder: contentOrder.length > 0 ? contentOrder : undefined,
                textBeforeWidgets: textBeforeWidgets || undefined,
                textAfterWidgets: textAfterWidgets || undefined
            }
        });
    }

}
