package dto

import "boxengage/backend/internal/app/memberbilling"

type StartMemberBillingRequest = memberbilling.StartInput
type ManualMemberPaymentRequest = memberbilling.ManualPaymentInput
type ReconcileMemberPaymentRequest = memberbilling.ReconcileInput
type RefundMemberPaymentRequest = memberbilling.RefundInput
