package src

import "testing"

func TestParseSuggestionsJSON(t *testing.T) {
	out, err := parseSuggestions(`["Définir le résultat", "Lister les étapes", "Bloquer 30 minutes"]`)
	if err != nil {
		t.Fatalf("parseSuggestions: %v", err)
	}
	if len(out) != 3 || out[0] != "Définir le résultat" {
		t.Fatalf("unexpected suggestions: %#v", out)
	}
}

func TestParseSuggestionsFenced(t *testing.T) {
	out, err := parseSuggestions("```json\n[\"Écrire le fil rouge\", \"Répéter à voix haute\"]\n```")
	if err != nil {
		t.Fatalf("parseSuggestions: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("unexpected suggestions: %#v", out)
	}
}

func TestParseSuggestionsProse(t *testing.T) {
	out, err := parseSuggestions(`Voici une proposition : ["Première étape", "Deuxième étape"] — bonne journée !`)
	if err != nil {
		t.Fatalf("parseSuggestions: %v", err)
	}
	if len(out) != 2 || out[1] != "Deuxième étape" {
		t.Fatalf("unexpected suggestions: %#v", out)
	}
}

func TestParseSuggestionsFallbackLines(t *testing.T) {
	out, err := parseSuggestions("1. Définir le résultat\n- Lister les étapes\n• Vérifier")
	if err != nil {
		t.Fatalf("parseSuggestions: %v", err)
	}
	want := []string{"Définir le résultat", "Lister les étapes", "Vérifier"}
	if len(out) != len(want) {
		t.Fatalf("unexpected suggestions: %#v", out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("suggestion %d: got %q want %q", i, out[i], want[i])
		}
	}
}

func TestParseSuggestionsDedupes(t *testing.T) {
	out, err := parseSuggestions(`["Faire", "faire", "  Faire  ", "Autre"]`)
	if err != nil {
		t.Fatalf("parseSuggestions: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected duplicates to be removed: %#v", out)
	}
}

func TestParseSuggestionsEmpty(t *testing.T) {
	if _, err := parseSuggestions("   "); err == nil {
		t.Fatal("empty input should produce an error")
	}
}
