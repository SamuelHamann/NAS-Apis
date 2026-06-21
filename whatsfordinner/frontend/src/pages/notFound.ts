import { h, mount } from "../dom";

export function renderNotFound(params: { path?: string } = {}): void {
  mount(
    "#app",
    h("h2", null, "Not found"),
    h(
      "p",
      { class: "muted" },
      `No route matches ${params.path ?? "this URL"}.`,
    ),
    h("p", null, h("a", { href: "#/" }, "Go home")),
  );
}
