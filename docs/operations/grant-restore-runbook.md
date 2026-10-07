# Runbook — khôi phục mức quyền tính phí AI cho tài khoản thiếu grant

Áp dụng khi: bộ đếm `cost_fallback_by_reason.tier1_default_policy` tăng, hoặc
`database/scripts/07-verify_policy_grants.sql` mục 1 trả về khác 0.

Bối cảnh kiến trúc và lý do không tự động hoá nằm ở
[ADR 013](../decisions/013-deferred-lazy-grant-self-heal.md). Runbook này chỉ mô tả
cách chạy công cụ đã có sẵn.

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

Đây là cái bẫy cần biết trước. `restore` quét `subscription.user_subscriptions`, nên
tài khoản không có dòng ở đó sẽ **không** bao giờ được chạm tới:

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

### 1c. Grant đang hiệu lực nhưng hết hạn trong kỳ

Bộ đếm báo thiếu grant thì tài khoản bị suy giảm, nhưng nếu tài khoản **có** grant thì
đang ở trạng thái bình thường — không cần làm gì. Chạy script đầy đủ để phân biệt:

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
  grant thiếu).

## 3. Chạy thật

Chỉ khi bước 2 cho `failed: 0`:

```bash
curl -s -X POST "https://api.closy.hycat.online/api/v1/admin/subscription-grants/restore" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"dryRun": false}' | python -m json.tool
```

Thao tác này ghi một bản ghi kiểm toán `subscription.grant_restore` kèm `requestIP`
và mã admin, nên luôn truy vết được ai chạy từ đâu.

## 4. Xác minh

Consumer của dịch vụ AI xử lý bất đồng bộ, nên **đợi vài giây** rồi chạy lại bước 1a:

```bash
psql "$DATABASE_URL" -f database/scripts/07-verify_policy_grants.sql
```

Mục 1 phải về 0. Nếu vẫn khác 0, kiểm tra theo thứ tự:

1. Hàng đỗi `subscription_plan_changed_queue` có còn đang chạy không.
2. Log dịch vụ AI có dòng `"msg":"subscription.plan.changed event rejected"` không —
   nếu có, đọc trường `reason`. `stale_event` nghĩa là bản ghi hiện tại đã có
   `occurred_at` mới hơn sự kiện; xem ADR 013 §1.
3. Bộ đếm `grant_rejected_by_reason` trên `GET /api/v1/ai/health` (xem
   `shared-reference.md` §4.1).

## 5. Bán kính và rủi ro — đọc trước khi bấm lần đầu

- **Lần chạy đầu đóng và tạo lại grant cho MỌI tài khoản còn hạn**, kể cả tài khoản
  đang có grant hoàn hảo. Cơ chế chống trùng ở phía AI dựa trên `SourceEventID`, mà
  sự kiện restore mang `SourceEventID` mới, nên nó không chặn được.
- **Snapshot được dựng lại từ cấu hình chính sách lúc chạy**, không phải giá trị lúc
  grant gốc được tạo. Nếu cấu hình hôm nay sai, sai số đó bị đóng băng vào snapshot
  của mọi tài khoản. **Đây là lý do bước 2 là bắt buộc, không phải cho có**: kiểm tra
  `ai_cost_policies` và `ai_cost_policy_operations` trước khi chạy thật.
- **Chạy lại được an toàn.** Event key là `RESTORE_GRANT:<userID>:<version>`, và bản
  sửa không đổi `version`, nên lần hai thấy key đã có và bỏ qua (báo `skipped` /
  `already_restored`).
- **Một transaction cho toàn lượt.** Hoặc mọi tài khoản đều được phát lại sự kiện và
  có bản ghi kiểm toán, hoặc không có gì xảy ra. Đây là chủ ý: một lượt sửa nửa vời
  thì tệ hơn không sửa.

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