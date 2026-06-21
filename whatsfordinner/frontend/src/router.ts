/**
 * A minimal hash-based router.
 *
 * Routes are registered with a path pattern and a render function. Patterns
 * may contain `:param` segments, which are extracted and passed to the render
 * function. The first matching route wins; otherwise the not-found render runs.
 */

export type RouteParams = Record<string, string>;
export type RouteRender = (params: RouteParams) => void | Promise<void>;

interface Route {
  pattern: string;
  segments: string[];
  render: RouteRender;
}

export class Router {
  private readonly routes: Route[] = [];
  private notFound: RouteRender = () => {
    /* overridden by setNotFound */
  };

  constructor(private readonly mountSelector: string) {}

  add(pattern: string, render: RouteRender): this {
    this.routes.push({
      pattern,
      segments: pattern.split("/").filter(Boolean),
      render,
    });
    return this;
  }

  setNotFound(render: RouteRender): this {
    this.notFound = render;
    return this;
  }

  start(): void {
    window.addEventListener("hashchange", () => this.resolve());
    this.resolve();
  }

  navigate(path: string): void {
    window.location.hash = path;
  }

  currentPath(): string {
    return window.location.hash.replace(/^#/, "") || "/";
  }

  private async resolve(): Promise<void> {
    const path = this.currentPath();
    const target = document.querySelector(this.mountSelector);
    if (!target) {
      throw new Error(`router mount not found: ${this.mountSelector}`);
    }

    this.highlightActiveNav(path);

    for (const route of this.routes) {
      const params = matchPath(route.segments, path);
      if (params) {
        try {
          await route.render(params);
        } catch (err) {
          console.error("route render failed:", err);
          target.textContent = `Error: ${(err as Error).message}`;
        }
        return;
      }
    }

    await this.notFound({ path });
  }

  private highlightActiveNav(path: string): void {
    const links = document.querySelectorAll<HTMLAnchorElement>(
      `${this.mountSelector === "#app" ? "" : ""}#nav a`,
    );
    for (const link of links) {
      const target = link.getAttribute("href")?.replace(/^#/, "") ?? "";
      link.classList.toggle("active", target === path);
    }
  }
}

function matchPath(
  patternSegments: string[],
  path: string,
): RouteParams | null {
  const pathSegments = path.split("/").filter(Boolean);
  if (patternSegments.length !== pathSegments.length) return null;

  const params: RouteParams = {};
  for (let i = 0; i < patternSegments.length; i++) {
    const pat = patternSegments[i]!;
    const seg = pathSegments[i]!;
    if (pat.startsWith(":")) {
      params[pat.slice(1)] = decodeURIComponent(seg);
    } else if (pat !== seg) {
      return null;
    }
  }
  return params;
}
