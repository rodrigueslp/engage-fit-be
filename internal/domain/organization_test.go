package domain

import "testing"

func TestOrganizationEffectiveStatus(t *testing.T) {
	if status := (Organization{}).EffectiveStatus(); status != OrganizationStatusActive {
		t.Fatalf("empty status = %q, want %q", status, OrganizationStatusActive)
	}
	if !(Organization{Status: OrganizationStatusActive}).IsActive() {
		t.Fatal("active organization should be active")
	}
	if (Organization{Status: OrganizationStatusSuspended}).IsActive() {
		t.Fatal("suspended organization should not be active")
	}
}
