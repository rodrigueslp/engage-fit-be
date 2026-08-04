package imports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/observability"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type ImportCheckinsInput struct {
	BoxID    domain.ID
	Source   domain.Source
	Filename string
	File     io.Reader
}

type ImportCheckinsOutput struct {
	ImportID     domain.ID
	TotalRecords int
	Students     int
	Checkins     int
}

type ImportCheckinsUseCase struct {
	parser       services.CheckinFileParser
	imports      repositories.ImportHistoryRepository
	students     repositories.StudentRepository
	checkins     repositories.CheckinRepository
	campaigns    repositories.CampaignRepository
	rewards      repositories.RewardRepository
	privacy      repositories.PrivacyRepository
	transactions repositories.TransactionManager
	activations  ContactActivationReprocessor
}

type ContactActivationReprocessor interface {
	ReprocessPending(ctx context.Context, boxID domain.ID, source domain.Source) error
}

func NewImportCheckinsUseCase(parser services.CheckinFileParser, imports repositories.ImportHistoryRepository, students repositories.StudentRepository, checkins repositories.CheckinRepository, campaigns repositories.CampaignRepository, rewards repositories.RewardRepository, privacy repositories.PrivacyRepository, transactions repositories.TransactionManager, activations ContactActivationReprocessor) ImportCheckinsUseCase {
	return ImportCheckinsUseCase{parser: parser, imports: imports, students: students, checkins: checkins, campaigns: campaigns, rewards: rewards, privacy: privacy, transactions: transactions, activations: activations}
}

func (uc ImportCheckinsUseCase) Execute(ctx context.Context, input ImportCheckinsInput) (output *ImportCheckinsOutput, resultErr error) {
	startedAt := time.Now()
	phase := "parse"
	defer func() {
		status, records, checkins := "failed", 0, 0
		if output != nil {
			records, checkins = output.TotalRecords, output.Checkins
		}
		if resultErr == nil {
			status = "success"
		}
		observability.RecordImport(ctx, string(input.Source), status, records, checkins, time.Since(startedAt))
	}()
	parsed, err := uc.parser.Parse(ctx, input.File, input.Source, input.Filename)
	if err != nil {
		logImportFailure(ctx, input, "", phase, len(parsed), err, startedAt)
		return nil, err
	}
	if uc.transactions == nil {
		err := errors.New("import transaction manager is not configured")
		logImportFailure(ctx, input, "", "transaction_setup", len(parsed), err, startedAt)
		return nil, err
	}

	now := time.Now()
	importHistory := domain.ImportHistory{
		BoxID:        input.BoxID,
		Filename:     input.Filename,
		Source:       input.Source,
		Status:       domain.ImportStatusProcessing,
		TotalRecords: len(parsed),
		ImportedAt:   now,
	}
	phase = "history_create"
	if err := uc.imports.Save(ctx, &importHistory); err != nil {
		logImportFailure(ctx, input, "", phase, len(parsed), err, startedAt)
		return nil, err
	}

	studentsCreated := 0
	checkins := make([]domain.Checkin, 0, len(parsed))
	insertedCheckins := 0
	transactionErr := uc.transactions.WithinTransaction(ctx, func(transactionContext context.Context) error {
		studentsByIdentity := make(map[string]*domain.Student)
		suppressedByIdentity := make(map[string]bool)

		phase = "student_processing"
		for _, parsedCheckin := range parsed {
			identity := studentIdentity(parsedCheckin)
			suppressed, checked := suppressedByIdentity[identity]
			if !checked {
				var err error
				suppressed, err = uc.privacy.IsIdentitySuppressed(transactionContext, input.BoxID, input.Source, identity)
				if err != nil {
					return err
				}
				suppressedByIdentity[identity] = suppressed
			}
			if suppressed {
				continue
			}

			student := studentsByIdentity[identity]
			if student == nil {
				var err error
				student, err = uc.students.FindByExternalID(transactionContext, input.BoxID, input.Source, identity)
				if err != nil {
					if !errors.Is(err, gorm.ErrRecordNotFound) {
						return err
					}
					student, err = uc.findSelfRegisteredStudent(transactionContext, input.BoxID, input.Source, parsedCheckin.StudentName)
					if err != nil {
						return err
					}
					if student != nil {
						student.ExternalID = identity
						student.Name = parsedCheckin.StudentName
						if student.Email == "" {
							student.Email = parsedCheckin.StudentEmail
						}
						if student.Phone == "" {
							student.Phone = parsedCheckin.StudentPhone
						}
						student.UpdatedAt = now
						if err := uc.students.Save(transactionContext, student); err != nil {
							return err
						}
					} else {
						student = &domain.Student{
							BoxID:                   input.BoxID,
							Name:                    parsedCheckin.StudentName,
							Email:                   parsedCheckin.StudentEmail,
							Phone:                   parsedCheckin.StudentPhone,
							Source:                  input.Source,
							ExternalID:              identity,
							MembershipStartedAt:     &parsedCheckin.CheckinDate,
							MembershipStartedSource: "first_checkin_inferred",
							CreatedAt:               now,
							UpdatedAt:               now,
						}
						if err := uc.students.Save(transactionContext, student); err != nil {
							return err
						}
						studentsCreated++
					}
				}
				studentsByIdentity[identity] = student
			}

			if student.MembershipStartedAt == nil || (student.MembershipStartedSource == "first_checkin_inferred" && parsedCheckin.CheckinDate.Before(*student.MembershipStartedAt)) {
				membershipStartedAt := parsedCheckin.CheckinDate
				student.MembershipStartedAt = &membershipStartedAt
				student.MembershipStartedSource = "first_checkin_inferred"
				student.UpdatedAt = now
				if err := uc.students.Save(transactionContext, student); err != nil {
					return err
				}
			}

			checkins = append(checkins, domain.Checkin{
				BoxID:           input.BoxID,
				StudentID:       student.ID,
				CheckinDate:     parsedCheckin.CheckinDate,
				CheckinTime:     parsedCheckin.CheckinTime,
				Source:          input.Source,
				ImportHistoryID: importHistory.ID,
				CreatedAt:       now,
			})
		}

		phase = "checkin_persistence"
		var err error
		insertedCheckins, err = uc.checkins.SaveMany(transactionContext, checkins)
		if err != nil {
			return err
		}
		phase = "campaign_recalculation"
		if err := uc.recalculateActiveCampaigns(transactionContext, input.BoxID); err != nil {
			return err
		}
		phase = "retention_baseline"
		if err := uc.imports.SetRetentionBaselineIfEmpty(transactionContext, input.BoxID, now); err != nil {
			return err
		}
		phase = "history_finalize"
		return uc.imports.MarkCompleted(transactionContext, input.BoxID, importHistory.ID, studentsCreated, insertedCheckins, time.Now().UTC())
	})
	if transactionErr != nil {
		errorCode, _ := importErrorDetails(phase, transactionErr)
		failureContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if statusErr := uc.imports.MarkFailed(failureContext, input.BoxID, importHistory.ID, errorCode, time.Now().UTC()); statusErr != nil {
			statusKind, statusSQLState := importErrorDetails("history_failure_finalize", statusErr)
			slog.ErrorContext(failureContext, "checkin_import_failure_status_update_failed",
				"box_id", input.BoxID,
				"import_id", importHistory.ID,
				"error_kind", statusKind,
				"sqlstate", statusSQLState,
			)
		}
		logImportFailure(ctx, input, importHistory.ID, phase, len(parsed), transactionErr, startedAt)
		return nil, transactionErr
	}
	if uc.activations != nil {
		if err := uc.activations.ReprocessPending(ctx, input.BoxID, input.Source); err != nil {
			slog.WarnContext(ctx, "contact_activation_reprocess_failed", "box_id", input.BoxID, "source", input.Source, "error", err)
		}
	}

	return &ImportCheckinsOutput{
		ImportID:     importHistory.ID,
		TotalRecords: len(parsed),
		Students:     studentsCreated,
		Checkins:     insertedCheckins,
	}, nil
}

func logImportFailure(ctx context.Context, input ImportCheckinsInput, importID domain.ID, phase string, records int, err error, startedAt time.Time) {
	errorKind, sqlState := importErrorDetails(phase, err)
	slog.ErrorContext(ctx, "checkin_import_failed",
		"box_id", input.BoxID,
		"import_id", importID,
		"source", input.Source,
		"phase", phase,
		"error_kind", errorKind,
		"sqlstate", sqlState,
		"records", records,
		"latency_ms", time.Since(startedAt).Milliseconds(),
	)
}

func importErrorDetails(phase string, err error) (string, string) {
	if errors.Is(err, context.Canceled) {
		return "request_canceled", ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded", ""
	}
	type sqlStateCarrier interface{ SQLState() string }
	var sqlError sqlStateCarrier
	if errors.As(err, &sqlError) {
		sqlState := sqlError.SQLState()
		switch {
		case strings.HasPrefix(sqlState, "08"):
			return "database_connection", sqlState
		case sqlState == "23503":
			return "foreign_key_violation", sqlState
		case sqlState == "23505":
			return "unique_violation", sqlState
		case sqlState == "40001":
			return "serialization_failure", sqlState
		case sqlState == "40P01":
			return "deadlock", sqlState
		case sqlState == "57014":
			return "query_canceled", sqlState
		default:
			return "database_error", sqlState
		}
	}
	if strings.Contains(strings.ToLower(err.Error()), "65535") && strings.Contains(strings.ToLower(err.Error()), "parameter") {
		return "database_parameter_limit", ""
	}
	return fmt.Sprintf("%s_failed", phase), ""
}

func (uc ImportCheckinsUseCase) findSelfRegisteredStudent(ctx context.Context, boxID domain.ID, source domain.Source, name string) (*domain.Student, error) {
	normalizedName := strings.Join(strings.Fields(strings.ToLower(name)), " ")
	candidates, err := uc.students.List(ctx, boxID, repositories.StudentFilters{Source: &source, Search: normalizedName})
	if err != nil {
		return nil, err
	}
	matches := make([]domain.Student, 0, 1)
	for _, candidate := range candidates {
		candidateName := strings.Join(strings.Fields(strings.ToLower(candidate.Name)), " ")
		if candidate.AnonymizedAt == nil && candidate.MembershipStartedSource == "self_registration" && candidateName == normalizedName {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return nil, nil
	}
	return &matches[0], nil
}

func (uc ImportCheckinsUseCase) recalculateActiveCampaigns(ctx context.Context, boxID domain.ID) error {
	activeCampaigns, err := uc.campaigns.ListActive(ctx, boxID)
	if err != nil {
		return err
	}

	allStudents, err := uc.students.List(ctx, boxID, repositories.StudentFilters{})
	if err != nil {
		return err
	}

	for _, campaign := range activeCampaigns {
		goals, err := uc.campaigns.ListGoals(ctx, campaign.ID)
		if err != nil {
			return err
		}
		campaignCheckins, err := uc.checkins.ListByRange(ctx, boxID, domain.TimeRange{Start: campaign.StartDate, End: campaign.EndDate})
		if err != nil {
			return err
		}

		progress := domain.BuildCampaignProgress(campaign.ID, allStudents, campaignCheckins, goals)

		if err := uc.campaigns.ReplaceProgress(ctx, campaign.ID, progress); err != nil {
			return err
		}

		eligibleStudentIDs := []domain.ID{}
		for _, item := range progress {
			if item.Achieved {
				eligibleStudentIDs = append(eligibleStudentIDs, item.StudentID)
			}
		}

		rewards, err := uc.rewards.ListByCampaign(ctx, boxID, campaign.ID)
		if err != nil {
			return err
		}
		for _, reward := range rewards {
			if err := uc.rewards.SyncPendingDeliveries(ctx, reward.ID, eligibleStudentIDs); err != nil {
				return err
			}
		}
	}

	return nil
}

func studentIdentity(parsed services.ParsedCheckin) string {
	if parsed.StudentExternalID != "" {
		return strings.TrimSpace(strings.ToLower(parsed.StudentExternalID))
	}
	if parsed.StudentEmail != "" {
		return strings.TrimSpace(strings.ToLower(parsed.StudentEmail))
	}
	if parsed.StudentPhone != "" {
		return strings.TrimSpace(strings.ToLower(parsed.StudentPhone))
	}
	return strings.Join(strings.Fields(strings.ToLower(parsed.StudentName)), " ")
}
