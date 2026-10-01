package repositories

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"boxengage/backend/internal/adapters/persistence/postgres"
	"boxengage/backend/internal/adapters/security"
	"boxengage/backend/internal/app/people"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
)

func TestPeopleRepositoryCreationIdempotencyDuplicatesPrivacyAndConcurrency(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	db, err := postgres.Open(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	boxRepository := NewBoxGormRepository(db)
	box := &domain.Box{Name: "People Tenant", Status: domain.BoxStatusActive, CreatedAt: now, UpdatedAt: now}
	owner := &domain.User{Name: "People Owner", Email: uuid.NewString() + "@example.test", PasswordHash: "hash", AuthVersion: 1, Role: domain.UserRoleOwner, Active: true, CreatedAt: now, UpdatedAt: now}
	if err := boxRepository.SaveWithOwner(context.Background(), box, owner); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		boxID := string(box.ID)
		organizationID := string(box.OrganizationID)
		_ = db.Exec("DELETE FROM privacy_audit_events WHERE box_id = ?", boxID).Error
		_ = db.Exec("DELETE FROM people_audit_events WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM person_identity_reviews WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM person_athlete_accounts WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM person_external_identities WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM person_creation_requests WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM person_addresses WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM athlete_student_links WHERE student_id IN (SELECT id FROM students WHERE box_id = ?)", boxID).Error
		_ = db.Exec("DELETE FROM athlete_box_memberships WHERE box_id = ?", boxID).Error
		_ = db.Exec("DELETE FROM athlete_accounts WHERE email LIKE 'people-identity-%@example.test'").Error
		_ = db.Exec("DELETE FROM students WHERE box_id = ?", boxID).Error
		_ = db.Exec("DELETE FROM person_unit_relationships WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM persons WHERE organization_id = ?", organizationID).Error
		_ = db.Exec("DELETE FROM users WHERE box_id = ?", boxID).Error
		_ = db.Exec("DELETE FROM boxes WHERE id = ?", boxID).Error
		_ = db.Exec("DELETE FROM organizations WHERE id = ?", organizationID).Error
	})

	cipherKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("e", 32)))
	cipher, err := security.NewSecretCipher("test", "test:"+cipherKey)
	if err != nil {
		t.Fatal(err)
	}
	lookupKey := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("h", 32)))
	repository := NewPeopleGormRepository(db)
	service, err := people.NewService(repository, boxRepository, cipher, lookupKey)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	input := people.CreateInput{Name: "Maria da Silva", Phone: "(11) 99999-9999", CPF: "529.982.247-25", Email: "maria@example.test", Address: &people.AddressInput{PostalCode: "01234-567", Street: "Rua Um", Number: "10", City: "São Paulo", State: "SP"}}
	created, replayed, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-001", input)
	if err != nil || replayed {
		t.Fatalf("create = %+v replayed=%v err=%v", created, replayed, err)
	}
	replayedResult, replayed, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-001", input)
	if err != nil || !replayed || replayedResult.PersonID != created.PersonID || replayedResult.StudentID != created.StudentID {
		t.Fatalf("replay = %+v replayed=%v err=%v", replayedResult, replayed, err)
	}
	changed := input
	changed.Name = "Outra Pessoa"
	if _, _, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-001", changed); !errors.Is(err, people.ErrIdempotencyKeyReused) {
		t.Fatalf("changed replay = %v", err)
	}
	if _, _, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-002", input); !errors.Is(err, people.ErrCPFConflict) {
		t.Fatalf("duplicate CPF = %v", err)
	}

	phoneDuplicate := people.CreateInput{Name: "Joana Silva", Phone: input.Phone, CPF: "168.995.350-09"}
	if _, _, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-003", phoneDuplicate); !errors.Is(err, people.ErrPossibleDuplicate) {
		t.Fatalf("unreviewed phone duplicate = %v", err)
	}
	phoneDuplicate.PhoneDuplicateReview = &people.PhoneDuplicateReviewInput{Decision: "create_separate", MatchedPersonID: string(created.PersonID), Reason: "Telefone compartilhado pela família"}
	separate, _, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-004", phoneDuplicate)
	if err != nil {
		t.Fatalf("reviewed phone duplicate: %v", err)
	}
	if separate.PersonID == created.PersonID {
		t.Fatal("phone review merged people")
	}
	listed, err := service.List(ctx, box.ID, box.ID, people.SearchInput{Query: "529.982.247-25"})
	if err != nil || len(listed) != 1 || listed[0].ID != created.PersonID || listed[0].CPFMasked != "***.***.***-25" {
		t.Fatalf("unified CPF search = %+v, %v", listed, err)
	}
	updated, err := service.Update(ctx, box.ID, box.ID, created.PersonID, owner.ID, people.UpdateInput{Revision: 1, Name: stringPointer("Maria Souza"), PreferredName: stringPointer("Mari"), Email: stringPointer("mari@example.test")})
	if err != nil || updated.Person.Name != "Maria Souza" || updated.Person.PreferredName != "Mari" || updated.Person.Revision != 2 {
		t.Fatalf("profile update = %+v, %v", updated, err)
	}
	reviewedUpdate, err := service.Update(ctx, box.ID, box.ID, created.PersonID, owner.ID, people.UpdateInput{
		Revision: 2,
		Phone:    stringPointer(input.Phone),
		PhoneReview: &people.PhoneDuplicateReviewInput{
			Decision:        "create_separate",
			MatchedPersonID: string(separate.PersonID),
			Reason:          "Telefone compartilhado pela família",
		},
	})
	if err != nil || reviewedUpdate.Person.Revision != 3 || len(reviewedUpdate.History) == 0 || reviewedUpdate.History[0].Metadata["phone_duplicate_reviewed"] != true {
		t.Fatalf("reviewed phone update = %+v, %v", reviewedUpdate, err)
	}
	if _, err := service.Update(ctx, box.ID, box.ID, created.PersonID, owner.ID, people.UpdateInput{Revision: 1, Name: stringPointer("Versão antiga")}); !errors.Is(err, people.ErrPersonRevisionConflict) {
		t.Fatalf("stale profile update = %v", err)
	}
	withIdentity, err := service.LinkExternalIdentity(ctx, box.ID, box.ID, created.PersonID, owner.ID, people.ExternalIdentityInput{Provider: "wellhub", ExternalID: "member-verified-001", Verified: true})
	if err != nil || len(withIdentity.Identities) != 1 || withIdentity.Identities[0].ExternalID != "member-verified-001" {
		t.Fatalf("external identity = %+v, %v", withIdentity, err)
	}
	athleteID, athleteMembershipID := uuid.NewString(), uuid.NewString()
	if err := db.Exec("INSERT INTO athlete_accounts (id, name, email, password_hash) VALUES (?, 'Maria', ?, 'hash')", athleteID, "people-identity-"+uuid.NewString()+"@example.test").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO athlete_box_memberships (id, athlete_account_id, box_id) VALUES (?, ?, ?)", athleteMembershipID, athleteID, string(box.ID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO athlete_student_links (id, membership_id, student_id, link_method) VALUES (?, ?, ?, 'manual_review')", uuid.NewString(), athleteMembershipID, string(created.StudentID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.ReconcileAthleteAccount(ctx, box.ID, created.StudentID, domain.ID(athleteID), now); err != nil {
		t.Fatal(err)
	}
	withAthlete, err := repository.Get(ctx, box.OrganizationID, box.ID, created.PersonID)
	if err != nil || withAthlete.AthleteAccount == nil || withAthlete.AthleteAccount.AthleteAccountID != domain.ID(athleteID) {
		t.Fatalf("athlete person link = %+v, %v", withAthlete, err)
	}
	mergeSource, _, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-merge", people.CreateInput{Name: "Perfil duplicado", Phone: "(31) 98888-0000", CPF: "123.456.789-09"})
	if err != nil {
		t.Fatal(err)
	}
	merged, err := service.Merge(ctx, box.ID, box.ID, separate.PersonID, owner.ID, people.MergeInput{SourcePersonID: string(mergeSource.PersonID), Reason: "Duplicidade confirmada pela revisão humana"})
	if err != nil || len(merged.Profiles) != 2 {
		t.Fatalf("safe merge = %+v, %v", merged, err)
	}
	var mergedInto *string
	if err := db.Table("persons").Select("merged_into_person_id").Where("id = ?", string(mergeSource.PersonID)).Row().Scan(&mergedInto); err != nil || mergedInto == nil || *mergedInto != string(separate.PersonID) {
		t.Fatalf("merged source = %v, %v", mergedInto, err)
	}
	staleReview := people.CreateInput{
		Name:  "Revisão Antiga",
		Phone: "(41) 98888-0000",
		CPF:   "390.533.447-05",
		PhoneDuplicateReview: &people.PhoneDuplicateReviewInput{
			Decision:        "create_separate",
			MatchedPersonID: string(created.PersonID),
			Reason:          "Telefone foi alterado após a revisão",
		},
	}
	if _, _, err := service.Create(ctx, box.ID, box.ID, owner.ID, "people-intent-005", staleReview); !errors.Is(err, people.ErrPossibleDuplicate) {
		t.Fatalf("stale phone review = %v", err)
	}

	var personRows, studentRows, consentRows, auditRows int64
	if err := db.Table("persons").Where("organization_id = ?", string(box.OrganizationID)).Count(&personRows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("students").Where("box_id = ?", string(box.ID)).Count(&studentRows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("contact_consent_events").Where("box_id = ?", string(box.ID)).Count(&consentRows).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("people_audit_events").Where("person_id = ?", string(created.PersonID)).Count(&auditRows).Error; err != nil {
		t.Fatal(err)
	}
	if personRows != 3 || studentRows != 3 || consentRows != 0 || auditRows != 6 {
		t.Fatalf("atomic creation counts persons=%d students=%d consents=%d audits=%d", personRows, studentRows, consentRows, auditRows)
	}
	rollbackInput := people.CreateInput{Name: "Rollback Test", Phone: "(31) 98888-0000", CPF: "123.456.789-09", Address: &people.AddressInput{Street: "Rua que não deve persistir"}}
	if _, _, err := service.Create(ctx, box.ID, box.ID, domain.ID(uuid.NewString()), "people-rollback-001", rollbackInput); err == nil {
		t.Fatal("invalid audit/address actor should fail the transaction")
	}
	var rollbackPeople int64
	if err := db.Table("persons").Where("organization_id = ? AND name = ?", string(box.OrganizationID), "Rollback Test").Count(&rollbackPeople).Error; err != nil {
		t.Fatal(err)
	}
	if rollbackPeople != 0 {
		t.Fatalf("failed transaction leaked %d people", rollbackPeople)
	}

	privacy := NewPrivacyGormRepository(db, cipher)
	exported, err := privacy.ExportStudent(ctx, box.ID, created.StudentID)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Person == nil || exported.Person.CPF != "52998224725" || exported.Person.Address == nil || exported.Person.Address.PostalCode != "01234567" || len(exported.Person.Identities) != 1 || exported.Person.AthleteAccount == nil {
		t.Fatalf("privacy export = %+v", exported.Person)
	}
	if err := privacy.AnonymizeStudent(ctx, box.ID, created.StudentID, owner.ID, "Solicitação do titular"); err != nil {
		t.Fatal(err)
	}
	var status string
	var encryptedCPF, phone *string
	if err := db.Table("persons").Select("profile_status, cpf_encrypted, phone_e164").Where("id = ?", string(created.PersonID)).Row().Scan(&status, &encryptedCPF, &phone); err != nil {
		t.Fatal(err)
	}
	var addresses, identities, athleteLinks int64
	if err := db.Table("person_addresses").Where("person_id = ?", string(created.PersonID)).Count(&addresses).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("person_external_identities").Where("person_id = ?", string(created.PersonID)).Count(&identities).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Table("person_athlete_accounts").Where("person_id = ?", string(created.PersonID)).Count(&athleteLinks).Error; err != nil {
		t.Fatal(err)
	}
	if status != "anonymized" || encryptedCPF != nil || phone != nil || addresses != 0 || identities != 0 || athleteLinks != 0 {
		t.Fatalf("person anonymization status=%s cpf=%v phone=%v addresses=%d identities=%d athlete_links=%d", status, encryptedCPF, phone, addresses, identities, athleteLinks)
	}

	concurrentInput := people.CreateInput{Name: "Concorrente", Phone: "(21) 98888-7777", CPF: "111.444.777-35"}
	type outcome struct{ err error }
	outcomes := make(chan outcome, 2)
	var wait sync.WaitGroup
	for _, key := range []string{"people-race-001", "people-race-002"} {
		wait.Add(1)
		go func(idempotencyKey string) {
			defer wait.Done()
			_, _, createErr := service.Create(ctx, box.ID, box.ID, owner.ID, idempotencyKey, concurrentInput)
			outcomes <- outcome{err: createErr}
		}(key)
	}
	wait.Wait()
	close(outcomes)
	successes, conflicts := 0, 0
	for outcome := range outcomes {
		if outcome.err == nil {
			successes++
		} else if errors.Is(outcome.err, people.ErrCPFConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected race outcome: %v", outcome.err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("race outcomes success=%d conflict=%d", successes, conflicts)
	}
}
