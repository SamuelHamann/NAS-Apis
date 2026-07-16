// Package units converts cooking quantities between units of measurement —
// metric and imperial, volume and mass — so callers (currently
// internal/store's recipe-cooking logic) never have to assume a recipe's
// ingredient unit matches whatever unit a pantry happens to track that
// ingredient in.
package units

import "strings"

// Kind categorizes a unit as a volume or mass measurement. Conversion never
// crosses between kinds (a cup of flour has no meaningful gram equivalent
// without knowing the ingredient's density, which this package has no way
// to know) — Convert refuses instead of guessing.
type Kind int

const (
	KindUnknown Kind = iota
	KindVolume
	KindMass
)

// unitDef is one recognized unit name's Kind and its conversion factor into
// that Kind's base unit — milliliters for volume, grams for mass.
type unitDef struct {
	kind   Kind
	factor float64
}

// unitDefs lists every unit name/alias this package recognizes, in both
// metric and the (US) imperial/customary units common on recipes and
// grocery receipts. Keys are already lower-cased; look them up via
// lookup(), which also normalizes the input the same way.
var unitDefs = map[string]unitDef{
	// --- Volume: metric --------------------------------------------------
	"ml":          {KindVolume, 1},
	"milliliter":  {KindVolume, 1},
	"milliliters": {KindVolume, 1},
	"millilitre":  {KindVolume, 1},
	"millilitres": {KindVolume, 1},
	"l":           {KindVolume, 1000},
	"liter":       {KindVolume, 1000},
	"liters":      {KindVolume, 1000},
	"litre":       {KindVolume, 1000},
	"litres":      {KindVolume, 1000},

	// --- Volume: imperial/US customary (standard US cooking measures) ----
	"tsp":          {KindVolume, 4.92892},
	"teaspoon":     {KindVolume, 4.92892},
	"teaspoons":    {KindVolume, 4.92892},
	"tbsp":         {KindVolume, 14.7868},
	"tablespoon":   {KindVolume, 14.7868},
	"tablespoons":  {KindVolume, 14.7868},
	"fl oz":        {KindVolume, 29.5735},
	"floz":         {KindVolume, 29.5735},
	"fluid ounce":  {KindVolume, 29.5735},
	"fluid ounces": {KindVolume, 29.5735},
	"cup":          {KindVolume, 236.588},
	"cups":         {KindVolume, 236.588},
	"pt":           {KindVolume, 473.176},
	"pint":         {KindVolume, 473.176},
	"pints":        {KindVolume, 473.176},
	"qt":           {KindVolume, 946.353},
	"quart":        {KindVolume, 946.353},
	"quarts":       {KindVolume, 946.353},
	"gal":          {KindVolume, 3785.41},
	"gallon":       {KindVolume, 3785.41},
	"gallons":      {KindVolume, 3785.41},

	// --- Mass: metric ------------------------------------------------------
	"g":         {KindMass, 1},
	"gram":      {KindMass, 1},
	"grams":     {KindMass, 1},
	"gramme":    {KindMass, 1},
	"grammes":   {KindMass, 1},
	"kg":        {KindMass, 1000},
	"kilogram":  {KindMass, 1000},
	"kilograms": {KindMass, 1000},

	// --- Mass: imperial ------------------------------------------------
	"oz":     {KindMass, 28.3495},
	"ounce":  {KindMass, 28.3495},
	"ounces": {KindMass, 28.3495},
	"lb":     {KindMass, 453.592},
	"lbs":    {KindMass, 453.592},
	"pound":  {KindMass, 453.592},
	"pounds": {KindMass, 453.592},
}

// normalize lowercases and trims a unit name for map lookup.
func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// lookup resolves a unit name to its definition. Besides the exact names in
// unitDefs, it also tries stripping a trailing "s" (a plural of a unit
// that isn't already listed explicitly) before giving up.
func lookup(name string) (unitDef, bool) {
	name = normalize(name)
	if def, ok := unitDefs[name]; ok {
		return def, true
	}
	if singular, ok := strings.CutSuffix(name, "s"); ok {
		if def, ok := unitDefs[singular]; ok {
			return def, true
		}
	}
	return unitDef{}, false
}

// Convert converts quantity from fromUnit to toUnit, returning the
// converted quantity and true on success. It succeeds when:
//
//   - fromUnit and toUnit are the same unit (case-insensitively, e.g. "Cup"
//     and "cup") — always true, even for a unit this package doesn't
//     otherwise recognize (e.g. "bunch" to "bunch"), since no actual
//     conversion is needed; or
//   - both are recognized units (see unitDefs) of the same Kind — volume
//     only converts to volume, mass only to mass. A cup can become
//     milliliters; it never becomes grams, since that depends on the
//     ingredient's density, which this package doesn't know.
//
// Anything else — an unrecognized unit, or a cross-Kind conversion (e.g.
// cups to grams) — returns (0, false), leaving the fallback (if any) up to
// the caller.
func Convert(quantity float64, fromUnit, toUnit string) (float64, bool) {
	if normalize(fromUnit) == normalize(toUnit) {
		return quantity, true
	}

	from, ok := lookup(fromUnit)
	if !ok {
		return 0, false
	}
	to, ok := lookup(toUnit)
	if !ok {
		return 0, false
	}
	if from.kind != to.kind {
		return 0, false
	}
	return quantity * from.factor / to.factor, true
}
