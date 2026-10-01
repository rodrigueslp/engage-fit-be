package services

import (
	"context"
	"time"

	"boxengage/backend/internal/domain"
)

type MemberPaymentRequest struct {
	ChargeID         domain.ID
	AmountCents      int
	Currency, Method string
	DueOn            time.Time
}

type MemberPaymentCheckout struct {
	Provider, Reference string
	Instructions        map[string]any
}

type MemberPaymentGateway interface {
	CreateCheckout(context.Context, MemberPaymentRequest) (MemberPaymentCheckout, error)
}
