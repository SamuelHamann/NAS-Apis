package handlers

import (
	"sort"
	"time"

	"github.com/SamuelHamann/NAS-Apis/whatsfordinner/internal/store"
)

// PantrySort chooses how ListPantryDetailed rows are grouped and ordered on
// the /pantry page.
type PantrySort string

const (
	// PantrySortExpiration groups items into three cards based on how close
	// their expiration_date is: expired / expiring soon / other. Within each
	// card items are sorted by expiration ASC (nulls last), then by name.
	PantrySortExpiration PantrySort = "expiration"
	// PantrySortAlphabetical puts every item into a single card sorted by
	// ingredient name. Secondary sort is expiration.
	PantrySortAlphabetical PantrySort = "alphabetical"
	// PantrySortLocation groups items by food_location (one card per
	// location, plus "Unassigned"). Within each card items are sorted by
	// expiration then name.
	PantrySortLocation PantrySort = "location"
)

// parsePantrySort returns the sort matching s, defaulting to expiration
// when s is empty or unknown so a bad query string never breaks the page.
func parsePantrySort(s string) PantrySort {
	switch PantrySort(s) {
	case PantrySortAlphabetical:
		return PantrySortAlphabetical
	case PantrySortLocation:
		return PantrySortLocation
	default:
		return PantrySortExpiration
	}
}

// ExpirationClass tags an item with how close to expiration it is, which
// drives the row's background color in templates/pantry.html.
type ExpirationClass string

const (
	ExpirationExpired ExpirationClass = "expired" // red
	ExpirationSoon    ExpirationClass = "soon"    // orange
	ExpirationOther   ExpirationClass = "other"   // default
)

// expiringSoonWindow is the "watch out — use this before it goes bad" window.
// Two days as per the pantry page spec.
const expiringSoonWindow = 2 * 24 * time.Hour

// classifyExpiration returns the ExpirationClass for exp given "today" (a
// midnight timestamp in the user's TZ; see startOfDay). Items with no
// expiration date always land in "other".
func classifyExpiration(exp *time.Time, today time.Time) ExpirationClass {
	if exp == nil {
		return ExpirationOther
	}
	if exp.Before(today) {
		return ExpirationExpired
	}
	if exp.Sub(today) <= expiringSoonWindow {
		return ExpirationSoon
	}
	return ExpirationOther
}

// startOfDay returns midnight of t in t's own location. Used so "expires
// today" doesn't flicker in/out of the "expired" bucket depending on the
// current hour.
func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// PantryItemView is one row on the pantry page. It is the store row plus a
// pre-computed ExpirationClass, so the template stays dumb ("apply this CSS
// class") and never has to do date math.
type PantryItemView struct {
	store.PantryDetailedRow
	ExpirationClass ExpirationClass
}

// PantryCard is one visual card on the pantry page (e.g. "Expired",
// "Fridge", or the whole list when sorted alphabetically). Cards are
// rendered in the order returned; empty cards are omitted by the handler.
type PantryCard struct {
	Title string
	// Tone is a hint for the template to color the whole card ("danger",
	// "warning" or "" for default). Only used by the expiration sort.
	Tone  string
	Items []PantryItemView
}

// GroupPantry turns the flat store rows into the ordered cards the pantry
// page renders. The grouping / secondary sort mirror what the store returns
// (expiration NULLS LAST, then name), but with the coarse buckets and
// alphabetical / location overrides applied on top.
//
// `now` is injected so tests can pin "today" and not race the clock.
func GroupPantry(rows []store.PantryDetailedRow, sortMode PantrySort, now time.Time) []PantryCard {
	today := startOfDay(now)

	views := make([]PantryItemView, len(rows))
	for i, r := range rows {
		views[i] = PantryItemView{
			PantryDetailedRow: r,
			ExpirationClass:   classifyExpiration(r.ExpirationDate, today),
		}
	}

	switch sortMode {
	case PantrySortAlphabetical:
		return groupAlphabetical(views)
	case PantrySortLocation:
		return groupByLocation(views)
	default:
		return groupByExpiration(views)
	}
}

// sortByExpirationThenName is the shared secondary sort: expiration date
// ascending, NULLs last, then ingredient name.
func sortByExpirationThenName(items []PantryItemView) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i].ExpirationDate, items[j].ExpirationDate
		switch {
		case a == nil && b == nil:
			return items[i].IngredientName < items[j].IngredientName
		case a == nil:
			return false // nil sorts last
		case b == nil:
			return true
		case !a.Equal(*b):
			return a.Before(*b)
		default:
			return items[i].IngredientName < items[j].IngredientName
		}
	})
}

func groupByExpiration(items []PantryItemView) []PantryCard {
	var expired, soon, other []PantryItemView
	for _, it := range items {
		switch it.ExpirationClass {
		case ExpirationExpired:
			expired = append(expired, it)
		case ExpirationSoon:
			soon = append(soon, it)
		default:
			other = append(other, it)
		}
	}

	sortByExpirationThenName(expired)
	sortByExpirationThenName(soon)
	sortByExpirationThenName(other)

	cards := make([]PantryCard, 0, 3)
	if len(expired) > 0 {
		cards = append(cards, PantryCard{Title: "Expired", Tone: "danger", Items: expired})
	}
	if len(soon) > 0 {
		cards = append(cards, PantryCard{Title: "Expiring in the next 2 days", Tone: "warning", Items: soon})
	}
	if len(other) > 0 {
		cards = append(cards, PantryCard{Title: "Everything else", Items: other})
	}
	return cards
}

func groupAlphabetical(items []PantryItemView) []PantryCard {
	sorted := append([]PantryItemView(nil), items...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].IngredientName != sorted[j].IngredientName {
			return sorted[i].IngredientName < sorted[j].IngredientName
		}
		// Secondary: expiration date (NULLs last), then id for a stable tiebreak.
		a, b := sorted[i].ExpirationDate, sorted[j].ExpirationDate
		switch {
		case a == nil && b == nil:
			return sorted[i].ID.String() < sorted[j].ID.String()
		case a == nil:
			return false
		case b == nil:
			return true
		default:
			return a.Before(*b)
		}
	})
	if len(sorted) == 0 {
		return nil
	}
	return []PantryCard{{Title: "All items", Items: sorted}}
}

// unassignedLocationTitle is the header for pantry items whose location_id
// is NULL. Kept as a constant so tests and the template stay in sync.
const unassignedLocationTitle = "Unassigned"

func groupByLocation(items []PantryItemView) []PantryCard {
	// Preserve first-seen order of real locations by walking `items` once
	// (they arrive from the DB sorted by expiration + name, giving each
	// location a deterministic initial position).
	type bucket struct {
		title string
		items []PantryItemView
	}

	buckets := make(map[string]*bucket) // key: location name or unassignedLocationTitle
	var order []string

	for _, it := range items {
		key := unassignedLocationTitle
		if it.LocationName != nil && *it.LocationName != "" {
			key = *it.LocationName
		}
		if _, ok := buckets[key]; !ok {
			buckets[key] = &bucket{title: key}
			order = append(order, key)
		}
		buckets[key].items = append(buckets[key].items, it)
	}

	// Named locations first (alphabetical for predictable UI), then
	// "Unassigned" pinned to the bottom.
	sort.Slice(order, func(i, j int) bool {
		ai, aj := order[i] == unassignedLocationTitle, order[j] == unassignedLocationTitle
		if ai != aj {
			return !ai // named ones (false) come before Unassigned (true)
		}
		return order[i] < order[j]
	})

	cards := make([]PantryCard, 0, len(order))
	for _, key := range order {
		sortByExpirationThenName(buckets[key].items)
		cards = append(cards, PantryCard{Title: buckets[key].title, Items: buckets[key].items})
	}
	return cards
}
