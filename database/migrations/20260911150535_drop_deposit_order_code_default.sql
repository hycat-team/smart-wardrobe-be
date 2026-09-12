-- +goose Up
-- +goose StatementBegin
-- PayOS orderCode chuyển sang do application sinh (unixMillis*1000+rand, xem
-- internal/modules/subscription/application/ordercode). Cột order_code giữ kiểu
-- BIGINT UNIQUE nhưng không còn DEFAULT nextval, để mọi đơn mới bắt buộc mang mã
-- entropy cao, không bao giờ quay lại dãy BIGSERIAL 1,2,3... mà PayOS đã ghi nhận
-- (lỗi business 231 "Đơn thanh toán đã tồn tại" sau mỗi lần reset DB).
-- Migration thuần schema (không đổi dữ liệu) → viết Down đầy đủ.
ALTER TABLE subscription.deposit_transactions ALTER COLUMN order_code DROP DEFAULT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE subscription.deposit_transactions ALTER COLUMN order_code SET DEFAULT nextval('subscription.deposit_transactions_order_code_seq'::regclass);
-- +goose StatementEnd
