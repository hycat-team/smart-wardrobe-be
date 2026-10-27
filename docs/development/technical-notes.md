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

### Đo báo cáo phân bố trên 200k khoảng (SC-004, 2026-10-20)

Đo trên database tạm `verify_plan031` (PostgreSQL 17.10, đã migrate tới 031).
Cấu hình planner mặc định của môi trường local: `random_page_cost = 4`,
`enable_bitmapscan = on`, `work_mem = 4MB`, `shared_buffers = 128MB`.

Dữ liệu đo nằm trong schema dùng một lần `perf031`, không đụng bảng thật:
bảng cùng shape với `subscription.user_plan_periods` nhưng **không** có
`EXCLUDE` constraint — mục đích là cô lập đúng `idx_user_plan_periods_range`
(GiST range index có mặt ở **cả hai** biến thể migration; index của EXCLUDE chỉ
có khi `btree_gist` khả dụng). Seed 200.000 khoảng của 200.000 user tổng hợp,
`valid_from` rải trong 365 ngày gần nhất, mỗi khoảng đóng dài 30 ngày, cứ 25
dòng thì một dòng `valid_to IS NULL` (free/lifetime), sau đó `ANALYZE`.
Mỗi truy vấn chạy **50 lần** trong một session; p95 lấy bằng
`percentile_cont(0.95) WITHIN GROUP (ORDER BY ms)`.

Kỳ đo là cửa sổ 30 ngày `[now() - 30 days, now())`; "snapshot" là
`tstzrange(valid_from, valid_to, '[)') @> now()`. Kết quả 200k:

| Shape | Interval khớp | p50 (ms) | **p95 (ms)** | max (ms) |
|---|---:|---:|---:|---:|
| cả hai bound (kỳ 30 ngày) | 39.565 | 34,58 | **37,47** | 43,46 |
| chỉ `from` | 39.565 | 35,41 | **36,14** | 36,65 |
| snapshot (`@> now()`) | 23.782 | 20,43 | **21,31** | 21,53 |
| chỉ `to` (toàn bảng) | 200.000 | 195,36 | **199,63** | 201,70 |

`EXPLAIN (ANALYZE, BUFFERS)` của kỳ 30 ngày xác nhận GiST range index:

```text
GroupAggregate  (actual time=39.891..43.193 rows=4 loops=1)
  ->  Sort  (actual time=38.708..40.108 rows=39565 loops=1)
        Sort Key: plan_code, user_id
        ->  Bitmap Heap Scan on user_plan_periods  (actual time=5.808..14.408 rows=39565 loops=1)
              Recheck Cond: (tstzrange(valid_from, valid_to, '[)'::text)
                             && tstzrange((now() - '30 days'::interval), now(), '[)'::text))
              ->  Bitmap Index Scan on idx_user_plan_periods_range  (actual time=5.380..5.380 rows=39565 loops=1)
                    Index Cond: (tstzrange(valid_from, valid_to, '[)'::text)
                                 && tstzrange((now() - '30 days'::interval), now(), '[)'::text))
Execution Time: 43.477 ms
```

Snapshot cũng dùng đúng index này (`Bitmap Index Scan on
idx_user_plan_periods_range`, 23.782 dòng, `Execution Time: 24.944 ms`).
Riêng shape "chỉ `to`" cố ý **không** dùng index: predicate `(-∞, now())`
khớp toàn bộ 200.000 dòng nên planner chọn `Seq Scan` (227 ms) — index không
thể thu hẹp khi mọi dòng đều thoả.

**Bố cục worst-case lúc deploy** (200.000 khoảng của 200.000 user, *tất cả*
giao kỳ 30 ngày): p50 223,43 ms, **p95 229,09 ms**, max 235,57 ms; snapshot p95
194,07 ms. `EXPLAIN` cho thấy `Seq Scan` + `GroupAggregate` + `Sort` (predicate
khớp 100% bảng nên index không thể thu hẹp); `Execution Time: 236.498 ms`. Dù
vậy vẫn thấp hơn ngưỡng 1 giây hơn 4 lần. Đây là hình dạng đúng của lần deploy
đầu (mỗi user đúng một cửa sổ hiện tại), nên SC-004 không chỉ đạt ở layout
"rải 365 ngày" mà cả ở worst-case.

**1 triệu khoảng (tham khảo)**: thêm 800.000 dòng cùng layout rồi đo lại 50
lần. Kỳ 30 ngày khớp 197.823 khoảng: p50 236,93 ms, **p95 395,59 ms**, max
533,64 ms; snapshot p95 164,73 ms. `EXPLAIN` vẫn là
`Bitmap Index Scan on idx_user_plan_periods_range` (211.902 dòng index,
`Execution Time: 252.158 ms`, có đọc đĩa lần đầu). Không có dấu hiệu plan
"xẹp" sang sequential scan khi dữ liệu tăng 5 lần.

**Kết luận SC-004: ĐẠT.** Với 200.000 khoảng, p95 của báo cáo kỳ 30 ngày là
**37,47 ms**, thấp hơn ngưỡng 1.000 ms khoảng 26 lần; ngay cả worst-case 200k
(tất cả khoảng giao kỳ, `Seq Scan` toàn bảng) cũng chỉ **229,09 ms**. Ở 1 triệu
khoảng, p95 vẫn dưới 1 giây (395,59 ms). Sau khi đo, schema `perf031` được
`DROP ... CASCADE`; `subscription.user_plan_periods` trở về 0 dòng, database
không còn dữ liệu seed.

---

## p95 của endpoint gợi ý phối đồ (033 SC-010) — **FAIL trên AI path, PASS trên fallback**

### Số đo 2026-10-27

Môi trường: `verify_033` (init + migrate 3 service), account 8081, ai 8082, commerce 8080, 5 active
item (ao/quan clone), plan `premium-monthly` (`personalization_enabled=true`), catalog footwear chưa
sở hữu. Embedding provider hoạt động.

| Path | n | p50 | p95 | p99 | max | Kết luận |
|---|---|---|---|---|---|---|
| HSL fallback (`provider_error`) | 50 | 563.4 ms | **673.3 ms** | 766.8 ms | 766.8 ms | **PASS** |
| AI `gemini-2.5-flash-lite` | 50 | 4903.7 ms | **21686.1 ms** | 24526.1 ms | 24526.1 ms | **FAIL** |
| AI `gemini-3.5-flash-lite` | 50 | 18604.7 ms | **50044.0 ms** | 60576.9 ms | 60576.9 ms | **FAIL** |

**Kết luận SC-010**: **FAIL trên AI path** (p95 21.7 s với `gemini-2.5`, 50.0 s với `gemini-3.5`);
fallback HSL PASS (0.67 s). Nguyên nhân là latency provider/model, không phải pipeline 033
(embedding + retrieval + ranking ~0.5 s). Hai hướng xử lý cần quyết định:

- đặt timeout recommendation text ~4 s để fallback HSL trong budget (sẽ mất AI path khi provider chậm);
- hoặc dùng provider/model nhanh hơn, rồi đo lại ở production.

**Quyết định 2026-10-27**: giữ timeout 30 s, chấp nhận SC-010 fail trên AI path; fallback PASS;
đo lại ở production khi provider/model tốt hơn.

`gemini-3.5-flash-lite` chậm hơn `gemini-2.5-flash-lite` trong môi trường này. Row pricing
`gemini-3.5-flash-lite` đã được thêm vào seed AI baseline; DB đã migrate trước đó cần data patch
(xem `specs/033-personalized-styling/deferred.md` D-11/D-12).

### Lệnh tái lập

```bash
# verify_033: database/init/*.sql + make migrate-up svc=account|commerce|ai (DB_NAME=verify_033)
# start bin/account.exe (8081), bin/ai.exe (8082), bin/commerce.exe (8080) với APP_MODE=all
# seed user premium + 5 active item (catalog-init ao/quan)
psql "$DATABASE_URL" -c "UPDATE subscription.user_daily_quotas SET outfit_recommend_count=0, last_reset_date=now() WHERE user_id='<id>';"
# 50 request, reset sau 25; đo Stopwatch; p95 = sorted[ceil(0.95*n)-1]
```

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
