package store

import "testing"

func TestSetThemePreferenceUpserts(t *testing.T) {
	st := openTestStore(t)

	if err := st.SetThemePreference("ai", ThemeStateFollow); err != nil {
		t.Fatalf("SetThemePreference follow: %v", err)
	}
	if err := st.SetThemePreference("ai", ThemeStateMute); err != nil {
		t.Fatalf("SetThemePreference mute: %v", err)
	}

	prefs, err := st.ListThemePreferences()
	if err != nil {
		t.Fatalf("ListThemePreferences: %v", err)
	}
	if got := prefs["ai"]; got != ThemeStateMute {
		t.Errorf("ai state = %q, want %q after upsert", got, ThemeStateMute)
	}
	if len(prefs) != 1 {
		t.Errorf("len(prefs) = %d, want 1", len(prefs))
	}
}

func TestDeleteThemePreferenceRemovesRow(t *testing.T) {
	st := openTestStore(t)

	if err := st.SetThemePreference("ai", ThemeStateFollow); err != nil {
		t.Fatalf("SetThemePreference: %v", err)
	}
	if err := st.DeleteThemePreference("ai"); err != nil {
		t.Fatalf("DeleteThemePreference: %v", err)
	}

	prefs, err := st.ListThemePreferences()
	if err != nil {
		t.Fatalf("ListThemePreferences: %v", err)
	}
	if len(prefs) != 0 {
		t.Errorf("len(prefs) = %d after delete, want 0", len(prefs))
	}
}

func TestSetThemePreferenceRejectsInvalidState(t *testing.T) {
	st := openTestStore(t)

	if err := st.SetThemePreference("ai", "maybe"); err == nil {
		t.Error("expected error for invalid state, got nil")
	}
}
