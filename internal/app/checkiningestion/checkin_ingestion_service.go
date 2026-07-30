package checkiningestion

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	"boxengage/backend/internal/app/imports"
	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

var (
	ErrInvalidSource      = errors.New("invalid check-in ingestion source")
	ErrInvalidCredential  = errors.New("invalid check-in ingestion credential")
	ErrIngestionDisabled  = errors.New("check-in ingestion source disabled")
	ErrInvalidIdempotency = errors.New("invalid idempotency key")
)

var idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)

type Service struct {
	repository repositories.CheckinIngestionRepository
	imports    imports.ImportCheckinsUseCase
	now        func() time.Time
}

type CreatedSource struct {
	Source domain.CheckinIngestionSource
	Token  string
}

type IngestInput struct {
	SourceID       domain.ID
	Token          string
	IdempotencyKey string
	Filename       string
	File           io.Reader
}

type IngestResult struct {
	Batch  domain.CheckinIngestionBatch
	Replay bool
}

func NewService(repository repositories.CheckinIngestionRepository, importCheckins imports.ImportCheckinsUseCase) *Service {
	return &Service{repository: repository, imports: importCheckins, now: time.Now}
}

func (s Service) ListSources(ctx context.Context, boxID domain.ID) ([]domain.CheckinIngestionSource, error) {
	return s.repository.ListSources(ctx, boxID)
}

func (s Service) CreateSource(ctx context.Context, boxID, userID domain.ID, name string, source domain.Source) (*CreatedSource, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 || !validSource(source) {
		return nil, ErrInvalidSource
	}
	token, hash, err := newToken()
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	item := domain.CheckinIngestionSource{
		BoxID: boxID, CreatedByUserID: userID, Name: name, Source: source,
		TokenHash: hash, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repository.SaveSource(ctx, &item); err != nil {
		return nil, err
	}
	return &CreatedSource{Source: item, Token: token}, nil
}

func (s Service) RotateToken(ctx context.Context, boxID, id domain.ID) (*CreatedSource, error) {
	item, err := s.repository.FindSourceForBox(ctx, boxID, id)
	if err != nil {
		return nil, err
	}
	token, hash, err := newToken()
	if err != nil {
		return nil, err
	}
	item.TokenHash = hash
	item.UpdatedAt = s.now().UTC()
	if err := s.repository.UpdateSource(ctx, *item); err != nil {
		return nil, err
	}
	return &CreatedSource{Source: *item, Token: token}, nil
}

func (s Service) UpdateSource(ctx context.Context, boxID, id domain.ID, name string, enabled bool) (*domain.CheckinIngestionSource, error) {
	item, err := s.repository.FindSourceForBox(ctx, boxID, id)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return nil, ErrInvalidSource
	}
	item.Name, item.Enabled, item.UpdatedAt = name, enabled, s.now().UTC()
	if err := s.repository.UpdateSource(ctx, *item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s Service) Ingest(ctx context.Context, input IngestInput) (*IngestResult, error) {
	if !idempotencyPattern.MatchString(input.IdempotencyKey) {
		return nil, ErrInvalidIdempotency
	}
	source, err := s.repository.FindSource(ctx, input.SourceID)
	if err != nil {
		return nil, ErrInvalidCredential
	}
	if !source.Enabled {
		return nil, ErrIngestionDisabled
	}
	if !tokenMatches(input.Token, source.TokenHash) {
		return nil, ErrInvalidCredential
	}
	now := s.now().UTC()
	batch := domain.CheckinIngestionBatch{
		SourceID: source.ID, BoxID: source.BoxID, IdempotencyKey: input.IdempotencyKey,
		Status: "processing", CreatedAt: now,
	}
	claimed, existing, err := s.repository.ClaimBatch(ctx, &batch)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return &IngestResult{Batch: *existing, Replay: true}, nil
	}
	output, err := s.imports.Execute(ctx, imports.ImportCheckinsInput{
		BoxID: source.BoxID, Source: source.Source, Filename: input.Filename, File: input.File,
	})
	if err != nil {
		_ = s.repository.CompleteBatch(ctx, batch.ID, "failed", "", 0, 0, 0, s.now().UTC())
		return nil, err
	}
	completedAt := s.now().UTC()
	if err := s.repository.CompleteBatch(ctx, batch.ID, "completed", output.ImportID, output.TotalRecords, output.Students, output.Checkins, completedAt); err != nil {
		return nil, err
	}
	batch.Status, batch.ImportHistoryID = "completed", output.ImportID
	batch.TotalRecords, batch.StudentsCreated, batch.CheckinsCreated = output.TotalRecords, output.Students, output.Checkins
	batch.CompletedAt = &completedAt
	return &IngestResult{Batch: batch}, nil
}

func validSource(source domain.Source) bool {
	return source == domain.SourceWellhub || source == domain.SourceTotalPass
}

func newToken() (string, string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(value)
	return token, tokenHash(token), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokenMatches(token, expectedHash string) bool {
	actual := tokenHash(strings.TrimSpace(token))
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}
