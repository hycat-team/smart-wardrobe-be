# 04 — CloudAMQP RabbitMQ (Tough Tiger)

## Tài nguyên

- CloudAMQP **Tough Tiger**, **100 connections**, **TLS bắt buộc** (`amqps`, 5671).
- Vhost riêng cho Closy; topology (quorum queue + DLX + delivery-limit, prefetch 5)
  do code tự khai báo.

## Kết nối (FR-004)

- `rabbitmq.tls_enabled=true` trên managed (`amqps` + `TLSClientConfig` TLS1.2);
  mặc định `false` cho VPS.
- Management/gauge API qua HTTPS khi `rabbitmq.management_tls_enabled=true` (T066).
- Env: `RABBITMQ_HOST/PORT/USER/PASSWORD`, `RABBITMQ_TLS_ENABLED=true`,
  `RABBITMQ_MANAGEMENT_TLS_ENABLED=true`.

## Quorum queue (edge case)

- Verify quorum được hỗ trợ trên shared plan; nếu không → fallback
  **classic durable** + bổ sung ADR.
- Outbox worker phát event qua poll; `LISTEN` mất thì không fatal (FR-009/FR-010).

## Ngân sách connection (FR-047)

- Tổng client conn ≤ **100** với headroom ≥ 20%; verify bằng test tải nhẹ (SC-021).

## Verify

- [ ] Kết nối `amqps` thành công từ Cloud Run.
- [ ] Topology (exchange/queue/DLX) khai báo đúng; quorum hoặc fallback có ADR.
- [ ] Outbox publish + consumer hoạt động; poll fallback khi mất `LISTEN`.
