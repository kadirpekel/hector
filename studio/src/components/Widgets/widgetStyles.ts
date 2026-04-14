import { cn } from '../../lib/utils';

export interface WidgetStatusStyles {
    border: string;
    bg: string;
    headerBg: string;
    iconColor: string;
    textColor: string;
    glow: string;
    opacity?: string;
    scale?: string;
}

/**
 * Get status-based styles for widgets
 * Returns less prominent styles when completed to reduce visual dominance
 */
export function getWidgetStatusStyles(
    status: string | undefined,
    isCompleted: boolean
): WidgetStatusStyles {
    // When completed, use subtle styles to reduce visual dominance
    if (isCompleted) {
        return {
            border: 'border-white/5',
            bg: 'bg-black/10',
            headerBg: 'bg-white/2',
            iconColor: 'text-gray-500',
            textColor: 'text-gray-400',
            glow: '',
            opacity: 'opacity-60',
            scale: 'scale-95',
        };
    }

    // Active/working styles - prominent
    switch (status) {
        case 'working':
        case 'pending':
            return {
                border: 'border-yellow-500/30',
                bg: 'bg-gradient-to-r from-yellow-500/10 to-purple-500/10',
                headerBg: 'bg-gradient-to-r from-yellow-500/5 to-purple-500/5',
                iconColor: 'text-yellow-400',
                textColor: 'text-yellow-200',
                glow: 'shadow-[0_0_8px_rgba(234,179,8,0.3)]',
            };
        case 'active':
            return {
                border: 'border-blue-500/30',
                bg: 'bg-gradient-to-r from-blue-500/10 to-purple-500/10',
                headerBg: 'bg-gradient-to-r from-blue-500/5 to-purple-500/5',
                iconColor: 'text-blue-400',
                textColor: 'text-blue-200',
                glow: 'shadow-[0_0_8px_rgba(59,130,246,0.3)]',
            };
        case 'success':
            return {
                border: 'border-green-500/15',
                bg: 'bg-black/10',
                headerBg: 'bg-white/2',
                iconColor: 'text-green-500',
                textColor: 'text-green-400',
                glow: '',
            };
        case 'failed':
            return {
                border: 'border-red-500/15',
                bg: 'bg-black/10',
                headerBg: 'bg-white/2',
                iconColor: 'text-red-500',
                textColor: 'text-red-400',
                glow: '',
            };
        default:
            return {
                border: 'border-white/10',
                bg: 'bg-black/20',
                headerBg: 'bg-white/5',
                iconColor: 'text-purple-400',
                textColor: 'text-purple-200',
                glow: '',
            };
    }
}

/**
 * Get widget container classes
 */
export function getWidgetContainerClasses(
    statusStyles: WidgetStatusStyles,
    isExpanded: boolean,
    isCompleted: boolean
): string {
    return cn(
        "border rounded-xl overflow-hidden text-sm transition-all duration-300",
        statusStyles.border,
        statusStyles.bg,
        statusStyles.glow,
        statusStyles.opacity,
        statusStyles.scale,
        isExpanded && !isCompleted && "shadow-lg", // Only show shadow when expanded and not completed
        isCompleted && "shadow-none" // Remove shadow when completed
    );
}

/**
 * Get widget header classes
 */
export function getWidgetHeaderClasses(
    statusStyles: WidgetStatusStyles,
    isActive: boolean
): string {
    return cn(
        "flex items-center gap-2.5 p-2 cursor-pointer transition-all duration-200",
        statusStyles.headerBg,
        "hover:bg-white/10",
        isActive && "hover:bg-white/15"
    );
}

