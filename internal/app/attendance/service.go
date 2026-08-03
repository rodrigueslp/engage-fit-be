package attendance

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"
	"unicode"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"golang.org/x/text/unicode/norm"
)

const selfCheckinSessionDuration = 10 * time.Minute

var (
	ErrInvalidInput       = errors.New("invalid attendance input")
	ErrSessionUnavailable = errors.New("self checkin session unavailable")
	ErrStudentNotFound    = errors.New("box member not found")
)

type Service struct {
	repository  repositories.AttendanceRepository
	campaigns   activeCampaignRepository
	recalculate campaignRecalculator
	now         func() time.Time
	location    *time.Location
}

type activeCampaignRepository interface {
	ListActive(ctx context.Context, boxID domain.ID) ([]domain.Campaign, error)
}

type campaignRecalculator interface {
	Execute(ctx context.Context, boxID, campaignID domain.ID) error
}

type SessionResult struct {
	Token     string
	BoxName   string
	ExpiresAt time.Time
}

type CheckinResult struct {
	StudentID       domain.ID
	StudentName     string
	CheckinDate     time.Time
	AlreadyRecorded bool
}

type ManualCheckinInput struct {
	BoxID     domain.ID
	StudentID domain.ID
	UserID    domain.ID
	Date      time.Time
}

type SelfCheckinInput struct {
	Token string
	Name  string
	Phone string
}

func NewService(repository repositories.AttendanceRepository, campaignRepository activeCampaignRepository, recalculate campaignRecalculator) *Service {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		location = time.UTC
	}
	return &Service{repository: repository, campaigns: campaignRepository, recalculate: recalculate, now: time.Now, location: location}
}

func (s *Service) CreateSession(ctx context.Context, boxID, userID domain.ID) (*SessionResult, error) {
	if boxID == "" || userID == "" {
		return nil, ErrInvalidInput
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	now := s.now().UTC()
	session := domain.SelfCheckinSession{
		BoxID: boxID, CreatedByUserID: userID, TokenHash: hashToken(token),
		ExpiresAt: now.Add(selfCheckinSessionDuration), CreatedAt: now,
	}
	if err := s.repository.SaveSelfCheckinSession(ctx, &session); err != nil {
		return nil, err
	}
	return &SessionResult{Token: token, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) PublicSession(ctx context.Context, token string) (*SessionResult, error) {
	if !validToken(token) {
		return nil, ErrSessionUnavailable
	}
	session, boxName, err := s.repository.FindValidSelfCheckinSession(ctx, hashToken(token), s.now().UTC())
	if err != nil {
		return nil, ErrSessionUnavailable
	}
	return &SessionResult{BoxName: boxName, ExpiresAt: session.ExpiresAt}, nil
}

func (s *Service) ManualCheckin(ctx context.Context, input ManualCheckinInput) (*CheckinResult, error) {
	if input.BoxID == "" || input.StudentID == "" || input.UserID == "" || input.Date.IsZero() {
		return nil, ErrInvalidInput
	}
	now := s.now().In(s.location)
	date := time.Date(input.Date.Year(), input.Date.Month(), input.Date.Day(), 0, 0, 0, 0, s.location)
	if date.After(dateOnly(now, s.location)) || date.Before(dateOnly(now.AddDate(-1, 0, 0), s.location)) {
		return nil, ErrInvalidInput
	}
	checkinTime := now
	checkin := domain.Checkin{
		BoxID: input.BoxID, StudentID: input.StudentID, CheckinDate: date, CheckinTime: &checkinTime,
		Source: domain.SourceBoxMember, EntryMethod: domain.CheckinEntryManual,
		RecordedByUserID: input.UserID, CreatedAt: s.now().UTC(),
	}
	created, err := s.repository.SaveBoxMemberCheckin(ctx, &checkin)
	if err != nil {
		return nil, err
	}
	if err := s.recalculateCampaigns(ctx, input.BoxID, date); err != nil {
		return nil, err
	}
	return &CheckinResult{StudentID: input.StudentID, CheckinDate: date, AlreadyRecorded: !created}, nil
}

func (s *Service) SelfCheckin(ctx context.Context, input SelfCheckinInput) (*CheckinResult, error) {
	name := strings.TrimSpace(input.Name)
	phone := normalizePhone(input.Phone)
	if !validToken(input.Token) || len([]rune(name)) < 3 || len([]rune(name)) > 160 || phone == "" {
		return nil, ErrInvalidInput
	}
	now := s.now().UTC()
	session, _, err := s.repository.FindValidSelfCheckinSession(ctx, hashToken(input.Token), now)
	if err != nil {
		return nil, ErrSessionUnavailable
	}
	students, err := s.repository.FindActiveBoxMembersByPhone(ctx, session.BoxID, phone)
	if err != nil {
		return nil, err
	}
	claimedName := normalizeName(name)
	var student *domain.Student
	for index := range students {
		if normalizeName(students[index].Name) != claimedName {
			continue
		}
		if student != nil {
			return nil, ErrStudentNotFound
		}
		student = &students[index]
	}
	if student == nil {
		return nil, ErrStudentNotFound
	}
	localNow := now.In(s.location)
	date := dateOnly(localNow, s.location)
	checkinTime := localNow
	checkin := domain.Checkin{
		BoxID: session.BoxID, StudentID: student.ID, CheckinDate: date, CheckinTime: &checkinTime,
		Source: domain.SourceBoxMember, EntryMethod: domain.CheckinEntrySelfService,
		SelfCheckinSessionID: session.ID, CreatedAt: now,
	}
	created, err := s.repository.SaveBoxMemberCheckin(ctx, &checkin)
	if err != nil {
		return nil, err
	}
	if err := s.recalculateCampaigns(ctx, session.BoxID, date); err != nil {
		return nil, err
	}
	return &CheckinResult{StudentID: student.ID, StudentName: student.Name, CheckinDate: date, AlreadyRecorded: !created}, nil
}

func (s *Service) recalculateCampaigns(ctx context.Context, boxID domain.ID, date time.Time) error {
	items, err := s.campaigns.ListActive(ctx, boxID)
	if err != nil {
		return err
	}
	for _, campaign := range items {
		dateKey := date.Format("2006-01-02")
		if dateKey < campaign.StartDate.Format("2006-01-02") || dateKey > campaign.EndDate.Format("2006-01-02") {
			continue
		}
		if err := s.recalculate.Execute(ctx, boxID, campaign.ID); err != nil {
			return err
		}
	}
	return nil
}

func hashToken(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func validToken(value string) bool {
	if len(value) < 32 || len(value) > 80 {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil
}

func normalizePhone(value string) string {
	var digits strings.Builder
	for _, char := range strings.TrimSpace(value) {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}
	if digits.Len() < 10 || digits.Len() > 15 {
		return ""
	}
	normalized := digits.String()
	if len(normalized) == 10 || len(normalized) == 11 {
		normalized = "55" + normalized
	}
	return normalized
}

func normalizeName(value string) string {
	decomposed := norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var cleaned strings.Builder
	for _, char := range decomposed {
		switch {
		case unicode.Is(unicode.Mn, char):
			continue
		case unicode.IsLetter(char) || unicode.IsDigit(char):
			cleaned.WriteRune(char)
		default:
			cleaned.WriteRune(' ')
		}
	}
	fields := strings.Fields(cleaned.String())
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		if field == "da" || field == "das" || field == "de" || field == "do" || field == "dos" || field == "e" {
			continue
		}
		result = append(result, field)
	}
	return strings.Join(result, " ")
}

func dateOnly(value time.Time, location *time.Location) time.Time {
	year, month, day := value.In(location).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, location)
}
