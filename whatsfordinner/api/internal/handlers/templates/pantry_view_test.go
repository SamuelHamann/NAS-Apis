package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse date %q: %v", s, err)
	}
	return d
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrString(s string) *string     { return &s }

// row is a shorthand for building a PantryDetailedRow in tests.
func row(name string, exp *time.Time, location *string) store.PantryDetailedRow {
	r := store.PantryDetailedRow{
		ID:             uuid.New(),
		IngredientName: name,
		ExpirationDate: exp,
		LocationName:   location,
		Quantity:       1,
		UnitName:       "u",
	}
	return r
}

func TestClassifyExpiration(t *testing.T) {
	today := mustDate(t, "2026-07-09")

	cases := []struct {
		name string
		exp  *time.Time
		want ExpirationClass
	}{
		{"nil expiration is 'other'", nil, ExpirationOther},
		{"yesterday is expired", ptrTime(mustDate(t, "2026-07-08")), ExpirationExpired},
		{"today is expiring soon (not expired)", ptrTime(today), ExpirationSoon},
		{"tomorrow is soon", ptrTime(mustDate(t, "2026-07-10")), ExpirationSoon},
		{"in 2 days is soon (inclusive)", ptrTime(mustDate(t, "2026-07-11")), ExpirationSoon},
		{"in 3 days is other (out of window)", ptrTime(mustDate(t, "2026-07-12")), ExpirationOther},
		{"far future is other", ptrTime(mustDate(t, "2027-01-01")), ExpirationOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyExpiration(tc.exp, today); got != tc.want {
				t.Errorf("classifyExpiration(%v) = %q, want %q", tc.exp, got, tc.want)
			}
		})
	}
}

func TestGroupByExpiration(t *testing.T) {
	now := mustDate(t, "2026-07-09")

	rows := []store.PantryDetailedRow{
		row("Milk", ptrTime(mustDate(t, "2026-07-05")), nil),   // expired
		row("Bread", ptrTime(mustDate(t, "2026-07-10")), nil),  // soon
		row("Eggs", ptrTime(mustDate(t, "2026-07-11")), nil),   // soon (2d out)
		row("Rice", ptrTime(mustDate(t, "2027-01-01")), nil),   // other
		row("Salt", nil, nil),                                  // other (no date)
		row("Yogurt", ptrTime(mustDate(t, "2026-07-04")), nil), // expired (older)
	}

	cards := GroupPantry(rows, PantrySortExpiration, now)

	if len(cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(cards))
	}
	if cards[0].Title != "Expired" || cards[0].Tone != "danger" {
		t.Errorf("card 0 = %+v, want Expired/danger", cards[0])
	}
	if cards[1].Title != "Expiring in the next 2 days" || cards[1].Tone != "warning" {
		t.Errorf("card 1 = %+v, want Expiring soon/warning", cards[1])
	}
	if cards[2].Title != "Everything else" || cards[2].Tone != "" {
		t.Errorf("card 2 = %+v, want Everything else / no tone", cards[2])
	}

	// Expired card is sorted by exp asc: Yogurt (Jul 4) then Milk (Jul 5).
	if names := itemNames(cards[0].Items); names[0] != "Yogurt" || names[1] != "Milk" {
		t.Errorf("expired card items = %v, want [Yogurt, Milk]", names)
	}
	// Soon card: Bread (Jul 10) then Eggs (Jul 11).
	if names := itemNames(cards[1].Items); names[0] != "Bread" || names[1] != "Eggs" {
		t.Errorf("soon card items = %v, want [Bread, Eggs]", names)
	}
	// Other card: Rice (has date) before Salt (no date, NULLs last).
	if names := itemNames(cards[2].Items); names[0] != "Rice" || names[1] != "Salt" {
		t.Errorf("other card items = %v, want [Rice, Salt]", names)
	}

	// Every item still carries its own ExpirationClass — the template relies
	// on this to colour rows individually inside the location grouping.
	for _, it := range cards[0].Items {
		if it.ExpirationClass != ExpirationExpired {
			t.Errorf("expired card item %q got class %q", it.IngredientName, it.ExpirationClass)
		}
	}
}

func TestGroupByExpirationOmitsEmptyCards(t *testing.T) {
	now := mustDate(t, "2026-07-09")
	rows := []store.PantryDetailedRow{
		row("Rice", ptrTime(mustDate(t, "2027-01-01")), nil), // other only
	}

	cards := GroupPantry(rows, PantrySortExpiration, now)
	if len(cards) != 1 || cards[0].Title != "Everything else" {
		t.Fatalf("expected single 'Everything else' card, got %+v", cards)
	}
}

func TestGroupAlphabetical(t *testing.T) {
	now := mustDate(t, "2026-07-09")
	rows := []store.PantryDetailedRow{
		row("Rice", ptrTime(mustDate(t, "2027-01-01")), nil),
		row("Apples", ptrTime(mustDate(t, "2026-07-05")), nil), // still tagged expired
		row("Milk", nil, nil),
	}

	cards := GroupPantry(rows, PantrySortAlphabetical, now)
	if len(cards) != 1 || cards[0].Title != "All items" {
		t.Fatalf("alphabetical sort should yield one 'All items' card, got %+v", cards)
	}
	if names := itemNames(cards[0].Items); names[0] != "Apples" || names[1] != "Milk" || names[2] != "Rice" {
		t.Errorf("alphabetical items = %v, want [Apples, Milk, Rice]", names)
	}
	// Colour class survives the switch of sort mode.
	if cards[0].Items[0].ExpirationClass != ExpirationExpired {
		t.Errorf("Apples should still be flagged expired in alphabetical mode, got %q", cards[0].Items[0].ExpirationClass)
	}
}

func TestGroupByLocation(t *testing.T) {
	now := mustDate(t, "2026-07-09")
	fridge := ptrString("Fridge")
	freezer := ptrString("Freezer")

	rows := []store.PantryDetailedRow{
		row("Milk", ptrTime(mustDate(t, "2026-07-05")), fridge),  // fridge, expired
		row("Peas", ptrTime(mustDate(t, "2027-01-01")), freezer), // freezer
		row("Ice", nil, freezer),                                 // freezer, no date
		row("Ketchup", nil, nil),                                 // unassigned
	}

	cards := GroupPantry(rows, PantrySortLocation, now)

	// Named locations first (alphabetical), then Unassigned.
	titles := make([]string, len(cards))
	for i, c := range cards {
		titles[i] = c.Title
	}
	want := []string{"Freezer", "Fridge", unassignedLocationTitle}
	for i := range want {
		if titles[i] != want[i] {
			t.Errorf("card %d title = %q, want %q (got %v)", i, titles[i], want[i], titles)
		}
	}

	// Freezer sorted by exp asc, NULLs last -> [Peas, Ice].
	if names := itemNames(cards[0].Items); names[0] != "Peas" || names[1] != "Ice" {
		t.Errorf("freezer items = %v, want [Peas, Ice]", names)
	}
	// Fridge single item — expired class preserved.
	if len(cards[1].Items) != 1 || cards[1].Items[0].ExpirationClass != ExpirationExpired {
		t.Errorf("fridge card = %+v, want [Milk expired]", cards[1].Items)
	}
}

func TestParsePantrySortFallback(t *testing.T) {
	for _, s := range []string{"", "wat", "invalid", "SORT"} {
		if got := parsePantrySort(s); got != PantrySortExpiration {
			t.Errorf("parsePantrySort(%q) = %q, want %q", s, got, PantrySortExpiration)
		}
	}
	for _, s := range []string{"alphabetical", "location", "expiration"} {
		if got := parsePantrySort(s); string(got) != s {
			t.Errorf("parsePantrySort(%q) = %q, want %q", s, got, s)
		}
	}
}

func itemNames(items []PantryItemView) []string {
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.IngredientName
	}
	return names
}
