import { getHealth, getReady } from "../api/health";
import { ApiError } from "../api/client";
import { h, mount } from "../dom";
import { getApiBaseUrl, getApiKey } from "../config";

export async function renderHome(): Promise<void> {
  const baseUrl = getApiBaseUrl();
  const hasKey = getApiKey() !== null;

  const healthEl = h("p", null, "Checking…");
  const readyEl = h("p", null, "Checking…");

  mount(
    "#app",
    h("h2", null, "What's for dinner?"),
    h(
      "p",
      { class: "muted" },
      "A small dashboard for the whatsfordinner API.",
    ),
    h(
      "div",
      { class: "card" },
      h("h3", null, "API status"),
      h("p", null, h("strong", null, "Base URL: "), baseUrl),
      h(
        "p",
        null,
        h("strong", null, "API key: "),
        hasKey ? "configured" : "not configured (see #/settings)",
      ),
      h("p", null, h("strong", null, "/health: "), healthEl),
      h("p", null, h("strong", null, "/ready: "), readyEl),
    ),
  );

  void probe(getHealth, healthEl);
  void probe(getReady, readyEl);
}

async function probe(
  fn: (signal?: AbortSignal) => Promise<{ status: string }>,
  target: HTMLElement,
): Promise<void> {
  try {
    const result = await fn();
    target.textContent = result.status;
  } catch (err) {
    target.classList.add("error");
    target.textContent =
      err instanceof ApiError
        ? `${err.status} — ${err.message}`
        : (err as Error).message;
  }
}
