import type { ApiError, ErrorResponse, ValidationIssue } from "./types";

export function isErrorResponse(x: unknown): x is ErrorResponse {
  return (
    x !== null &&
    typeof x === "object" &&
    typeof (x as ErrorResponse).error === "string" &&
    typeof (x as ErrorResponse).message === "string"
  );
}

export function normalizeApiError(e: unknown): { message: string; issues?: ValidationIssue[] } {
  // Check if it's ApiError-like (has status, error, message)
  if (e !== null && typeof e === "object" && "status" in e && "error" in e && "message" in e) {
    const apiErr = e as ApiError & { message?: unknown; issues?: unknown };
    const message = typeof apiErr.message === "string" ? apiErr.message : "Request failed";
    const issues = Array.isArray(apiErr.issues) ? (apiErr.issues as ValidationIssue[]) : undefined;
    return { message, issues };
  }

  // Fallback: Error instance
  if (e instanceof Error) {
    return { message: e.message };
  }

  // Final fallback
  return { message: String(e ?? "Request failed") };
}
