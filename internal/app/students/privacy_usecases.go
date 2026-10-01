package students

import (
	"context"
	"errors"
	"strings"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

var ErrInvalidPrivacyRequest = errors.New("invalid privacy request")

type ExportStudentDataUseCase struct {
	privacy       repositories.PrivacyRepository
	enrollments   repositories.EnrollmentPrivacyReader
	contracts     repositories.ContractPrivacyReader
	reservations  repositories.ReservationPrivacyReader
	memberBilling repositories.MemberBillingPrivacyReader
	healthForms   repositories.HealthFormRepository
	commerce      repositories.CommercePrivacyReader
}

func NewExportStudentDataUseCase(privacy repositories.PrivacyRepository, enrollmentReaders ...repositories.EnrollmentPrivacyReader) ExportStudentDataUseCase {
	var enrollmentReader repositories.EnrollmentPrivacyReader
	if len(enrollmentReaders) > 0 {
		enrollmentReader = enrollmentReaders[0]
	}
	return ExportStudentDataUseCase{privacy: privacy, enrollments: enrollmentReader}
}

func (uc *ExportStudentDataUseCase) WithContracts(contracts repositories.ContractPrivacyReader) {
	uc.contracts = contracts
}

func (uc *ExportStudentDataUseCase) WithReservations(reader repositories.ReservationPrivacyReader) {
	uc.reservations = reader
}

func (uc *ExportStudentDataUseCase) WithMemberBilling(reader repositories.MemberBillingPrivacyReader) {
	uc.memberBilling = reader
}

func (uc *ExportStudentDataUseCase) WithHealthForms(reader repositories.HealthFormRepository) {
	uc.healthForms = reader
}

func (uc *ExportStudentDataUseCase) WithCommerce(reader repositories.CommercePrivacyReader) {
	uc.commerce = reader
}

func (uc ExportStudentDataUseCase) Execute(ctx context.Context, boxID, studentID, actorUserID domain.ID) (*domain.StudentPrivacyExport, error) {
	result, err := uc.privacy.ExportStudent(ctx, boxID, studentID)
	if err != nil {
		return nil, err
	}
	if uc.enrollments != nil && result.Person != nil {
		result.Enrollments, err = uc.enrollments.ListPrivacy(ctx, boxID, result.Person.ID)
		if err != nil {
			return nil, err
		}
	}
	if uc.contracts != nil && result.Person != nil {
		result.Contracts, err = uc.contracts.ListPrivacy(ctx, boxID, result.Person.ID)
		if err != nil {
			return nil, err
		}
	}
	if uc.reservations != nil && result.Person != nil {
		result.Reservations, err = uc.reservations.ListReservationPrivacy(ctx, boxID, result.Person.ID)
		if err != nil {
			return nil, err
		}
	}
	if uc.memberBilling != nil && result.Person != nil {
		result.MemberBilling, err = uc.memberBilling.ListPrivacy(ctx, boxID, result.Person.ID)
		if err != nil {
			return nil, err
		}
	}
	if uc.healthForms != nil && result.Person != nil {
		result.HealthForms, err = uc.healthForms.ListPrivacy(ctx, boxID, result.Person.ID)
		if err != nil {
			return nil, err
		}
	}
	if uc.commerce != nil && result.Person != nil {
		result.CommerceSales, err = uc.commerce.ListPrivacy(ctx, boxID, result.Person.ID)
		if err != nil {
			return nil, err
		}
	}
	if err := uc.privacy.RecordAudit(ctx, boxID, studentID, actorUserID, "exported", ""); err != nil {
		return nil, err
	}
	return result, nil
}

type UpdateContactPreferenceUseCase struct {
	students repositories.StudentRepository
	privacy  repositories.PrivacyRepository
}

func NewUpdateContactPreferenceUseCase(students repositories.StudentRepository, privacy repositories.PrivacyRepository) UpdateContactPreferenceUseCase {
	return UpdateContactPreferenceUseCase{students: students, privacy: privacy}
}

func (uc UpdateContactPreferenceUseCase) Execute(ctx context.Context, boxID, studentID, actorUserID domain.ID, status domain.ContactStatus, source string) error {
	source = strings.TrimSpace(source)
	if (status != domain.ContactStatusUnknown && status != domain.ContactStatusOptedIn && status != domain.ContactStatusOptedOut) || source == "" || len(source) > 100 {
		return ErrInvalidPrivacyRequest
	}
	if _, err := uc.students.FindByID(ctx, boxID, studentID); err != nil {
		return err
	}
	if err := uc.students.UpdateContactPreference(ctx, boxID, studentID, status, source, time.Now().UTC()); err != nil {
		return err
	}
	return uc.privacy.RecordAudit(ctx, boxID, studentID, actorUserID, "contact_preference_updated", string(status)+":"+source)
}

type AnonymizeStudentUseCase struct {
	privacy repositories.PrivacyRepository
}

func NewAnonymizeStudentUseCase(privacy repositories.PrivacyRepository) AnonymizeStudentUseCase {
	return AnonymizeStudentUseCase{privacy: privacy}
}

func (uc AnonymizeStudentUseCase) Execute(ctx context.Context, boxID, studentID, actorUserID domain.ID, confirmed bool, reason string) error {
	reason = strings.TrimSpace(reason)
	if !confirmed || len(reason) < 5 || len(reason) > 500 {
		return ErrInvalidPrivacyRequest
	}
	return uc.privacy.AnonymizeStudent(ctx, boxID, studentID, actorUserID, reason)
}
