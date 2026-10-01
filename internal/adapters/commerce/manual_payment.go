package commerce

import (
	"context"
	"errors"
	"fmt"

	"boxengage/backend/internal/ports/services"
)

type ManualPaymentRecorder struct{}

func NewManualPaymentRecorder() ManualPaymentRecorder { return ManualPaymentRecorder{} }

func (ManualPaymentRecorder) Record(_ context.Context, request services.CommercePaymentRequest) (services.CommercePaymentReceipt, error) {
	if request.AmountCents <= 0 || (request.Method != "pix" && request.Method != "cash" && request.Method != "card_machine") {
		return services.CommercePaymentReceipt{}, errors.New("invalid manual commerce payment")
	}
	return services.CommercePaymentReceipt{Reference: fmt.Sprintf("manual:%s", request.SaleID)}, nil
}
