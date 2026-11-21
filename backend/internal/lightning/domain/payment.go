package domain

import "fmt"

// PaymentStatus is the state of an outgoing payment.
type PaymentStatus string

const (
	PaymentInitiated PaymentStatus = "INITIATED"
	PaymentInFlight  PaymentStatus = "IN_FLIGHT"
	PaymentSucceeded PaymentStatus = "SUCCEEDED"
	PaymentFailed    PaymentStatus = "FAILED"
	PaymentUnknown   PaymentStatus = "UNKNOWN"
)

// PaymentUpdate is a progress notification about an outgoing payment.
type PaymentUpdate struct {
	AmountSat int64
	Status    PaymentStatus
}

// String is the human readable message shown to the user.
func (u PaymentUpdate) String() string {
	return fmt.Sprintf("💸 Paiement de %d sats — statut : %s", u.AmountSat, u.Status)
}
