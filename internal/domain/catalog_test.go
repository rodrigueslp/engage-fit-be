package domain

import (
	"strings"
	"testing"
)

func TestNormalizeCatalogName(t *testing.T) {
	name, key, ok := NormalizeCatalogName("  Cross\t  Training\n")
	if !ok || name != "Cross Training" || key != "cross training" {
		t.Fatalf("unexpected normalization: name=%q key=%q ok=%v", name, key, ok)
	}
	if _, _, ok := NormalizeCatalogName(" \n\t "); ok {
		t.Fatal("blank name should be rejected")
	}
	if _, _, ok := NormalizeCatalogName(strings.Repeat("á", 81)); ok {
		t.Fatal("name over 80 Unicode characters should be rejected")
	}
}

func TestNormalizeModalityColor(t *testing.T) {
	if color, ok := NormalizeModalityColor(""); !ok || color != DefaultModalityColor {
		t.Fatalf("unexpected default color: %q %v", color, ok)
	}
	if color, ok := NormalizeModalityColor("#a1b2c3"); !ok || color != "#A1B2C3" {
		t.Fatalf("unexpected normalized color: %q %v", color, ok)
	}
	for _, value := range []string{"0F766E", "#FFF", "#GG0000"} {
		if _, ok := NormalizeModalityColor(value); ok {
			t.Fatalf("invalid color %q was accepted", value)
		}
	}
}
