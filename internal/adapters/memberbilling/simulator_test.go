package memberbilling

import (
	"context"
	"testing"
	"time"

	"boxengage/backend/internal/domain"
	"boxengage/backend/internal/ports/services"
)

func TestSimulatorProducesExplicitlyNonPayableInstructions(t *testing.T) {
	for _, method := range []string{"pix", "boleto", "card"} {
		checkout, err := NewSimulator().CreateCheckout(context.Background(), services.MemberPaymentRequest{ChargeID: domain.ID("charge-id"), AmountCents: 1000, Currency: "BRL", Method: method, DueOn: time.Now()})
		if err != nil || checkout.Provider != "local_simulator" || checkout.Instructions["simulation"] != true {
			t.Fatalf("%s: %+v %v", method, checkout, err)
		}
	}
}
