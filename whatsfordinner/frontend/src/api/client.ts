/**
 * Low-level HTTP client.
 *
 * Wraps `fetch` and:
 *   - prefixes paths with the configured API base URL;
 *   - serializes query parameters (supports repeated keys for arrays);
 *   - attaches the `X-Api-Key` header from runtime config;
 *   - parses JSON responses and turns non-2xx responses into `ApiError`s.
 */

import { getApiBaseUrl, getApiKey } from "../config";

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/**
 * Accepted values for a query parameter. Arrays produce repeated keys
 * (`?tags=1&tags=2`), `null`/`undefined` values are skipped, everything else
 * is coerced with `String(...)`.
 */
export type QueryValue =
  | string
  | number
  | boolean
  | null
  | undefined
  | ReadonlyArray<string | number | boolean>;

/**
 * Query parameters are typed as `object` so the per-resource interfaces in
 * `types/models.ts` can be passed in directly. (TypeScript will not assign a
 * closed interface to `Record<string, unknown>`, but `object` accepts any
 * non-primitive value.) Values are still handled defensively at runtime.
 */
export type QueryParams = object;

export interface RequestOptions {
  query?: QueryParams;
  body?: unknown;
  signal?: AbortSignal;
}

function buildUrl(path: string, query?: QueryParams): string {
  const base = getApiBaseUrl().replace(/\/$/, "");
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  const url = new URL(`${base}${cleanPath}`, window.location.origin);
  if (query) {
    for (const [key, value] of Object.entries(query)) {
      if (value === undefined || value === null) continue;
      if (Array.isArray(value)) {
        for (const item of value as ReadonlyArray<unknown>) {
          if (item === undefined || item === null) continue;
          url.searchParams.append(key, String(item));
        }
      } else {
        url.searchParams.append(key, String(value));
      }
    }
  }
  return url.toString();
}

async function request<T>(
  method: string,
  path: string,
  opts: RequestOptions = {},
): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
  };
  const apiKey = getApiKey();
  if (apiKey) headers["X-Api-Key"] = apiKey;

  let body: BodyInit | undefined;
  if (opts.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(opts.body);
  }

  const init: RequestInit = { method, headers };
  if (body !== undefined) init.body = body;
  if (opts.signal) init.signal = opts.signal;

  const res = await fetch(buildUrl(path, opts.query), init);

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  let data: unknown;
  if (text.length > 0) {
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
  }

  if (!res.ok) {
    const message =
      data &&
      typeof data === "object" &&
      "error" in (data as Record<string, unknown>)
        ? String((data as { error: unknown }).error)
        : `HTTP ${res.status} ${res.statusText}`;
    throw new ApiError(res.status, message);
  }

  return data as T;
}

export const api = {
  get: <T>(path: string, opts?: RequestOptions) =>
    request<T>("GET", path, opts),
  post: <T>(
    path: string,
    body?: unknown,
    opts?: Omit<RequestOptions, "body">,
  ) => request<T>("POST", path, { ...opts, body }),
  put: <T>(
    path: string,
    body?: unknown,
    opts?: Omit<RequestOptions, "body">,
  ) => request<T>("PUT", path, { ...opts, body }),
  del: <T>(path: string, opts?: RequestOptions) =>
    request<T>("DELETE", path, opts),
};
