package attendance

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
)

type attendanceRepositoryStub struct {
	session  domain.SelfCheckinSession
	students []domain.Student
	saved    *domain.Checkin
	created  bool
	phone    string
}

func (stub *attendanceRepositoryStub) SaveSelfCheckinSession(context.Context, *domain.SelfCheckinSession) error {
	return nil
}

func (stub *attendanceRepositoryStub) FindValidSelfCheckinSession(context.Context, string, time.Time) (*domain.SelfCheckinSession, string, error) {
	return &stub.session, "CrossFit Teste", nil
}

func (stub *attendanceRepositoryStub) FindActiveBoxMembersByPhone(_ context.Context, _ domain.ID, phone string) ([]domain.Student, error) {
	stub.phone = phone
	return stub.students, nil
}

func (stub *attendanceRepositoryStub) SaveBoxMemberCheckin(_ context.Context, checkin *domain.Checkin) (bool, error) {
	copy := *checkin
	stub.saved = &copy
	return stub.created, nil
}

type activeCampaignRepositoryStub struct{}

func (activeCampaignRepositoryStub) ListActive(context.Context, domain.ID) ([]domain.Campaign, error) {
	return nil, nil
}

type campaignRecalculatorStub struct{}

func (campaignRecalculatorStub) Execute(context.Context, domain.ID, domain.ID) error { return nil }

func TestSelfCheckinMatchesActivatedBoxMemberAndRecordsAttendance(t *testing.T) {
	token := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	repository := &attendanceRepositoryStub{
		session: domain.SelfCheckinSession{ID: "session-1", BoxID: "box-1", TokenHash: hashToken(token)},
		students: []domain.Student{{
			ID: "student-1", BoxID: "box-1", Name: "Maria de Ávila", Phone: "5511999999999",
			Source: domain.SourceBoxMember, ContactStatus: domain.ContactStatusOptedIn,
		}},
		created: true,
	}
	service := NewService(repository, activeCampaignRepositoryStub{}, campaignRecalculatorStub{})
	service.now = func() time.Time { return time.Date(2026, 8, 3, 15, 30, 0, 0, time.UTC) }

	result, err := service.SelfCheckin(context.Background(), SelfCheckinInput{
		Token: token, Name: "  MARIA AVILA ", Phone: "+55 (11) 99999-9999",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.StudentID != "student-1" || result.StudentName != "Maria de Ávila" || result.AlreadyRecorded {
		t.Fatalf("unexpected result: %+v", result)
	}
	if repository.phone != "5511999999999" {
		t.Fatalf("unexpected normalized phone: %s", repository.phone)
	}
	if repository.saved == nil || repository.saved.Source != domain.SourceBoxMember || repository.saved.EntryMethod != domain.CheckinEntrySelfService || repository.saved.SelfCheckinSessionID != "session-1" {
		t.Fatalf("unexpected checkin: %+v", repository.saved)
	}
	if got := repository.saved.CheckinDate.Format("2006-01-02"); got != "2026-08-03" {
		t.Fatalf("unexpected local checkin date: %s", got)
	}
}
