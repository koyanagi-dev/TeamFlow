import type { ApiError, ErrorResponse, ValidationIssue } from './types';

export function isErrorResponse(x: unknown): x is ErrorResponse {
  return (
    x !== null &&
    typeof x === 'object' &&
    typeof (x as ErrorResponse).error === 'string' &&
    typeof (x as ErrorResponse).message === 'string'
  );
}

export function normalizeApiError(
  e: unknown
): { message: string; issues?: ValidationIssue[] } {
  if (e !== null && typeof e === 'object' && 'message' in e) {
    const anyErr = e as ApiError & { message?: unknown; issues?: unknown };
    const message =
      typeof anyErr.message === 'string' ? anyErr.message : 'Request failed';
    const issues = Array.isArray(anyErr.issues)
      ? (anyErr.issues as ValidationIssue[])
      : undefined;
    return { message, issues };
  }
  return { message: String(e ?? 'Request failed') };
}
