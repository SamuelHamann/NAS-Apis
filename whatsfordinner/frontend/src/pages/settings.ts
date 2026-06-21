import { h, mount } from "../dom";
import {
  DEFAULT_API_BASE_URL,
  getApiBaseUrl,
  getApiKey,
  setApiBaseUrl,
  setApiKey,
} from "../config";

export async function renderSettings(): Promise<void> {
  const baseUrlInput = h("input", {
    type: "url",
    name: "baseUrl",
    value: getApiBaseUrl(),
    placeholder: DEFAULT_API_BASE_URL,
    required: true,
  });

  const apiKeyInput = h("input", {
    type: "text",
    name: "apiKey",
    value: getApiKey() ?? "",
    placeholder: "UUID from the api_keys table",
    autocomplete: "off",
    spellcheck: false,
  });

  const status = h("p", { class: "muted" }, "");

  const form = h(
    "form",
    {
      onsubmit: (e: Event) => {
        e.preventDefault();
        setApiBaseUrl(baseUrlInput.value);
        setApiKey(apiKeyInput.value);
        status.textContent = "Saved.";
        status.classList.remove("error");
      },
    },
    h("label", null, "API base URL", baseUrlInput),
    h("label", null, "API key (X-Api-Key)", apiKeyInput),
    h("button", { type: "submit" }, "Save"),
    status,
  );

  mount(
    "#app",
    h("h2", null, "Settings"),
    h(
      "p",
      { class: "muted" },
      "Stored in this browser's localStorage. Reload the page after saving.",
    ),
    h("div", { class: "card" }, form),
  );
}
