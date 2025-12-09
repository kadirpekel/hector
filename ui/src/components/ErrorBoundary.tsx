import React, { Component, type ReactNode } from "react";
import { AlertCircle, RefreshCw } from "lucide-react";

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  onReset?: () => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

/**
 * Error Boundary component to catch and handle React errors gracefully.
 *
 * Prevents widget crashes from bringing down the entire message list.
 * Shows a user-friendly error message with option to retry.
 *
 * Usage:
 * ```tsx
 * <ErrorBoundary>
 *   <MessageList messages={messages} />
 * </ErrorBoundary>
 * ```
 */
export class ErrorBoundary extends Component<
  ErrorBoundaryProps,
  ErrorBoundaryState
> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
    };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return {
      hasError: true,
      error,
    };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo): void {
    // Log error to console in development
    if (import.meta.env.DEV) {
      console.error("ErrorBoundary caught an error:", error, errorInfo);
    }

    // In production, you might want to log to an error reporting service
    // e.g., Sentry, LogRocket, etc.
  }

  handleReset = (): void => {
    this.setState({
      hasError: false,
      error: null,
    });
    this.props.onReset?.();
  };

  render(): ReactNode {
    if (this.state.hasError) {
      // Use custom fallback if provided
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // Default error UI
      return (
        <div className="flex flex-col items-center justify-center p-6 bg-red-500/10 border border-red-500/30 rounded-xl">
          <div className="flex items-center gap-2 mb-3 text-red-400">
            <AlertCircle size={20} />
            <h3 className="text-lg font-semibold">Something went wrong</h3>
          </div>
          <p className="text-sm text-gray-400 mb-4 text-center max-w-md">
            An error occurred while rendering this component. Please try
            refreshing or contact support if the problem persists.
          </p>
          {import.meta.env.DEV && this.state.error && (
            <details className="mb-4 w-full">
              <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-400">
                Error Details (Development Only)
              </summary>
              <pre className="mt-2 p-3 bg-black/50 rounded text-xs text-red-300 overflow-auto max-h-40">
                {this.state.error.message}
                {"\n"}
                {this.state.error.stack}
              </pre>
            </details>
          )}
          <button
            onClick={this.handleReset}
            className="flex items-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors text-sm font-medium"
          >
            <RefreshCw size={16} />
            Try Again
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

/**
 * Lightweight error boundary specifically for individual widgets.
 * Shows inline error without taking up much space.
 */
export function WidgetErrorBoundary({ children }: { children: ReactNode }) {
  return (
    <ErrorBoundary
      fallback={
        <div className="flex items-center gap-2 p-2 bg-red-500/10 border border-red-500/20 rounded text-xs text-red-400">
          <AlertCircle size={14} />
          <span>Failed to render widget</span>
        </div>
      }
    >
      {children}
    </ErrorBoundary>
  );
}
