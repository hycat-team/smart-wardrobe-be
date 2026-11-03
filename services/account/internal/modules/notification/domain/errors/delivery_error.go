// Package errors holds the delivery outcome vocabulary of the notification
// module.
//
// It lives in domain because both ends of the port need it: the SMTP adapter
// (infrastructure) classifies a transport failure, and the email worker
// (presentation) decides whether the broker should try again. Domain is the only
// layer both may import.
package errors

import "errors"

// DeliveryError marks WHY a delivery failed, which is the only thing the caller
// needs in order to decide its next move: retry, or give up.
type DeliveryError struct {
	Transient bool
	Err       error
}

func (e *DeliveryError) Error() string { return e.Err.Error() }

func (e *DeliveryError) Unwrap() error { return e.Err }

// Transient marks a failure that redelivery could fix: the relay was unreachable,
// timed out, or answered with a 4xx.
func Transient(err error) error {
	return &DeliveryError{Transient: true, Err: err}
}

// Permanent marks a failure that will fail again identically: invalid recipient,
// 5xx from the relay, malformed message. Retrying it only delays the DLQ.
func Permanent(err error) error {
	return &DeliveryError{Transient: false, Err: err}
}

// IsTransient reports whether redelivery is worth attempting. An error that was
// never classified is treated as transient: the cost of one more attempt is a few
// minutes, while the cost of a premature DLQ is a customer who never receives the
// message.
func IsTransient(err error) bool {
	var deliveryErr *DeliveryError
	if errors.As(err, &deliveryErr) {
		return deliveryErr.Transient
	}
	return true
}
