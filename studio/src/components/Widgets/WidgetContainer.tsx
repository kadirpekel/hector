import type { ReactNode } from "react";
import React from "react";
import { ChevronDown, Sparkles } from "lucide-react";
import { cn } from "../../lib/utils";
import {
  getWidgetContainerClasses,
  getWidgetHeaderClasses,
} from "./widgetStyles";
import type { WidgetStatusStyles } from "./widgetStyles";

interface WidgetContainerProps {
  /** Accessible label for the widget region */
  ariaLabel: string;
  /** Icon component to display in the header */
  icon: ReactNode;
  /** Text label displayed in the header */
  label: string;
  /** Current status text for accessibility */
  statusText: string;
  /** Whether the widget is expanded */
  isExpanded: boolean;
  /** Whether the widget is in an active state */
  isActive: boolean;
  /** Whether the widget is completed */
  isCompleted: boolean;
  /** Status styles from getWidgetStatusStyles */
  statusStyles: WidgetStatusStyles;
  /** Whether to apply animation styles */
  shouldAnimate?: boolean;
  /** Toggle handler */
  onToggle: () => void;
  /** Optional status indicator component (replaces default) */
  statusIndicator?: ReactNode;
  /** Content to render when expanded */
  children: ReactNode;
  /** Maximum height for expanded content */
  maxHeight?: string;
}

/**
 * Shared container component for all widget types.
 * Provides consistent:
 * - Accessibility (ARIA attributes, keyboard navigation)
 * - Header styling and expand/collapse behavior
 * - Animation and status indicators
 *
 * Usage:
 * ```tsx
 * <WidgetContainer
 *   ariaLabel="Tool: my-tool"
 *   icon={<Wrench size={16} />}
 *   label="Tool: my-tool"
 *   statusText="working"
 *   isExpanded={isExpanded}
 *   isActive={isActive}
 *   isCompleted={isCompleted}
 *   statusStyles={statusStyles}
 *   onToggle={handleToggle}
 * >
 *   {expanded content}
 * </WidgetContainer>
 * ```
 */
export const WidgetContainer: React.FC<WidgetContainerProps> = ({
  ariaLabel,
  icon,
  label,
  statusText,
  isExpanded,
  isActive,
  isCompleted,
  statusStyles,
  shouldAnimate = false,
  onToggle,
  statusIndicator,
  children,
  maxHeight = "300px",
}) => {
  return (
    <div
      className={getWidgetContainerClasses(
        statusStyles,
        isExpanded,
        isCompleted,
      )}
      role="region"
      aria-label={ariaLabel}
    >
      <div
        className={getWidgetHeaderClasses(statusStyles, isActive)}
        onClick={onToggle}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            onToggle();
          }
        }}
        role="button"
        tabIndex={0}
        aria-expanded={isExpanded}
        aria-label={`Toggle ${label} details. Status: ${statusText}`}
      >
        <div className={cn("relative", statusStyles.iconColor)}>
          {isActive && (
            <Sparkles
              size={12}
              className="absolute -top-1 -right-1 animate-pulse opacity-70"
            />
          )}
          <div
            className={cn(
              "transition-transform duration-200",
              shouldAnimate &&
                "animate-[badgeLifecycle_2s_ease-in-out_infinite]",
              isExpanded && !isCompleted && "rotate-12",
            )}
          >
            {icon}
          </div>
        </div>

        <span
          className={cn("font-medium flex-1 text-sm", statusStyles.textColor)}
        >
          {label}
        </span>

        <div className="ml-auto flex items-center gap-2">
          {statusIndicator}
          <ChevronDown
            size={14}
            className={cn(
              "transition-transform duration-300 text-gray-400",
              isExpanded ? "rotate-0" : "-rotate-90",
            )}
          />
        </div>
      </div>

      <div
        className={cn(
          "overflow-hidden transition-all duration-300 ease-in-out",
          isExpanded ? `max-h-[${maxHeight}] opacity-100` : "max-h-0 opacity-0",
        )}
      >
        {children}
      </div>
    </div>
  );
};
