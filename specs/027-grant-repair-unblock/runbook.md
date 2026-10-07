# Runbook vận hành — sửa 8 tài khoản có mức quyền AI không đọc được

> **Chạy SAU KHI** bản sửa của spec 027 đã được deploy lên production và đã xác nhận
> qua smoke test. Chạy trước thì thao tác sẽ thất bại y hệt lần hôm nay.
>
> **Không dùng lại `POST /admin/subscription-grants/restore`.** Khoá sự kiện của nó
> (`RESTORE_GRANT:<userID>:<version>`) đã được ghi cho cả 28 tài khoản trong lượt chạy
> ngày 2026-10-07, nên nó sẽ báo `already_restored` vĩnh viễn. Chi tiết ở
> [`docs/operations/grant-restore-runbook.md`](../../docs/operations/grant-restore-runbook.md).

## Ai cần sửa, và vì sao chỉ 8

Lượt khôi phục ngày 2026-10-07 không sửa được 18 tài khoản, nhưng **chỉ 8 trong số đó
thực sự cần sửa**:

| Nhóm | Số | Tình trạng grant | Cần sửa? |
|---|---|---|---|
| Không đọc được `envelope` | 8 | Grant gốc **vẫn hỏng** | **Có** |
| Vi phạm ràng buộc dữ liệu | 10 | Giao dịch lùi nguyên tử, grant gốc **vẫn tốt** | Không |

Nhóm thứ hai chỉ thất bại ở bước "sửa lại một grant vốn đã đúng". Đụng vào chúng là
gây hại, không phải sửa.

### Danh sách 8 tài khoản

Lấy lại danh sách ngay trước khi thao tác — nó có thể đã thay đổi:

```sql
SELECT s.user_id, u.username, s.current_plan_code AS plan_slug
FROM aiexecutor.ai_policy_grants g
JOIN subscription.user_subscriptions s ON s.user_id = g.user_id
LEFT JOIN identity.users u ON u.id = g.user_id
WHERE g.status IN ('active','future')
  AND g.effective_from <= now() AND (g.effective_to IS NULL OR g.effective_to > now())
  AND NOT (g.policy_snapshot ? 'envelope' AND g.policy_snapshot ? 'entitlements')
ORDER BY s.current_plan_code, u.created_at;
```

Tại thời điểm viết runbook này (2026-10-07), truy vấn trả về:

| # | UUID tài khoản | Tên đăng nhập | Gói cần cấp lại |
|---|---|---|---|
| 1 | `ad11ad11-ad11-ad11-ad11-ad11ad11ad11` | `admin` | `free` |
| 2 | `736445e9-5046-4c63-9611-8baa3f2dc657` | `trinh` | `free` |
| 3 | `8523cbb9-7082-4177-99fd-3c6e26575df6` | `huyngse183274` | `free` |
| 4 | `e8808dea-dd64-4f30-afa2-a4bcebf47a23` | `Kyoel` | `free` |
| 5 | `1aaa85cf-a219-4e75-aad3-36475025a2ef` | `linh` | `free` |
| 6 | `01b168be-f497-443c-8383-226e952b6d8e` | `Duong2405.ts` | `free` |
| 7 | `c24024c5-0905-4926-903d-9b8c7fb2cb27` | `Nguyet.No1` | `free` |
| 8 | `ad11ad11-ad11-ad11-ad11-ad11ad11ad12` | `user` | `premium-monthly` |

`admin` và `user` là **tài khoản demo** (`admin@smartwardrobe.com`, `user@smartwardrobe.com`).
Sáu tài khoản còn lại là người thật, tất cả đều ở gói `free` và đăng ký ngày 22–25/06/2026.

> **Không có khách trả tiền nào trong danh sách.** Tài khoản `premium-monthly` duy nhất là
> tài khoản demo `user`.

## Vì sao cấp lại đúng gói đó vẫn sửa được

Cấp lại **cùng một mã gói** tạo ra sự kiện loại `EXTENDED`, và `EXTENDED` **không** đóng bản
ghi cũ — nó **cắt** bản ghi cũ ở thời điểm bắt đầu của kỳ mới (`TruncateGrantsBefore`,
`ai_policy_grant_writer.go:77`). Với 8 tài khoản ở đây thì:

- Bản ghi cũ: `effective_from` = ngày bắt đầu kỳ cước (lâu trước), `effective_to` = NULL →
  được cắt thành `effective_to = now`, trạng thái `đã đóng`.
- Bản ghi mới: `effective_from` = now, ảnh chụp chính sách **đọc được**.

Kết quả: không có hai bản ghi đang hiệu lực chồng lấn, ràng buộc `effective_to >
effective_from` vẫn thoả, và bản ghi cũ rời khỏi cửa sổ hiệu lực nên kiểm tra toàn vẹn
không còn đếm nó.

## Vì sao phải tắt email

Đường cấp gói hành chính **luôn** gửi email cho người nhận
(`admin_grant_batch_uc.go:509`). Tám tài khoản này gồm sáu người thật, và gửi cho họ một
email *"bạn đã được cấp gói"* khi họ không hề mua gói là thông tin sai.

`EMAIL_ENABLED=false` là công tắc dựng riêng để loại email: worker **ack** từng dòng ngay
lúc đọc, nên email bị lo **vĩnh viễn** chứ không phải xếp hàng rồi bắn lại sau khi bật
lại (`email_worker.go:67-71`).

**Đánh đổi của việc tắt:** trong khoảng thời gian tắt, **mọi email giao dịch thật cũng bị
lo** — ví dụ xác nhận thanh toán, xác nhận đăng ký. Cửa sổ này càng ngắn càng tốt. Thao tác
cấp gói là **một** lệnh HTTP, nên toàn bộ phần giữa dưới đây có thể xong dưới 2 phút.

---

## Thao tác

Chạy trên VPS. Thư mục làm việc là thư mục gốc repo trên VPS (`/opt/closy`).

Đặt biến cho gọn:

```bash
cd /opt/closy
COMPOSE="docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml"
PSQL="$COMPOSE exec -T postgres psql -U closy -d closy_db -q"
export ADMIN_TOKEN='<token quản trị viên>'
```

Tên người dùng và tên cơ sở dữ liệu viết thẳng vì chúng **không** phải biến của shell —
`.env.production` là file dành cho docker, không phải để `source`. Nếu production của bạn
đổi tên, đọc `DB_USER` và `DB_NAME` trong file đó rồi thay vào.

### Bước 1 — Chụp số liệu trước khi đụng vào

```bash
$PSQL -c "
SELECT g.status, count(*) AS n
FROM aiexecutor.ai_policy_grants g
JOIN subscription.user_subscriptions s ON s.user_id = g.user_id
WHERE g.status IN ('active','future')
  AND g.effective_from <= now() AND (g.effective_to IS NULL OR g.effective_to > now())
  AND NOT (g.policy_snapshot ? 'envelope' AND g.policy_snapshot ? 'entitlements')
GROUP BY 1;"
```

Kỳ vọng: đúng **8** dòng. Nếu khác, **dừng lại** — có thể đã có ai đó sửa, hoặc môi
trường đích không phải production.

Ghi lại kết quả. Bước 6 sẽ so với nó.

### Bước 2 — Chắc chắn bản sửa đã lên

```bash
$COMPOSE logs --tail=200 ai | grep -c 'recovered from panic'
```

Và kiểm tra dịch vụ AI đang chạy đúng phiên bản mới — xem `docker compose ps` cùng tag image
đang chạy. Nếu chưa deploy xong, **dừng lại**, không làm tiếp.

### Bước 3 — Tắt email và khởi động lại dịch vụ account

Toàn bộ `.env.production` được nạp vào container qua `env_file`
(`docker-compose.prod.yml:111-112`), nên thêm dòng vào đó là đủ — không cần sửa compose.

```bash
cp .env.production .env.production.bak-027
echo 'EMAIL_ENABLED=false' >> .env.production
$COMPOSE up -d --force-recreate account
$COMPOSE exec -T account printenv EMAIL_ENABLED   # phải in ra false
```

`--force-recreate` là để chắc chắn container được dựng lại với nội dung file mới.

Chỉ tiếp tục khi lệnh cuối in ra `false`. Nếu nó in ra rỗng, biến chưa tới được container —
**dừng lại**, không chạy tiếp, vì khi đó bước 4 sẽ gửi 8 email thật.

Bản sao lưu ở trên dùng để khôi phục nếu bạn bỏ sót bước 5.

### Bước 4 — Cấp lại gói cho 8 tài khoản

Dán đúng danh sách UUID và **đúng mã gói** ở bảng phía trên. `planSlug` phải là gói mà
tài khoản **đang** ở, không phải gói bạn muốn cho họ.

```bash
cat > /tmp/grant-027.json <<'JSON'
{
  "items": [
    { "userIdentifier": "ad11ad11-ad11-ad11-ad11-ad11ad11ad11", "planSlug": "free" },
    { "userIdentifier": "736445e9-5046-4c63-9611-8baa3f2dc657", "planSlug": "free" },
    { "userIdentifier": "8523cbb9-7082-4177-99fd-3c6e26575df6", "planSlug": "free" },
    { "userIdentifier": "e8808dea-dd64-4f30-afa2-a4bcebf47a23", "planSlug": "free" },
    { "userIdentifier": "1aaa85cf-a219-4e75-aad3-36475025a2ef", "planSlug": "free" },
    { "userIdentifier": "01b168be-f497-443c-8383-226e952b6d8e", "planSlug": "free" },
    { "userIdentifier": "c24024c5-0905-4926-903d-9b8c7fb2cb27", "planSlug": "free" },
    { "userIdentifier": "ad11ad11-ad11-ad11-ad11-ad11ad11ad12", "planSlug": "premium-monthly" }
  ]
}
JSON

curl -s -X POST "https://api.closy.hycat.online/api/v1/admin/subscription-grants" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Idempotency-Key: 027-grant-repair-2026-10-07" \
  -H "Content-Type: application/json" \
  --data-binary @/tmp/grant-027.json | python3 -m json.tool
```

Đọc `data.succeeded`, `data.failed`, `data.results[]`.

- `succeeded` phải bằng **8**. Nếu có `status: "error"`, đọc `reason` trong `results[]`:
  - `user_not_found` → sai UUID
  - `plan_not_found` / `plan_inactive` → sai mã gói
  - `target_is_admin` → danh sách có tài khoản admin thật; tài khoản demo `admin` ở đây
    có `role_slug = user` nên không dính, nhưng nếu gặp thì bỏ tài khoản đó ra
  - `duplicate_request` → đã chạy với khoá này rồi, không cần chạy lại
  - `grant_failed` → xem log dịch vụ account

Chỉ đi tiếp khi **không** có dòng nào `status: "error"`.

### Bước 5 — Bật lại email ngay

**Không được bỏ qua bước này.** Email còn tắt là mọi email giao dịch của production đều
bị lo.

```bash
sed -i '/^EMAIL_ENABLED=false$/d' .env.production
$COMPOSE up -d --force-recreate account
$COMPOSE exec -T account printenv EMAIL_ENABLED   # phải in ra true
```

Xác nhận lại rồi mới sang bước 6. Nếu nó in ra rỗng hoặc `false`, dịch vụ account đang chạy
với email tắt — **đó là sự cố đang âm thầm bỏ mọi email giao dịch**, xử lý ngay.

### Bước 6 — Xác minh

Consumer của dịch vụ AI xử lý bất đồng bộ. **Đợi 10–15 giây** rồi chạy lại truy vấn ở bước
1. Kỳ vọng: **0 dòng**.

Nếu vẫn còn dòng, kiểm tra:

```bash
$COMPOSE logs --tail=300 ai | grep -E 'stale_event|user_ai_policy_grants_check|transient'
```

Sau đó chạy toàn bộ kiểm tra toàn vẹn:

```bash
$COMPOSE exec -T postgres psql -U closy -d closy_db -q -f database/scripts/07-verify_policy_grants.sql
```

Mục 1 và mục 4 phải bằng **0**. Mục 4 chỉ bằng 0 sau khi đã áp dụng FR-014 của spec 027 —
trước đó nó vẫn đếm cả bản ghi lịch sử đã đóng, và đó là hành vi cũ đã được sửa.

Kiểm tra thêm không có bản ghi đang hiệu lực chồng lấn:

```bash
$PSQL -c "
SELECT user_id, count(*) AS n
FROM aiexecutor.ai_policy_grants
WHERE status IN ('active','future')
  AND effective_from <= now() AND (effective_to IS NULL OR effective_to > now())
GROUP BY user_id HAVING count(*) > 1;"
```

Kỳ vọng: **không có dòng nào**.

### Dọn dẹp

```bash
rm -f /tmp/grant-027.json
```

Giữ lại `.env.production.bak-027` cho tới khi xác nhận xong, rồi xoá.

## Kiểm tra lại bằng hành vi thật

Sau khi mọi thứ về 0, thử đổi gói thật một tài khoản free trong danh sách và xác nhận
quyền AI đi theo. Đây là kiểm chứng duy nhất chứng minh FR-001 đã hết hiệu lực — trước đó
thay đổi gói của tài khoản này bị bỏ qua âm thầm.

## Nếu có gì đó hỏng

**Sự kiện đã phát thì không có đường lùi.** Bản ghi mức quyền cũ đã bị cắt và bản ghi mới
đã tạo. Cách khôi phục là dựng lại từ dữ liệu hiện tại bằng một lần cấp gói nữa với
`Idempotency-Key` khác — cùng cơ chế, và vẫn an toàn về ràng buộc cửa sổ.

## Ghi chú về ngữ nghĩa kiểm toán

Bản ghi kiểm toán của thao tác này ghi là **"cấp gói"** cho tám tài khoản, trong khi thực
tế chúng **vốn đã ở đúng gói đó** — không có ai mua gì ở đây. Đây là hệ quả trực tiếp của
việc dùng đường ghi hành chính để sửa dữ liệu, và nó được chấp nhận có chủ đích: quyền
không đổi, chỉ ảnh chụp chính sách được dựng lại đúng như mong muốn.

Sự kiện mang nhãn `admin_restore` sẽ **không** được tạo. Ai đọc lịch sử sự kiện sau này sẽ
thấy một lần cấp gói hành chính, không phải một lần khôi phục — đó là dấu vết trung thực
của việc đã làm gì thật.