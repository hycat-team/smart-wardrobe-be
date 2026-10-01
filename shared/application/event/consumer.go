package event

import (
	"context"
	"encoding/json"
)

// Message là dạng trừu tượng của message nhận từ message broker
// (không lộ công nghệ broker cụ thể: AMQP/Kafka/NATS...).
// Ack/nack do infrastructure xử lý dựa trên lỗi handler trả về
// (err != nil → nack/drop, nil → ack) — worker không cần biết broker.
type Message struct {
	Body    []byte
	Headers map[string]any
}

// CorrelationId trả về correlationId từ message headers (nếu có),
// dùng để log/trace cùng phía publisher (outbox ghi x-correlation-id).
// CorrelationId returns the correlation identifier carried by the message.
//
// BOTH header names are read. The outbox publisher writes "correlationId",
// while an explicitly supplied header arrives as "x-correlation-id" — the
// standard AMQP property name. Reading only the second one meant the
// correlation id was empty for every message this system produced itself, so no
// rejection could be traced back to its publisher (024 FR-086).
func (m Message) CorrelationId() string {
	if m.Headers == nil {
		return ""
	}
	for _, key := range []string{"x-correlation-id", "correlationId"} {
		if v, ok := m.Headers[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// PermanentError đánh dấu poison message (JSON hỏng, schema lệch) —
// broker sẽ đưa vào DLQ, không requeue.
type PermanentError struct{ Err error }

func (e PermanentError) Error() string { return e.Err.Error() }
func (e PermanentError) Unwrap() error { return e.Err }

// RetryableError đánh dấu lỗi tạm thời (DB timeout, dependency offline) —
// broker sẽ requeue; vượt delivery-limit của queue thì tự vào DLQ.
type RetryableError struct{ Err error }

func (e RetryableError) Error() string { return e.Err.Error() }
func (e RetryableError) Unwrap() error { return e.Err }

// ConsumerOpts điều chỉnh QoS/concurrency của consumer. Giá trị zero
// nghĩa là "dùng default per-queue của broker implementation"
// (với RabbitMQ: đọc từ Topology registry).
type ConsumerOpts struct {
	Prefetch    int
	Concurrency int
}

// TypedHandler xử lý payload đã unmarshal.
type TypedHandler[T any] func(ctx context.Context, payload T) error

// correlationIDContextKey là key lưu correlationId trong context per-message.
type correlationIDContextKey struct{}

// ContextWithCorrelationId gắn correlationId của message vào ctx.
// Broker implementation gọi cho từng message trước khi chạy handler;
// worker đọc lại bằng CorrelationIdFromContext để log cùng ID với publisher.
func ContextWithCorrelationId(ctx context.Context, msg Message) context.Context {
	return context.WithValue(ctx, correlationIDContextKey{}, msg.CorrelationId())
}

// CorrelationIdFromContext đọc correlationId mà ContextWithCorrelationId đã gắn.
func CorrelationIdFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(correlationIDContextKey{}).(string); ok {
		return v
	}
	return ""
}

// IMessageConsumer là contract trừu tượng cho việc consume message từ broker.
// Đặt trong application layer để presentation worker phụ thuộc abstraction,
// không phụ thuộc implementation cụ thể (dependency inversion).
type IMessageConsumer interface {
	Consume(queueName string) (<-chan Message, error)
	RegisterConsumer(queueName string, handler func(msg Message) error)
	RegisterTypedHandler(queueName string, handler func(ctx context.Context, msg Message) error, opts ConsumerOpts) error
	Close()
}

// RegisterTypedConsumer đăng ký typed consumer: helper tự unmarshal JSON
// vào T (fail → PermanentError → DLQ) rồi delegate xuống
// IMessageConsumer.RegisterTypedHandler — implementation inject
// correlationId vào ctx và spawn Concurrency consumer goroutines.
func RegisterTypedConsumer[T any](mc IMessageConsumer, queue string, handler TypedHandler[T], opts ConsumerOpts) error {
	wrapper := func(ctx context.Context, msg Message) error {
		var payload T
		if err := json.Unmarshal(msg.Body, &payload); err != nil {
			return PermanentError{Err: err}
		}
		return handler(ctx, payload)
	}
	return mc.RegisterTypedHandler(queue, wrapper, opts)
}
