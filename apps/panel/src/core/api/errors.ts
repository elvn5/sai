import { HttpErrorResponse } from '@angular/common/http';

// The API's error message (NestJS-style { message }), or `fallback`.
export function apiErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof HttpErrorResponse) {
    const message = (error.error as { message?: unknown } | null)?.message;
    if (typeof message === 'string' && message) return message;
  }
  return fallback;
}

export function hasStatus(error: unknown, status: number): boolean {
  return error instanceof HttpErrorResponse && error.status === status;
}
