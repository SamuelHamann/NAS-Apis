/**
 * Tiny DOM helpers used in place of a framework.
 *
 * `h()` builds an element tree imperatively (avoids `innerHTML` so user data
 * cannot accidentally inject HTML). `mount()` replaces the children of a
 * target element with the given nodes.
 */

export type Child = Node | string | number | null | undefined | false;
export type Attrs = Record<
  string,
  string | number | boolean | EventListener | null | undefined
>;

export function h<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  attrs?: Attrs | null,
  ...children: Child[]
): HTMLElementTagNameMap[K] {
  const el = document.createElement(tag);
  if (attrs) {
    for (const [name, value] of Object.entries(attrs)) {
      if (value == null || value === false) continue;
      if (name.startsWith("on") && typeof value === "function") {
        el.addEventListener(
          name.slice(2).toLowerCase(),
          value as EventListener,
        );
      } else if (name === "class") {
        el.className = String(value);
      } else if (value === true) {
        el.setAttribute(name, "");
      } else {
        el.setAttribute(name, String(value));
      }
    }
  }
  appendChildren(el, children);
  return el;
}

export function clear(el: Element): void {
  while (el.firstChild) {
    el.removeChild(el.firstChild);
  }
}

export function mount(selector: string, ...children: Child[]): void {
  const el = document.querySelector(selector);
  if (!el) throw new Error(`mount target not found: ${selector}`);
  clear(el);
  appendChildren(el, children);
}

function appendChildren(el: Element, children: Child[]): void {
  for (const child of children) {
    if (child == null || child === false) continue;
    el.append(child instanceof Node ? child : String(child));
  }
}
