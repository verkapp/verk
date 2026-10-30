package src

import "testing"

func TestCleanTitle(t *testing.T) {
	if got := cleanTitle("  un   titre\navec des   espaces "); got != "un titre avec des espaces" {
		t.Fatalf("unexpected title: %q", got)
	}
	long := ""
	for i := 0; i < 300; i++ {
		long += "a"
	}
	if got := cleanTitle(long); len([]rune(got)) != 240 {
		t.Fatalf("title should be clipped to 240 runes, got %d", len([]rune(got)))
	}
}
