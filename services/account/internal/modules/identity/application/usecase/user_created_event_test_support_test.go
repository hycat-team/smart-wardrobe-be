package usecase_test

import (
	"context"
	"errors"
	"reflect"
	"sync"

	identitydto "smart-wardrobe-be/services/account/dto/identity"

	"github.com/google/uuid"
)

// reflectTypeOfPayload exposes the payload's fields to the FR-087 shape tests.
//
// It is a function so the reflection lives in one place: the two tests that use it
// care about different things (names versus json tags) but must see the same
// type.
func reflectTypeOfPayload() reflect.Type {
	return reflect.TypeOf(identitydto.UserCreatedEventPayload{})
}

// errUnexpectedPayload marks a publish whose body is not the declared payload
// type — a contract break on `identity.user.created`.
var errUnexpectedPayload = errors.New("unexpected payload type for identity.user.created")

// newCorrelationID mints the identifier the outbox publisher would.
func newCorrelationID() string { return uuid.NewString() }

// recordingPublisher is the `identity.user.created` spy (025 T029b).
//
// It stands in for the outbox publisher: `RabbitMQClient.Publish` mints a
// correlationId, writes it into the outbox record's headers, and the outbox
// worker later copies those headers onto the AMQP message. Reproducing that mint
// here is what lets FR-064 be asserted without a broker: the test reads the
// correlation id off the recorded record exactly as the worker's log would.
//
// `err` is the lever for FR-042. Setting it makes the publisher fail, which is
// the only way to prove that a failed event write takes the account creation down
// with it — an account that exists but whose event nobody ever heard about is the
// failure FR-042 exists to forbid.
type recordingPublisher struct {
	mu sync.Mutex
	// records is every publish that was attempted, in order.
	records []recordedPublish
	// err fails every publish when set.
	err error
}

// recordedPublish is one captured publish.
type recordedPublish struct {
	topic   string
	payload identitydto.UserCreatedEventPayload
	// correlationId mirrors what the outbox publisher writes into the record
	// headers and the broker then delivers (FR-064).
	correlationId string
	// inTx records whether the publish happened inside the caller's transaction.
	inTx bool
}

// txProbeKey marks a context as carrying the test's transaction, the same way
// the real unit of work does with `db.InjectTx`.
type txProbeKey struct{}

func inTestTx(ctx context.Context) bool {
	v, _ := ctx.Value(txProbeKey{}).(bool)
	return v
}

func newRecordingPublisher() *recordingPublisher { return &recordingPublisher{} }

func (p *recordingPublisher) Publish(ctx context.Context, topic string, payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	body, ok := payload.(identitydto.UserCreatedEventPayload)
	if !ok {
		// A different payload shape on this topic is a contract break, not
		// something to paper over: the receiver unmarshals into the declared
		// struct and would silently get zero values.
		p.err = errUnexpectedPayload
		return p.err
	}

	// A fresh id per publish, as the outbox publisher does.
	p.records = append(p.records, recordedPublish{
		topic:         topic,
		payload:       body,
		correlationId: newCorrelationID(),
		inTx:          inTestTx(ctx),
	})
	return p.err
}

// all returns a copy of the recorded publishes.
func (p *recordingPublisher) all() []recordedPublish {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]recordedPublish, len(p.records))
	copy(out, p.records)
	return out
}

// failWith makes every subsequent publish return err.
func (p *recordingPublisher) failWith(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.err = err
}
