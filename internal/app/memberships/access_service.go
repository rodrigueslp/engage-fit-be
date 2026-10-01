package memberships

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
)

type AccessService struct {
	repository repositories.MembershipAccessRepository
	usage      repositories.MembershipAccessUsageReader
	legacy     repositories.LegacyMembershipReader
	prereqs    repositories.MembershipAccessPrerequisiteReader
}

func NewAccessService(repository repositories.MembershipAccessRepository, usage repositories.MembershipAccessUsageReader, legacy repositories.LegacyMembershipReader) *AccessService {
	return &AccessService{repository: repository, usage: usage, legacy: legacy}
}

func (s *AccessService) WithPrerequisiteReader(value repositories.MembershipAccessPrerequisiteReader) *AccessService {
	s.prereqs = value
	return s
}

func (s *AccessService) Evaluate(ctx context.Context, request domain.MembershipAccessRequest) (*domain.MembershipAccessDecision, error) {
	decision := &domain.MembershipAccessDecision{Outcome: domain.MembershipAccessDenied, ReasonCode: domain.MembershipAccessReasonNoEnrollment, Reason: "Nenhuma matrícula ativa encontrada."}
	values, err := s.repository.ResolveMembershipAccessCandidates(ctx, request)
	if err != nil {
		return nil, err
	}
	limited := false
	bookingRequired := false
	var prerequisiteDenial *domain.MembershipAccessDecision
	prerequisites := map[domain.ID]domain.MembershipAccessPrerequisites{}
	for _, candidate := range values {
		if request.PreferredEnrollment != "" && candidate.EnrollmentID != request.PreferredEnrollment {
			continue
		}
		if request.PreferredEntitlement != "" && candidate.EntitlementID != request.PreferredEntitlement {
			continue
		}
		if s.prereqs != nil {
			status, ok := prerequisites[candidate.EnrollmentID]
			if !ok {
				status, err = s.prereqs.ResolveMembershipAccessPrerequisites(ctx, request.OrganizationID, request.UnitID, request.PersonID, candidate.EnrollmentID, request.OccurredAt)
				if err != nil {
					return nil, err
				}
				prerequisites[candidate.EnrollmentID] = status
			}
			if denial := prerequisiteDecision(candidate, status); denial != nil {
				if prerequisiteDenial == nil {
					prerequisiteDenial = denial
				}
				continue
			}
		}
		if request.Purpose == domain.MembershipAccessCheckin && candidate.BookingRequired && !request.UsageAlreadyCounted {
			bookingRequired = true
			continue
		}
		remaining, available, err := s.available(ctx, request, candidate)
		if err != nil {
			return nil, err
		}
		if !available {
			limited = true
			continue
		}
		return &domain.MembershipAccessDecision{Outcome: domain.MembershipAccessAllowed, ReasonCode: domain.MembershipAccessReasonEligible, Reason: "Direito ativo para esta unidade e atividade.", EnrollmentID: candidate.EnrollmentID, EntitlementID: candidate.EntitlementID, RemainingUsage: remaining}, nil
	}
	if prerequisiteDenial != nil {
		return prerequisiteDenial, nil
	}
	if limited {
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonLimitReached, "Limite de uso do plano atingido."
		return decision, nil
	}
	if bookingRequired {
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonBookingRequired, "Esta atividade exige reserva confirmada."
		return decision, nil
	}
	activeHere, err := s.repository.HasActiveEnrollmentInUnit(ctx, request.OrganizationID, request.UnitID, request.PersonID, request.OccurredAt)
	if err != nil {
		return nil, err
	}
	if activeHere {
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonTargetNotIncluded, "A atividade não está incluída no plano."
		return decision, nil
	}
	anyHere, err := s.repository.HasEnrollmentInUnit(ctx, request.OrganizationID, request.UnitID, request.PersonID)
	if err != nil {
		return nil, err
	}
	if anyHere {
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonOutsideValidity, "A matrícula está fora da vigência."
		return decision, nil
	}
	outside, err := s.repository.HasEnrollmentOutsideRequestedUnit(ctx, request.OrganizationID, request.UnitID, request.PersonID, request.OccurredAt)
	if err != nil {
		return nil, err
	}
	if outside {
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonUnitNotIncluded, "A matrícula não inclui esta unidade."
		return decision, nil
	}
	legacy, err := s.legacy.HasLegacyMembership(ctx, request.OrganizationID, request.UnitID, request.PersonID)
	if err != nil {
		return nil, err
	}
	if legacy && request.Purpose == domain.MembershipAccessCheckin {
		return &domain.MembershipAccessDecision{Outcome: domain.MembershipAccessAllowed, ReasonCode: domain.MembershipAccessReasonLegacyTransition, Reason: "Mensalista legado liberado durante a transição para matrículas.", LegacyFallback: true}, nil
	}
	return decision, nil
}

func prerequisiteDecision(candidate domain.MembershipAccessCandidate, value domain.MembershipAccessPrerequisites) *domain.MembershipAccessDecision {
	decision := &domain.MembershipAccessDecision{Outcome: domain.MembershipAccessDenied, EnrollmentID: candidate.EnrollmentID, EntitlementID: candidate.EntitlementID}
	switch {
	case !value.ContractAccepted:
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonContractPending, "O contrato da matrícula ainda não foi aceito."
	case !value.HealthFormSubmitted:
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonHealthFormPending, "O questionário PAR-Q da matrícula ainda não foi preenchido."
	case value.PaymentOverdue:
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonPaymentOverdue, "A matrícula possui pagamento vencido em aberto."
	case !value.InitialPaymentPaid:
		decision.ReasonCode, decision.Reason = domain.MembershipAccessReasonPaymentRequired, "O pagamento inicial da matrícula ainda não foi confirmado."
	default:
		return nil
	}
	return decision
}

func (s *AccessService) available(ctx context.Context, request domain.MembershipAccessRequest, candidate domain.MembershipAccessCandidate) (*int, bool, error) {
	if request.IgnoreUsage {
		return nil, true, nil
	}
	if err := s.usage.LockMembershipAccessUsage(ctx, request.OrganizationID, request.PersonID, candidate.EntitlementID); err != nil {
		return nil, false, err
	}
	var dailyRemaining *int
	if candidate.DailyLimit != nil {
		from, until := membershipUsageWindow(request.OccurredAt, candidate.Timezone, domain.MembershipPeriodDay, candidate.StartsOn, candidate.ValidUntil)
		used, err := s.usage.CountMembershipAccessUsage(ctx, domain.MembershipAccessUsageQuery{OrganizationID: request.OrganizationID, PersonID: request.PersonID, EnrollmentID: candidate.EnrollmentID, EntitlementID: candidate.EntitlementID, From: from, Until: until})
		if err != nil {
			return nil, false, err
		}
		if used > *candidate.DailyLimit || (!request.UsageAlreadyCounted && used >= *candidate.DailyLimit) {
			zero := 0
			return &zero, false, nil
		}
		value := *candidate.DailyLimit - used
		if !request.UsageAlreadyCounted {
			value--
		}
		dailyRemaining = &value
	}
	if candidate.AccessType == domain.MembershipAccessOpen || candidate.Quantity == nil {
		return dailyRemaining, true, nil
	}
	period := domain.MembershipPeriodMonth
	if candidate.PeriodUnit != nil {
		period = *candidate.PeriodUnit
	}
	from, until := membershipUsageWindow(request.OccurredAt, candidate.Timezone, period, candidate.StartsOn, candidate.ValidUntil)
	if candidate.AccessType == domain.MembershipAccessClassPack {
		from = candidate.StartsOn
		if candidate.ValidUntil != nil {
			until = candidate.ValidUntil.AddDate(0, 0, 1)
		}
	}
	used, err := s.usage.CountMembershipAccessUsage(ctx, domain.MembershipAccessUsageQuery{OrganizationID: request.OrganizationID, PersonID: request.PersonID, EnrollmentID: candidate.EnrollmentID, EntitlementID: candidate.EntitlementID, From: from, Until: until})
	if err != nil {
		return nil, false, err
	}
	remaining := *candidate.Quantity - used
	if remaining < 0 {
		remaining = 0
	}
	available := remaining > 0 || (request.UsageAlreadyCounted && remaining == 0)
	if available && !request.UsageAlreadyCounted {
		remaining--
	}
	if dailyRemaining != nil && *dailyRemaining < remaining {
		remaining = *dailyRemaining
	}
	return &remaining, available, nil
}

func membershipUsageWindow(at time.Time, timezone string, period domain.MembershipPeriodUnit, startsOn time.Time, validUntil *time.Time) (time.Time, time.Time) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		location = time.UTC
	}
	local := at.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	switch period {
	case domain.MembershipPeriodWeek:
		start = start.AddDate(0, 0, -((int(local.Weekday()) + 6) % 7))
	case domain.MembershipPeriodMonth:
		start = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, location)
	}
	end := start.AddDate(0, 0, 1)
	if period == domain.MembershipPeriodWeek {
		end = start.AddDate(0, 0, 7)
	} else if period == domain.MembershipPeriodMonth {
		end = start.AddDate(0, 1, 0)
	}
	return start.UTC(), end.UTC()
}
