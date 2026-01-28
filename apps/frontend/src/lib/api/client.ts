import type { ApiError } from "./types";
import { isErrorResponse } from "./error";

function extractMessage(maybeJson: unknown, fallbackText: string): string {
  if (maybeJson !== null && typeof maybeJson === "object") {
    const obj = maybeJson as {
      message?: string;
      detail?: string;
      error?: string;
    };
    if (typeof obj.message === "string") return obj.message;
    if (typeof obj.detail === "string") return obj.detail;
    if (typeof obj.error === "string") return obj.error;
  }
  return fallbackText;
}

export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  // Normalize headers safely (init.headers can be Headers | string[][] | Record<string,string>)
  const headers = new Headers(init.headers);

  // Add Content-Type only when request has body, and only if caller didn't specify it.
  if (init.body && !headers.has("content-type")) {
    headers.set("Content-Type", "application/json");
  }

  const res = await fetch(path, {
    ...init,
    headers,
    credentials: "include",
    cache: "no-store",
  });

  if (res.status === 204) return null as unknown as T;

  const text = await res.text();
  const maybeJson = (() => {
    try {
      return text ? (JSON.parse(text) as unknown) : null;
    } catch {
      return null;
    }
  })();

  if (res.ok) return maybeJson as T;

  // TeamFlow standard error response (ErrorResponse)
  if (isErrorResponse(maybeJson)) {
    const err: ApiError = {
      status: res.status,
      error: maybeJson.error,
      message: maybeJson.message,
      issues: maybeJson.details?.issues,
      raw: maybeJson,
    };
    throw err;
  }

  // Fallback: extract message from non-standard JSON/text
  const message = extractMessage(maybeJson, text) || `HTTP ${res.status}`;

  throw {
    status: res.status,
    error: "UNKNOWN_ERROR",
    message,
    raw: maybeJson ?? text,
  } as ApiError;
}
