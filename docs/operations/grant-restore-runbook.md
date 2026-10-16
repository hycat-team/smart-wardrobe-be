# Runbook — khôi phục mức quyền tính phí AI cho tài khoản thiếu grant

Áp dụng khi: bộ đếm `cost_fallback_by_reason.tier1_default_policy` tăng, hoặc
`database/scripts/07-verify_policy_grants.sql` mục 1 hoặc mục 4 trả về khác 0.

Bối cảnh kiến trúc và lý do không tự động hoá nằm ở
[ADR 013](../decisions/013-deferred-lazy-grant-self-heal.md).

---

## ⚠ ĐỌC TRƯỚC KHI CHẠY — hai giới hạn đã biết của công cụ này

Cả hai đã được xác minh trên production ngày 2026-10-07 bằng cách chạy thật và đọc
log. Chúng **không phải** giới hạn lý thuyết.

### Cảnh báo 1 — Không sửa được grant mất `envelope`

`rejectStaleEvent` cần so thứ tự sự kiện với `occurred_at` nằm trong
`policy_snapshot->'envelope'` của grant hiện hành. Grant đời trước 024 không có
`envelope` (xem mục 1c), nên bước này trả về `false` và **từ chối vĩnh viễn**:

```
"Bản ghi mức quyền hiện hành không đọc được thời điểm xảy ra,
 không thể xác định thứ tự sự kiện."  →  [TypedHandler] Poison message, dead-lettering
```

Đây là **deadlock khởi tạo**: công cụ sửa duy nhất bị chặn bởi chính dữ liệu hỏng mà
nó được gọi để sửa. Không có cách nào sửa bằng chính endpoint này.

Hệ quả nghiêm trọng hơn con số: **bất kỳ lần đổi gói nào sau này của tài khoản đó
cũng sẽ bị từ chối y hệt.** Tài khoản sẽ nâng cấp thành công trên bản ghi đăng ký
nhưng quyền AI vẫn ở mức cũ, và sự kiện nằm chết trong hàng đỗi chết không tự phục
hồi.

### Cảnh báo 2 — Trên production hiện tại, endpoint này hỏng với MỌI tài khoản

Đã đo ngày 2026-10-07: **28/28 tài khoản** trên production có
`subscription.started_at == grant.effective_from` tuyệt đối.

`RestorePolicyGrants` ghim `EffectiveAt = subscription.started_at`, rồi
`retirePreviousGrants` đóng grant cũ với cutoff bằng đúng giá trị đó. Khi hai giá
trị bằng nhau thì `effective_to == effective_from`, và constraint
`ai_policy_grants_effective_window_check` (`CHECK (effective_to IS NULL OR effective_to > effective_from)`)
từ chối:

```
ERROR: new row for relation "ai_policy_grants" violates check constraint
        "ai_policy_grants_effective_window_check" (SQLSTATE 23514)
```

Lỗi này bị phân loại **nhầm là transient**, nên thông điệp bị requeue rồi thử lại 5
lần — mỗi lần đều chắc chắn thất bại — trước khi bị dead-letter. Nó không phải sự
cố hạ tầng và không tự hết.

Kết quả lần chạy thật 2026-10-07 trên 28 tài khoản: **10 được sửa, 18 bị từ chối**
(8 theo cảnh báo 1, 10 theo cảnh báo 2).

### Trước khi chạy, phải chạy mục 1c và 1d

Hai truy vấn đó dự đoán chính xác tài khoản nào sẽ thất bại và vì lý do gì. Nếu
chúng trả về số khác 0 thì **endpoint này không sửa được những tài khoản đó** — đừng
chạy, và đừng chạy lại lần nữa sau khi nó đã báo `already_restored`.

Trạng thái hiện tại: hai lỗi này đã được ghi nhận để sửa trong spec 027. Tới lúc đó
mốc cảnh báo ở đây được gỡ bỏ. **Tới lúc đó, đừng xoá các mục 1c và 1d** — chúng là
cách biết môi trường đích đã sẵn sàng hay chưa.

---

## 0. Điều kiện tiên quyết

- Đăng nhập được tài khoản admin (role `admin` hoặc `brand_admin`).
- Kết nối tới Postgres của môi trường đích.
- Biết dịch vụ AI đang chạy và hàng đỗi `subscription_plan_changed_queue` không bị
  tắt — nếu hàng đỗi tắt, sự kiện phát ra sẽ nằm lại trong outbox và bước 4 sẽ
  "thành công" trong khi không có gì được dựng.

## 1. Chẩn đoán trước khi sửa (read-only)

### 1a. Tài khoản có gói nhưng không có grant đang hiệu lực

Đây là câu trả lời cho "ai đang thiếu grant":

```sql
SELECT s.user_id, s.current_plan_code, s.started_at, s.expires_at
FROM subscription.user_subscriptions s
WHERE NOT EXISTS (
    SELECT 1 FROM aiexecutor.ai_policy_grants g
     WHERE g.user_id = s.user_id
       AND g.status IN ('active', 'future')
       AND g.effective_from <= now()
       AND (g.effective_to IS NULL OR g.effective_to > now())
)
ORDER BY s.started_at;
```

Kỳ vọng: **không có dòng nào**. Có dòng thì đó là danh sách tài khoản cần sửa.

### 1b. Tài khoản không có cả dòng đăng ký — trường hợp endpoint **không** sửa được

`restore` quét `subscription.user_subscriptions`, nên tài khoản không có dòng ở đó sẽ
**không** bao giờ được chạm tới:

```sql
SELECT u.id AS user_id, u.created_at
FROM identity.users u
WHERE NOT EXISTS (
    SELECT 1 FROM subscription.user_subscriptions s WHERE s.user_id = u.id
)
ORDER BY u.created_at;
```

Nếu câu này có dòng: những tài khoản đó tự được cấp gói mặc định ở lần `GET /me` kế
tiếp (lazy door), và sự kiện `ACTIVATED` sinh ra lúc đó sẽ tới dịch vụ AI theo
đường bình thường. **Cứ để chạy `GET /me`** thay vì sửa tay — cùng một cơ chế đã
được dùng và có test.

### 1c. Grant có snapshot không đọc được (cảnh báo 1)

Phân rã theo nguyên nhân và trạng thái, để biết cái nào thực sự đang hỏng:

```sql
SELECT g.status,
       CASE WHEN g.effective_from <= now() AND (g.effective_to IS NULL OR g.effective_to > now())
            THEN 'in_force' ELSE 'not_in_force' END AS win,
       CASE WHEN g.policy_snapshot = '{}'::jsonb THEN 'empty_doc'
            WHEN NOT (g.policy_snapshot ? 'envelope') THEN 'no_envelope'
            WHEN NOT (g.policy_snapshot ? 'entitlements') THEN 'no_entitlements'
            ELSE 'partial_ops' END AS why,
       count(*) AS n
FROM aiexecutor.ai_policy_grants g
WHERE g.policy_snapshot = '{}'::jsonb
   OR NOT (g.policy_snapshot ? 'envelope')
   OR NOT (g.policy_snapshot ? 'entitlements')
   OR NOT (g.policy_snapshot->'entitlements' ? 'chat'
       AND g.policy_snapshot->'entitlements' ? 'outfit'
       AND g.policy_snapshot->'entitlements' ? 'summary'
       AND g.policy_snapshot->'entitlements' ? 'rewriter')
GROUP BY 1, 2, 3 ORDER BY 1, 2, 3;
```

**Chỉ dòng `in_force` là đáng lo.** Dòng `not_in_force` là grant `closed` đã hết hạn,
không bao giờ được đọc, và vẫn bị mục 4 của script verify đếm vào — nên con số mục 4
**không** tương đương số tài khoản đang hỏng. Trên production ngày 2026-10-07, mục 4
trả 22 gồm 8 `in_force` và 14 `closed`.

Kiểm tra thêm một tài khoản có thực sự đang suy giảm:

```sql
WITH bad AS (
  SELECT DISTINCT user_id FROM aiexecutor.ai_policy_grants
  WHERE status IN ('active','future') AND effective_from <= now()
    AND (effective_to IS NULL OR effective_to > now())
    AND NOT (policy_snapshot ? 'envelope' AND policy_snapshot ? 'entitlements'
      AND (policy_snapshot->'entitlements' ? 'chat' AND policy_snapshot->'entitlements' ? 'outfit'
       AND policy_snapshot->'entitlements' ? 'summary' AND policy_snapshot->'entitlements' ? 'rewriter'))
)
SELECT b.user_id,
       NOT EXISTS (SELECT 1 FROM aiexecutor.ai_policy_grants g2
                    WHERE g2.user_id = b.user_id AND g2.status IN ('active','future')
                      AND g2.effective_from <= now()
                      AND (g2.effective_to IS NULL OR g2.effective_to > now())
                      AND g2.policy_snapshot ? 'envelope') AS co_grant_tot_khac
FROM bad b;
```

`co_grant_tot_khac = false` nghĩa là grant hỏng là grant duy nhất của tài khoản đó,
nên nó đang lấy hạn mức từ cấu hình live thay vì từ snapshot.

### 1d. Grant sẽ không đóng được khi sửa (cảnh báo 2)

Truy vấn này dự đoán chính xác lỗi CHECK constraint:

```sql
SELECT count(*) AS accounts_that_restore_cannot_fix
FROM subscription.user_subscriptions s
JOIN aiexecutor.ai_policy_grants g ON g.user_id = s.user_id
 AND g.status IN ('active','future') AND g.effective_from <= now()
 AND (g.effective_to IS NULL OR g.effective_to > now())
WHERE s.started_at = g.effective_from;
```

Kỳ vọng: **0**. Trên production ngày 2026-10-07 câu này trả **28**, tức toàn bộ.

### 1e. Grant đang hiệu lực nhưng hết hạn trong kỳ

Chạy script đầy đủ để phân biệt phần còn lại:

```bash
psql "$DATABASE_URL" -f database/scripts/07-verify_policy_grants.sql
```

Cả 6 mục đều phải bằng 0 (mục 6 có một ngoại lệ duy nhất đã được chấp thuận, script
tự loại).

## 2. Chạy thử (dry run)

`dryRun` là **mặc định**, không phải lựa chọn thay thế. Bỏ trống body cũng ra dry run.

```bash
curl -s -X POST "https://api.closy.hycat.online/api/v1/admin/subscription-grants/restore" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dryRun": true}' | python -m json.tool
```

Đọc 4 số trong `data`:

- `scanned` — tổng số tài khoản có dòng đăng ký.
- `restored` — số sẽ phát lại sự kiện. **Với lần chạy đầu, con số này gần bằng
  `scanned`**, không phải bằng số tài khoản thiếu grant ở bước 1a. Xem mục 5.
- `skipped` — gói hết hạn, hoặc lần chạy trước đã sửa rồi.
- `failed` — có dòng nào `status=error` thì **dừng lại**, không chạy thật. Xem
  `results[]` để biết lý do (`plan_not_in_catalog` nghĩa là dữ liệu sai, không phải
  grant hỏng).

**`dryRun` không phát hiện được hai cảnh báo ở trên.** Nó chỉ quét và đếm; nó không
mô phỏng việc dịch vụ AI có chấp nhận sự kiện hay không. Mục 1c và 1d mới làm việc đó.

## 3. Chạy thật

Chỉ khi bước 2 cho `failed: 0` **và** mục 1c + 1d đều bằng 0:

```bash
curl -s -X POST "https://api.closy.hycat.online/api/v1/admin/subscription-grants/restore" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dryRun": false}' | python -m json.tool
```

Thao tác này ghi một bản ghi kiểm toán `subscription.grant_restore` kèm `requestIP`
và mã admin, nên luôn truy vết được ai chạy từ đâu.

## 4. Xác minh

Consumer của dịch vụ AI xử lý bất đồng bộ, nên **đợi vài giây** rồi chạy lại mục 1a
và 1c.

**`restored` trong phản hồi KHÔNG phải là con số đã sửa được.** Nó đếm số sự kiện
đã *phát đi*, không phải số grant đã *dựng*. Tài khoản bị từ chối ở phía AI vẫn được
tính vào `restored`. Cách duy nhất để biết là đếm lại ở mục 1c.

Nếu vẫn còn tài khoản hỏng, kiểm tra theo thứ tự:

1. Hàng đỗi `subscription_plan_changed_queue` có còn đang chạy không.
2. Log dịch vụ AI:
   - `"reason":"stale_event"` → cảnh báo 1. Sự kiện đã dead-letter, không retry.
    - `SQLSTATE 23514` hoặc `ai_policy_grants_effective_window_check` → cảnh báo 2. Sự kiện đã
     bị requeue 5 lần rồi dead-letter.
3. Bộ đếm `grant_rejected_by_reason` trên `GET /api/v1/ai/health` (xem
   `shared-reference.md` §4.1). Bộ đếm này **process-local**, reset mỗi lần deploy —
   đọc sau khi restart thì nó chỉ phản ánh chuyện từ lúc đó.

## 5. Bán kính và rủi ro — đọc trước khi bấm lần đầu

- **Lần chạy đầu đóng và tạo lại grant cho MỌI tài khoản còn hạn**, kể cả tài khoản
  đang có grant hoàn hảo. Cơ chế chống trùng ở phía AI dựa trên `SourceEventID`, mà
  sự kiện restore mang `SourceEventID` mới, nên nó không chặn được.
- **Snapshot được dựng lại từ cấu hình chính sách lúc chạy**, không phải giá trị lúc
  grant gốc được tạo. Nếu cấu hình hôm nay sai, sai số đó bị đóng băng vào snapshot
  của mọi tài khoản. **Đây là lý do bước 2 là bắt buộc, không phải cho có**: kiểm tra
  `ai_cost_policies` và `ai_cost_policy_operations` trước khi chạy thật.
- **Chạy lại được an toàn về phía Account, nhưng vô ích về phía AI.** Event key là
  `RESTORE_GRANT:<userID>:<version>` và bản sửa không đổi `version`, nên lần hai bị
  bỏ qua với `already_restored`. Hệ quả là nếu lần đầu thất bại, **bạn không có cách
  nào làm lại bằng endpoint này** — nó sẽ báo `skipped` mãi. Khi đó phải dùng
  `POST /api/v1/admin/subscription-grants` cho đúng danh sách tài khoản, và bản ghi
  kiểm toán sẽ ghi là "cấp gói" dù họ vốn đã có gói đó.
- **Một transaction cho toàn lượt.** Hoặc mọi tài khoản đều được phát lại sự kiện và
  có bản ghi kiểm toán, hoặc không có gì xảy ra. Đây là chủ ý: một lượt sửa nửa vời
  thì tệ hơn không sửa. Lưu ý ranh giới này dừng ở phía Account — sự kiện đã phát
  thì việc dịch vụ AI có xử lý được hay không là câu chuyện khác, nằm ngoài
  transaction.

## 6. Rollback

Không có. Không có cách nào đưa lại giá trị cũ của snapshot vì các bản ghi cũ đã bị
đóng, và dữ liệu giá trị gốc không tồn tại.

Đây là lý do bước 5 phải đọc trước khi chạy, không phải sau. Cách phòng ngừa duy
nhất là `dryRun` và kiểm tra cấu hình chính sách trước.

## 7. Sau khi chạy

- Đọc lại `cost_fallback_by_reason` trên `GET /api/v1/ai/health` sau vài phút: giá
  trị `tier1_default_policy` phải ngừng tăng.
- Nếu nó **vẫn tăng trên tài khoản tạo sau hôm nay**, thì đây không còn là dữ liệu đời
  trước nữa — đó là điều kiện mở lại hạng mục trong ADR 013 §2.

## 8. Ghi chú môi trường

- **Development và production không giống nhau.** Ngày 2026-10-07: production có
  28/28 tài khoản đều có grant, còn development có **12 tài khoản không có grant row
  nào**. Nếu bạn thấy `tier1_default_policy` tăng ở development thì đó là chuyện bình
  thường và được xử lý bằng suy giảm hai tầng — không phải lỗi, và cũng không phải
  lý do để chạy endpoint này trên development.
- **`ResolvePolicy` không lọc `status`** (`ai_cost_policy_reader.go:32`), chỉ lọc theo
  cửa sổ hiệu lực. Hiện mục 5 của script verify giữ được điều đó bằng 0, nhưng đây là
  bất biến chỉ ở tầng ứng dụng — không có ràng buộc nào ở tầng dữ liệu bảo vệ nó.
- **Sửa trực tiếp bảng `ai_policy_grants` bằng SQL là không được.** Bảng đó có đúng
  một chủ sở hữu là dịch vụ AI, và hiến pháp cấm mọi hình thức ghi từ bên ngoài kể
  cả SQL thô. Cách sửa hợp lệ duy nhất là phát sự kiện qua hàng đỗi.