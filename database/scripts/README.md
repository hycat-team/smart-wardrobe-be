# Script dữ liệu — hướng dẫn dùng

Thư mục này chứa các script SQL chạy tay, dùng để **chuẩn bị dữ liệu**, **sửa chữa
dữ liệu cũ** và **kiểm chứng** trong môi trường phát triển.

> ⚠️ **Không dùng các script này trên production.** Ở production, schema do
> migration quản lý (`services/*/database/migrations`, chạy qua `make
> migrate-up`). Script trong thư mục này là SQL thô, không có lịch trình, không
> tự rollback, và một số trong số đó **xoá dữ liệu thật**.

---

## Cách kết nối

Cần `psql` (đi kèm PostgreSQL client). Hai cách dùng trong dự án:

**A. Qua container (tiện cho môi trường phát triển)**

```bash
docker exec -i postgres-closy-dev psql -U "$DB_USER" -d "$DB_NAME"
```

Chạy một file:

```bash
docker exec -i postgres-closy-dev psql -U "$DB_USER" -d "$DB_NAME" \
  < database/scripts/07-verify_policy_grants.sql
```

> **Trên PowerShell:** `ps1` không có toán tử chuyển hướng `<`, nên lệnh trên sẽ báo
> lỗi ngay. Dùng đường ống thay cho chuyển hướng:
>
> ```powershell
> Get-Content -Raw -Encoding UTF8 database/scripts/07-verify_policy_grants.sql `
>   | docker exec -i postgres-closy-dev psql -U $env:DB_USER -d $env:DB_NAME
> ```
>
> `-Encoding UTF8` là bắt buộc. Không có nó, PowerShell mặc định đọc theo encoding
> của máy và các dấu tiếng Việt trong chú thích sẽ thành ký tự hỏng — vô hại với
> SQL vì chú thích không được phân tích, nhưng khiến đầu ra không đọc được.

**B. Trực tiếp từ máy host** — dùng định dạng DSN giống hệt biến `MIGRATION_DSN`
trong `Makefile`:

```bash
psql "host=localhost user=$DB_USER password=$DB_PASSWORD dbname=$DB_NAME port=5432 sslmode=disable" \
  -f database/scripts/07-verify_policy_grants.sql
```

Giá trị `DB_USER` / `DB_NAME` / `DB_PASSWORD` lấy từ `.env`.

---

## ⚠️ Đọc trước khi chạy script có GHI

`psql` **mặc định vẫn chạy tiếp sau khi một câu lệnh hỏng**. Với script nhiều câu
lệnh, điều đó có nghĩa là nó vừa in ra một dòng `ERROR` vừa tiếp tục `DELETE` /
`UPDATE` những câu phía sau — và người chạy tưởng đã dừng.

Các script ghi trong thư mục này đã bật sẵn:

```sql
\set ON_ERROR_STOP on
```

Nếu bạn tự viết script mới ở đây, **hãy đặt dòng đó ở đầu file**. Đây không phải
chuyện hình thức: nó là thứ đứng giữa một script hỏng và việc dữ liệu bị xoá âm
thầm.

Ngoài ra, hãy bọc phần ghi trong `BEGIN; … COMMIT;` để lỗi giữa chừng được hoàn
tác thay vì để lại trạng thái nửa vời.

---

## Danh mục — feature 024 (đường cấp quyền tính phí AI)

Sáu script dưới đây thuộc feature 024. Chúng theo một **thứ tự bắt buộc**, mỗi bước
là điều kiện để bước sau có ý nghĩa.

### `03-dry_run_update_ai_cost_limits.sql` — XEM, không ghi

In ra giá trị `hard_cost_micro_vnd` hiện tại của mọi chính sách phí AI.

Chỉ đọc. Chạy bao nhiêu lần cũng được, và mỗi lần phải cho ra kết quả y hệt nhau
trên cùng một cơ sở dữ liệu.

Mục đích: nâng trần chi phí cứng của Premium từ 25.000.000.000 lên
30.000.000.000 microVND, đủ cho hạn mức 30 outfit + 10 chat mỗi ngày.
`microVND` = VND × 1.000.000, nên 30.000.000.000 microVND **= 30.000đ**.

### `04-update_ai_cost_limits.sql` — GHI

Thực thi thay đổi trần chi phí cứng, trong một transaction.

Chỉ đụng tới dòng `code = 'premium-default'`. Chính sách `free` cố ý để
`hard_cost_micro_vnd = NULL` (không có trần cứng) và **không được** gán trần.

Idempotent: câu `UPDATE` có điều kiện `IS DISTINCT FROM`, nên chạy lần hai tác
động 0 dòng. Cột `version` cũng được tăng để thay đổi truy vết được.

Cách hoàn tác: đặt `hard_cost_micro_vnd` về `25000000000` cho `premium-default`
và trả `version` về giá trị cũ.

### `05-dry_run_cleanup_policy_grants.sql` — XEM, không ghi

Đo trước phạm vi sửa chữa dữ liệu grant cũ: bao nhiêu ảnh chụp rỗng, bao nhiêu grant
chết, bao nhiêu grant đã hết hạn, và **bao nhiêu grant vẫn còn hiệu lực**.

Chạy **ngay trước** `06` và so số của nó với số mà `06` báo ra. Đừng so với con số
ghi trong tài liệu: tín hiệu đến liên tục, một mốc cứng sẽ không khớp và bạn không
biết là dữ liệu dịch chuyển hay tài liệu đã cũ.

Trong số bốn con số, **con số cuối là bất biến phải giữ nguyên**: đó là số grant
thực sự còn hiệu lực. Dọn dẹp không được lấy mất quyền của ai. Ba con số kia chỉ
cần về 0.

### `06-cleanup_policy_grants.sql` — GHI

Dựng lại ảnh chụp chính sách, bỏ grant chết, đóng grant quá hạn. Tất cả trong một
transaction, nên lỗi giữa chừng thì bảng giữ nguyên.

Idempotent: chạy lần hai báo toàn 0 và không đổi gì.

**Thứ tự ba bước bên trong là cố ý, và thứ tự đó chính là ý nghĩa của script:**

1. Dựng lại ảnh chụp cho **mọi** dòng, ghi đè cái cũ. Rất dễ chỉ điền tám dòng rỗng
   — và đó là sai, vì tám dòng còn lại thiếu cả nhóm `entitlements` (chúng được
   viết trước khi bảng cấu hình thao tác tồn tại), lại trộn hai kiểu ghi khoá
   (`Code` và `code` cùng tồn tại). Đọc một giới hạn vắng mặt là đọc thành 0, và
   0 trông hợp lệ nhưng không chặn được gì cả.
2. Xoá các dòng đã chết.
3. Đóng các dòng đã quá hạn.

### `07-verify_policy_grants.sql` — XEM, không ghi

Sáu bất biến của bảng `aiexecutor.ai_policy_grants`. Chạy bao nhiêu lần cũng
được.

**Cả sáu đều phải bằng 0.** Mỗi cái tương ứng với một kiểu bảng sai, và mỗi kiểu
từng gây ra lỗi khách hàng nhìn thấy được trước khi feature này tồn tại:

| # | Bất biến | Sai lệch nghĩa là |
|---|---|---|
| 1 | Có gói nhưng thiếu bản ghi mức quyền | Khách trả tiền không dùng được AI trả phí; hệ thống tự hạ cấp mô hình rẻ mà không báo lỗi |
| 2 | Nhiều bản ghi mức quyền cùng hiệu lực | Bị cấp trùng — nghĩa là đóng và tạo không nguyên tử |
| 3 | `plan_code` của grant lệch với gói hiện tại | Một sự kiện bị mất hoặc đến sai thứ tự |
| 4 | Ảnh chụp chính sách rỗng hoặc thiếu thao tác | **Không có giới hạn nào được áp dụng, và không gì báo lỗi** |
| 5 | Bản ghi `active`/`future` đã hết hạn | Grant chết che mất quyền thật |
| 6 | Cổng 1 và cổng 2 lệch nhau | Khách thấy "còn 20 lượt" nhưng thực tế bị chặn 10 — **loại lỗi đắt nhất**, vì nó hoàn toàn im lặng |

Mục 6 **có hai ngoại lệ có chủ đích** và mã loại trừ chúng:

1. Chính sách `premium-default` ở chế độ `strict`: cột
   `max_paid_attempts_per_day` không bao giờ được đọc, nên NULL ở đây là cấu hình
   chết chứ không phải lỗi.
2. Gói `free` cố ý thấp hơn **đúng một lượt** so với con số khách nhìn thấy.

Mục **6b** in ra mọi dòng kèm phán đoán, kể cả hai ngoại lệ, để chúng luôn hiện
diện thay vì bị ẩn đi. Bất kỳ chênh lệch nào khác là chưa giải thích được và phải
điều tra.

### `08-seed_grant_test_data.sql` — dữ liệu kiểm thử (tùy chọn)

Tạo dữ liệu riêng cho kiểm thử feature 024. **Không liên quan gì tới dữ liệu
thật** — mọi tài khoản đều có tiền tố `t024_`.

Tạo:

- **1 quản trị viên thứ hai** (`t024_admin2`) — cần để kiểm tra hai quản trị viên
  gọi cùng một `Idempotency-Key` song song.
- **500 tài khoản khách hàng** trên gói free, **cố ý không có bản ghi mức quyền
  AI** — đó là nhóm chạy qua bộ giải quyết dự phòng, dùng để đo thời gian một yêu
  cầu 500 phần tử và xác minh hạn mức lượt của gói miễn phí khi không có grant.

```bash
# tạo
psql -f database/scripts/08-seed_grant_test_data.sql

# xoá sạch
psql -f database/scripts/08-seed_grant_test_data.sql --set cleanup=1
```

Cả hai chế độ đều chạy trong một transaction và **idempotent**: chạy tạo lần hai
không sinh trùng, chạy xoá lần hai báo 0 dòng.

Mật khẩu của các tài khoản này được lấy từ tài khoản quản trị sẵn có trong cơ sở
dữ liệu phát triển. Chúng được tạo để kiểm thử, **không dùng ở bất kỳ môi trường
nào có dữ liệu thật**.

⚠️ **08 KHÔNG kiểm thử được FR-088, và đó không phải lỗi của nó.** Những tài khoản
`t024_*` được tạo bằng `INSERT` thẳng và cố ý không có bản ghi gói — đúng trạng
thái cần đo cho FR-005b/c/d. Nhưng công cụ khôi phục quét
`subscription.user_subscriptions` (`restore_policy_grants_uc.go:249`), nên chúng
không bao giờ xuất hiện trong `res.Scanned`. Dùng `11` cho FR-088.

### `11-seed_restore_test_data.sql` — dữ liệu kiểm thử (tùy chọn)

Tạo sáu tài khoản **có bản ghi gói nhưng cố ý không có bản ghi mức quyền AI** —
tức là đúng trạng thái mà FR-088 sinh ra để sửa. Không liên quan gì tới dữ liệu
thật: tiền tố `t024restore_` và sáu UUID cố định `f0240001-…`.

Trước T204 **không tồn tại fixture nào thật sự kiểm thử được FR-088**, vì công cụ
quét bảng `user_subscriptions` mà không fixture nào tạo dòng trong đó.

| Tài khoản | Gói | Kết quả mong đợi của `dryRun: false` |
|---|---|---|
| `t024restore01`…`03` | free | `restored` — FR-088 không phân biệt gói miễn phí |
| `t024restore04`…`05` | `premium-monthly` | `restored` — gói trả phí |
| `t024restore06` | `premium-monthly`, đã hết hạn | `skipped` / `subscription_expired` |

```bash
# tạo
psql -f database/scripts/11-seed_restore_test_data.sql

# xoá sạch
psql -f database/scripts/11-seed_restore_test_data.sql --set cleanup=1
```

Cả hai chế độ đều trong một transaction và **idempotent**. Phần tạo **tự kiểm tra**:
in ra `accounts_missing_grant` — phải là `6`. Nếu là `0` thì fixture đã hỏng (ví dụ
đã chạy công cụ khôi phục một lần rồi tạo lại); chạy `--set cleanup=1` rồi tạo lại.

Tiền tố `t024restore_` **cố ý không trùng** với `t024_` của `08`: mẫu
`LIKE 't024\_%'` của 08 không khớp `t024restore01`, nên hai tệp này xoá đúng phần
của mình và chạy được cùng lúc trên một cơ sở dữ liệu.

### `09-dry_run_backfill_auto_renew_history.sql` — XEM, không ghi

Báo cáo những bản ghi lịch sử `AUTO_RENEW_UPDATED` mà một lệnh `UPDATE` hàng loạt
đã bỏ sót.

**Khi nào cần chạy:** chỉ khi bạn đã tắt tự động gia hạn hàng loạt bằng SQL
thay vì qua ứng dụng. Đường ứng dụng (`SetAutoRenewStatus`) ghi bản ghi lịch sử,
ghi outbox và tăng `version` trong một giao dịch; `UPDATE` trực tiếp thì không làm
gì cả, nên FR-012 bị vi phạm cho những dòng đó.

**Phạm vi là ĐÚNG MỘT MỐC THỜI GIAN**, không phải "mọi dòng trông như chưa có
lịch sử". Chọn rộng hơn sẽ tạo bản ghi lịch sử cho những tài khoản **chưa từng
bật** tự động gia hạn — tức là ghi lại một thay đổi không ai thực hiện. Một dòng
ghi bịa đặt tệ hơn một dòng ghi thiếu, vì dòng thiếu thì nhìn ra là thiếu.

Mốc mặc định nằm trong chính file. Truyền mốc khác nếu cần:

```bash
psql -f database/scripts/09-dry_run_backfill_auto_renew_history.sql \
     -v bulk_update_at='2026-09-30 09:41:48.638213+00'
```

Năm mục, và điều chúng kiểm tra:

| # | Ý nghĩa |
|---|---|
| 1 | Các dòng sẽ được bù lịch sử. Sau khi chạy `10` thì phải **0** |
| 2 | Khoá sự kiện sẽ dùng. Trùng khoá sẽ khiến `ON CONFLICT` bỏ qua |
| 3 | Số khoá **đụng động** (khoá đã tồn tại nhưng thuộc loại khác). Mong đợi **0** — nếu khác 0, `version` sẽ bị tăng mà lịch sử không có |
| 4 | Gói không có trong danh mục. Mong đợi **0** — thiếu thì không dựng được payload outbox |
| 5 | Dòng đang **bật** tự động gia hạn mà không có lịch sử. Mong đợi **0**, và script này **không** đụng tới chúng |

### `10-backfill_auto_renew_history.sql` — GHI

Bù lại lịch sử cho các dòng mà `09` báo cáo. Một transaction, idempotent.

Mỗi dòng nhận **ba** thứ, và cả ba đều bắt buộc:

1. Bản ghi `user_subscription_events` loại `AUTO_RENEW_UPDATED` — FR-012.
2. Bản ghi `shared.outbox_records` trên `subscription.plan.changed` — FR-015 đòi
   lịch sử **và** sự kiện trong *một* giao dịch.
3. `version + 1` — ứng dụng luôn tăng version cho thay đổi này; `version` là một
   phần của khoá `RESTORE_GRANT:{user}:{version}` và của kiểm tra xung đột lạc quan.

`updated_at` được **giữ nguyên có chủ đích**: nó đã ghi đúng thời điểm thay đổi xảy
ra, và việc bù lại chạy muộn hơn không được phép ghi đè bằng đồng hồ của chính nó.

```bash
# chạy thử trước, rồi mới áp dụng
psql -f database/scripts/09-dry_run_backfill_auto_renew_history.sql
psql -f database/scripts/10-backfill_auto_renew_history.sql
```

Chạy lại `10` báo `0 | 0 | 0` — đó là hợp đồng idempotent của nó. Chạy lại `09` thì
mục 1 và mục 2 phải trống.

---

## Thứ tự chạy cho feature 024

```
03 (xem)  →  04 (ghi)  →  05 (xem)  →  06 (ghi)  →  07 (xem, phải 0 hết)
                                                        ↑
                                     08 (tuỳ chọn, chỉ khi cần dữ liệu kiểm thử)
                                     11 (tuỳ chọn, chỉ khi cần kiểm thử FR-088)

09 (xem)  →  10 (ghi)     ← chỉ khi đã tắt tự động gia hạn bằng SQL thủ công
```

`08` và `11` không nối tiếp nhau và không phụ thuộc nhau: `08` phục vụ FR-005b/c/d
(đường dự phòng), `11` phục vụ FR-088 (công cụ khôi phục). Chạy cái nào cũng được.

Nếu đang triển khai lại từ đầu trên một cơ sở dữ liệu cũ, chạy `03` và `05` trước
để **ghi lại kết quả ra giấy**, rồi chạy `04` và `06`, rồi chạy `07` để xác nhận
sáu con số về 0.

---

## Vài lưu ý vận hành liên quan

- **`x-queue-type` không đổi được trên queue đã tồn tại.** Đổi khai báo sẽ khiến
  RabbitMQ trả `406 PRECONDITION_FAILED` và dịch vụ **không khởi động được**. CI đã
  check trước mỗi deploy; cách migrate: xem
  [`deployments/scripts/rabbitmq-topology/README.md`](../../deployments/scripts/rabbitmq-topology/README.md).
- **Sau khi dựng lại container `account` hoặc `commerce`**, nginx giữ DNS cũ và mọi
  route sẽ trả `404` cho tới khi chạy `make nginx-reload-local` hoặc restart nginx.
  Đây là triệu chứng quen thuộc, không phải lỗi route.
- Chi tiết về DDL của `ai_policy_grants` và `ai_usage_period_ledgers`, cùng lý do
  `grant_id` phải nullable: xem `docs/development/technical-notes.md`.
- Sơ đồ đường cấp quyền AI: xem mục XI của `docs/system-design/architecture.md`.

---

## Script không thuộc feature 024

Phần còn lại của thư mục (`01`, `02`, `rollback_cloudinary_urls`, `seed_*`,
`truncate_all_tables`) có từ các feature trước và không nằm trong chuỗi ở trên.

⚠️ **`truncate_all_tables.sql` xoá toàn bộ dữ liệu.** Không chạy nó trên cơ sở
dữ liệu mà bạn không sẵn sàng mất.

---

## Chuẩn bị chạy thật — dọn dữ liệu seed ở production

Hai script dưới đây **không thuộc feature nào**. Chúng dọn dữ liệu demo đã tích
tụ trong lúc demo sản phẩm, chuẩn bị đưa production lên vận hành thật.

Chúng là ngoại lệ duy nhất so với cảnh báo ở đầu file này. Cảnh báo đó nói
"Không dùng script này trên production" vì các script còn lại **xoá dữ liệu thật
mà không hỏi**. Hai script này khác ở chỗ chúng phân biệt rõ dữ liệu seed với
dữ liệu người dùng thật, và chúng từ chối chạy nếu không xác minh được ranh
giới đó.

### `12-dry_run_preprod_cleanup.sql` — XEM, không ghi

Bảy bảng báo cáo, chạy bao nhiêu lần cũng được:

| # | Nội dung |
|---|---|
| 1, 1b | 70 tài khoản seed sẽ bị xoá, liệt kê **đầy đủ tên** — đọc trước khi chạy file 13 |
| 2 | 26 tài khoản được giữ, kèm số wardrobe/outfit/subscription/deposit của từng người |
| 3 | Số dòng sẽ xoá theo từng bảng, kèm lý do thứ tự |
| 4 | Dòng bị xoá bởi `CASCADE`, không có lệnh `DELETE` riêng |
| 5 | Dữ liệu **giữ nguyên** — nếu một bảng ở đây bằng 0 mà bạn không xoá nó, hãy dừng lại |
| 6 | Baseline phải còn lại (bảng này phải khớp tuyệt đối) |
| 7 | Kỳ vọng sau khi dọn, để đối chiếu với file 13 |

### `13-preprod_cleanup.sql` — GHI

Một transaction, `\set ON_ERROR_STOP on`, xoá theo 5 bước. **Ba guard dừng
script trước khi ghi bất cứ thứ gì:**

1. Bắt buộc `-v apply=1`. Thiếu, hoặc bằng bất kỳ giá trị nào khác, script
   thoát ngay.
2. Danh sách tài khoản được giữ phải khớp **DANH TÍCH**, không chỉ khớp số
   lượng. Thiếu một, thừa một, hoặc truyền sai định dạng đều `RAISE EXCEPTION`
   → rollback.
3. Baseline config phải còn nguyên — 2 plan, 9 category, 2 cost policy,
   8 operation, 7 model config, 32 + 3 dòng goose. Sai số → rollback.

**Danh sách tài khoản được giữ nằm ngoài git**, ở
`docs/temp/preprod-kept-users.txt` (thư mục `docs/temp` đã nằm trong
`.gitignore:730`). Truyền vào lúc chạy bằng `-v kept_users=`. Lý do không để
trong file SQL: đó là tài khoản người dùng thật, và repo không nên chứa nó.

**`COMMIT` nằm ở cuối file**, sau cả khối báo cáo. Nếu câu báo cáo hỏng thì
`ON_ERROR_STOP` dừng script trước `COMMIT` và việc dọn được hoàn tác. Đặt
`COMMIT` trước báo cáo nghĩa là báo cáo hỏng không cứu được dữ liệu — bạn mất
dữ liệu mà không có bản đối chiếu để biết đã mất gì.

Kiểm danh sách được truyền vào **trước mọi lệnh `DELETE`**, không phải ở guard 2
cuối file. Nếu kiểm cuối thì script đã xoá xong 5 bước rồi mới phát hiện thiếu
đầu vào — đúng trường hợp mà transaction phải cứu ta.

#### Chạy

Chạy từ thư mục gốc repo. **Cần có dump backup trước** — script xoá vĩnh viễn.

Trên máy này **không có `psql` trên host** — `psql` chỉ nằm trong container
`postgres-closy-dev`. PowerShell cũng không có toán tử `<`, nên phải nạp file
qua đường ống.

Mật khẩu production đặt sẵn ở biến môi trường user scope `PGPROD_PW`, nên không
gõ mật khẩu vào lệnh (tránh lọt vào PowerShell history và log của container):

```powershell
# đặt biến (chỉ cần một lần, mở cửa sổ PowerShell mới để nó nạp)
[Environment]::SetEnvironmentVariable("PGPROD_PW", "<pwd>", "User")
```

Đọc danh sách tài khoản được giữ (bỏ dòng comment và dòng trống):

```powershell
$kept = (Get-Content docs/temp/preprod-kept-users.txt | Where-Object { $_ -and $_ -notmatch '^\s*#' }) -join ','
```

Ba lệnh dưới đây là dạng chạy được:

```powershell
# 1. Xem kế hoạch
Get-Content -Raw -Encoding UTF8 database/scripts/12-dry_run_preprod_cleanup.sql | docker exec -i -e PGPASSWORD=$env:PGPROD_PW postgres-closy-dev psql -h host.docker.internal -p 15432 -U closy -d closy_db -q
```

```powershell
# 2. Chạy thật
Get-Content -Raw -Encoding UTF8 database/scripts/13-preprod_cleanup.sql | docker exec -i -e PGPASSWORD=$env:PGPROD_PW postgres-closy-dev psql -h host.docker.internal -p 15432 -U closy -d closy_db -q -v apply=1 -v kept_users="$kept"
```

```powershell
# 3. Chạy lại file 12 — bảng [7] phải khớp cột actual_remaining
Get-Content -Raw -Encoding UTF8 database/scripts/12-dry_run_preprod_cleanup.sql | docker exec -i -e PGPASSWORD=$env:PGPROD_PW postgres-closy-dev psql -h host.docker.internal -p 15432 -U closy -d closy_db -q
```

Ba chỗ dễ nhầm trong các lệnh trên:

- **`host.docker.internal`**, không phải `localhost`. Bạn đứng ngoài container
  nên `localhost` là máy bạn; container cần tên đó để vòng về host.
- **`-e PGPASSWORD=$env:PGPROD_PW`** truyền mật khẩu vào container. Thiếu nó,
  `psql` hỏi interactive và đọc từ đường ống sẽ hỏng.
- **Không dùng `-f`.** File phải nạp qua đường ống vì PowerShell không có `<`.

DSN đầy đủ (`host=... user=... password=... dbname=...`) — cùng format với biến
`MIGRATION_DSN` ở `Makefile:33` — dùng cho `psql`, `pg_dump` và `goose`, và cho
môi trường có `psql` trên host.

#### Ai bị xoá, ai được giữ

**Xoá 70 tài khoản seed**, khớp với bốn script trong thư mục này:

| Mẫu username | SL | Nguồn |
|---|---|---|
| `fb_cust_*` (`@closy.local`) | 60 | `seed_fill_customer_user_ids.sql` |
| `conv_user_*` (`@test.com`) | 6 | `seed_feature_006_dashboard.sql` |
| `owner_branda`, `staff_branda` | 2 | `seed_feature_006_dashboard.sql` |
| `demo*` (`@hycat.online`) | 2 | seed demo users |

Kèm 3 brand seed (Thời Trang ABC, Fashion XYZ, Demo Regis) với toàn bộ dữ liệu
brand/loyalty/dashboard đi theo, và 31 `fashion_items` không còn ai dùng.

**Giữ 26 tài khoản** — danh sách nằm ở
`docs/temp/preprod-kept-users.txt` — cùng **toàn bộ** dữ liệu của họ: `admin`,
`user`, và 24 tài khoản Gmail/FPT đã đăng ký thật. `user` giữ nguyên 184
wardrobe items, 19 outfits, 31 deposit, 225 AI event; `admin` giữ 81 wardrobe
items.

#### Baseline KHÔNG được xoá

`database/init/03-seed.sql` được mount vào `/docker-entrypoint-initdb.d`, và
Postgres chỉ chạy nó **một lần** lúc data dir còn rỗng. Production đã có dữ
liệu nên file đó không bao giờ chạy lại. Xoá một trong những dòng dưới đây là
mất vĩnh viễn:

| Bảng | SL | Mất thì |
|---|---|---|
| `subscription.subscription_plans` | 2 | Account service **không boot** — Catalog Validator cần đúng một plan `free` đang active ([`catalog_validator.go`](../../services/account/internal/modules/subscription/application/validator/catalog_validator.go)) |
| `fashion.categories` | 9 | Mất danh mục, không tự tạo lại |
| `aiexecutor.ai_cost_policies` | 2 | Plan mất `ai_cost_policy_id` ⇒ account không boot |
| `aiexecutor.ai_cost_policy_operations` | 8 | Mất cấu hình route AI |
| `aiexecutor.ai_model_configs` | 7 | Mất bảng giá model |
| `public.goose_db_version` | 32 | Goose coi là chưa migration nào và chạy lại từ đầu |
| `public.goose_db_version_commerce` | 3 | Như trên, cho commerce |

#### Kiểm chứng

Cả hai script đã chạy thử trên bản sao database (`closy_clone`) dựng từ
`pg_dump` của production. Bảy tình huống, tất cả đều đúng:

| Tình huống | Kết quả |
|---|---|
| Danh sách đúng 26 tên | Chạy xong, guard 2 và 3 pass, số dòng xoá khớp dự đoán |
| Chạy lần hai | **0 dòng** ở mọi bảng (idempotent) |
| Không truyền `-v apply` | Dừng, DB không đổi |
| `apply=0` / `yes` / `TRUE` | Dừng, DB không đổi |
| Không truyền `-v kept_users` | Dừng **trước** lệnh `DELETE` đầu tiên |
| `kept_users` rỗng | Dừng trước lệnh `DELETE` đầu tiên |
| Thêm tài khoản lạ vào DB | Guard 2 chặn → rollback |
| Danh sách thiếu 1 tên | Guard 2 chặn → rollback |
| Danh sách thừa 1 tên không tồn tại | Guard 2 chặn → rollback |
| Tắt plan `free` | Guard 3 chặn → rollback |

#### Điều script này **không** làm

- Không xoá `outbox_records` (422), `refresh_tokens` (308), `campaign_claims`
  (4), `user_daily_quotas`, `admin_audit_logs`. Đã quyết định giữ.
- Không xoá `subscription.campaigns` — campaign `launch-2026-10` do migration
  `20261003165500` tạo, xoá thì campaign biến mất khỏi sản phẩm.
- Không xoá `deposit_transactions`, `ai_usage_events`, `ai_usage_period_ledgers`,
  `ai_policy_grants` — đều là dữ liệu của 26 tài khoản được giữ.
- Không `VACUUM`. Bảng vẫn còn dung lượng cũ trên đĩa cho tới khi chạy
  `VACUUM FULL` (giữ mọi thứ, không cần thiết).
