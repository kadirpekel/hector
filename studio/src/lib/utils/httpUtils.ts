/**
 * HTTP utility functions for API calls with retry logic
 */

interface RetryOptions {
  retries?: number;
  timeout?: number;
  backoffMs?: number;
}

/**
 * Fetch with exponential backoff retry logic
 */
export async function fetchWithRetry(
  url: string,
  init: RequestInit = {},
  options: RetryOptions = {}
): Promise<Response> {
  const { retries = 3, timeout = 5000, backoffMs = 1000 } = options;

  let lastError: Error | null = null;

  for (let attempt = 0; attempt <= retries; attempt++) {
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), timeout);

      const response = await fetch(url, {
        ...init,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);
      return response;
    } catch (error) {
      lastError = error as Error;

      // Don't retry on abort (user-initiated)
      if (error instanceof Error && error.name === 'AbortError' && !error.message.includes('timeout')) {
        throw error;
      }

      // Last attempt - throw
      if (attempt === retries) {
        break;
      }

      // Exponential backoff: wait before retry
      const delay = backoffMs * Math.pow(2, attempt);
      console.log(`[httpUtils] Retry ${attempt + 1}/${retries} after ${delay}ms:`, error);
      await new Promise((resolve) => setTimeout(resolve, delay));
    }
  }

  throw lastError || new Error('Fetch failed with unknown error');
}
