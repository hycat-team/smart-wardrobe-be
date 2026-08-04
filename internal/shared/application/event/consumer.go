package event

// Message là dạng trừu tượng của message nhận từ message broker
// (không lộ công nghệ broker cụ thể: AMQP/Kafka/NATS...).
// Ack/nack do infrastructure xử lý dựa trên lỗi handler trả về
// (err != nil → nack/drop, nil → ack) — worker không cần biết broker.
type Message struct {
	Body []byte
}

// IMessageConsumer là contract trừu tượng cho việc consume message từ broker.
// Đặt trong application layer để presentation worker phụ thuộc abstraction,
// không phụ thuộc implementation cụ thể (dependency inversion).
type IMessageConsumer interface {
	Consume(queueName string) (<-chan Message, error)
	RegisterConsumer(queueName string, handler func(msg Message) error)
	Close()
}
