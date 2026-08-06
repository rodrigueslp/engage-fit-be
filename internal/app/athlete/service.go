package athlete

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

var (
	ErrInvalidInput       = errors.New("invalid athlete input")
	ErrInvalidCredentials = errors.New("invalid athlete credentials")
	ErrInvitationExpired  = errors.New("athlete invitation expired")
)

const (
	invitationLifetime = 7 * 24 * time.Hour
	sessionLifetime    = 30 * 24 * time.Hour
)

type Service struct {
	athletes  repositories.AthleteRepository
	students  repositories.StudentRepository
	passwords services.PasswordService
	mailer    services.AthleteAccountMailer
	explainer services.AthleteExplanationGenerator
	now       func() time.Time
}

type InvitationOutput struct {
	Token     string
	ExpiresAt time.Time
}

type ClaimInput struct {
	Token    string
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type SessionOutput struct {
	Token     string
	ExpiresAt time.Time
	Context   domain.AthleteContext
}

type ResultInput struct {
	WorkoutID   domain.ID
	Scale       string
	Entries     []domain.AthleteResultEntry
	RPE         int
	Notes       string
	PerformedAt time.Time
}

type ResultOutput struct {
	Result          domain.AthleteWorkoutResult
	PossibleRecords []domain.AthletePersonalRecord
}

func NewService(athletes repositories.AthleteRepository, students repositories.StudentRepository, passwords services.PasswordService) *Service {
	return &Service{athletes: athletes, students: students, passwords: passwords, now: time.Now}
}

func (s *Service) WithMailer(mailer services.AthleteAccountMailer) *Service {
	s.mailer = mailer
	return s
}
func (s *Service) WithExplainer(explainer services.AthleteExplanationGenerator) *Service {
	s.explainer = explainer
	return s
}

func (s *Service) CreateInvitation(ctx context.Context, boxID, studentID, actorUserID domain.ID) (*InvitationOutput, error) {
	student, err := s.students.FindByID(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	if student.AnonymizedAt != nil {
		return nil, ErrInvalidInput
	}
	token, tokenHash, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := s.now()
	invitation := domain.AthleteInvitation{BoxID: boxID, StudentID: studentID, TokenHash: tokenHash, CreatedByUserID: actorUserID, ExpiresAt: now.Add(invitationLifetime), CreatedAt: now}
	if err := s.athletes.SaveInvitation(ctx, &invitation); err != nil {
		return nil, err
	}
	return &InvitationOutput{Token: token, ExpiresAt: invitation.ExpiresAt}, nil
}

func (s *Service) PreviewInvitation(ctx context.Context, token string) (*domain.AthleteInvitation, error) {
	invitation, err := s.athletes.FindInvitationByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, ErrInvitationExpired
	}
	if invitation.ClaimedAt != nil || !invitation.ExpiresAt.After(s.now()) {
		return nil, ErrInvitationExpired
	}
	invitation.TokenHash = ""
	return invitation, nil
}

func (s *Service) ClaimInvitation(ctx context.Context, input ClaimInput) (*SessionOutput, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if name == "" || len([]rune(name)) > 160 || !validEmail(email) || len(input.Password) < 12 || len(input.Password) > 128 {
		return nil, ErrInvalidInput
	}
	invitation, err := s.PreviewInvitation(ctx, input.Token)
	if err != nil {
		return nil, err
	}

	existing, findErr := s.athletes.FindAccountByEmail(ctx, email)
	var account *domain.AthleteAccount
	var athleteID domain.ID
	if findErr == nil {
		if err := s.passwords.Compare(ctx, existing.PasswordHash, input.Password); err != nil {
			return nil, ErrInvalidCredentials
		}
		athleteID = existing.ID
	} else if errors.Is(findErr, repositories.ErrAthleteAccountNotFound) {
		passwordHash, err := s.passwords.Hash(ctx, input.Password)
		if err != nil {
			return nil, err
		}
		now := s.now()
		account = &domain.AthleteAccount{Name: name, Email: email, PasswordHash: passwordHash, Status: "active", CreatedAt: now, UpdatedAt: now}
	} else {
		return nil, findErr
	}

	if _, err := s.athletes.ClaimInvitation(ctx, invitation.ID, account, athleteID, s.now()); err != nil {
		if errors.Is(err, repositories.ErrAthleteInvitationUnavailable) {
			return nil, ErrInvitationExpired
		}
		return nil, err
	}
	if account != nil {
		athleteID = account.ID
	}
	return s.newSession(ctx, athleteID)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (*SessionOutput, error) {
	account, err := s.athletes.FindAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil || account.Status != "active" {
		return nil, ErrInvalidCredentials
	}
	if err := s.passwords.Compare(ctx, account.PasswordHash, input.Password); err != nil {
		return nil, ErrInvalidCredentials
	}
	return s.newSession(ctx, account.ID)
}

func (s *Service) Authenticate(ctx context.Context, token string) (*domain.AthleteContext, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidCredentials
	}
	return s.athletes.FindContextBySessionHash(ctx, hashToken(token), s.now())
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.athletes.RevokeSession(ctx, hashToken(token), s.now())
}

func (s *Service) Workouts(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkout, error) {
	items, err := s.athletes.ListPublishedWorkouts(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	results, err := s.athletes.ListWorkoutResults(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	records, err := s.athletes.ListPersonalRecords(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	byWorkout := make(map[domain.ID]domain.AthleteWorkoutResult, len(results))
	for _, result := range results {
		byWorkout[result.WorkoutID] = result
	}
	for i := range items {
		if result, ok := byWorkout[items[i].ID]; ok {
			copy := result
			items[i].Result = &copy
		}
		items[i].Personalization = personalizeWorkout(items[i], records, results)
	}
	return items, nil
}

func (s *Service) SaveResult(ctx context.Context, athleteID domain.ID, input ResultInput) (*ResultOutput, error) {
	workout, err := s.athletes.FindPublishedWorkout(ctx, athleteID, input.WorkoutID)
	if err != nil {
		return nil, err
	}
	if input.Scale != "rx" && input.Scale != "scaled" && input.Scale != "adapted" {
		return nil, ErrInvalidInput
	}
	if input.RPE < 0 || input.RPE > 10 || len([]rune(input.Notes)) > 2000 || len(input.Entries) == 0 || len(input.Entries) > 20 {
		return nil, ErrInvalidInput
	}
	for _, entry := range input.Entries {
		if !validResultEntry(entry) {
			return nil, ErrInvalidInput
		}
	}
	now := s.now()
	performedAt := input.PerformedAt
	if performedAt.IsZero() {
		performedAt = now
	}
	if performedAt.After(now.Add(24 * time.Hour)) {
		return nil, ErrInvalidInput
	}
	result := domain.AthleteWorkoutResult{AthleteID: athleteID, WorkoutID: workout.ID, MembershipID: workout.MembershipID, Scale: input.Scale, Entries: input.Entries, RPE: input.RPE, Notes: strings.TrimSpace(input.Notes), PerformedAt: performedAt, UpdatedAt: now}
	records, err := s.athletes.UpsertWorkoutResult(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &ResultOutput{Result: result, PossibleRecords: records}, nil
}

func (s *Service) Results(ctx context.Context, athleteID domain.ID) ([]domain.AthleteWorkoutResult, error) {
	return s.athletes.ListWorkoutResults(ctx, athleteID)
}
func (s *Service) PersonalRecords(ctx context.Context, athleteID domain.ID) ([]domain.AthletePersonalRecord, error) {
	return s.athletes.ListPersonalRecords(ctx, athleteID)
}
func (s *Service) ConfirmPersonalRecord(ctx context.Context, athleteID, recordID domain.ID) error {
	return s.athletes.ConfirmPersonalRecord(ctx, athleteID, recordID, s.now())
}

func (s *Service) ExplainWorkout(ctx context.Context, athleteID, workoutID domain.ID) (*domain.AthleteWorkoutInsight, error) {
	items, err := s.Workouts(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	var workout *domain.AthleteWorkout
	for i := range items {
		if items[i].ID == workoutID {
			workout = &items[i]
			break
		}
	}
	if workout == nil {
		return nil, ErrInvalidInput
	}
	account, err := s.athletes.FindAccountByID(ctx, athleteID)
	if err != nil {
		return nil, err
	}
	contextBytes, _ := json.Marshal(workout.Personalization)
	digest := sha256.Sum256(append([]byte(workout.RawText), contextBytes...))
	inputHash := hex.EncodeToString(digest[:])
	if cached, err := s.athletes.FindWorkoutInsight(ctx, athleteID, workoutID, inputHash); err == nil {
		return cached, nil
	}
	body := workout.Personalization.Summary + " " + workout.Personalization.Pacing
	provider, model := "rules", "rules-v1"
	if s.explainer != nil {
		output, generationErr := s.explainer.GenerateAthleteExplanation(ctx, services.AthleteExplanationInput{AthleteName: account.Name, BoxName: workout.BoxName, WorkoutText: workout.RawText, DeterministicContext: string(contextBytes)})
		if generationErr == nil && output != nil && strings.TrimSpace(output.Body) != "" {
			body, provider, model = strings.TrimSpace(output.Body), output.Provider, output.Model
		}
	}
	insight := &domain.AthleteWorkoutInsight{AthleteID: athleteID, WorkoutID: workoutID, InputHash: inputHash, Provider: provider, Model: model, Body: body, CreatedAt: s.now()}
	if err := s.athletes.SaveWorkoutInsight(ctx, insight); err != nil {
		return nil, err
	}
	return insight, nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	account, err := s.athletes.FindAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil
	}
	// Keep the public response generic even when delivery is unavailable, so the
	// endpoint cannot be used to discover registered athlete e-mails.
	_ = s.issueAccountToken(ctx, account, "reset_password", time.Hour)
	return nil
}
func (s *Service) RequestEmailVerification(ctx context.Context, athleteID domain.ID) error {
	account, err := s.athletes.FindAccountByID(ctx, athleteID)
	if err != nil {
		return err
	}
	if account.EmailVerifiedAt != nil {
		return nil
	}
	return s.issueAccountToken(ctx, account, "verify_email", 24*time.Hour)
}
func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if len(password) < 12 || len(password) > 128 {
		return ErrInvalidInput
	}
	account, err := s.athletes.ConsumeAccountToken(ctx, hashToken(token), "reset_password", s.now())
	if err != nil {
		return ErrInvitationExpired
	}
	hash, err := s.passwords.Hash(ctx, password)
	if err != nil {
		return err
	}
	return s.athletes.UpdateAthletePassword(ctx, account.ID, hash, s.now())
}
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	account, err := s.athletes.ConsumeAccountToken(ctx, hashToken(token), "verify_email", s.now())
	if err != nil {
		return ErrInvitationExpired
	}
	return s.athletes.VerifyAthleteEmail(ctx, account.ID, s.now())
}
func (s *Service) issueAccountToken(ctx context.Context, account *domain.AthleteAccount, purpose string, lifetime time.Duration) error {
	if s.mailer == nil {
		return errors.New("athlete account email delivery is not configured")
	}
	token, tokenHash, err := randomToken()
	if err != nil {
		return err
	}
	now := s.now()
	item := domain.AthleteAccountToken{AthleteID: account.ID, Purpose: purpose, TokenHash: tokenHash, ExpiresAt: now.Add(lifetime), CreatedAt: now}
	if err := s.athletes.SaveAccountToken(ctx, &item); err != nil {
		return err
	}
	return s.mailer.SendAthleteAccountLink(ctx, account.ID, account.Email, account.Name, purpose, token)
}

func (s *Service) newSession(ctx context.Context, athleteID domain.ID) (*SessionOutput, error) {
	token, tokenHash, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := s.now()
	session := domain.AthleteSession{AthleteID: athleteID, TokenHash: tokenHash, ExpiresAt: now.Add(sessionLifetime), CreatedAt: now}
	if err := s.athletes.SaveSession(ctx, &session); err != nil {
		return nil, err
	}
	context, err := s.athletes.FindContextBySessionHash(ctx, tokenHash, now)
	if err != nil {
		return nil, err
	}
	return &SessionOutput{Token: token, ExpiresAt: session.ExpiresAt, Context: *context}, nil
}

func validResultEntry(entry domain.AthleteResultEntry) bool {
	if entry.SectionIndex < 0 || entry.SectionIndex > 30 || len([]rune(entry.Movement)) > 160 {
		return false
	}
	if entry.TimeSeconds < 0 || entry.Rounds < 0 || entry.Repetitions < 0 || entry.LoadKG < 0 || entry.LoadKG > 1000 || entry.DistanceM < 0 || entry.Calories < 0 {
		return false
	}
	switch entry.ScoreType {
	case "time":
		return entry.TimeSeconds > 0
	case "rounds_reps":
		return entry.Rounds > 0 || entry.Repetitions > 0
	case "reps":
		return entry.Repetitions > 0
	case "load":
		return entry.LoadKG > 0
	case "distance":
		return entry.DistanceM > 0
	case "calories":
		return entry.Calories > 0
	case "completed":
		return entry.Completed
	default:
		return false
	}
}

var movementCleaner = regexp.MustCompile(`[^a-z0-9]+`)

func simpleMovementKey(value string) string {
	return strings.Trim(movementCleaner.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-"), "-")
}

func personalizeWorkout(workout domain.AthleteWorkout, records []domain.AthletePersonalRecord, results []domain.AthleteWorkoutResult) domain.AthletePersonalization {
	personalization := domain.AthletePersonalization{Summary: "Use seus resultados anteriores como referência e ajuste qualquer carga com o coach.", Pacing: pacingFor(workout.Classification.Formats), Guidance: []domain.AthleteGuidance{}, GeneratedBy: "rules-v1"}
	for _, movement := range workout.Classification.MovementMentions {
		key := simpleMovementKey(movement)
		for _, record := range records {
			if record.Status != "confirmed" || simpleMovementKey(record.MovementName) != key || record.Metric != "load" {
				continue
			}
			min, max := record.BestValue*.4, record.BestValue*.6
			for _, section := range workout.Classification.Sections {
				if section.Type == domain.WorkoutSectionStrength {
					min, max = record.BestValue*.7, record.BestValue*.85
					break
				}
			}
			personalization.Guidance = append(personalization.Guidance, domain.AthleteGuidance{Movement: movement, Message: fmt.Sprintf("Seu PR confirmado é %.1f kg. A faixa de %.1f–%.1f kg é apenas uma referência inicial; confirme a intenção com o coach.", record.BestValue, roundHalf(min), roundHalf(max)), ReferenceValue: record.BestValue, ReferenceUnit: "kg", SuggestedMin: roundHalf(min), SuggestedMax: roundHalf(max)})
		}
	}
	if len(results) > 0 && results[0].RPE >= 9 {
		personalization.Summary = "Seu último esforço registrado foi alto. Considere começar conservador e converse com o coach antes de aumentar a intensidade."
	}
	return personalization
}

func pacingFor(formats []string) string {
	for _, format := range formats {
		switch format {
		case "amrap":
			return "Comece em ritmo repetível, preserve transições curtas e acelere apenas no trecho final se a execução continuar estável."
		case "emom":
			return "Busque terminar cada minuto com uma margem consistente de descanso, sem sacrificar a técnica."
		case "for_time":
			return "Evite abrir no máximo. Divida as séries antes da falha e sustente um ritmo semelhante até a parte final."
		case "max_effort":
			return "Faça saltos graduais de carga, preserve tentativas e pare se a técnica se deteriorar."
		case "interval":
			return "Use o primeiro intervalo para calibrar um ritmo que consiga repetir nos seguintes."
		}
	}
	return "Priorize execução consistente e ajuste intensidade, escala e carga com o coach."
}

func roundHalf(value float64) float64 { return math.Round(value*2) / 2 }

func randomToken() (string, string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(value)
	return token, hashToken(token), nil
}

func hashToken(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:])
}

func validEmail(value string) bool {
	parts := strings.Split(value, "@")
	return len(parts) == 2 && parts[0] != "" && strings.Contains(parts[1], ".")
}
