package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"smart-wardrobe-be/shared/infrastructure/messaging"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DefaultSpillDir holds the drained messages when the tool runs on the host.
// Inside the container image (distroless nonroot) /tmp is not writable, so the
// migrate scripts pass --spill-dir pointing at a mounted host directory —
// without that mount the drained messages would disappear with the container.
const DefaultSpillDir = "/tmp/rabbitmq-topology-drain"

// drainedMessage is one line of the spill file. Payload is base64 because
// message bodies are binary JSON and the file has to stay line-delimited.
type drainedMessage struct {
	RoutingKey    string     `json:"routing_key"`
	Exchange      string     `json:"exchange"`
	Headers       amqp.Table `json:"headers,omitempty"`
	ContentType   string     `json:"content_type,omitempty"`
	DeliveryMode  uint8      `json:"delivery_mode,omitempty"`
	CorrelationID string     `json:"correlation_id,omitempty"`
	MessageID     string     `json:"message_id,omitempty"`
	Timestamp     int64      `json:"timestamp,omitempty"`
	Payload       string     `json:"payload"`
}

// Migrator performs the drain/delete/redeclare/replay cycle for one queue.
type Migrator struct {
	conn    *amqp.Connection
	spillIn string
}

// NewMigrator dials the broker and prepares the spill directory.
func NewMigrator(s Settings, spillDir string) (*Migrator, error) {
	if spillDir == "" {
		spillDir = DefaultSpillDir
	}
	conn, err := s.Dial()
	if err != nil {
		return nil, fmt.Errorf("dial amqp: %w", err)
	}
	if err := os.MkdirAll(spillDir, 0o755); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("create spill dir %s: %w", spillDir, err)
	}
	return &Migrator{conn: conn, spillIn: spillDir}, nil
}

func (m *Migrator) Close() error {
	if m.conn == nil {
		return nil
	}
	return m.conn.Close()
}

// Migrate brings one drifted queue to the type the registry declares. Live
// consumers are refused outright: deleting a queue under a running worker races
// the delivery in flight.
func (m *Migrator) Migrate(binding messaging.Binding, want string) (spillPath string, drained int, err error) {
	ch, err := m.conn.Channel()
	if err != nil {
		return "", 0, fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	queueName := binding.Queue
	state, err := ch.QueueInspect(queueName)
	if err != nil {
		return "", 0, fmt.Errorf("inspect queue %s (does it exist?): %w", queueName, err)
	}
	if state.Consumers > 0 {
		return "", 0, fmt.Errorf("queue %s still has %d consumer(s); stop the workers before migrating",
			queueName, state.Consumers)
	}

	spillPath, drained, err = m.drain(ch, queueName)
	if err != nil {
		return spillPath, drained, err
	}

	// Re-check after draining: a publisher may have enqueued while we drained.
	if state, err = ch.QueueInspect(queueName); err != nil {
		return spillPath, drained, fmt.Errorf("re-inspect queue %s: %w", queueName, err)
	}
	if state.Messages > 0 {
		return spillPath, drained, fmt.Errorf("queue %s still holds %d message(s) after drain; retry when the publisher is quiet",
			queueName, state.Messages)
	}

	if _, err := ch.QueueDelete(queueName, false, false, false); err != nil {
		return spillPath, drained, fmt.Errorf("delete queue %s: %w", queueName, err)
	}
	if _, err := ch.QueueDeclare(queueName, true, false, false, false, binding.Args); err != nil {
		return spillPath, drained, fmt.Errorf("redeclare queue %s as %s: %w", queueName, want, err)
	}
	for _, rk := range binding.RoutingKeys {
		if err := ch.QueueBind(queueName, rk, binding.Exchange, false, nil); err != nil {
			return spillPath, drained, fmt.Errorf("rebind %s -> %s on %s: %w", queueName, rk, binding.Exchange, err)
		}
	}

	if err := m.replay(ch, spillPath, binding); err != nil {
		return spillPath, drained, err
	}
	return spillPath, drained, nil
}

// drain empties the queue with basic.get, writing each message to the spill file
// before acking it. Acking only after the write is what makes this safe to
// interrupt: an unacked message stays on the queue, never lost.
//
// basic.get rather than a Consume loop: a consumer's delivery channel stays open
// after the queue empties, so `for range deliveries` would block forever on an
// already-drained queue. basic.get reports ok=false at empty and the loop ends.
func (m *Migrator) drain(ch *amqp.Channel, queueName string) (string, int, error) {
	spillPath := filepath.Join(m.spillIn, fmt.Sprintf("%s-%s.jsonl",
		sanitize(queueName), time.Now().UTC().Format("20060102T150405Z")))

	f, err := os.Create(spillPath)
	if err != nil {
		return "", 0, fmt.Errorf("create spill file %s: %w", spillPath, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	count := 0
	for {
		d, ok, err := ch.Get(queueName, false)
		if err != nil {
			return spillPath, count, fmt.Errorf("get from %s: %w", queueName, err)
		}
		if !ok {
			break
		}
		msg := drainedMessage{
			RoutingKey:    d.RoutingKey,
			Exchange:      d.Exchange,
			Headers:       d.Headers,
			ContentType:   d.ContentType,
			DeliveryMode:  d.DeliveryMode,
			CorrelationID: d.CorrelationId,
			MessageID:     d.MessageId,
			Timestamp:     d.Timestamp.Unix(),
			Payload:       base64.StdEncoding.EncodeToString(d.Body),
		}
		if err := enc.Encode(msg); err != nil {
			_ = d.Nack(false, true)
			return spillPath, count, fmt.Errorf("write spill record: %w", err)
		}
		if err := f.Sync(); err != nil {
			_ = d.Nack(false, true)
			return spillPath, count, fmt.Errorf("flush spill file: %w", err)
		}
		if err := d.Ack(false); err != nil {
			return spillPath, count, fmt.Errorf("ack drained message: %w", err)
		}
		count++
	}
	return spillPath, count, nil
}

// replay republishes the spill file through the default exchange keyed by queue
// name, so every drained message returns to the queue it came from. Headers and
// identity are restored verbatim so consumers cannot tell the difference.
func (m *Migrator) replay(ch *amqp.Channel, spillPath string, binding messaging.Binding) error {
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

	f, err := os.Open(spillPath)
	if err != nil {
		return fmt.Errorf("open spill file %s: %w", spillPath, err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	count := 0
	for {
		var msg drainedMessage
		if err := dec.Decode(&msg); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("read spill file: %w", err)
		}
		payload, err := base64.StdEncoding.DecodeString(msg.Payload)
		if err != nil {
			return fmt.Errorf("decode payload %d: %w", count, err)
		}
		// Replay through the DEFAULT exchange keyed by the queue name, not
		// through the queue's own exchange with the drained routing key.
		//
		// A drained message carries the routing key it was published with,
		// which is not necessarily one of this queue's binding keys. Sending
		// it back through the topic exchange can therefore match no binding,
		// and the message is silently dropped — the queue comes out empty
		// after a "successful" migration. The default exchange routes by
		// queue name, so the message lands in exactly the queue being
		// restored. The bindings are already back in place for new traffic.
		err = ch.PublishWithContext(context.Background(), "", binding.Queue, false, false, amqp.Publishing{
			Headers:       msg.Headers,
			ContentType:   msg.ContentType,
			DeliveryMode:  msg.DeliveryMode,
			CorrelationId: msg.CorrelationID,
			MessageId:     msg.MessageID,
			Timestamp:     time.Unix(msg.Timestamp, 0).UTC(),
			Body:          payload,
		})
		if err != nil {
			return fmt.Errorf("replay message %d: %w", count, err)
		}
		if confirmed := <-confirms; !confirmed.Ack {
			return fmt.Errorf("broker did not confirm replayed message %d; %d messages are still in %s",
				count, count, spillPath)
		}
		count++
	}
	if count == 0 {
		return nil
	}
	return nil
}

// sanitize turns a queue name into something safe for a filename.
func sanitize(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == '-' || r == '_' || r == '.':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
