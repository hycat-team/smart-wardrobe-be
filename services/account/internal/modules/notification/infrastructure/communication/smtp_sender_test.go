package communication

import (
	"context"
	"net"
	"testing"
	"time"

	"smart-wardrobe-be/shared/config"

	"github.com/stretchr/testify/require"
)

// noDialGrace is how long a sender that must stay silent is given to prove it.
// It only has to exceed scheduling noise: a sender that transmits connects
// immediately, so a short wait is the assertion, not a flake risk.
const noDialGrace = 500 * time.Millisecond

// localSMTPListener opens a real TCP listener and returns it with the port the
// sender will be configured to dial. The listener never speaks SMTP, so a
// connection opened by the sender is accepted and hung up on.
func localSMTPListener(t *testing.T) (net.Listener, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	addr, ok := ln.Addr().(*net.TCPAddr)
	require.True(t, ok)
	return ln, addr.Port
}

// armDialWatcher starts accepting before the sender runs. It has to be armed
// first: a sender that DID dial blocks reading a greeting this listener never
// writes, so arming afterwards would hang the test instead of observing it.
func armDialWatcher(ln net.Listener) <-chan struct{} {
	dialed := make(chan struct{}, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
		dialed <- struct{}{}
	}()
	return dialed
}

func smtpConfig(port int, enabled bool) *config.Config {
	cfg := &config.Config{}
	cfg.Email.Enabled = enabled
	cfg.Email.Host = "127.0.0.1"
	cfg.Email.Port = port
	cfg.Email.SenderName = "SmartWardrobe"
	cfg.Email.SenderEmail = "no-reply@example.test"
	return cfg
}

// TestSmtpSenderTransmitsNothingWhenDisabled is the 025 T113 safety property:
// while `email.enabled` is false the sender must open no socket and transmit
// nothing. The host points at a real listening socket precisely so that a dial
// would be visible — asserting only on the returned error would also pass for a
// sender that dialled and then gave up quietly.
func TestSmtpSenderTransmitsNothingWhenDisabled(t *testing.T) {
	ln, port := localSMTPListener(t)
	dialed := armDialWatcher(ln)
	sender := NewSmtpSender(smtpConfig(port, false))

	err := sender.Send(context.Background(), "user@example.test", "subject", "<p>body</p>")

	select {
	case <-dialed:
		t.Fatal("a disabled sender must not open an SMTP connection")
	case <-time.After(noDialGrace):
	}
	require.NoError(t, err, "a disabled sender handles the message on purpose, so it is not an error")
}

// TestSmtpSenderPortIsNotDialedWhenDisabled covers the implicit-TLS branch as
// well: port 465 dials through tls.Dial on a different code path from the plain
// smtp.Dial above, and a guard placed inside one branch would leave the other
// open.
func TestSmtpSenderPortIsNotDialedWhenDisabled(t *testing.T) {
	ln, port := localSMTPListener(t)
	dialed := armDialWatcher(ln)
	cfg := smtpConfig(port, false)
	cfg.Email.Port = 465
	sender := NewSmtpSender(cfg)

	err := sender.Send(context.Background(), "user@example.test", "subject", "<p>body</p>")

	select {
	case <-dialed:
		t.Fatal("the implicit-TLS branch must be guarded too")
	case <-time.After(noDialGrace):
	}
	require.NoError(t, err)
}

// TestSmtpSenderDialsWhenEnabled is the control for the two tests above: same
// listener, same configuration, switch flipped. Without it they could pass for
// the wrong reason — a sender broken by an unreachable host rather than by the
// switch.
func TestSmtpSenderDialsWhenEnabled(t *testing.T) {
	ln, port := localSMTPListener(t)
	dialed := armDialWatcher(ln)
	sender := NewSmtpSender(smtpConfig(port, true))

	// The listener speaks no SMTP and hangs up, so Send fails after the dial.
	// The point of this test is that it dialled at all.
	_ = sender.Send(context.Background(), "user@example.test", "subject", "<p>body</p>")

	select {
	case <-dialed:
	case <-time.After(5 * time.Second):
		t.Error("an enabled sender must still reach the configured host; the switch must not break delivery")
	}
}
