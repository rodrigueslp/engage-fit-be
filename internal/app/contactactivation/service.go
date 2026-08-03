package contactactivation

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- Twilio's webhook signature protocol requires HMAC-SHA1.
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"golang.org/x/text/unicode/norm"
	"gorm.io/gorm"
)

const (
	ConsentVersion = "whatsapp-engagement-v1"
	ConsentText    = "Quero receber pelo WhatsApp da academia mensagens sobre meu progresso de check-ins, metas, brindes e lembretes de frequência. Posso cancelar gratuitamente a qualquer momento enviando SAIR."
)

var (
	ErrInvalidInput        = errors.New("invalid contact activation input")
	ErrWhatsappUnavailable = errors.New("whatsapp activation unavailable")
	ErrInvalidSignature    = errors.New("invalid twilio signature")
	ErrActivationExpired   = errors.New("contact activation expired")
	ErrUnsupportedMessage  = errors.New("unsupported inbound message")
	ErrReviewConflict      = errors.New("activation review conflicts with an existing student")
)

type SettingsResolver interface {
	Resolve(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, error)
}

type Service struct {
	repository repositories.ContactActivationRepository
	students   repositories.StudentRepository
	settings   SettingsResolver
	webhookURL string
	now        func() time.Time
}

type StartInput struct {
	ActivationCode    string
	Name              string
	Source            domain.Source
	RecentCheckinDate *time.Time
	IsNewStudent      bool
	ConsentAccepted   bool
}

type StartResult struct {
	WhatsappURL string
	ExpiresAt   time.Time
}

type InboundResult struct {
	Message string
}

type matchDecision struct {
	studentID domain.ID
	strategy  string
	pending   bool
}

func NewService(repository repositories.ContactActivationRepository, students repositories.StudentRepository, settings SettingsResolver, webhookURL string) *Service {
	return &Service{repository: repository, students: students, settings: settings, webhookURL: strings.TrimSpace(webhookURL), now: time.Now}
}

func (s *Service) PublicConfig(ctx context.Context, activationCode string) (*domain.ContactActivationConfig, error) {
	boxID, boxName, err := s.repository.FindPublicBox(ctx, activationCode)
	if err != nil {
		return nil, err
	}
	settings, sender, err := s.activationSettings(ctx, boxID)
	if err != nil {
		return nil, err
	}
	_ = settings
	return &domain.ContactActivationConfig{
		BoxID: boxID, BoxName: boxName, ActivationCode: activationCode, SenderPhone: sender,
		ConsentVersion: ConsentVersion, ConsentText: ConsentText,
	}, nil
}

func (s *Service) Start(ctx context.Context, input StartInput) (*StartResult, error) {
	name := strings.TrimSpace(input.Name)
	now := s.now().UTC()
	if len([]rune(name)) < 3 || len([]rune(name)) > 160 || !validSource(input.Source) || !input.ConsentAccepted {
		return nil, ErrInvalidInput
	}
	if input.IsNewStudent {
		input.RecentCheckinDate = nil
	} else if input.RecentCheckinDate == nil {
		return nil, ErrInvalidInput
	} else {
		checkinDate := input.RecentCheckinDate.UTC()
		if checkinDate.IsZero() || checkinDate.After(now) || checkinDate.Before(now.AddDate(-1, 0, 0)) {
			return nil, ErrInvalidInput
		}
	}
	boxID, _, err := s.repository.FindPublicBox(ctx, input.ActivationCode)
	if err != nil {
		return nil, err
	}
	_, sender, err := s.activationSettings(ctx, boxID)
	if err != nil {
		return nil, err
	}
	studentID := domain.ID("")
	matchStrategy := "new_student"
	if !input.IsNewStudent {
		matchData, matchErr := s.repository.FindActivationMatchData(ctx, boxID, input.Source, *input.RecentCheckinDate)
		if matchErr != nil {
			return nil, matchErr
		}
		decision := decideActivationMatch(name, *input.RecentCheckinDate, matchData)
		if input.Source == domain.SourceBoxMember && decision.pending {
			decision.pending = false
			decision.strategy = "no_matching_checkin"
		}
		studentID, matchStrategy = decision.studentID, decision.strategy
	}
	return s.createRequest(ctx, boxID, studentID, name, input.Source, input.RecentCheckinDate, input.IsNewStudent, matchStrategy, sender)
}

func (s *Service) StartForStudent(ctx context.Context, boxID, studentID domain.ID) (*StartResult, error) {
	student, err := s.students.FindByID(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	_, sender, err := s.activationSettings(ctx, boxID)
	if err != nil {
		return nil, err
	}
	return s.createRequest(ctx, boxID, student.ID, student.Name, student.Source, nil, false, "assisted_student", sender)
}

func (s *Service) AdminSummary(ctx context.Context, boxID domain.ID) (*domain.ContactActivationSummary, error) {
	summary, err := s.repository.Summary(ctx, boxID)
	if err != nil {
		return nil, err
	}
	code, err := s.repository.ActivationCode(ctx, boxID)
	if err != nil {
		return nil, err
	}
	summary.ActivationCode = code
	if _, sender, settingsErr := s.activationSettings(ctx, boxID); settingsErr == nil {
		summary.SenderPhone = sender
		summary.WhatsappReady = true
	}
	return &summary, nil
}

func (s *Service) List(ctx context.Context, boxID domain.ID) ([]domain.ContactActivationRequest, error) {
	items, err := s.repository.ListActivations(ctx, boxID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	for index := range items {
		if items[index].Status == domain.ContactActivationAwaitingMessage && !items[index].ExpiresAt.After(now) {
			items[index].Status = domain.ContactActivationExpired
		}
	}
	return items, nil
}

func (s *Service) Resolve(ctx context.Context, boxID, activationID, studentID domain.ID) (*domain.ContactActivationRequest, error) {
	if activationID == "" || studentID == "" {
		return nil, ErrInvalidInput
	}
	return s.repository.ResolveActivation(ctx, boxID, activationID, studentID, "manual_review", s.now().UTC())

}

func (s *Service) CreateStudentFromReview(ctx context.Context, boxID, activationID domain.ID) (*domain.ContactActivationRequest, error) {
	if boxID == "" || activationID == "" {
		return nil, ErrInvalidInput
	}
	item, err := s.repository.CreateStudentFromReview(ctx, boxID, activationID, s.now().UTC())
	if errors.Is(err, repositories.ErrContactActivationConflict) {
		return nil, ErrReviewConflict
	}
	return item, err
}

func (s *Service) CancelReview(ctx context.Context, boxID, activationID domain.ID) (*domain.ContactActivationRequest, error) {
	if boxID == "" || activationID == "" {
		return nil, ErrInvalidInput
	}
	return s.repository.CancelReview(ctx, boxID, activationID, s.now().UTC())
}

func (s *Service) ReprocessPending(ctx context.Context, boxID domain.ID, source domain.Source) error {
	items, err := s.repository.ListPendingSyncActivations(ctx, boxID, source)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.RecentCheckinDate == nil {
			if err := s.repository.MarkActivationNeedsReview(ctx, boxID, item.ID, "missing_checkin_date", s.now().UTC()); err != nil {
				return err
			}
			continue
		}
		matchData, matchErr := s.repository.FindActivationMatchData(ctx, boxID, source, *item.RecentCheckinDate)
		if matchErr != nil {
			return matchErr
		}
		decision := decideActivationMatch(item.ClaimedName, *item.RecentCheckinDate, matchData)
		if decision.studentID != "" {
			if _, resolveErr := s.repository.ResolveActivation(ctx, boxID, item.ID, decision.studentID, "after_import_"+decision.strategy, s.now().UTC()); resolveErr != nil {
				return resolveErr
			}
			continue
		}
		if !decision.pending {
			if err := s.repository.MarkActivationNeedsReview(ctx, boxID, item.ID, decision.strategy, s.now().UTC()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) HandleInbound(ctx context.Context, requestURL, signature string, values url.Values) (*InboundResult, error) {
	body := strings.TrimSpace(values.Get("Body"))
	from := normalizePhone(values.Get("From"))
	to := normalizePhone(values.Get("To"))
	if body == "" || from == "" || to == "" {
		return nil, ErrInvalidInput
	}
	if token := activationToken(body); token != "" {
		activation, err := s.repository.FindActivationByTokenHash(ctx, tokenHash(token))
		if err != nil {
			return nil, err
		}
		if err := s.validateSignature(ctx, activation.BoxID, requestURL, signature, values); err != nil {
			return nil, err
		}
		if activation.SenderPhone != to {
			return nil, ErrInvalidSignature
		}
		if !activation.ExpiresAt.After(s.now().UTC()) {
			return nil, ErrActivationExpired
		}
		confirmed, err := s.repository.ConfirmActivation(ctx, activation.ID, from, s.now().UTC())
		if err != nil {
			return nil, err
		}
		if confirmed.Status == domain.ContactActivationNeedsReview {
			return &InboundResult{Message: "Recebemos sua ativação. A academia vai conferir o vínculo e avisaremos quando estiver pronto."}, nil
		}
		if confirmed.Status == domain.ContactActivationPendingSync {
			return &InboundResult{Message: "Recebemos sua ativação. Seu check-in mais recente ainda será importado pela academia e o vínculo será conferido automaticamente."}, nil
		}
		name := firstName(confirmed.StudentName)
		if name == "" {
			name = "Tudo certo"
		}
		if confirmed.IsNewStudent {
			return &InboundResult{Message: name + ", seu cadastro e seu WhatsApp foram ativados. Você já faz parte da academia no EngageFit. Para cancelar mensagens, envie SAIR."}, nil
		}
		return &InboundResult{Message: name + ", seu WhatsApp foi ativado. Você receberá avisos sobre check-ins, metas e brindes. Para cancelar, envie SAIR."}, nil
	}

	if isOptOut(body) {
		activations, err := s.repository.FindActivationsByPhoneAndSender(ctx, from, to)
		if err != nil {
			return nil, err
		}
		if len(activations) == 0 {
			return nil, gorm.ErrRecordNotFound
		}
		if err := s.validateSignature(ctx, activations[0].BoxID, requestURL, signature, values); err != nil {
			return nil, err
		}
		ids := make([]domain.ID, 0, len(activations))
		for _, activation := range activations {
			if activation.Status != domain.ContactActivationCancelled {
				ids = append(ids, activation.ID)
			}
		}
		if len(ids) > 0 {
			if err := s.repository.OptOutActivations(ctx, ids, from, s.now().UTC()); err != nil {
				return nil, err
			}
		}
		return &InboundResult{Message: "Tudo certo. Você não receberá novas mensagens de engajamento. Se quiser voltar, faça uma nova ativação com a academia."}, nil
	}
	return nil, ErrUnsupportedMessage
}

func (s *Service) createRequest(ctx context.Context, boxID, studentID domain.ID, name string, source domain.Source, recentCheckinDate *time.Time, isNewStudent bool, matchStrategy, sender string) (*StartResult, error) {
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	activation := domain.ContactActivationRequest{
		BoxID: boxID, StudentID: studentID, ClaimedName: strings.TrimSpace(name), Source: source,
		RecentCheckinDate: recentCheckinDate, IsNewStudent: isNewStudent, SenderPhone: sender, TokenHash: tokenHash(token),
		MatchStrategy: matchStrategy, Status: domain.ContactActivationAwaitingMessage, ConsentVersion: ConsentVersion,
		ConsentText: ConsentText, ExpiresAt: now.Add(30 * time.Minute), CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repository.CreateActivation(ctx, &activation); err != nil {
		return nil, err
	}
	message := "Autorizo mensagens da academia sobre check-ins, metas, brindes e lembretes de frequência. Posso cancelar enviando SAIR. Código: EF-" + token
	return &StartResult{
		WhatsappURL: "https://wa.me/" + sender + "?text=" + url.QueryEscape(message),
		ExpiresAt:   activation.ExpiresAt,
	}, nil
}

func decideActivationMatch(claimedName string, recentCheckinDate time.Time, data domain.ContactActivationMatchData) matchDecision {
	claimed := activationNameTokens(claimedName)
	if len(claimed) < 2 {
		return matchDecision{strategy: "name_too_short"}
	}
	exact := make([]domain.ContactActivationCandidate, 0, 1)
	compatible := make([]domain.ContactActivationCandidate, 0, 1)
	for _, candidate := range data.Candidates {
		candidateTokens := activationNameTokens(candidate.Student.Name)
		if equalTokens(claimed, candidateTokens) {
			exact = append(exact, candidate)
		}
		if orderedSubset(claimed, candidateTokens) {
			compatible = append(compatible, candidate)
		}
	}
	if match := uniqueCandidateWithCheckin(exact); match != nil {
		return matchDecision{studentID: match.Student.ID, strategy: "exact_name_checkin"}
	}
	if len(exact) == 1 && data.LatestCheckinDate != nil && recentCheckinDate.After(dateOnly(*data.LatestCheckinDate)) {
		return matchDecision{studentID: exact[0].Student.ID, strategy: "exact_unique_source_lag"}
	}
	if match := uniqueCandidateWithCheckin(compatible); match != nil {
		return matchDecision{studentID: match.Student.ID, strategy: "compatible_name_checkin"}
	}
	if len(compatible) > 0 && (data.LatestCheckinDate == nil || recentCheckinDate.After(dateOnly(*data.LatestCheckinDate))) {
		return matchDecision{strategy: "awaiting_source_sync", pending: true}
	}
	if len(compatible) > 1 {
		return matchDecision{strategy: "ambiguous_name"}
	}
	return matchDecision{strategy: "no_matching_checkin"}
}

func uniqueCandidateWithCheckin(candidates []domain.ContactActivationCandidate) *domain.ContactActivationCandidate {
	var match *domain.ContactActivationCandidate
	for index := range candidates {
		if !candidates[index].HasRecentCheckin {
			continue
		}
		if match != nil {
			return nil
		}
		match = &candidates[index]
	}
	return match
}

func activationNameTokens(value string) []string {
	decomposed := norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var normalized strings.Builder
	for _, char := range decomposed {
		switch {
		case unicode.Is(unicode.Mn, char):
			continue
		case unicode.IsLetter(char) || unicode.IsDigit(char):
			normalized.WriteRune(char)
		default:
			normalized.WriteByte(' ')
		}
	}
	all := strings.Fields(normalized.String())
	result := make([]string, 0, len(all))
	for _, token := range all {
		if !activationNameParticle(token) {
			result = append(result, token)
		}
	}
	return result
}

func activationNameParticle(value string) bool {
	switch value {
	case "da", "das", "de", "do", "dos", "e":
		return true
	default:
		return false
	}
}

func equalTokens(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func orderedSubset(claimed, candidate []string) bool {
	if len(claimed) < 2 || len(claimed) > len(candidate) {
		return false
	}
	position := 0
	for _, token := range candidate {
		if token == claimed[position] {
			position++
			if position == len(claimed) {
				return true
			}
		}
	}
	return false
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func (s *Service) activationSettings(ctx context.Context, boxID domain.ID) (*domain.WhatsappSettings, string, error) {
	settings, err := s.settings.Resolve(ctx, boxID)
	if err != nil {
		return nil, "", ErrWhatsappUnavailable
	}
	if !settings.Enabled || strings.ToLower(strings.TrimSpace(settings.Provider)) != "twilio" {
		return nil, "", ErrWhatsappUnavailable
	}
	sender := normalizePhone(settings.InstanceName)
	if sender == "" || strings.HasPrefix(strings.ToUpper(strings.TrimSpace(settings.InstanceName)), "MG") {
		return nil, "", ErrWhatsappUnavailable
	}
	return settings, sender, nil
}

func (s *Service) validateSignature(ctx context.Context, boxID domain.ID, requestURL, signature string, values url.Values) error {
	settings, _, err := s.activationSettings(ctx, boxID)
	if err != nil {
		return err
	}
	_, authToken, ok := strings.Cut(settings.APIKeyEncrypted, ":")
	if !ok || strings.TrimSpace(authToken) == "" || strings.TrimSpace(signature) == "" {
		return ErrInvalidSignature
	}
	signingURL := requestURL
	if s.webhookURL != "" {
		signingURL = s.webhookURL
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	payload := signingURL
	for _, key := range keys {
		for _, value := range values[key] {
			payload += key + value
		}
	}
	mac := hmac.New(sha1.New, []byte(authToken))
	_, _ = mac.Write([]byte(payload))
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}
	return nil
}

func randomToken() (string, error) {
	bytes := make([]byte, 18)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func activationToken(body string) string {
	index := strings.Index(strings.ToUpper(body), "EF-")
	if index < 0 {
		return ""
	}
	value := body[index+3:]
	end := 0
	for end < len(value) {
		char := value[end]
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_') {
			break
		}
		end++
	}
	if end < 16 {
		return ""
	}
	return value[:end]
}

func normalizePhone(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "whatsapp:"))
	var digits strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			digits.WriteRune(char)
		}
	}
	if digits.Len() < 10 || digits.Len() > 15 {
		return ""
	}
	return digits.String()
}

func isOptOut(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SAIR", "PARAR", "CANCELAR", "STOP":
		return true
	default:
		return false
	}
}

func validSource(source domain.Source) bool {
	return source == domain.SourceWellhub || source == domain.SourceTotalPass || source == domain.SourceBoxMember
}

func firstName(value string) string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}
