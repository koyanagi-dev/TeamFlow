import type { ApiError } from './types';
import { isErrorResponse } from './error';

export async function apiFetch<T>(
  path: string,
  init: RequestInit = {}
): Promise<T> {
  // Content-Type は body がある場合のみ付与する
  // 既に init.headers に Content-Type が指定されている場合はそれを尊重
  const existingHeaders = init.headers as Record<string, string> | undefined;
  const headers: Record<string, string> = existingHeaders ? { ...existingHeaders } : {};
  
  if (init.body && !headers['Content-Type'] && !headers['content-type']) {
    headers['Content-Type'] = 'application/json';
  }

  const res = await fetch(path, {
    ...init,
    headers: Object.keys(headers).length > 0 ? headers : undefined,
    credentials: 'include',
    cache: 'no-store',
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

  let message = text;
  if (maybeJson !== null && typeof maybeJson === 'object') {
    const obj = maybeJson as { message?: string; detail?: string };
    if (typeof obj.message === 'string') message = obj.message;
    else if (typeof obj.detail === 'string') message = obj.detail;
  }
  if (typeof message !== 'string' || !message.length) {
    message = `HTTP ${res.status}`;
  }

  throw {
    status: res.status,
    error: 'UNKNOWN_ERROR',
    message,
    raw: maybeJson ?? text,
  } as ApiError;
}
