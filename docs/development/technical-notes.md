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

## `subscription.user_plan_periods` — lịch sử gói theo khoảng hiệu lực

### Bảng này là gì

Mỗi dòng là **một khoảng liên tục** mà một người dùng giữ một gói. Bảng do
recorder ghi trong cùng transaction với thao tác đổi gói: đổi gói thêm khoảng
mới và đóng khoảng cũ, còn gia hạn chỉ dời mốc kết thúc. Không có job định kỳ
và không có endpoint tra cứu riêng (feature 031).

- `valid_from` là mốc bắt đầu, **bao gồm**.
- `valid_to` là mốc kết thúc, **không bao gồm** (khoảng nửa mở `[valid_from, valid_to)`).
- `valid_to IS NULL` = khoảng **không có điểm kết thúc** (gói free hoặc lifetime),
  không phải "thiếu dữ liệu" và cũng không phải "đã hết hạn".
- `plan_code`, `plan_name`, `tier_rank`, `plan_kind` là **ảnh chụp** lúc ghi.
  Đổi tên hoặc đổi giá gói sau này không viết lại lịch sử.
- `source_event_key` truy vết về sự kiện đã tạo/cập nhật khoảng; giá trị
  `BACKFILL:<user_id>` là khoảng được backfill lúc deploy.
- `subscription.plan_history_meta` với `id = 'cutover'` cho biết lịch sử đầy đủ
  bắt đầu từ thời điểm nào. Khoảng thời gian trước mốc đó là không đủ dữ liệu,
  không phải "không có user".

### Tra cứu lịch sử của một người dùng (chỉ đọc)

Chạy trên `psql`, chỉ dùng `SELECT`:

```sql
\set user_id '00000000-0000-0000-0000-000000000000'

SELECT plan_code,
       plan_name,
       tier_rank,
       plan_kind,
       valid_from,
       valid_to,
       valid_to IS NULL AS is_open
FROM subscription.user_plan_periods
WHERE user_id = :'user_id'
ORDER BY valid_from;
```

`idx_user_plan_periods_user_valid_from (user_id, valid_from)` phục vụ đúng truy
vấn này: vừa lọc theo user, vừa trả về đúng thứ tự `valid_from`. Luôn giữ điều
kiện `WHERE user_id = ...`; bỏ nó đi là quét toàn bảng.

Muốn thấy khoảng hở hoặc chồng lấn giữa các khoảng:

```sql
SELECT valid_from,
       valid_to,
       valid_from - lag(valid_to) OVER (ORDER BY valid_from) AS gap
FROM subscription.user_plan_periods
WHERE user_id = :'user_id'
ORDER BY valid_from;
```

- `gap = 0`: khoảng sau nối tiếp khoảng trước (đổi gói liên tục).
- `gap > 0`: có khoảng hở — thường là gói hết hạn rồi mua lại, đúng như thực tế.
- `gap < 0`: chồng lấn — bất thường. Với biến thể có `EXCLUDE` thì database đã
  chặn; với biến thể fallback (partial unique chỉ bảo vệ khoảng có
  `valid_to IS NULL`) phải báo ngay để điều tra.

Truy vấn `lag(...)` chỉ nên chạy cho một user; đừng chạy toàn bảng.

Cách đọc `valid_to`:

- `NULL` nghĩa là gói vẫn còn hiệu lực vô thời hạn (free/lifetime). **Không**
  dùng `valid_to < now()` để kết luận hết hạn: NULL không thoả điều kiện đó và
  khoảng sẽ bị bỏ sót.
- Khi cần tính như một mốc, dùng `COALESCE(valid_to, 'infinity'::timestamptz)`.
- Một khoảng **trả phí** có `valid_to IS NULL` là bất thường (trừ khi dữ liệu
  vừa được sửa tay); ghi lại `id`, `user_id`, `source_event_key` và điều tra
  thay vì tự sửa.

### Cảnh báo: không sửa bảng này bằng tay

Không `INSERT`, `UPDATE` hay `DELETE` trực tiếp vào `subscription.user_plan_periods`,
kể cả khi thấy một dòng có vẻ sai:

- Lịch sử được ghi bởi recorder **trong cùng transaction** với thao tác đổi gói.
  Sửa tay phá vỡ tính nguyên tử, tính idempotent và khả năng truy vết về sự kiện.
- Báo cáo phân bố đọc trực tiếp bảng này. Một dòng bị sửa hoặc xoá làm sai số
  liệu của mọi kỳ báo cáo liên quan mà không để lại dấu vết.
- Sửa tay có thể vi phạm `chk_user_plan_periods_valid_range`,
  `ex_user_plan_periods_no_overlap` / `ux_user_plan_periods_open`, hoặc
  `ux_user_plan_periods_source_event_key`; trường hợp xấu hơn là tạo khoảng
  chồng lấn mà biến thể fallback không phát hiện được.
- `plan_history_meta.cutover` do migration ghi. Đổi mốc này làm báo cáo khai báo
  sai mức đầy đủ của lịch sử (complete/partial/unavailable).
- Khi buộc phải sửa dữ liệu, dừng lại, xác định nguyên nhân, rồi làm một
  migration/one-off có review. Chạy lại backfill **không** tái tạo được lịch sử
  đã mất; nó chỉ điền khoảng hiện tại cho user còn thiếu. Mốc `cutover` đã có
  không bị dịch (chỉ khi bảng bị drop và tạo lại thì mốc mới đổi).

### Đo truy vấn tra cứu (2026-10-20)

Đo trên database tạm `verify_plan031` (đã migrate tới 031), dữ liệu tổng hợp
200.010 khoảng của 20.001 user (10 khoảng/user), `ANALYZE` trước khi đo; toàn bộ
dữ liệu được `ROLLBACK` sau khi đo. Truy vấn trả 10 dòng.

Cấu hình planner mặc định của môi trường local (`random_page_cost = 4`, bitmap
scan bật):

```text
Sort  (actual time=0.022..0.023 rows=10 loops=1)
  Sort Key: valid_from
  Buffers: shared hit=4
  ->  Bitmap Heap Scan on user_plan_periods  (actual time=0.015..0.017 rows=10 loops=1)
        Recheck Cond: (user_id = '...'::uuid)
        Heap Blocks: exact=1
        Buffers: shared hit=4
        ->  Bitmap Index Scan on ex_user_plan_periods_no_overlap  (actual time=0.011..0.011 rows=10 loops=1)
              Index Cond: (user_id = '...'::uuid)
              Buffers: shared hit=3
Planning Time: 0.104 ms
Execution Time: 0.059 ms
```

Với 10 dòng kết quả, planner chọn index GiST của `EXCLUDE` constraint (cũng lọc
theo `user_id`) rồi sort; đây là index chỉ tồn tại ở biến thể có `btree_gist`.

Khi planner ưu tiên index scan — cấu hình SSD (`random_page_cost = 1.1`) hoặc
`enable_bitmapscan = off` — index chuyên dụng được chọn và **không cần Sort**:

```text
Index Scan using idx_user_plan_periods_user_valid_from on user_plan_periods
  (cost=0.42..12.69 rows=10 width=55) (actual time=0.007..0.014 rows=10 loops=1)
  Index Cond: (user_id = '...'::uuid)
  Buffers: shared hit=24
Planning Time: 0.099 ms
Execution Time: 0.020 ms
```

Cùng 200k dòng với `enable_bitmapscan = off`: `Execution Time: 0.023 ms`. Kết
luận: `idx_user_plan_periods_user_valid_from` phục vụ đúng truy vấn tra cứu. Ở
biến thể fallback (không có `EXCLUDE`/`btree_gist`), đây là index **duy nhất**
lọc được theo `user_id` — GiST range không dùng được cho điều kiện `user_id =` —
nên truy vấn vận hành phải giữ `WHERE user_id = ...`.

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
