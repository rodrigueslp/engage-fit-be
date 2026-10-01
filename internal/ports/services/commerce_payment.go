package services

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type CommercePaymentRequest struct {
	SaleID, ActorUserID domain.ID
	Method              string
	AmountCents         int
	OccurredAt          time.Time
}

type CommercePaymentReceipt struct{ Reference string }

// CommercePaymentRecorder keeps commerce independent from memberbilling and
// lets a future provider replace today's local, manual-only implementation.
type CommercePaymentRecorder interface {
	Record(context.Context, CommercePaymentRequest) (CommercePaymentReceipt, error)
}
