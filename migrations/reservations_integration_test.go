package migrations

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	pgrepo "boxengage/backend/internal/adapters/persistence/postgres/repositories"
	"boxengage/backend/internal/app/memberships"
	reservationsapp "boxengage/backend/internal/app/reservations"
	schedulingapp "boxengage/backend/internal/app/scheduling"
	"boxengage/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type reservationTestContexts struct{ organizationID domain.ID }

func (r reservationTestContexts) ResolveSchedulingUnitContext(context.Context, domain.ID, domain.ID) (*domain.SchedulingUnitContext, error) {
	return &domain.SchedulingUnitContext{OrganizationID: r.organizationID, UnitID: r.organizationID, Role: "OWNER", Authorized: true}, nil
}

func TestReservationsPostgresConcurrencyAndPromotion(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	adminConfig, err := pgx.ParseConfig(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	adminDB := stdlib.OpenDB(*adminConfig)
	defer adminDB.Close()
	databaseName := "spec007_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = adminDB.ExecContext(ctx, "CREATE DATABASE "+databaseName); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		if _, dropErr := adminDB.ExecContext(cleanup, "DROP DATABASE IF EXISTS "+databaseName+" WITH (FORCE)"); dropErr != nil {
			t.Errorf("drop database: %v", dropErr)
		}
	}()
	testConfig := adminConfig.Copy()
	testConfig.Database = databaseName
	sqlDB := stdlib.OpenDB(*testConfig)
	if _, err = New(sqlDB).Up(ctx); err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	gormDB, err := gorm.Open(gormpostgres.New(gormpostgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	org := uuid.NewString()
	actor := uuid.NewString()
	modality := uuid.NewString()
	plan := uuid.NewString()
	version := uuid.NewString()
	entitlement := uuid.NewString()
	professional := uuid.NewString()
	session := uuid.NewString()
	personOne := uuid.NewString()
	personTwo := uuid.NewString()
	enrollmentOne := uuid.NewString()
	enrollmentTwo := uuid.NewString()
	starts := time.Date(2026, 9, 16, 18, 30, 0, 0, time.FixedZone("BRT", -3*3600)).UTC()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO organizations(id,name) VALUES (?, 'Academia')`, []any{org}},
		{`INSERT INTO boxes(id,organization_id,name,status,created_at,updated_at) VALUES (?,?, 'Centro','active',NOW(),NOW())`, []any{org, org}},
		{`INSERT INTO users(id,box_id,name,email,password_hash,role,auth_version,active,created_at,updated_at) VALUES (?,?, 'Owner',?,'hash','OWNER',1,true,NOW(),NOW())`, []any{actor, org, uuid.NewString() + "@example.test"}},
		{`INSERT INTO persons(id,organization_id,name,phone_e164) VALUES (?,?,'Ana Lima','+5511990000001'),(?,?,'Bia Luz','+5511990000002')`, []any{personOne, org, personTwo, org}},
		{`INSERT INTO person_unit_relationships(id,organization_id,person_id,unit_id,started_at) VALUES (?,?,?,?,NOW()),(?,?,?,?,NOW())`, []any{uuid.NewString(), org, personOne, org, uuid.NewString(), org, personTwo, org}},
		{`INSERT INTO modalities(id,organization_id,name,name_key) VALUES (?,?,'CrossFit','crossfit')`, []any{modality, org}},
		{`INSERT INTO unit_modalities(organization_id,unit_id,modality_id) VALUES (?,?,?)`, []any{org, org, modality}},
		{`INSERT INTO membership_plans(id,organization_id,home_unit_id,name,name_key) VALUES (?,?,?,'Mensal','mensal')`, []any{plan, org, org}},
		{`INSERT INTO membership_plan_versions(id,organization_id,plan_id,version_number,status,name,price_cents,currency,billing_cycle) VALUES (?,?,?,1,'draft','Mensal',20000,'BRL','monthly')`, []any{version, org, plan}},
		{`INSERT INTO membership_plan_entitlements(id,organization_id,plan_version_id,modality_id,access_type,quantity,booking_required) VALUES (?,?,?,?,'class_pack',10,true)`, []any{entitlement, org, version, modality}},
		{`UPDATE membership_plan_versions SET status='published',published_at=NOW() WHERE id=?`, []any{version}},
		{`INSERT INTO enrollments(id,organization_id,person_id,home_unit_id,plan_version_id,starts_on,created_by_user_id) VALUES (?,?,?,?,?,'2026-09-01',?),(?,?,?,?,?,'2026-09-01',?)`, []any{enrollmentOne, org, personOne, org, version, actor, enrollmentTwo, org, personTwo, org, version, actor}},
		{`INSERT INTO scheduling_professionals(id,organization_id,display_name,display_name_key,created_by_user_id,updated_by_user_id) VALUES (?,?,'Marina','marina',?,?)`, []any{professional, org, actor, actor}},
		{`INSERT INTO scheduling_professional_units(organization_id,unit_id,professional_id) VALUES (?,?,?)`, []any{org, org, professional}},
		{`INSERT INTO scheduling_sessions(id,organization_id,unit_id,origin,status,target_kind,target_id,target_name_snapshot,professional_id,professional_name_snapshot,delivery_mode,capacity,starts_at,ends_at,local_date,local_start_time,timezone,published_at,created_by_user_id,updated_by_user_id) VALUES (?,?,?,'extra','scheduled','modality',?,'CrossFit',?,'Marina','group',1,?,?,?,'18:30','America/Sao_Paulo',NOW(),?,?)`, []any{session, org, org, modality, professional, starts, starts.Add(time.Hour), "2026-09-16", actor, actor}},
	}
	for _, statement := range statements {
		if err = gormDB.Exec(statement.query, statement.args...).Error; err != nil {
			t.Fatalf("seed %s: %v", statement.query, err)
		}
	}
	repository := pgrepo.NewReservationGormRepository(gormDB)
	enrollmentRepo := pgrepo.NewEnrollmentGormRepository(gormDB)
	peopleRepo := pgrepo.NewPeopleGormRepository(gormDB)
	schedulingRepo := pgrepo.NewSchedulingGormRepository(gormDB)
	grid := schedulingapp.NewGridService(schedulingRepo, nil, nil, nil)
	tx := pgrepo.NewGormTransactionManager(gormDB)
	now := starts.Add(-time.Hour)
	access := memberships.NewAccessService(enrollmentRepo, pgrepo.NewMembershipAccessUsageGormAdapter(gormDB), pgrepo.NewAttendanceGormRepository(gormDB))
	service := reservationsapp.NewService(repository, grid, enrollmentRepo, peopleRepo, reservationTestContexts{organizationID: domain.ID(org)}, tx, reservationsapp.WithClock(func() time.Time { return now }), reservationsapp.WithAccessEvaluator(access))
	type result struct {
		value *domain.Reservation
		err   error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for index, person := range []string{personOne, personTwo} {
		wg.Add(1)
		go func(index int, person string) {
			defer wg.Done()
			value, _, bookErr := service.Book(ctx, domain.ID(actor), domain.ID(org), "book.concurrent."+string(rune('a'+index)), reservationsapp.BookInput{SessionID: domain.ID(session), PersonID: domain.ID(person)})
			results <- result{value, bookErr}
		}(index, person)
	}
	wg.Wait()
	close(results)
	var confirmed, waitlisted *domain.Reservation
	for item := range results {
		if item.err != nil {
			t.Fatal(item.err)
		}
		if item.value.Status == domain.ReservationConfirmed {
			confirmed = item.value
		} else if item.value.Status == domain.ReservationWaitlisted {
			waitlisted = item.value
		}
	}
	if confirmed == nil || waitlisted == nil {
		t.Fatalf("expected one confirmed and one waitlisted: confirmed=%+v waitlisted=%+v", confirmed, waitlisted)
	}
	_, promoted, _, err := service.Cancel(ctx, domain.ID(actor), domain.ID(org), confirmed.ID, "cancel.confirmed", reservationsapp.CancelInput{ExpectedRevision: 1, Reason: "mudança de horário"})
	if err != nil {
		t.Fatal(err)
	}
	if promoted == nil || promoted.ID != waitlisted.ID || promoted.Status != domain.ReservationConfirmed {
		t.Fatalf("unexpected promotion: %+v", promoted)
	}
	_, replayedPromotion, fresh, err := service.Cancel(ctx, domain.ID(actor), domain.ID(org), confirmed.ID, "cancel.confirmed", reservationsapp.CancelInput{ExpectedRevision: 1, Reason: "mudança de horário"})
	if err != nil || fresh || replayedPromotion == nil || replayedPromotion.ID != waitlisted.ID {
		t.Fatalf("stable cancellation replay failed: %+v fresh=%v err=%v", replayedPromotion, fresh, err)
	}
	if crossTenant, err := repository.Get(ctx, domain.ID(uuid.NewString()), domain.ID(org), promoted.ID, false); err == nil || crossTenant != nil {
		t.Fatalf("cross tenant lookup leaked reservation: %+v %v", crossTenant, err)
	}
	athleteID := uuid.NewString()
	if err := gormDB.Exec(`INSERT INTO athlete_accounts(id,name,email,password_hash,status,created_at,updated_at) VALUES (?, 'Ana Lima', ?, 'hash', 'active', NOW(), NOW())`, athleteID, uuid.NewString()+"@example.test").Error; err != nil {
		t.Fatal(err)
	}
	athleteScope := domain.AthletePortalScope{MembershipID: domain.ID(uuid.NewString()), OrganizationID: domain.ID(org), UnitID: domain.ID(org), PersonID: confirmed.PersonID}
	athleteReservation, _, err := service.BookForAthlete(ctx, athleteScope, domain.ID(athleteID), domain.ID(session), "athlete.book.0001")
	if err != nil || athleteReservation.Status != domain.ReservationWaitlisted {
		t.Fatalf("athlete booking: value=%+v err=%v", athleteReservation, err)
	}
	var audit struct {
		ActorAthleteAccountID *string
		ActorUserID           *string
	}
	if err := gormDB.Raw(`SELECT actor_athlete_account_id, actor_user_id FROM reservation_audit_events WHERE reservation_id = ? AND action = 'reservation.waitlisted' ORDER BY created_at DESC LIMIT 1`, string(athleteReservation.ID)).Scan(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if audit.ActorAthleteAccountID == nil || *audit.ActorAthleteAccountID != athleteID || audit.ActorUserID != nil {
		t.Fatalf("athlete audit: %+v", audit)
	}
}
