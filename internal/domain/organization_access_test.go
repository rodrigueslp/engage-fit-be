package domain

import "testing"

func TestNormalizeUnitName(t *testing.T) {
	name, key, ok := NormalizeUnitName("  Unidade\tCentro  ")
	if !ok || name != "Unidade Centro" || key != "unidade centro" {
		t.Fatalf("NormalizeUnitName = %q, %q, %v", name, key, ok)
	}
	if _, _, ok := NormalizeUnitName(" \n\t "); ok {
		t.Fatal("expected blank unit name to be rejected")
	}
}
