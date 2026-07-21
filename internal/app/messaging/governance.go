package messaging

import (
	"context"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/repositories"
	"boxengage/backend/internal/ports/services"
)

type GovernanceService struct {
	repository repositories.MessagingGovernanceRepository
}

func NewGovernanceService(repository repositories.MessagingGovernanceRepository) GovernanceService {
	return GovernanceService{repository: repository}
}

func (s GovernanceService) Reserve(ctx context.Context, request services.MessagingReservationRequest) (*domain.MessageDispatch, error) {
	return s.repository.Reserve(ctx, repositories.MessagingReservation{
		BoxID: request.BoxID, RequestedByUserID: request.RequestedByUserID,
		SourceType: request.SourceType, SourceID: request.SourceID,
		ConnectionMode: request.ConnectionMode, Recipients: request.Recipients,
	})
}

func (s GovernanceService) Complete(ctx context.Context, dispatchID domain.ID, accepted, failed int) error {
	return s.repository.Complete(ctx, dispatchID, accepted, failed)
}
