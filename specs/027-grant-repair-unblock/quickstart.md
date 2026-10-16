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
go test ./internal/modules/aiexecutor/application/usecase/grant/ -v
```

Chạy **cả gói** chứ không `-run` một mẫu. Tên hàm cụ thể dễ trôi, và một `-run` không khớp gì
sẽ in `[no tests to run]` rồi thoát 0 — một cổng chạy xanh trên không có gì, đúng kiểu sai
mà §6 của skill gọi là "xanh giả". Tên hàm đầy đủ nằm trong `tasks.md`.

| Trường hợp | Kết quả mong đợi |
| --- | --- |
| Không có bản ghi trước | cửa sổ gốc, không dịch |
| Sự kiện mới hơn mọi bản ghi | cửa sổ gốc, **không dịch** |
| Nhiều bản ghi, sự kiện mới hơn tất cả | cửa sổ gốc, không dịch |

Ba hàng này kiểm **việc không dịch**, nên so `EffectiveFrom` với đầu ra của `EventGrantWindow(event)` chứ **không** với `OccurredAt`. Hai giá trị đó chỉ bằng nhau ở nhánh thường; ở nhánh `EXTENDED` chúng khác nhau, và một kỳ vọng viết nhầm thành `OccurredAt` sẽ đỏ ngay cả khi mã đúng.

Sự kiện `EXTENDED` phải cho **cùng** kết quả: mốc bắt đầu vẫn là
`EffectiveTo − DurationDays`, không phải `OccurredAt`.

---

## K2 — Dịch đúng 1 microsecond quanh sàn khi sự kiện đến muộn (FR-004, FR-006, FR-008)

Đây là thay đổi cốt lõi. So sánh **trước** và **sau** trên cùng một tình huống.

Nhớ ký hiệu: `sàn_cũ` chỉ chống hồi sinh, `sàn_mới = max(sàn_cũ, prior.effective_from)` mới
là chỗ để dịch. Bảng dưới đây dùng `sàn_mới` làm mốc so sánh.

`sàn_cũ` có **ba mức**, theo thứ tự ưu tiên: `occurred_at` đọc được từ ảnh chụp → `created_at`
khi đọc được và không ở tương lai → `effective_from`. Hai mức lùi phát hai dòng cảnh báo
**khác nhau** (FR-003 cho `created_at`, FR-018 cho `effective_from`).

| Trường hợp | `sàn_mới` | Kết quả mong đợi |
| --- | --- | --- |
| Sự kiện mới hơn | bất kỳ | Không dịch — cửa sổ gốc giữ nguyên |
| Sự kiện kém hơn 1 µs | `s + 100µs` | `EffectiveFrom == s + 101µs` |
| Sự kiện mới hơn sàn nhưng kém mốc gốc | `s + 100µs`, mốc gốc `s + 50µs` | `EffectiveFrom == s + 100µs` |
| Lệch trên 1 µs | `s + 5s` | `EffectiveFrom == s + 5s + 1µs` — tổng dịch 5 s, **vượt quanh sàn chỉ 1 µs** |
| Ảnh chụp đọc được `occurred_at` | lấy từ `occurred_at` | Sàn là thời điểm **nghiệp vụ**; **không** có dòng cảnh báo nào |
| Ảnh chụp không đọc được | lùi `created_at` | Lùi về `created_at`, **có** dòng cảnh báo FR-003 |
| `created_at` ở tương lai | dùng `effective_from` cho sàn cũ | Lùi về `effective_from`, **có** dòng cảnh báo FR-018 |
| `created_at` bằng 0 | dùng `effective_from` cho sàn cũ | Lùi về `effective_from`, **có** dòng cảnh báo FR-018 |

**Hàng hồi quy cho lỗi từ chối nhầm** — lý do `occurred_at` đứng đầu ba mức:

| Trường hợp | Kết quả mong đợi |
| --- | --- |
| Bản ghi mới nhất có `occurred_at = T1`, `created_at = T1+10s`; sự kiện mới có `OccurredAt = T1+5s` | **Được chấp nhận**, không bị từ chối `stale_event` |

Sự kiện mới xảy ra sau sự kiện cũ (`T1+5s > T1`) nên nó hợp lệ. Nếu sàn lấy từ `created_at`
(`T1+10s`) thì `T1+5s ≤ T1+10s` và nó bị từ chối oan. Đây là lỗi mà việc ưu tiên `occurred_at`
loại bỏ, và là ca kiểm thử bắt buộc của T035.

Hàng thứ tư là nơi dễ đọc sai nhất: tổng dịch là 5 giây, nhưng FR-006 đo **độ dư quanh
sàn**, và độ dư đó là 1 µs. Nếu ai đó đọc FR-006 là "tổng dịch tối đa 1 µs" thì sẽ đi tìm
một cách sửa mà không tồn tại, vì cách đó phá FR-004.

**Hàng cốt lõi của cả đợt này** — bản ghi `tương lai`:

| Trường hợp | `created_at` | `effective_from` | `sàn_mới` | Kết quả mong đợi |
| --- | --- | --- | --- | --- |
| Bản ghi `tương lai` 30 ngày nữa | quá khứ | `t + 30d` | `t + 30d` | `EffectiveFrom == t + 30d + 1µs`, bản ghi `tương lai` **bị cắt**, **không** chồng lấn |

Nếu hàng này đỏ, đó chính là lỗi H2 của vòng phân tích: sàn lấy từ `created_at` một mình sẽ
để bản ghi mới bắt đầu **trước** bản ghi `tương lai`, và `TruncateGrantsBefore` không cắt
được dòng đó. Đây là kiểm thử bắt buộc, không phải kiểm thử bổ sung.

**Không hợp lệ** — phải là từ chối vĩnh viễn, mã `stale_event`:

| Trường hợp | FR | Kết quả mong đợi |
| --- | --- | --- |
| `OccurredAt == sàn cũ` | FR-002 | Từ chối |
| `OccurredAt < sàn cũ` | FR-002 | Từ chối — bảo vệ FR-033 của spec 024 |
| Dịch xong `EffectiveFrom ≥ EffectiveTo` | FR-017 | Từ chối — cửa sổ rỗng |

```bash
cd services/ai
go test ./internal/modules/aiexecutor/application/usecase/grant/ -run TestResolveGrantWindow -v
go test ./internal/modules/aiexecutor/application/usecase/grant/ -race -count=3
```

---

## K3 — `23514` là từ chối vĩnh viễn, không phải thử lại (FR-009, FR-010)

**Cần Postgres thật.** Xem lý do bắt buộc ở
[contracts/error-classification.md](contracts/error-classification.md) §4.

> **Cảnh báo về lược đồ kiểm thử**: thực thể `AIPolicyGrant`
> (`domain/entities/policy.go:52`) **không có** thẻ `check:` trên `EffectiveTo`, nên
> `AutoMigrate` trong `openGrantTestDB` **không** tạo ràng buộc mà production có — ràng buộc
> kiểm tra cửa sổ hiệu lực trên production (hiện tên `ai_policy_grants_effective_window_check`;
> trước spec 030 nó mang tên tự sinh `user_ai_policy_grants_check` từ migration
> `20260622131631_add_ai_cost_policy.sql` của dịch vụ Account) vẫn phải được kiểm thử **tự
> thêm** bằng `ALTER TABLE ... ADD CONSTRAINT ... CHECK (...)` trước khi ép vi
> phạm; nếu không, `UPDATE` sẽ thành công và kiểm thử sẽ đỏ vì lý do sai — hoặc tệ hơn,
> một biến thể lười sẽ xoá phần kiểm tra đó và báo "đã đúng".

```bash
cd services/ai
go test ./internal/modules/aiexecutor/application/usecase/grant/ -v
```

Chạy **cả gói**, không `-run` một mẫu. Ba hàm kiểm thử của T016 là
`TestIsCheckViolationThroughGorm`, `TestUniqueViolationStaysRetryable` và
`TestDeadlockStaysRetryable`; một `-run TestIsCheckViolation` chỉ khớp hàm đầu và bỏ qua hai
hàng còn lại — tức bảng dưới đây sẽ có hai hàng không được chạy mà vẫn trông như xanh.

| Cần chứng minh | Vì sao |
| --- | --- |
| `errors.As` bóc được `*pgconn.PgError` **qua GORM** | Nếu không, phân loại hỏng âm thầm và ta quay lại lỗi gốc |
| `23514` thật sinh ra khi `effective_to == effective_from` | Tránh `PgError` dựng tay — nó không chứng minh được điều gì |
| `23505` thật **không** bị xếp vĩnh viễn | FR-010; đây là hồi quy dễ gây nhất |
| `40001` deadlock thật **không** bị xếp vĩnh viễn | FR-010 |

> **Hai hàng cuối phải là Postgres thật, không phải unit test.** Một `PgError` dựng tay cho
> `23505` sẽ đỏ xanh mà không chứng minh được đường đi qua GORM — đúng cái bẫy mà R-001 cảnh
> báo. Ép `unique_violation` thật bằng cách chèn hai dòng trùng khoá, và deadlock thật bằng
> hai transaction gọi `pg_advisory_xact_lock` theo thứ tự ngược. Nếu bạn không dựng được
> deadlock thật trong thời gian hợp lý, ghi việc đó vào `deferred.md` và **không** tick hai
> hàng này — một hàng kiểm thử báo xanh trên bằng chứng yếu là nguy hiểm hơn một hàng còn
> trống.

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
go test ./internal/modules/aiexecutor/application/usecase/grant/ -v
```

Chạy **cả gói**. Một `-run TestGrantMetrics` chỉ khớp các hàm `TestGrantMetrics*` **cũ** và sẽ
bỏ qua hai hàm mới của T018 (`TestConstraintViolationReasonRecorded`,
`TestConstraintViolationIsNotRequeued`) — tức là bảng dưới đây có bốn hàng không được chạy mà
vẫn trông như xanh. Đây đúng là lỗi đã xuất hiện ở K3.

| Cần chứng minh | Vì sao |
| --- | --- |
| `policy_grant_constraint_violation` xuất hiện trong `grant_rejected_by_reason` | FR-011 |
| Mỗi lần xảy ra ghi **đúng một** dòng cảnh báo | FR-011 |
| Dòng cảnh báo mang `user_id` | `grant_reject_metrics.go:33` — đây là điều 024 T114 đã chốt |
| Không có ngưỡng cảnh báo nào cho mã mới | FR-011 nói rõ **không** tự đặt ngưỡng |
| Lỗi trả về là `sharedevent.PermanentError`, **không phải** `RetryableError` | SC-004 — số lần thử lại bằng 0 |
| `startConsumerLoop` **không** requeue với lỗi này, rơi thẳng vào nhánh ACK/DLQ | SC-004 — "đi thẳng vào hàng đỗi chết" |
| `transient_failure` vẫn là `RetryableError` và vẫn requeue | SC-004 nghịch đảo + FR-010 |
| Mọi đường từ chối đều ghi đủ `user_id` + mã lý do + thông điệp nguyên nhân | SC-006 — "0 thư chết mà không có dấu vết" |

Hai hàng cuối là nơi SC-004 và SC-006 thật sự được kiểm chứng. Bộ đếm bảo đảm **một** lần
ghi; nó không bảo đảm thông điệp **không** bị thử lại, và cũng không bảo đảm thư chết có dấu
vết. Nếu chỉ kiểm bộ đếm thì hai tiêu chí thành công đó không được đo.

### Đóng CHK026 bằng chứng, không bằng cảm giác

R-004 kết luận mã mới là **tương thích ngược**, nên không cần `frontend-guide.md`. Kết luận
đó chỉ đúng nếu nhãn của bộ đếm là `string` tự do:

```bash
cd services/ai
Select-String -Path internal/modules/aiexecutor/application/usecase/grant/grant_reject_metrics.go `
  -Pattern 'func \(m \*GrantMetrics\) Snapshot'
Select-String -Path internal/bootstrap/router.go -Pattern 'grant_rejected_by_reason'
```

**Kết quả mong đợi — và đã kiểm chứng:** `Snapshot() map[string]int64`, và
`"grant_rejected_by_reason": grantMetrics.Snapshot()`. Khoá là `string` tự do, **không** phải
kiểu liệt kê đóng, nên một khoá mới chỉ xuất hiện khi từ chối thật xảy ra và không làm vỡ
client đọc theo khoá.

Vì vậy T023 chỉ còn việc **xác nhận lại trên mã sau thay đổi**, và CHK026 đóng được. Nếu ai
đó sau này đổi `Snapshot()` sang kiểu đóng thì mới phải mở lại và sinh `frontend-guide.md`.

---

## K5 — Không có tác dụng phụ khi từ chối (FR-012)

```bash
cd services/ai
go test ./internal/modules/aiexecutor/application/usecase/grant/ -v
```

Chạy **cả gói**, không `-run` một mẫu: các ca từ chối của đường mới (FR-002, FR-017) nằm trong
`TestResolveGrantWindow` của T005, không nằm trong các hàm `TestReject*`/`TestStale*` cũ.

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

**Giá trị đo được trên cơ sở dữ liệu dev thật** (chạy ngày 2026-10-07, chỉ đọc): mục 4 =
**8**, truy vấn thông tin = **14**, tổng **22** — khớp chính xác số bản ghi mà sự cố để lại, và
8 đúng bằng số đang hiệu lực. Đây là bằng chứng rằng FR-014 thu hẹp đúng phạm vi và FR-016
giữ phần lịch sử hiển thị thay vì xoá nó.

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