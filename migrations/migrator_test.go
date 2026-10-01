package migrations

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadOrdersAndChecksumsMigrations(t *testing.T) {
	files := fstest.MapFS{
		"002_second.sql": {Data: []byte("SELECT 2;")},
		"001_first.sql":  {Data: []byte("SELECT 1;")},
	}
	loaded, err := Load(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[0].Version != 1 || loaded[1].Version != 2 {
		t.Fatalf("unexpected migration order: %+v", loaded)
	}
	if loaded[0].Checksum == "" || loaded[0].Checksum == loaded[1].Checksum {
		t.Fatal("expected stable distinct checksums")
	}
}

func TestLoadRejectsSequenceGap(t *testing.T) {
	_, err := Load(fstest.MapFS{"002_second.sql": {Data: []byte("SELECT 2;")}})
	if err == nil {
		t.Fatal("expected a sequence gap error")
	}
}

func TestLoadRejectsInvalidFilename(t *testing.T) {
	_, err := Load(fstest.MapFS{"migration.sql": {Data: []byte("SELECT 1;")}})
	if err == nil {
		t.Fatal("expected an invalid filename error")
	}
}

func TestEmbeddedMigrationsIncludeOrganizationFoundation(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 53 {
			if migration.Name != "create_organizations" || !strings.Contains(migration.SQL, "ADD COLUMN organization_id") {
				t.Fatalf("unexpected organization migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("organization foundation migration not found")
}

func TestEmbeddedMigrationsIncludeMemberBilling(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 68 {
			if migration.Name != "create_member_billing" || !strings.Contains(migration.SQL, "CREATE TABLE member_billing_contracts") || !strings.Contains(migration.SQL, "CREATE TABLE member_payments") {
				t.Fatalf("unexpected member billing migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("member billing migration not found")
}

func TestEmbeddedMigrationsIncludeCompleteContractExperience(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 69 {
			if migration.Name != "complete_contract_experience" || !strings.Contains(migration.SQL, "CREATE TABLE health_form_submissions") || !strings.Contains(migration.SQL, "CREATE TABLE contract_signature_envelopes") || !strings.Contains(migration.SQL, "CREATE TABLE contract_legal_actions") {
				t.Fatalf("unexpected contract experience migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("complete contract experience migration not found")
}

func TestEmbeddedMigrationsIncludeHealthFormSessions(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 72 {
			if migration.Name != "create_health_form_sessions" || !strings.Contains(migration.SQL, "CREATE TABLE health_form_sessions") || !strings.Contains(migration.SQL, "token_hash CHAR(64)") || !strings.Contains(migration.SQL, "idx_health_form_session_active") {
				t.Fatalf("unexpected health form session migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("health form session migration not found")
}

func TestEmbeddedMigrationsIncludeUnitAccessControlMode(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 74 {
			if migration.Name != "add_unit_access_control_mode" || !strings.Contains(migration.SQL, "access_control_mode") || !strings.Contains(migration.SQL, "'disabled', 'manual', 'integrated'") {
				t.Fatalf("unexpected access control migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("unit access control migration not found")
}

func TestEmbeddedMigrationsIncludeAnnouncements(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 70 {
			if migration.Name != "create_announcements" || !strings.Contains(migration.SQL, "CREATE TABLE announcements") || !strings.Contains(migration.SQL, "CREATE TABLE announcement_recipients") {
				t.Fatalf("unexpected announcements migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("announcements migration not found")
}

func TestEmbeddedMigrationsIncludeCommerce(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 71 {
			if migration.Name != "create_commerce" || !strings.Contains(migration.SQL, "CREATE TABLE commerce_products") || !strings.Contains(migration.SQL, "CREATE TABLE commerce_sales") || !strings.Contains(migration.SQL, "enforce_commerce_sale_totals") {
				t.Fatalf("unexpected commerce migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("commerce migration not found")
}

func TestEmbeddedMigrationsIncludeCatalogStructure(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 54 {
			if migration.Name != "create_catalog_structure" || !strings.Contains(migration.SQL, "CREATE TABLE modalities") || !strings.Contains(migration.SQL, "CREATE TABLE spaces") {
				t.Fatalf("unexpected catalog migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("catalog structure migration not found")
}

func TestEmbeddedMigrationsIncludeServiceOfferings(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 55 {
			if migration.Name != "create_service_offerings" || !strings.Contains(migration.SQL, "CREATE TABLE service_offerings") || !strings.Contains(migration.SQL, "unit_service_offerings") {
				t.Fatalf("unexpected service offerings migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("service offerings migration not found")
}

func TestEmbeddedMigrationsIncludePeopleFoundation(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 56 {
			if migration.Name != "create_people_foundation" || !strings.Contains(migration.SQL, "CREATE TABLE persons") || !strings.Contains(migration.SQL, "UPDATE students SET person_id = id") {
				t.Fatalf("unexpected people migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("people foundation migration not found")
}

func TestEmbeddedMigrationsIncludeCompletedPeopleIdentity(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 63 {
			if migration.Name != "complete_people_identity" || !strings.Contains(migration.SQL, "CREATE TABLE person_external_identities") || !strings.Contains(migration.SQL, "CREATE TABLE person_identity_reviews") {
				t.Fatalf("unexpected completed people migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("completed people identity migration not found")
}

func TestEmbeddedMigrationsIncludeCompleteScheduling(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 64 {
			if migration.Name != "complete_scheduling" || !strings.Contains(migration.SQL, "CREATE TABLE scheduling_series") || !strings.Contains(migration.SQL, "CREATE TABLE scheduling_sessions") || !strings.Contains(migration.SQL, "CREATE TABLE scheduling_session_events") {
				t.Fatalf("unexpected scheduling migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("complete scheduling migration not found")
}

func TestEmbeddedMigrationsIncludeMembershipPlanCatalog(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 57 {
			if migration.Name != "create_membership_plan_catalog" || !strings.Contains(migration.SQL, "CREATE TABLE membership_plans") || !strings.Contains(migration.SQL, "prevent_membership_plan_version_mutation") {
				t.Fatalf("unexpected membership migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("membership plan catalog migration not found")
}

func TestEmbeddedMigrationsIncludePersonPlanEnrollments(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 58 {
			if migration.Name != "create_person_plan_enrollments" || !strings.Contains(migration.SQL, "CREATE TABLE enrollments") || !strings.Contains(migration.SQL, "idx_enrollments_one_open_per_person_unit") {
				t.Fatalf("unexpected enrollment migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("person plan enrollment migration not found")
}

func TestEmbeddedMigrationsIncludeEnrollmentWhatsAppInvariant(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 73 {
			if migration.Name != "require_enrollment_whatsapp" || !strings.Contains(migration.SQL, "enrollments_person_phone_required") || !strings.Contains(migration.SQL, "persons_open_enrollment_phone_required") {
				t.Fatalf("unexpected enrollment WhatsApp migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("enrollment WhatsApp migration not found")
}

func TestEmbeddedMigrationsIncludeMembershipAddonsAndComposition(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 59 {
			if migration.Name != "create_membership_addons_and_composition" || !strings.Contains(migration.SQL, "CREATE TABLE membership_plan_version_addons") || !strings.Contains(migration.SQL, "CREATE TABLE enrollment_addons") || !strings.Contains(migration.SQL, "prevent_published_plan_addon_mutation") || !strings.Contains(migration.SQL, "prevent_enrollment_addon_mutation") {
				t.Fatalf("unexpected addons migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("membership addons and composition migration not found")
}

func TestEmbeddedMigrationsIncludeMultiUnitAccess(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 62 {
			if migration.Name != "create_multi_unit_access" || !strings.Contains(migration.SQL, "CREATE TABLE organization_user_memberships") || !strings.Contains(migration.SQL, "CREATE TABLE user_unit_access") {
				t.Fatalf("unexpected multi-unit migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("multi-unit migration not found")
}

func TestEmbeddedMigrationsIncludeCompletedMembershipAccess(t *testing.T) {
	loaded, err := Load(Files)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range loaded {
		if migration.Version == 67 {
			if migration.Name != "complete_membership_access" || !strings.Contains(migration.SQL, "legacy_access_eligible") || !strings.Contains(migration.SQL, "remaining_usage") {
				t.Fatalf("unexpected membership access migration: %+v", migration)
			}
			return
		}
	}
	t.Fatal("completed membership access migration not found")
}
