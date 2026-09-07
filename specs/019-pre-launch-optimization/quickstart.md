# Quickstart: Validation Guide — Tối Ưu Hóa Pre-Launch (019)

Hướng dẫn chạy xác minh end-to-end sau khi implement. Mỗi section ánh xạ 1 user story trong [spec.md](spec.md). Chi tiết kỹ thuật xem [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/http-behavior.md](contracts/http-behavior.md).

## Prerequisites

- Go 1.25+, `make dev` đang chạy (Postgres/Redis/RabbitMQ qua docker) hoặc môi trường test tương đương.
- Biến môi trường integration: `TEST_DATABASE_URL` (Postgres test) — các test integration tự skip nếu thiếu.
- Swagger đã regenerate: `make generate`.

## US1 — HTTP Hardening (timeouts + body limit)

```bash
# 1. Body-limit: vượt ngưỡng tầng mặc định (2 MB) → 413 chuẩn
python -c "print('x'*3*1024*1024)" > /tmp/big.json   # hoặc head -c 3M /dev/urandom
curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/api/v1/me/avatar \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" --data @/tmp/big.json
# Expected: 413

# 2. Body 1.5 MB vào route tầng nâng (vd chat AI) → KHÔNG 413 (đi tiếp, lỗi nghiệp vụ khác nếu payload rác)
# Expected: không phải 413

# 3. Slowloris mô phỏng (mở kết nối, không gửi gì):
#    dùng test unit trong body_limit_test/timeout hoặc telnet giữ kết nối
# Expected: kết nối bị đóng sau ~30s (read timeout), log có cấu trúc xuất hiện

# 4. SSE dài không bị cắt bởi write timeout:
#    chạy stream chat (phân tích nhiều item) kéo dài > 60s
# Expected: stream vẫn truyền dữ liệu liên tục qua mốc 60s (handler tự gia hạn deadline), không bị đóng giữa chừng
```

Kiểm chứng tự động: `go test -race ./internal/shared/presentation/middleware/...` pass (unit 413 + tier phân route + free registry).

## US2 — Billing/Quota/RBAC test suite

```bash
make test-race          # toàn bộ unit mới + cũ pass
make test-integration   # webhook reconciliation + renewal integration pass (cần TEST_DATABASE_URL; tự skip khi thiếu)
```

Expected: mọi test xanh; ma trận kịch bản khớp [data-model.md §5](data-model.md). Không sửa code production nào để test pass.

## US3 — Cache tổng quan gói cước

```bash
# 1. Gọi 5 lần liên tiếp API AI (vd outfit-recommendations) với cùng user
# 2. Quan sát log invalidation/hit theo contracts/http-behavior.md §5
# 3. Test unit: subscription cache hit/miss/fallback + invalidation tại mutation points
go test -race ./internal/modules/subscription/...
```

Expected: chỉ ≤1 lượt truy vấn overview trong cửa sổ TTL; quota vẫn ghi mỗi request; mua/gia hạn xong → cache bị xóa ngay (log invalidation); tắt Redis → request AI vẫn hoạt động.

## US4 — Gzip + Migration đa instance

```bash
# Gzip:
curl -s -H "Accept-Encoding: gzip" -o /tmp/resp.gz -w "%{size_download}\n" http://localhost:8080/api/v1/admin/dashboard/overview -H "Authorization: Bearer $ADMIN_TOKEN"
curl -s -o /tmp/resp.raw -w "%{size_download}\n" http://localhost:8080/api/v1/admin/dashboard/overview -H "Authorization: Bearer $ADMIN_TOKEN"
# Expected: size gzipped ≤ 50% raw; giải nén ra JSON khớp 100% (diff)
# SSE không nén: stream endpoint trả Content-Encoding rỗng

# Migration 2 instance đồng thời:
#   chạy 2 tiến trình server khởi động cùng lúc (hoặc test integration migration)
# Expected: không xung đột schema; log cho thấy lock được thực thi theo thứ tự
```

## US5 — Hygiene pass

```bash
wc -l internal/modules/wardrobe/presentation/handler/wardrobe_item*.go internal/modules/wardrobe/presentation/handler/wardrobe_catalog*.go
wc -l internal/modules/subscription/application/usecase/subscription/subscription_renewal*.go
grep -rn "fmt.Println" internal --include="*.go" | grep -v _test
make generate && git diff --stat api/swagger   # swagger chỉ phi phá vỡ
```

Expected: mọi file tách < 500 dòng, hàm < 80 dòng; `grep fmt.Println` trả rỗng (production code); `make test-race` pass sau tách (pure refactor).

## Gate cuối (SC-008)

```bash
git diff api/swagger/swagger.json | grep -E '^[-+]\s+"' | grep -vE 'definitions|(\$ref)'
# Expected: rỗng — không field nào của API công khai bị xóa/đổi
```
