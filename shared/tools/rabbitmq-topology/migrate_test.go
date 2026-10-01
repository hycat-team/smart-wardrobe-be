package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"smart-wardrobe-be/shared/infrastructure/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

// These exercise the real broker and are skipped unless TEST_RABBITMQ_URL is
// set, matching the other messaging integration tests in this repo.
//
// They cover the part that cannot be unit-tested: that draining a classic queue
// and replaying the spill file loses nothing and preserves message identity.

func testSettings(t *testing.T) Settings {
	t.Helper()
	url := os.Getenv("TEST_RABBITMQ_URL")
	if url == "" {
		t.Skip("TEST_RABBITMQ_URL not set")
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Skipf("TEST_RABBITMQ_URL not reachable: %v", err)
	}
	_ = conn.Close()

	return Settings{
		Host:         "127.0.0.1",
		Port:         5672,
		User:         "guest",
		Password:     "123456",
		MgmtUser:     "guest",
		MgmtPassword: "123456",
		MgmtPort:     15672,
	}
}

// TestMigratePreservesMessages is the regression test for the whole point of the
// drain-then-replay design: a queue carrying traffic must come out the other side
// as quorum with every message intact.
func TestMigratePreservesMessages(t *testing.T) {
	s := testSettings(t)

	conn, err := s.Dial()
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Close()

	const q = "rabbitmq_topology_migrate_test"
	const dlq = q + ".dlq"

	for _, name := range []string{q, dlq} {
		if _, err := ch.QueueDelete(name, false, false, false); err != nil {
			t.Logf("pre-clean %s: %v", name, err)
		}
	}

	// Start from a CLASSIC queue carrying messages — the exact stale state that
	// makes a service fail with 406 on startup.
	if _, err := ch.QueueDeclare(q, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    messaging.ExchangeDLXName,
		"x-dead-letter-routing-key": dlq,
	}); err != nil {
		t.Fatalf("declare classic: %v", err)
	}

	wantBodies := []string{`{"n":1}`, `{"n":2}`, `{"n":3}`}

	// Publisher confirms: without them Publish is fire-and-forget and the
	// seeding assertion below would race the deliveries.
	if err := ch.Confirm(false); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, len(wantBodies)))

	for i, body := range wantBodies {
		if err := ch.Publish("", q, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    "probe-" + string(rune('a'+i)),
			Body:         []byte(body),
		}); err != nil {
			t.Fatalf("publish %d: %v", i, err)
		}
	}
	for i := 0; i < len(wantBodies); i++ {
		if c := <-confirms; !c.Ack {
			t.Fatalf("seed message %d not acknowledged", i)
		}
	}

	if state, err := ch.QueueInspect(q); err != nil || state.Messages != len(wantBodies) {
		t.Fatalf("seed: state=%+v err=%v, want %d messages", state, err, len(wantBodies))
	}

	spill := t.TempDir()
	m, err := NewMigrator(s, spill)
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}
	defer m.Close()

	binding := messaging.Binding{
		Queue:       q,
		RoutingKeys: []string{"probe.probe"},
		Exchange:    messaging.ExchangeName,
		Args: amqp.Table{
			"x-queue-type":              messaging.QueueTypeQuorum,
			"x-delivery-limit":          5,
			"x-dead-letter-exchange":    messaging.ExchangeDLXName,
			"x-dead-letter-routing-key": dlq,
		},
	}

	spillPath, drained, err := m.Migrate(binding, messaging.QueueTypeQuorum)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if drained != len(wantBodies) {
		t.Errorf("drained = %d, want %d", drained, len(wantBodies))
	}

	// The queue must now be quorum.
	state, err := ch.QueueInspect(q)
	if err != nil {
		t.Fatalf("inspect after migrate: %v", err)
	}
	if state.Messages != len(wantBodies) {
		t.Errorf("messages after migrate = %d, want %d (replay lost messages)", state.Messages, len(wantBodies))
	}

	// Every original body must have survived the round trip.
	got := drainBodies(t, ch, q)
	if len(got) != len(wantBodies) {
		t.Fatalf("read back %d message(s), want %d: %v", len(got), len(wantBodies), got)
	}
	for i, want := range wantBodies {
		if got[i] != want {
			t.Errorf("message %d = %q, want %q", i, got[i], want)
		}
	}

	// The spill file is the audit trail; it must exist even for a clean run.
	if _, err := os.Stat(spillPath); err != nil {
		t.Errorf("spill file missing: %v", err)
	}

	if _, err := ch.QueueDelete(q, false, false, false); err != nil {
		t.Logf("cleanup: %v", err)
	}
}

// TestMigrateRefusesLiveConsumers guards the race the tool exists to avoid:
// deleting a queue under a running worker.
func TestMigrateRefusesLiveConsumers(t *testing.T) {
	s := testSettings(t)

	conn, err := s.Dial()
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Close()

	const q = "rabbitmq_topology_consumer_test"
	if _, err := ch.QueueDelete(q, false, false, false); err != nil {
		t.Logf("pre-clean: %v", err)
	}
	if _, err := ch.QueueDeclare(q, true, false, false, false, nil); err != nil {
		t.Fatalf("declare: %v", err)
	}
	defer ch.QueueDelete(q, false, false, false)

	consumerTag := "topology-test-consumer"
	cons, err := ch.Consume(q, consumerTag, false, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	go func() {
		for range cons {
		}
	}()

	m, err := NewMigrator(s, t.TempDir())
	if err != nil {
		t.Fatalf("NewMigrator: %v", err)
	}
	defer m.Close()

	binding := messaging.Binding{
		Queue:    q,
		Args:     amqp.Table{"x-queue-type": messaging.QueueTypeQuorum},
		Exchange: messaging.ExchangeName,
	}
	if _, _, err := m.Migrate(binding, messaging.QueueTypeQuorum); err == nil {
		t.Error("Migrate succeeded with a live consumer attached; it must refuse")
	} else {
		t.Logf("refused as expected: %v", err)
	}

	_ = ch.Cancel(consumerTag, false)
}

func drainBodies(t *testing.T, ch *amqp.Channel, q string) []string {
	t.Helper()
	var out []string
	for {
		d, ok, err := ch.Get(q, true)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if !ok {
			return out
		}
		out = append(out, string(d.Body))
	}
}

// TestSpillRecordRoundTripsPayload checks the on-disk format independently: the
// spill file is the only copy of a message between drain and replay, so a
// payload that survives a write/read cycle must be byte-identical.
func TestSpillRecordRoundTripsPayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "spill.jsonl")

	payload := []byte(`{"nested":{"unicode":"héllo→","bin":"\u0000"}}`)
	rec := drainedMessage{
		RoutingKey:   "probe.probe",
		Exchange:     "smart_wardrobe_exchange",
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageID:    "probe-a",
		Payload:      base64.StdEncoding.EncodeToString(payload),
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := json.NewEncoder(f).Encode(rec); err != nil {
		t.Fatalf("encode: %v", err)
	}
	_ = f.Close()

	rf, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer rf.Close()

	var back drainedMessage
	if err := json.NewDecoder(rf).Decode(&back); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := base64.StdEncoding.DecodeString(back.Payload)
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("payload = %q, want %q", got, payload)
	}
	if back.MessageID != "probe-a" || back.RoutingKey != "probe.probe" {
		t.Errorf("message identity lost: %+v", back)
	}
}
