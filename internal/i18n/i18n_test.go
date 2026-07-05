package i18n

import "testing"

func TestCatalogParity(t *testing.T) {
	if err := ValidateCatalogs(); err != nil {
		t.Fatal(err)
	}
}

func TestDetectAndToggle(t *testing.T) {
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	translator := New("auto")
	if translator.Language != "de" {
		t.Fatalf("expected German, got %s", translator.Language)
	}
	translator.Toggle()
	if translator.Language != "en" {
		t.Fatalf("expected English, got %s", translator.Language)
	}
}
