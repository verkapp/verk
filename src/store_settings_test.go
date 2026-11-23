package src

import "testing"

func TestSettings(t *testing.T) {
	st := newTestStore(t)
	defaults, err := st.getSettings()
	if err != nil {
		t.Fatalf("getSettings: %v", err)
	}
	if defaults.Accent != "lavande" || !defaults.CoachEnabled || defaults.CoachOnStart || !defaults.TimeTracking || !defaults.ShowMomentum || !defaults.ClaudeEnabled || defaults.ClaudePath != "" {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	want := Settings{Accent: "menthe", CoachEnabled: false, CoachOnStart: true, TimeTracking: false, ShowMomentum: false, ClaudePath: "/opt/claude"}
	if err := st.saveSettings(want); err != nil {
		t.Fatalf("saveSettings: %v", err)
	}
	got, err := st.getSettings()
	if err != nil {
		t.Fatalf("getSettings: %v", err)
	}
	if got != want {
		t.Fatalf("settings mismatch: got %+v want %+v", got, want)
	}

	if err := st.saveSettings(Settings{}); err != nil {
		t.Fatalf("save empty settings: %v", err)
	}
	got, _ = st.getSettings()
	if got.Accent != "lavande" {
		t.Fatalf("empty accent should fall back to lavande, got %q", got.Accent)
	}
}
