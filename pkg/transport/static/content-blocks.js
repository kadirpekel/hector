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
    const type = parseInt(block.type);

    // Route to specialized renderers based on type
    switch(type) {
        case ContentBlockType.THINKING:
        case ContentBlockType.REASONING:
            return renderThinkingSpecial(block, style);
        case ContentBlockType.PLANNING:
            return renderPlanningSpecial(block, style);
        case ContentBlockType.REFLECTION:
            return renderReflectionSpecial(block, style);
        case ContentBlockType.ANALYSIS:
            return renderAnalysisSpecial(block, style);
        case ContentBlockType.DEBUG:
            return renderDebugSpecial(block, style);
        default:
            return renderGenericThinkingBlock(block, style);
    }
}

/**
 * Render thinking blocks with grayed out, dimmed styling
 */
function renderThinkingSpecial(block, style) {
    const container = document.createElement('div');
    container.className = 'content-block thinking-block-special';
    container.style.cssText = `
        margin: 8px 0;
        padding: 8px 12px;
        background: rgba(149, 165, 166, 0.05);
        border-left: 2px solid rgba(149, 165, 166, 0.2);
        border-radius: 4px;
        opacity: 0.7;
        font-size: 12px;
        color: #95a5a6;
        font-style: italic;
        animation: fadeIn 0.5s ease-out;
    `;

    const thinkingBlock = block.block?.thinkingBlock;
    if (!thinkingBlock) return null;

    const metadata = thinkingBlock.metadata || {};
    const content = thinkingBlock.thinking || '';

    // Collapsible by default for thinking
    const isCollapsible = metadata.collapsible !== false;

    if (metadata.title || isCollapsible) {
        const header = document.createElement('div');
        header.style.cssText = `
            display: flex;
            align-items: center;
            gap: 6px;
            margin-bottom: 6px;
            font-size: 11px;
            font-weight: 500;
            color: #95a5a6;
        `;

        const titleText = metadata.title || 'Thinking...';
        header.innerHTML = `
            <span style="font-size: 14px;">💭</span>
            <span>${titleText}</span>
        `;

        if (isCollapsible) {
            header.style.cursor = 'pointer';
            const indicator = document.createElement('span');
            indicator.textContent = '▼';
            indicator.style.cssText = 'font-size: 9px; margin-left: auto; transition: transform 0.2s; opacity: 0.6;';
            header.appendChild(indicator);

            const contentDiv = createContentDiv(content, {color: '#95a5a6', ...style});
            contentDiv.style.marginTop = '6px';
            contentDiv.style.display = 'none'; // Start collapsed

            header.onclick = () => {
                const isCollapsed = contentDiv.style.display === 'none';
                contentDiv.style.display = isCollapsed ? 'block' : 'none';
                indicator.style.transform = isCollapsed ? 'rotate(0deg)' : 'rotate(-90deg)';
            };

            container.appendChild(header);
            container.appendChild(contentDiv);
        } else {
            container.appendChild(header);
            container.appendChild(createContentDiv(content, {color: '#95a5a6', ...style}));
        }
    } else {
        container.appendChild(createContentDiv(content, {color: '#95a5a6', ...style}));
    }

    return container;
}

/**
 * Render planning blocks with checkbox list styling
 */
function renderPlanningSpecial(block, style) {
    const container = document.createElement('div');
    container.className = 'content-block planning-block-special';
    container.style.cssText = `
        margin: 12px 0;
        padding: 14px 18px;
        background: ${style.bgColor};
        border-left: 4px solid ${style.borderColor};
        border-radius: 8px;
        animation: slideInBlock 0.3s ease-out;
    `;

    const thinkingBlock = block.block?.thinkingBlock;
    if (!thinkingBlock) return null;

    const metadata = thinkingBlock.metadata || {};
    const content = thinkingBlock.thinking || '';

    // Header
    if (metadata.title) {
        const header = document.createElement('div');
        header.style.cssText = `
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 12px;
            color: ${style.color};
            font-weight: 700;
            font-size: 14px;
        `;
        header.innerHTML = `
            <span style="font-size: 18px;">${style.icon}</span>
            <span>${metadata.title}</span>
        `;
        container.appendChild(header);
    }

    // Parse content for checklist items
    const contentDiv = createChecklistDiv(content, style);
    container.appendChild(contentDiv);

    return container;
}

/**
 * Create a checklist div from markdown list content
 */
function createChecklistDiv(content, style) {
    const div = document.createElement('div');
    div.style.cssText = `
        display: flex;
        flex-direction: column;
        gap: 8px;
    `;

    const lines = content.split('\n');
    let inList = false;

    lines.forEach(line => {
        const trimmed = line.trim();

        // Check if line is a list item
        const listMatch = trimmed.match(/^[-*•+]\s+(.+)/) || trimmed.match(/^\d+\.\s+(.+)/);

        if (listMatch) {
            inList = true;
            const itemText = listMatch[1];

            const item = document.createElement('div');
            item.style.cssText = `
                display: flex;
                align-items: flex-start;
                gap: 10px;
                padding: 6px 10px;
                background: rgba(255, 255, 255, 0.3);
                border-radius: 4px;
                transition: background 0.2s;
            `;

            item.innerHTML = `
                <span style="font-size: 16px; color: ${style.color}; flex-shrink: 0;">☐</span>
                <span style="color: ${style.color}; font-size: 13px; line-height: 1.5;">${itemText}</span>
            `;

            div.appendChild(item);
        } else if (trimmed && !inList) {
            // Regular text between lists
            const p = document.createElement('p');
            p.style.cssText = `
                margin: 4px 0;
                color: ${style.color};
                font-size: 13px;
                line-height: 1.6;
            `;
            p.textContent = trimmed;
            div.appendChild(p);
        } else if (trimmed) {
            // Text after list started
            const p = document.createElement('p');
            p.style.cssText = `
                margin: 8px 0 4px 0;
                color: ${style.color};
                font-size: 13px;
                line-height: 1.6;
            `;
            p.textContent = trimmed;
            div.appendChild(p);
            inList = false;
        }
    });

    return div;
}

/**
 * Render reflection blocks with emphasized box styling
 */
function renderReflectionSpecial(block, style) {
    const container = document.createElement('div');
    container.className = 'content-block reflection-block-special';
    container.style.cssText = `
        margin: 16px 0;
        padding: 16px 20px;
        background: ${style.bgColor};
        border: 2px solid ${style.borderColor};
        border-radius: 10px;
        box-shadow: 0 2px 8px ${style.borderColor};
        animation: slideInBlock 0.4s ease-out;
    `;

    const thinkingBlock = block.block?.thinkingBlock;
    if (!thinkingBlock) return null;

    const metadata = thinkingBlock.metadata || {};
    const content = thinkingBlock.thinking || '';

    // Header with prominent styling
    if (metadata.title) {
        const header = document.createElement('div');
        header.style.cssText = `
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 14px;
            padding-bottom: 10px;
            border-bottom: 1px solid ${style.borderColor};
            color: ${style.color};
            font-weight: 700;
            font-size: 15px;
        `;
        header.innerHTML = `
            <span style="font-size: 20px;">${style.icon}</span>
            <span>${metadata.title}</span>
        `;
        container.appendChild(header);
    }

    // Content with emphasis
    const contentDiv = createContentDiv(content, style);
    contentDiv.style.fontSize = '14px';
    contentDiv.style.lineHeight = '1.7';
    container.appendChild(contentDiv);

    return container;
}

/**
 * Render analysis blocks with structured data layout
 */
function renderAnalysisSpecial(block, style) {
    const container = document.createElement('div');
    container.className = 'content-block analysis-block-special';
    container.style.cssText = `
        margin: 12px 0;
        padding: 14px 18px;
        background: ${style.bgColor};
        border-left: 3px solid ${style.borderColor};
        border-radius: 6px;
        animation: slideInBlock 0.3s ease-out;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, monospace;
    `;

    const thinkingBlock = block.block?.thinkingBlock;
    if (!thinkingBlock) return null;

    const metadata = thinkingBlock.metadata || {};
    const content = thinkingBlock.thinking || '';

    // Header
    if (metadata.title) {
        const header = document.createElement('div');
        header.style.cssText = `
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 12px;
            color: ${style.color};
            font-weight: 600;
            font-size: 14px;
        `;
        header.innerHTML = `
            <span style="font-size: 18px;">${style.icon}</span>
            <span>${metadata.title}</span>
        `;
        container.appendChild(header);
    }

    // Parse content for key-value pairs and structure
    const contentDiv = createStructuredContentDiv(content, style);
    container.appendChild(contentDiv);

    return container;
}

/**
 * Create structured content div highlighting key-value pairs
 */
function createStructuredContentDiv(content, style) {
    const div = document.createElement('div');
    div.style.cssText = `
        display: flex;
        flex-direction: column;
        gap: 4px;
    `;

    const lines = content.split('\n');
    lines.forEach(line => {
        const trimmed = line.trim();
        if (!trimmed) return;

        // Check for key-value pairs
        const kvMatch = trimmed.match(/^([^:=]+)[:=]\s*(.+)$/);

        if (kvMatch) {
            const [, key, value] = kvMatch;
            const row = document.createElement('div');
            row.style.cssText = `
                display: flex;
                gap: 8px;
                padding: 4px 0;
                font-size: 13px;
            `;
            row.innerHTML = `
                <span style="color: ${style.color}; font-weight: 500;">${key.trim()}:</span>
                <span style="color: ${style.color}; font-weight: 700; opacity: 1;">${value.trim()}</span>
            `;
            div.appendChild(row);
        } else {
            const p = document.createElement('p');
            p.style.cssText = `
                margin: 4px 0;
                color: ${style.color};
                font-size: 13px;
                line-height: 1.6;
            `;
            p.textContent = trimmed;
            div.appendChild(p);
        }
    });

    return div;
}

/**
 * Render debug blocks with technical/monospace styling
 */
function renderDebugSpecial(block, style) {
    const container = document.createElement('div');
    container.className = 'content-block debug-block-special';
    container.style.cssText = `
        margin: 8px 0;
        padding: 10px 14px;
        background: rgba(149, 165, 166, 0.05);
        border: 1px dashed rgba(149, 165, 166, 0.3);
        border-radius: 4px;
        font-family: 'Monaco', 'Courier New', monospace;
        font-size: 11px;
        color: #7f8c8d;
        animation: fadeIn 0.3s ease-out;
    `;

    const thinkingBlock = block.block?.thinkingBlock;
    if (!thinkingBlock) return null;

    const metadata = thinkingBlock.metadata || {};
    const content = thinkingBlock.thinking || '';

    // Debug header
    if (metadata.title) {
        const header = document.createElement('div');
        header.style.cssText = `
            display: flex;
            align-items: center;
            gap: 6px;
            margin-bottom: 8px;
            padding-bottom: 6px;
            border-bottom: 1px dotted rgba(149, 165, 166, 0.2);
            color: #7f8c8d;
            font-weight: 600;
            font-size: 11px;
        `;
        header.innerHTML = `
            <span style="font-size: 14px;">🐛</span>
            <span>[DEBUG]</span>
            <span>${metadata.title}</span>
        `;
        container.appendChild(header);
    }

    // Content with line numbers
    const contentDiv = createDebugContentDiv(content);
    container.appendChild(contentDiv);

    return container;
}

/**
 * Create debug content div with line numbers
 */
function createDebugContentDiv(content) {
    const div = document.createElement('div');
    div.style.cssText = `
        display: flex;
        flex-direction: column;
        gap: 2px;
    `;

    const lines = content.split('\n');
    lines.forEach((line, i) => {
        const row = document.createElement('div');
        row.style.cssText = `
            display: flex;
            gap: 10px;
            font-size: 11px;
            line-height: 1.5;
        `;
        row.innerHTML = `
            <span style="color: #95a5a6; opacity: 0.6; user-select: none; min-width: 30px; text-align: right;">${String(i + 1).padStart(3, ' ')}</span>
            <span style="color: #7f8c8d;">│</span>
            <span style="color: #7f8c8d; flex: 1;">${line || ' '}</span>
        `;
        div.appendChild(row);
    });

    return div;
}

/**
 * Generic thinking block renderer (fallback)
 */
function renderGenericThinkingBlock(block, style) {
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
 * Render a progress block with animated indicator
 */
function renderProgressBlock(block, style) {
    const thinkingBlock = block.block?.thinkingBlock;
    if (!thinkingBlock) return null;

    const container = document.createElement('div');
    container.className = 'content-block progress-block';
    container.style.cssText = `
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 6px 12px;
        background: ${style.bgColor};
        border: 1px solid ${style.borderColor};
        border-radius: 20px;
        font-size: 12px;
        color: ${style.color};
        margin: 6px 0;
        animation: fadeIn 0.3s ease-out, pulse 2s ease-in-out infinite;
    `;

    // Create animated spinner
    const spinner = document.createElement('span');
    spinner.innerHTML = style.icon;
    spinner.style.cssText = `
        font-size: 14px;
        animation: spin 2s linear infinite;
    `;

    const text = document.createElement('span');
    text.textContent = thinkingBlock.thinking;
    text.style.cssText = `
        font-weight: 500;
    `;

    container.appendChild(spinner);
    container.appendChild(text);

    // Add progress bar if content suggests progress
    const progressMatch = thinkingBlock.thinking.match(/(\d+)%/);
    if (progressMatch) {
        const percentage = parseInt(progressMatch[1]);
        const progressBar = document.createElement('div');
        progressBar.style.cssText = `
            width: 60px;
            height: 4px;
            background: rgba(26, 188, 156, 0.2);
            border-radius: 2px;
            overflow: hidden;
        `;

        const progressFill = document.createElement('div');
        progressFill.style.cssText = `
            width: ${percentage}%;
            height: 100%;
            background: ${style.color};
            transition: width 0.5s ease-out;
        `;

        progressBar.appendChild(progressFill);
        container.appendChild(progressBar);
    }

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
