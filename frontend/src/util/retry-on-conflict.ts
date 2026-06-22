import { HttpErrorResponse } from '@angular/common/http';

/**
 * Runs an async operation, retrying when the API responds with HTTP 409
 * (optimistic-lock conflict on a shared event stream). The backend rolls the
 * conflicting transaction back and asks the client to retry, so the operation
 * had no effect and is safe to re-run. Any other error, or exhausting the
 * attempts, rethrows the last error.
 */
export async function retryOnConflict<T>(fn: () => Promise<T>, attempts = 3): Promise<T> {
  for (let attempt = 0; attempt < attempts; attempt++) {
    try {
      return await fn();
    } catch (e) {
      if (e instanceof HttpErrorResponse && e.status === 409 && attempt < attempts - 1) {
        continue;
      }
      throw e;
    }
  }
  // Unreachable: the final attempt either returns or throws above.
  throw new Error('retryOnConflict exhausted attempts');
}
