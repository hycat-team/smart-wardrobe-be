# Hướng dẫn kiểm chứng nhanh — 027 Sửa tắc hai nguyên nhân chặn vĩnh viễn

**Ngày**: 2026-10-07 | **Kế hoạch**: [plan.md](plan.md)

Cách chứng minh tính năng chạy đúng, không lặp lại mã nguồn. Chi tiết cửa sổ ở
[contracts/grant-window-resolution.md](contracts/grant-window-resolution.md), phân loại lỗi ở
[contracts/error-classification.md](contracts/error-classification.md).

## Yêu cầu trước

```bash
cd <gốc repo>
make lint                 # bắt buộc: hiến pháp §Code Style
```

Postgres thật là bắt buộc cho K3 — xem [contracts/error-classification.md](contracts/error-classification.md)
§4. Chỉ dùng Postgres thật, **không** dùng mock cho K3.

---

## K1 — Cửa sổ không đổi khi sự kiện mới hơn (FR-007)

Kịch bản cơ sở của FR-007: khi sự kiện đã mới hơn mọi bản ghi trước, hệ thống phải **không
làm gì cả**. Nếu K1 đỏ thì mọi thay đổi của 027 đang lấy đi một đường không cần đi.

```bash
cd services/ai
go test ./internal/modules/aiexecutor/application/usecase/grant/ -run 'TestGrantWindow|TestResolveGrantWindow' -v
```

| Trường hợp | Kết quả mong đợi |
| --- | --- |
| Không có bản ghi trước | `EffectiveFrom == OccurredAt`, không dịch |
| Sự kiện mới hơn mọi bản ghi | `EffectiveFrom == OccurredAt`, **không dịch** |
| Nhiều bản ghi, sự kiện mới hơn tất cả | `EffectiveFrom == OccurredAt`, không dịch |

Sự kiện `EXTENDED` phải cho **cùng** kết quả: mốc bắt đầu vẫn là
`EffectiveTo − DurationDays`, không phải `OccurredAt`.

---

## K2 — Dịch đúng 1 microsecond khi sự kiện đến muộn (FR-004, FR-006, FR-008)

Đây là thay đổi cốt lõi. So sánh **trước** và **sau** trên cùng một tình huống.

| Trường hợp | `floor` | Kết quả mong đợi |
| --- | --- | --- |
| Sự kiện mới hơn | bất kỳ | Không dịch |
| Sự kiện kém hơn 1 µs | `s + 100µs` | `EffectiveFrom == s + 101µs` |
| Sự kiện mới hơn `floor` nhưng kém mốc gốc | `s + 100µs`, mốc gốc `s + 50µs` | `EffectiveFrom == s + 100µs` |
| Lệch trên 1 µs | `s + 5s` | `EffectiveFrom == s + 5s + 1µs` |
| `started_at` ở tương lai | dùng `effective_from` | Lùi về `effective_from`, bỏ qua `started_at` |
| `started_at` bằng 0 | dùng `effective_from` | Lùi về `effective_from` |

Hai hàng cuối là toàn bộ R-003. Nếu chúng đỏ thì `started_at` tương lai vẫn đang đóng vai
đồng hồ quyền hạn — và ADR 014 đã nói rõ đó là điều không được chấp nhận.

**Không hợp lệ** — phải là từ chối vĩnh viễn, mã `stale_event`:

| Trường hợp | Kết quả mong đợi |
| --- | --- |
| `OccurredAt == floor` | Từ chối — FR-002 |
| `OccurredAt < floor` | Từ chối — FR-002, bảo vệ FR-033 của spec 024 |
| Dịch xong `EffectiveFrom ≥ EffectiveTo` | Từ chối — cửa sổ rỗng |

```bash
cd services/ai
go test ./internal/modules/aiexecutor/application/usecase/grant/ -run TestResolveGrantWindow -v
go test ./internal/modules/aiexecutor/application/usecase/grant/ -race -count=3
```

---

## K3 — `23514` là từ chối vĩnh viễn, không phải thử lại (FR-009, FR-010)

**Cần Postgres thật.** Xem lý do bắt buộc ở
[contracts/error-classification.md](contracts/error-classification.md) §4.

```bash
cd services/ai
go test ./internal/modules/aiexecutor/infrastructure/persistence/ -run TestCheckViolation -v
```

| Cần chứng minh | Vì sao |
| --- | --- |
| `errors.As` bóc được `*pgconn.PgError` **qua GORM** | Nếu không, phân loại hỏng âm thầm và ta quay lại lỗi gốc |
| `23514` thật sinh ra khi `effective_to == effective_from` | Tránh `PgError` dựng tay — nó không chứng minh được điều gì |
| `23505` thật **không** bị xếp vĩnh viễn | FR-010; đây là hồi quy dễ gây nhất |
| `40001` deadlock **không** bị xếp vĩnh viễn | FR-010 |

Sau khi tidy:

```bash
cd services/ai
go mod tidy && git diff --stat go.mod go.sum
```

`pgx/v5` phải chuyển từ `// indirect` sang phụ thuộc trực tiếp. Nếu nó vẫn ghi `// indirect`
sau khi mã đã dùng `pgconn`, nghĩa là mã đang import sai đường.

---

## K4 — Bộ đếm công khai có mã lý do mới (FR-011)

```bash
cd services/ai
go test ./internal/modules/aiexecutor/application/usecase/grant/ -run TestGrantMetrics -v
```

| Cần chứng minh | Vì sao |
| --- | --- |
| `policy_grant_constraint_violation` xuất hiện trong `grant_rejected_by_reason` | FR-011 |
| Mỗi lần xảy ra ghi **đúng một** dòng cảnh báo | FR-011 |
| Dòng cảnh báo mang `user_id` | `grant_reject_metrics.go:33` — đây là điều 024 T114 đã chốt |
| Không có ngưỡng cảnh báo nào cho mã mới | FR-011 nói rõ **không** tự đặt ngưỡng |

### Đóng CHK026 bằng chứng, không bằng cảm giác

R-004 kết luận mã mới là **tương thích ngược**, nên không cần `frontend-guide.md`. Kết luận
đó chỉ đúng nếu nhãn của bộ đếm là `string` tự do:

```bash
# Xác nhận nhãn là string tự do, KHÔNG phải kiểu liệt kê đóng
cd services/ai
Select-String -Path internal/modules/aiexecutor/presentation/**/health*.go -Pattern 'grant_rejected_by_reason'
```

Nếu phát hiện một danh sách giá trị đóng được sinh từ hằng, hoặc một kiểu liệt kê, thì
R-004 **sai**: CHK026 mở lại và `frontend-guide.md` trở thành bắt buộc.

---

## K5 — Không có tác dụng phụ khi từ chối (FR-012)

```bash
cd services/ai
go test ./internal/modules/aiexecutor/application/usecase/grant/ -run 'TestReject|TestStale' -v
```

Mỗi lần từ chối phải để lại bảng `ai_policy_grants` **y hệt** trước khi gọi: không dòng
mới, không dòng bị cắt. Đây là điều 024 T163 đã chốt và 027 giữ nguyên.

---

## K6 — Kiểm tra toàn vẹn: mục 4 đếm đúng phạm vi (FR-014, FR-015, FR-016)

Trước khi sửa 8 tài khoản, mục 4 phải **không** về 0 — 22 bản ghi có ảnh chụp không đọc
được, và 8 trong số đó đang hiệu lực.

```bash
psql -U closy -d closy_db -q -f database/scripts/07-verify_policy_grants.sql
```

| Mục | Trước khi sửa | Sau khi sửa |
| --- | --- | --- |
| 4. Ảnh chụp chính sách rỗng hoặc thiếu | **khác 0** | **0** |
| Truy vấn thông tin (mới) | khác 0 | khác 0 — **giữ nguyên** |

Hai dòng cuối là kiểm tra chống gian lận của chính bản sửa: nếu mục 4 về 0 **và** số thông
tin cũng về 0, có nghĩa là dữ liệu đã bị xoá thay vì được đóng lại — đúng thứ FR-016 cấm.

---

## K7 — Sửa dữ liệu thật: CHẠY SAU khi triển khai

Không phải kiểm thử — đây là thao tác vận hành, và nó **chỉ chạy được sau khi K1..K6 đã
deploy**. Toàn bộ quy trình nằm ở [runbook.md](runbook.md).

Cổng chặn bắt buộc trước khi gọi API sửa:

```bash
# Phải in ra false. Nếu in rỗng thì bước tiếp theo sẽ gửi email thật cho khách.
docker compose --env-file .env.production --env-file .env.deploy \
  -f deployments/docker-compose.prod.yml \
  exec -T account printenv EMAIL_ENABLED
```

Sau khi chạy xong:

```bash
psql -U closy -d closy_db -q -f database/scripts/07-verify_policy_grants.sql   # mục 4 = 0

# Không được có dòng nào trả về
psql -U closy -d closy_db -q -c "
SELECT user_id, count(*) AS n
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active','future')
  AND effective_from <= now() AND (effective_to IS NULL OR effective_to > now())
GROUP BY user_id HAVING count(*) > 1;"
```

Truy vấn thứ hai là kiểm tra trực tiếp FR-008: không có tài khoản nào giữ hai cửa sổ đang
hiệu lực chồng lấn.

---

## Cổng chặn trước khi triển khai

| Cổng | Lệnh | Điều kiện |
| --- | --- | --- |
| Lint | `make lint` | Sạch |
| Kiểm thử gói đã đụng | `go test -race -count=3` trong `services/ai/...` | Xanh |
| Phụ thuộc | `go mod tidy` rồi `git diff` | `pgx/v5` là trực tiếp |
| Toàn bộ | `make check` | Xanh — bắt buộc trước khi push hoặc tag |

Theo [AGENTS.md](../../../AGENTS.md) §2: `make lint` cộng kiểm thử gói đã đụng là đủ cho một
lần sửa một gói; `make check` đầy đủ là bắt buộc trước khi push hoặc tag.