// Content Blocks Display System for Hector Web UI
// This provides a unified design system for all content block types

// Content block type constants (matching protobuf enum)
const ContentBlockType = {
    TEXT: 1,
    THINKING: 2,
    REASONING: 3,
    TOOL_CALL: 4,
    TOOL_RESULT: 5,
    REFLECTION: 6,
    PLANNING: 7,
    PROGRESS: 8,
    DEBUG: 9,
    ANALYSIS: 10
};

// Design system configuration for each block type
const BlockStyles = {
    [ContentBlockType.THINKING]: {
        icon: '💭',
        color: '#f39c12',  // Yellow
        borderColor: 'rgba(243, 156, 18, 0.3)',
        bgColor: 'rgba(243, 156, 18, 0.08)',
        label: 'Thinking'
    },
    [ContentBlockType.REASONING]: {
        icon: '💭',
        color: '#f39c12',  // Yellow
        borderColor: 'rgba(243, 156, 18, 0.3)',
        bgColor: 'rgba(243, 156, 18, 0.08)',
        label: 'Reasoning'
    },
    [ContentBlockType.REFLECTION]: {
        icon: '🔍',
        color: '#3498db',  // Blue
        borderColor: 'rgba(52, 152, 219, 0.3)',
        bgColor: 'rgba(52, 152, 219, 0.08)',
        label: 'Reflection'
    },
    [ContentBlockType.PLANNING]: {
        icon: '📋',
        color: '#9b59b6',  // Purple
        borderColor: 'rgba(155, 89, 182, 0.3)',
        bgColor: 'rgba(155, 89, 182, 0.08)',
        label: 'Planning'
    },
    [ContentBlockType.PROGRESS]: {
        icon: '⏳',
        color: '#1abc9c',  // Teal
        borderColor: 'rgba(26, 188, 156, 0.3)',
        bgColor: 'rgba(26, 188, 156, 0.08)',
        label: 'Progress'
    },
    [ContentBlockType.DEBUG]: {
        icon: '🐛',
        color: '#95a5a6',  // Gray
        borderColor: 'rgba(149, 165, 166, 0.3)',
        bgColor: 'rgba(149, 165, 166, 0.05)',
        label: 'Debug'
    },
    [ContentBlockType.ANALYSIS]: {
        icon: '📊',
        color: '#27ae60',  // Green
        borderColor: 'rgba(39, 174, 96, 0.3)',
        bgColor: 'rgba(39, 174, 96, 0.08)',
        label: 'Analysis'
    },
    [ContentBlockType.TOOL_CALL]: {
        icon: '🔧',
        color: '#128c7e',
        borderColor: 'rgba(18, 140, 126, 0.3)',
        bgColor: 'rgba(18, 140, 126, 0.1)',
        label: 'Tool'
    },
    [ContentBlockType.TOOL_RESULT]: {
        icon: '✓',
        color: '#25d366',
        borderColor: 'rgba(37, 211, 102, 0.3)',
        bgColor: 'rgba(37, 211, 102, 0.1)',
        label: 'Result'
    }
};

/**
 * Check if a part is a content block
 */
function isContentBlock(part) {
    return part.metadata && part.metadata.content_block === true;
}

/**
 * Extract content block from a part
 */
function extractContentBlock(part) {
    if (!isContentBlock(part)) {
        return null;
    }

    // Content block is in part.data.data
    return part.data?.data;
}

/**
 * Render a content block element
 */
function renderContentBlock(block) {
    if (!block) return null;

    const type = parseInt(block.type);
    const style = BlockStyles[type];

    if (!style) {
        console.warn('Unknown content block type:', type);
        return null;
    }

    // Create container based on block type
    if (type === ContentBlockType.TOOL_CALL) {
        return renderToolCallBlock(block, style);
    } else if (type === ContentBlockType.TOOL_RESULT) {
        return renderToolResultBlock(block, style);
    } else if (type === ContentBlockType.PROGRESS) {
        return renderProgressBlock(block, style);
    } else {
        return renderThinkingBlock(block, style);
    }
}

/**
 * Render a thinking/reasoning/reflection/planning/analysis block
 */
function renderThinkingBlock(block, style) {
    const container = document.createElement('div');
    container.className = 'content-block thinking-block';
    container.style.cssText = `
        margin: 12px 0;
        padding: 12px 16px;
        background: ${style.bgColor};
        border-left: 3px solid ${style.borderColor};
        border-radius: 6px;
        animation: slideInBlock 0.3s ease-out;
    `;

    // Get thinking content
    const thinkingBlock = block.block?.thinkingBlock || block.block?.reasoningBlock;
    if (!thinkingBlock) return null;

    const metadata = thinkingBlock.metadata || {};
    const content = thinkingBlock.thinking || thinkingBlock.reasoning || '';

    // Header
    if (metadata.title) {
        const header = document.createElement('div');
        header.style.cssText = `
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 8px;
            color: ${style.color};
            font-weight: 600;
            font-size: 13px;
        `;
        header.innerHTML = `
            <span style="font-size: 16px;">${style.icon}</span>
            <span>${metadata.title}</span>
        `;

        // Add collapsible indicator
        if (metadata.collapsible) {
            header.style.cursor = 'pointer';
            const indicator = document.createElement('span');
            indicator.textContent = '▼';
            indicator.style.cssText = 'font-size: 10px; margin-left: auto; transition: transform 0.2s;';
            header.appendChild(indicator);

            const contentDiv = createContentDiv(content, style);
            contentDiv.style.marginTop = '8px';

            header.onclick = () => {
                const isCollapsed = contentDiv.style.display === 'none';
                contentDiv.style.display = isCollapsed ? 'block' : 'none';
                indicator.style.transform = isCollapsed ? 'rotate(0deg)' : 'rotate(-90deg)';
            };

            container.appendChild(header);
            container.appendChild(contentDiv);
        } else {
            container.appendChild(header);
            container.appendChild(createContentDiv(content, style));
        }
    } else {
        // No header, just content
        container.appendChild(createContentDiv(content, style));
    }

    return container;
}

/**
 * Create content div with markdown rendering
 */
function createContentDiv(content, style) {
    const contentDiv = document.createElement('div');
    contentDiv.style.cssText = `
        color: ${style.color};
        font-size: 13px;
        line-height: 1.6;
        opacity: 0.9;
    `;

    // Render markdown
    contentDiv.innerHTML = marked.parse(content);

    // Highlight code blocks
    contentDiv.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
    });

    return contentDiv;
}

/**
 * Render a tool call block (inline badge style)
 */
function renderToolCallBlock(block, style) {
    const toolCallBlock = block.block?.toolCallBlock;
    if (!toolCallBlock) return null;

    const container = document.createElement('span');
    container.className = 'content-block tool-call-block';
    container.setAttribute('data-tool-id', toolCallBlock.id);

    const badge = document.createElement('span');
    badge.className = 'tool-call-badge';
    badge.style.cssText = `
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 2px 8px;
        background: ${style.bgColor};
        border: 1px solid ${style.borderColor};
        border-radius: 12px;
        font-size: 11px;
        color: ${style.color};
        font-weight: 500;
        cursor: pointer;
        transition: all 0.3s ease;
        margin: 0 4px;
        vertical-align: middle;
    `;

    badge.innerHTML = `
        <span class="tool-icon" style="font-size: 13px;">${style.icon}</span>
        <span class="tool-name" style="font-family: 'Monaco', 'Courier New', monospace;">${toolCallBlock.name}</span>
        <span class="tool-status working" style="animation: toolWorking 1.5s ease-in-out infinite;">⏳</span>
    `;

    container.appendChild(badge);
    return container;
}

/**
 * Render a tool result block (inline status)
 */
function renderToolResultBlock(block, style) {
    const toolResultBlock = block.block?.toolResultBlock;
    if (!toolResultBlock) return null;

    const container = document.createElement('span');
    container.className = 'content-block tool-result-block';

    const icon = toolResultBlock.isError ? '✗' : '✓';
    const color = toolResultBlock.isError ? '#ff4444' : '#25d366';

    container.innerHTML = `<span style="color: ${color}; font-size: 14px;">${icon}</span>`;
    return container;
}

/**
 * Render a progress block (ephemeral style)
 */
function renderProgressBlock(block, style) {
    const thinkingBlock = block.block?.thinkingBlock;
    if (!thinkingBlock) return null;

    const container = document.createElement('div');
    container.className = 'content-block progress-block';
    container.style.cssText = `
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 10px;
        background: ${style.bgColor};
        border: 1px solid ${style.borderColor};
        border-radius: 16px;
        font-size: 11px;
        color: ${style.color};
        margin: 4px 0;
        animation: fadeIn 0.3s ease-out;
    `;

    container.innerHTML = `
        <span style="font-size: 14px;">${style.icon}</span>
        <span>${thinkingBlock.thinking}</span>
    `;

    return container;
}

/**
 * Update tool call status
 */
function updateToolCallStatus(toolCallId, success) {
    const elements = document.querySelectorAll(`[data-tool-id="${toolCallId}"]`);
    elements.forEach(el => {
        const status = el.querySelector('.tool-status');
        if (status) {
            status.classList.remove('working');
            if (success) {
                status.classList.add('success');
                status.textContent = '✓';
                status.style.color = '#25d366';
            } else {
                status.classList.add('failed');
                status.textContent = '✗';
                status.style.color = '#ff4444';
            }
        }
    });
}

// Export functions for use in main script
if (typeof window !== 'undefined') {
    window.ContentBlocks = {
        isContentBlock,
        extractContentBlock,
        renderContentBlock,
        updateToolCallStatus,
        ContentBlockType
    };
}
