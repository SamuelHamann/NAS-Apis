/**
 * Runtime configuration.
 *
 * The frontend is a static SPA, so there are no build-time secrets. Instead,
 * the API base URL and the API key are kept in `localStorage` and edited from
 * the in-app Settings page.
 */

const STORAGE_PREFIX = "wfd:";
const KEY_API_BASE_URL = `${STORAGE_PREFIX}apiBaseUrl`;
const KEY_API_KEY = `${STORAGE_PREFIX}apiKey`;

/**
 * Default API base URL.
 *
 * `/api` matches the path that the bundled nginx config proxies to the Go
 * backend (see `whatsfordinner/frontend/nginx.conf`) — so the Dockerised
 * deployment works without any manual configuration. For local development
 * with `npm run dev` against a directly-exposed API, override this from the
 * in-app Settings page (#/settings), e.g. `http://localhost:8080`.
 */
export const DEFAULT_API_BASE_URL = "/api";

export function getApiBaseUrl(): string {
  return localStorage.getItem(KEY_API_BASE_URL) ?? DEFAULT_API_BASE_URL;
}

export function setApiBaseUrl(url: string): void {
  localStorage.setItem(KEY_API_BASE_URL, url.trim());
}

export function getApiKey(): string | null {
  const v = localStorage.getItem(KEY_API_KEY);
  return v && v.length > 0 ? v : null;
}

export function setApiKey(key: string): void {
  const trimmed = key.trim();
  if (trimmed.length === 0) {
    localStorage.removeItem(KEY_API_KEY);
  } else {
    localStorage.setItem(KEY_API_KEY, trimmed);
  }
}
