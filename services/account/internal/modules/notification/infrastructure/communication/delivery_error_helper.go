package communication

import (
	"errors"
	"io"
	"net"
	"net/textproto"

	notificationerrors "smart-wardrobe-be/services/account/internal/modules/notification/domain/errors"
)

// classifyDeliveryError labels a transport failure so the worker can decide
// between "try again in a few minutes" and "this will never work".
//
// The split follows SMTP itself: 4xx is the server telling us to come back later,
// 5xx is the server telling us the message or the recipient is the problem.
// Anything else (dial, TLS, read/write, timeout) is treated as transient, because
// the alternative — dead-lettering on an unclassified error — is what silently
// lost messages before.
func classifyDeliveryError(err error) error {
	if err == nil {
		return nil
	}

	var smtpErr *textproto.Error
	if errors.As(err, &smtpErr) {
		if smtpErr.Code >= 500 {
			return notificationerrors.Permanent(err)
		}
		return notificationerrors.Transient(err)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return notificationerrors.Transient(err)
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return notificationerrors.Transient(err)
	}

	return notificationerrors.Transient(err)
}
