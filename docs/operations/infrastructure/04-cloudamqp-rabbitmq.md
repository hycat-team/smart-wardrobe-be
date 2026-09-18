# 04 — CloudAMQP RabbitMQ (Tough Tiger)

Điều kiện: đã có instance CloudAMQP (`00` §0.5). Topology (quorum queue + DLX +
delivery-limit, prefetch 5) do code tự khai báo.

## 1. Tạo instance + vhost

1. Mở https://customer.cloudamqp.com → **Create New Instance** → plan **Tough Tiger**
   (có thể dùng gói nhỏ/free để thử).
2. Mở instance → tab **Vhosts** → **Create vhost** (vhost riêng cho Closy).

## 2. Lấy credential

Tab **Details** → copy **AMQP URL** dạng `amqps://<user>:<pass>@<host>/<vhost>`:

```bash
gh secret set RABBITMQ_HOST     --body "<host>.rmq.cloudamqp.com"
gh secret set RABBITMQ_PORT     --body "5671"
gh secret set RABBITMQ_USER     --body "<user>"
gh secret set RABBITMQ_PASSWORD --body "<password>"
```
Workflow managed tự set: `RABBITMQ_TLS_ENABLED=true`,
`RABBITMQ_MANAGEMENT_TLS_ENABLED=true`.

## 3. Kiểm tra kết nối TLS

```bash
# TLS handshake tới AMQPS port
openssl s_client -connect "$RABBITMQ_HOST:5671" -servername "$RABBITMQ_HOST" </dev/null
# Kỳ vọng: "Verify return code: 0 (ok)"

# Management API qua HTTPS (dùng gauge budget)
curl -sf -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" \
  "https://$RABBITMQ_HOST/api/overview" | head -c 200
```

## 4. Quorum queue (edge case)

- Verify instance hỗ trợ quorum queue; nếu không → fallback **classic durable** + bổ sung ADR.
- Outbox worker phát event qua poll; mất `LISTEN` không fatal (FR-009/FR-010).

## 5. Ngân sách connection (FR-047)

- Tổng client conn ≤ **100**, headroom ≥ 20%; verify bằng test tải nhẹ (SC-021).

## 6. Verify

- [ ] Kết nối `amqps` thành công từ Cloud Run.
- [ ] Topology (exchange/queue/DLX) khai báo đúng; quorum hoặc fallback có ADR.
- [ ] Outbox publish + consumer hoạt động; poll fallback khi mất `LISTEN`.

## 7. Xử lý sự cố

| Hiện tượng | Xử lý |
|---|---|
| `tls: bad certificate` | Dùng port `5671` (amqps), không phải `5672` |
| `ACCESS_REFUSED` | Sai vhost/user; kiểm tra lại **AMQP URL** và vhost đã tạo |
| Queue không mirror | Instance không hỗ trợ quorum → dùng classic durable + ADR |
