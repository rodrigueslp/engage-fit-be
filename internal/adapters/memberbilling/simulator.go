package memberbilling

import (
	"context"
	"fmt"

	"boxengage/backend/internal/ports/services"
)

// Simulator never calls a network and intentionally returns non-payable data.
type Simulator struct{}

func NewSimulator() Simulator { return Simulator{} }

func (Simulator) CreateCheckout(_ context.Context, request services.MemberPaymentRequest) (services.MemberPaymentCheckout, error) {
	reference := "sim_" + string(request.ChargeID)
	instructions := map[string]any{"simulation": true, "message": "Cobrança simulada; não efetue pagamento real."}
	switch request.Method {
	case "pix":
		instructions["pix_copy_paste"] = fmt.Sprintf("ENGAGEFIT-SIMULATOR-%s", request.ChargeID)
	case "boleto":
		instructions["digitable_line"] = "00000.00000 00000.000000 00000.000000 0 00000000000000"
	case "card":
		instructions["checkout_url"] = fmt.Sprintf("#/simulator/member-payment/%s", request.ChargeID)
	default:
		return services.MemberPaymentCheckout{}, fmt.Errorf("unsupported simulated payment method")
	}
	return services.MemberPaymentCheckout{Provider: "local_simulator", Reference: reference, Instructions: instructions}, nil
}
