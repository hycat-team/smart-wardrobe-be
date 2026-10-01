# Ghi chú kỹ thuật

Ghi chú cho những chi tiết DDL và hành vi mà tài liệu kiến trúc không nêu, và
thường là nơi một sửa đổi sai sẽ tạo ra lỗi âm thầm.

---

## `aiexecutor.ai_policy_grants` — bảng cấp quyền AI

### Vì sao nằm ở schema `aiexecutor` mà được `subscription` đọc

Đây là điểm dễ hiểu sai nhất của kiến trúc. Bảng thuộc **sở hữu của `aiexecutor`**,
và `subscription` **không bao giờ ghi** vào nó.

Cột nối `subscription_plans.ai_cost_policy_id` tồn tại để Account **tham chiếu**
tới danh mục chính sách, không phải để Account ghi grant. Sách đúng một sở hữu
viết, và đó là dịch vụ AI, chỉ tiếp cận được qua việc tiêu thụ
`subscription.plan.changed`.

Lý do tách như vậy: một grant sai không chỉ làm hỏng hạn mức, nó làm **sai lệch
tiền**. Vì vậy đường ghi phải là đường duy nhất, có kiểm soát, và có thể truy vết
tới một sự kiện cụ thể.

### Các cột đáng chú ý

| Cột | Vì sao quan trọng |
|---|---|
| `policy_snapshot` | `jsonb NOT NULL`, chứa `envelope` + `entitlements`. Đây là **ảnh chụp đóng băng** lúc đổi gói, không phải cấu hình sống. Đổi chính sách sau này không viết lại lịch sử. |
| `effective_from` / `effective_to` | `effective_to NULL` = không hết hạn. Cột này quyết định vị trí **cửa sổ gia hạn**; ghim sai là grant không bao giờ chuyển sang `future` đúng lúc. |
| `status` | `active` / `future` / `revoked` / `expired`. `future` = kỳ đã trả nhưng chưa bắt đầu; trao quyền sớm ở trạng thái này là cấp quyền chưa mua. |
| `source_event_id` | Khóa chống trùng của phía nhận. Không có nó, một lần giao hóa at-least-once sẽ tạo hai grant cho một lần đổi gói. |
| `plan_id` | `NOT NULL` **không còn foreign key**. Receiver phải tự điền từ `toPlanId` trong payload. |

### Bẫy đã gặp: `policy_snapshot` phải có đủ 4 operation

Reader cần `entitlements.chat`, `.outfit`, `.summary`, `.rewriter`. Thiếu một
operation **không gây lỗi** — giới hạn của nó đơn giản là không tồn tại, và
request vẫn thành công. Đó là loại lỗi đắt nhất: không ai thấy cho tới khi
khách hàng khiếu nại.

Vì vậy `database/scripts/07-verify_policy_grants.sql` mục 4 kiểm tra riêng điều
này, và phải trả về `0`.

---

## `aiexecutor.ai_usage_period_ledgers` — sổ dùng theo kỳ

### `grant_id` là NULLABLE, và đó là điểm cố ý

Một lượt dùng khi tài khoản **không có grant** vẫn phải được ghi vào ledger, với
`grant_id IS NULL`.

Đây là kết quả của thiết kế hai tầng (feature 024), không phải sự lơi trong ràng
buộc. Cơ chế:

1. Tài khoản không có grant → **tầng giải quyết dự phòng** cấp quyền miễn phí.
2. Request vẫn chạy, vẫn trả lời. Ledger ghi `grant_id = NULL`.
3. Báo cáo thấy dòng ledger không gắn với grant nào → **đó chính là tín hiệu cho
   biết dự phòng đang chạy**.

Nếu `grant_id` là `NOT NULL`, các lượt dùng này hoặc bị ghi đè, hoặc làm hỏng
request — và cách duy nhất để "sửa" là viết grant giả, tức là đánh dấu một tài
khoản miễn phí là đang được trả phí.

Hệ quả bắt buộc: mọi câu `INSERT` vào bảng này phải truyền `grant_id` có thể
NULL. Đừng "siết" cột này lại.

### Ràng buộc duy nhất

`UNIQUE (grant_id, period_index)`.

Với `grant_id IS NULL`, PostgreSQL coi NULL là không bằng nhau trong unique index
(thông qua `NULLS DISTINCT` mặc định), nên **nhiều lượt dùng không có grant vẫn
ghi được**. Đây chính là hành vi cần có. Nếu ai đó đổi thành `NULLS NOT DISTINCT`,
các lượt dùng miễn phí thứ hai trở đi sẽ đụng nhau.

### `period_index` không phải số ngày

Nó là **chỉ số kỳ tính phí** (`cost_index`), tính từ grant, không phải ngày.
Cột `period_start` / `period_end` mới là mốc thời gian. Gộp hai khái niệm này là
nguyên nhân của những lỗi trừ tiền sai theo mùa.

---

## Quy trình kiểm chứng (feature 024)

| Bước | Mục đích |
|---|---|
| `database/scripts/05-dry_run_cleanup_policy_grants.sql` | xem trước phạm vi sửa chữa dữ liệu cũ |
| `database/scripts/06-cleanup_policy_grants.sql` | sửa chữa dữ liệu grant sinh ra trước 024 |
| `database/scripts/07-verify_policy_grants.sql` | 6 bất biến, **tất cả phải bằng 0** |
| `POST /api/v1/admin/subscription-grants/restore` | phát lại tín hiệu cho tài khoản thiếu grant |

Script `07` là read-only và idempotent, chạy bao nhiêu lần cũng được. Mục 6b in ra
mọi dòng kèm phán đoán, kể cả hai ngoại lệ có chủ đích, để chúng luôn hiện diện
thay vì bị ẩn đi.

### Hai ngoại lệ có chủ đích ở mục 6

1. `premium-default` là chính sách **strict**: cột `max_paid_attempts_per_day`
   không bao giờ được đọc, nên NULL ở đây là cấu hình chết, không phải lỗi.
2. Gói `free` cố ý thấp hơn đúng một lượt so với con số khách hàng nhìn thấy
   (FR-005d). Bất kỳ chênh lệch **nào khác** là chưa giải thích được và phải điều
   tra.

---

## Lưu ý vận hành

- **`x-queue-type` không đổi được trên queue đã tồn tại.** Queue đang là classic mà
  khai báo quorum sẽ bị RabbitMQ trả `406 PRECONDITION_FAILED` và dịch vụ **không
  khởi động được**. Đây không phải tình huống riêng của
  `subscription_plan_changed_queue` — nó áp dụng cho **mọi** queue khi đổi
  `isQuorum` trong `topology.go`.
  - CI chạy check trước mỗi deploy (bước `Check RabbitMQ queue topology` ở
    `deploy-vps-services.yml` + `deploy-aws.yml`) nên lệch bị bắt trước khi có
    service nào crash-loop trên box.
  - Cách migrate (drain → xoá → khai báo lại, kèm script):
    [`deployments/scripts/rabbitmq-topology/README.md`](../../deployments/scripts/rabbitmq-topology/README.md).
  - Kinh nghiệm thực tế (2026-10-01): `subscription_plan_changed_queue` đã là quorum
    trên VPS nhưng AWS box chưa đồng bộ — quên sync broker là nguyên nhân gốc, và đọc
    log crash-loop từ xa tốn nhiều thời gian hơn hẳn một bước CI đỏ.
- **Sau khi dựng lại container `account` hoặc `commerce`**, nginx giữ DNS cũ và
  mọi route sẽ trả `404` cho tới khi `make nginx-reload-local` hoặc restart
  nginx. Đây là triệu chứng quen thuộc, không phải lỗi route.
