package handlers

import (
	"bytes"
	"strings"
	"testing"
)

// navbarActiveCases enumerate one fixture per page that sets
// PageData.ActiveNav, plus the href that should be highlighted as a result.
// Shared by the tests below so each renders through the exact same
// production templates (via parseTestTemplates) rather than a hand-rolled
// navbar fixture.
func navbarActiveCases(t *testing.T) []struct {
	name       string
	template   string
	data       any
	activeHref string
} {
	t.Helper()
	return []struct {
		name       string
		template   string
		data       any
		activeHref string
	}{
		{"pantry", "pantry.html", pantryPageFixture(t), "/pantry"},
		{"recipes", "recipes.html", recipesPageFixture(t), "/recipes"},
		{"recipe_detail", "recipe_detail.html", recipeDetailPageFixture(t), "/recipes"},
	}
}

// TestNavbarHighlightsActivePage checks that PageData.ActiveNav (set by
// each page's handler — see pantry_page.go, recipes_page.go,
// recipe_detail_page.go) results in exactly the matching navbar link (both
// the inline copy and its twin inside the burger-menu dropdown) getting
// wfd-nav-link--active, and no other link does.
func TestNavbarHighlightsActivePage(t *testing.T) {
	allHrefs := []string{"/pantry", "/recipes", "/ingredients"}

	for _, tc := range navbarActiveCases(t) {
		t.Run(tc.name, func(t *testing.T) {
			ts, ok := parseTestTemplates(t)[tc.template]
			if !ok {
				t.Fatalf("%s not found in template cache", tc.template)
			}

			var buf bytes.Buffer
			if err := ts.Execute(&buf, tc.data); err != nil {
				t.Fatalf("execute %s: %v", tc.template, err)
			}
			body := buf.String()

			for _, href := range allHrefs {
				active := activeLinkClassFor(body, href)
				if href == tc.activeHref {
					// Expect two matches: one in the inline .wfd-nav-links,
					// one in the .wfd-mobile-nav__panel dropdown.
					if active != 2 {
						t.Errorf("expected %q to carry wfd-nav-link--active twice (inline + burger menu), got %d",
							href, active)
					}
				} else if active != 0 {
					t.Errorf("expected %q not to carry wfd-nav-link--active, got %d occurrences", href, active)
				}
			}
		})
	}
}

// activeLinkClassFor counts how many times href's anchor tag in body carries
// the wfd-nav-link--active class, regardless of attribute order.
func activeLinkClassFor(body, href string) int {
	count := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, `href="`+href+`"`) && strings.Contains(line, "wfd-nav-link--active") {
			count++
		}
	}
	return count
}

// TestNavbarNoActiveLinkOnHome renders the home page (whose PageData never
// sets ActiveNav) and checks that none of the three nav links are
// highlighted — a page not in {pantry, recipes, ingredients} should never
// accidentally match the zero-value "".
func TestNavbarNoActiveLinkOnHome(t *testing.T) {
	ts, ok := parseTestTemplates(t)["home.html"]
	if !ok {
		t.Fatal(`template "home.html" not found`)
	}

	data := HomeData{PageData: PageData{Title: "Home", SignedIn: true, CurrentUser: "Alice"}}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute home.html: %v", err)
	}
	body := buf.String()

	// Note: ".wfd-nav-link--active" (with a leading dot) always appears in
	// the embedded <style> block regardless of data, so assert on the
	// class attribute a highlighted link would actually render instead of
	// the bare class-name substring.
	if strings.Contains(body, `class="wfd-nav-link wfd-nav-link--active"`) {
		t.Errorf("expected no nav link to be highlighted on the home page, got:\n%s", body)
	}
}

// TestNavbarRendersMobileBurgerMenu checks that every page ships the no-JS
// <details> burger menu (left side, see templates/navbar.html) with the
// same three destinations as the inline nav, so it works even before the
// CSS breakpoint that actually reveals it on narrow viewports.
func TestNavbarRendersMobileBurgerMenu(t *testing.T) {
	ts, ok := parseTestTemplates(t)["home.html"]
	if !ok {
		t.Fatal(`template "home.html" not found`)
	}

	data := HomeData{PageData: PageData{Title: "Home"}}

	var buf bytes.Buffer
	if err := ts.Execute(&buf, data); err != nil {
		t.Fatalf("execute home.html: %v", err)
	}
	body := buf.String()

	for _, want := range []string{
		`class="wfd-mobile-nav"`,
		`id="mobile-nav-menu"`,
		`class="wfd-mobile-nav__panel"`,
		`aria-label="Menu"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered body", want)
		}
	}

	// Each destination must appear twice: once in the always-in-the-DOM
	// inline nav, once inside the burger dropdown. Match on the nav-link
	// class + href together (not a bare href) since the home page's
	// dashboard tiles *also* link to /pantry and /recipes.
	for _, href := range []string{`href="/pantry"`, `href="/recipes"`, `href="/ingredients"`} {
		want := `class="wfd-nav-link" ` + href
		if got := strings.Count(body, want); got != 2 {
			t.Errorf("expected %q to appear twice (inline nav + burger menu), got %d", want, got)
		}
	}
}
