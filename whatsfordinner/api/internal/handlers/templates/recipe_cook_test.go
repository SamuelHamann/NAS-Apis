package handlers

import "testing"

func TestParseCookForm(t *testing.T) {
	form := map[string][]string{
		"pantry_id":         {"7"},
		"multiplier":        {" 1.2 "},
		"create_combined":   {"on"},
		"combined_quantity": {" 1.2 "},
		"combined_unit_id":  {"14"},
		"confirmed":         {"true"},
	}

	state, msg := parseCookForm(form)
	if msg != "" {
		t.Fatalf("expected no validation message, got %q", msg)
	}
	if state.PantryID != 7 {
		t.Errorf("expected pantry_id 7, got %d", state.PantryID)
	}
	if state.Multiplier != 1.2 {
		t.Errorf("expected multiplier 1.2, got %v", state.Multiplier)
	}
	if state.MultiplierRaw != "1.2" {
		t.Errorf("expected trimmed raw multiplier %q, got %q", "1.2", state.MultiplierRaw)
	}
	if !state.CreateCombined {
		t.Error("expected CreateCombined true")
	}
	if !state.Confirmed {
		t.Error("expected Confirmed true")
	}
	if state.CombinedQuantity != 1.2 {
		t.Errorf("expected combined quantity 1.2, got %v", state.CombinedQuantity)
	}
	if state.CombinedUnitID == nil || *state.CombinedUnitID != 14 {
		t.Errorf("expected combined unit_id 14, got %v", state.CombinedUnitID)
	}
}

func TestParseCookFormRequiresCombinedQuantityWhenCreateCombinedChecked(t *testing.T) {
	form := map[string][]string{
		"pantry_id":       {"1"},
		"multiplier":      {"1"},
		"create_combined": {"on"},
	}
	if _, msg := parseCookForm(form); msg == "" {
		t.Error("expected a validation message for a missing combined ingredient amount")
	}
}

func TestParseCookFormIgnoresCombinedFieldsWhenUnchecked(t *testing.T) {
	form := map[string][]string{
		"pantry_id":         {"1"},
		"multiplier":        {"1"},
		"combined_quantity": {"not-a-number"},
		"combined_unit_id":  {"also-not-a-number"},
	}
	state, msg := parseCookForm(form)
	if msg != "" {
		t.Fatalf("expected no validation message when the checkbox isn't checked, got %q", msg)
	}
	if state.CombinedQuantity != 0 || state.CombinedUnitID != nil {
		t.Errorf("expected combined fields to stay zero-valued, got quantity=%v unit_id=%v", state.CombinedQuantity, state.CombinedUnitID)
	}
}

func TestParseCookFormRequiresPantry(t *testing.T) {
	form := map[string][]string{
		"multiplier": {"1"},
	}
	_, msg := parseCookForm(form)
	if msg == "" {
		t.Error("expected a validation message for a missing pantry")
	}
}

func TestParseCookFormRequiresPositiveMultiplier(t *testing.T) {
	for _, raw := range []string{"0", "-1", "not-a-number", ""} {
		form := map[string][]string{
			"pantry_id":  {"1"},
			"multiplier": {raw},
		}
		if _, msg := parseCookForm(form); msg == "" {
			t.Errorf("expected a validation message for multiplier %q", raw)
		}
	}
}

func TestParseCookFormCreateCombinedDefaultsFalse(t *testing.T) {
	form := map[string][]string{
		"pantry_id":  {"1"},
		"multiplier": {"1"},
	}
	state, msg := parseCookForm(form)
	if msg != "" {
		t.Fatalf("expected no validation message, got %q", msg)
	}
	if state.CreateCombined {
		t.Error("expected CreateCombined false when the checkbox isn't submitted")
	}
	if state.Confirmed {
		t.Error("expected Confirmed false when not submitted")
	}
}
