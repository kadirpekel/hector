/**
 * Structured logging utility with debug mode support
 */

type LogLevel = 'debug' | 'info' | 'warn' | 'error';

interface LogContext {
  [key: string]: any;
}

class Logger {
  private debugMode: boolean;

  constructor() {
    // Check if running in development or debug mode
    this.debugMode = import.meta.env.DEV || import.meta.env.VITE_DEBUG === 'true';
  }

  private sanitize(obj: any): any {
    if (typeof obj !== 'object' || obj === null) {
      return obj;
    }

    const sanitized: Record<string, any> = Array.isArray(obj) ? [] : {};
    for (const [key, value] of Object.entries(obj)) {
      // Mask sensitive fields
      if (
        key.toLowerCase().includes('token') ||
        key.toLowerCase().includes('secret') ||
        key.toLowerCase().includes('key') ||
        key.toLowerCase().includes('password')
      ) {
        sanitized[key] = value ? `${String(value).substring(0, 8)}...` : value;
      } else if (typeof value === 'object' && value !== null) {
        sanitized[key] = this.sanitize(value);
      } else {
        sanitized[key] = value;
      }
    }
    return sanitized;
  }

  private log(level: LogLevel, message: string, context?: LogContext) {
    // Only show debug logs in debug mode
    if (level === 'debug' && !this.debugMode) {
      return;
    }

    const timestamp = new Date().toISOString();
    const sanitizedContext = context ? this.sanitize(context) : undefined;

    const logFn = level === 'error' ? console.error : level === 'warn' ? console.warn : console.log;

    if (sanitizedContext) {
      logFn(`[${timestamp}] [${level.toUpperCase()}] ${message}`, sanitizedContext);
    } else {
      logFn(`[${timestamp}] [${level.toUpperCase()}] ${message}`);
    }
  }

  debug(message: string, context?: LogContext) {
    this.log('debug', message, context);
  }

  info(message: string, context?: LogContext) {
    this.log('info', message, context);
  }

  warn(message: string, context?: LogContext) {
    this.log('warn', message, context);
  }

  error(message: string, context?: LogContext) {
    this.log('error', message, context);
  }
}

export const logger = new Logger();
