package units

import (
	"math"
	"testing"
)

// Tolerance is relative, not absolute, so it stays meaningful across wildly
// different magnitudes (a few ml vs a few kg) — 1e-4 is loose enough to
// absorb the rounding in unitDefs' own conversion factors (which are
// themselves rounded to 6 significant figures) while still catching a
// genuinely wrong factor, which would be off by orders of magnitude.
func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-4*math.Max(1, math.Abs(b))
}

func TestConvertMetricVolume(t *testing.T) {
	got, ok := Convert(2, "l", "ml")
	if !ok || !almostEqual(got, 2000) {
		t.Errorf("2 l -> ml: got (%v, %v), want (2000, true)", got, ok)
	}
}

func TestConvertMetricMass(t *testing.T) {
	got, ok := Convert(1500, "g", "kg")
	if !ok || !almostEqual(got, 1.5) {
		t.Errorf("1500 g -> kg: got (%v, %v), want (1.5, true)", got, ok)
	}
}

func TestConvertImperialToMetricVolume(t *testing.T) {
	// A US cup is ~236.588 ml.
	got, ok := Convert(1, "cup", "ml")
	if !ok || !almostEqual(got, 236.588) {
		t.Errorf("1 cup -> ml: got (%v, %v), want (~236.588, true)", got, ok)
	}
}

func TestConvertImperialToMetricMass(t *testing.T) {
	// 16 oz ~= 1 lb ~= 453.592 g.
	got, ok := Convert(16, "oz", "g")
	if !ok || !almostEqual(got, 453.592) {
		t.Errorf("16 oz -> g: got (%v, %v), want (~453.592, true)", got, ok)
	}
}

func TestConvertBetweenImperialUnitsSameKind(t *testing.T) {
	// 1 cup = 16 tbsp.
	got, ok := Convert(1, "cup", "tbsp")
	if !ok || !almostEqual(got, 16) {
		t.Errorf("1 cup -> tbsp: got (%v, %v), want (16, true)", got, ok)
	}
}

func TestConvertRefusesVolumeToMass(t *testing.T) {
	if _, ok := Convert(1, "cup", "g"); ok {
		t.Error("expected cup -> g to be refused (volume can't become mass)")
	}
	if _, ok := Convert(1, "kg", "l"); ok {
		t.Error("expected kg -> l to be refused (mass can't become volume)")
	}
}

func TestConvertRefusesUnknownUnits(t *testing.T) {
	if _, ok := Convert(1, "bunch", "g"); ok {
		t.Error("expected bunch -> g to be refused (bunch isn't a recognized unit)")
	}
	if _, ok := Convert(1, "g", "clove"); ok {
		t.Error("expected g -> clove to be refused (clove isn't a recognized unit)")
	}
}

func TestConvertSameUnitIsAlwaysIdentityEvenIfUnrecognized(t *testing.T) {
	got, ok := Convert(3, "bunch", "bunch")
	if !ok || got != 3 {
		t.Errorf("bunch -> bunch: got (%v, %v), want (3, true)", got, ok)
	}
	// Case-insensitive.
	got, ok = Convert(2, "Cup", "cup")
	if !ok || got != 2 {
		t.Errorf("Cup -> cup: got (%v, %v), want (2, true)", got, ok)
	}
}

func TestConvertHandlesPluralsAndCase(t *testing.T) {
	got, ok := Convert(3, "CUPS", "ML")
	if !ok || !almostEqual(got, 709.764) {
		t.Errorf("3 CUPS -> ML: got (%v, %v), want (~709.764, true)", got, ok)
	}
}

func TestConvertZeroQuantity(t *testing.T) {
	got, ok := Convert(0, "cup", "ml")
	if !ok || got != 0 {
		t.Errorf("0 cup -> ml: got (%v, %v), want (0, true)", got, ok)
	}
}
