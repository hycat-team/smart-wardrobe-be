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

> **Cổng cũ ở đây đã bị gỡ, và lý do đáng ghi lại.** Bản trước chạy
> `$COMPOSE logs --tail=200 ai | grep -c 'recovered from panic'`. Lệnh đó **không đo phiên
> bản**: dòng `recovered from panic` chỉ được ghi khi có panic thật và nó được recover, nên
> một bản deploy khoẻ mạnh in ra `0` — y hệt một bản chưa deploy. Tệ hơn, dòng đó thuộc
> interceptor của bản deploy **trước** 027 (`4990fd59`), nên nó cho cùng kết quả dù 027 có lên
> hay không. Và `grep -c` trả exit code 1 khi tìm thấy 0 dòng, nên dưới `set -e` chính nó làm
> script dừng lại và trông như deploy hỏng.

Kiểm tra đúng là đối chiếu **ảnh đang chạy** với ảnh mà bản phát hành trỏ tới:

```bash
# Ảnh AI đang chạy, và ảnh mà .env.deploy ghim (do lần deploy này ghi)
$COMPOSE ps ai
grep -E '^AI_IMAGE=' .env.deploy

# Container phải VỪA được tạo lại: STATUS ghi "Up vài phút", không phải "Up vài tuần"
```

Đạt khi: cột IMAGE của `$COMPOSE ps ai` bằng đúng giá trị `AI_IMAGE` trong `.env.deploy`, **và**
STATUS cho thấy container vừa khởi động lại. Nếu STATUS ghi vài tuần thì deploy chưa recreate
container — **dừng lại**, không làm tiếp.

#### Chụp bộ đếm từ chối trước khi sửa

Một sự kiện bị bỏ qua hầu như không để lại dấu vết: consumer vẫn khoẻ, hàng đợi vẫn rỗng, và
quyền thì không tới. Thứ duy nhất nói ra điều đó là bộ đếm `grant_rejected_by_reason` mà dịch
vụ AI phơi trong phản hồi sức khoẻ (`services/ai/internal/bootstrap/router.go`;
`internal/modules/aiexecutor/application/usecase/grant/grant_reject_metrics.go`). Chụp lại
trước khi sửa để làm mốc đối chiếu:

```bash
curl -s https://api.closy.hycat.online/api/v1/ai/health \
  | python3 -c 'import json,sys; d=json.load(sys.stdin); print(json.dumps(d.get("grant_rejected_by_reason", {}), indent=2, ensure_ascii=False))'
```

Hai điều phải nhớ khi đọc kết quả:

- Bộ đếm là **theo tiến trình** và **về 0 khi dịch vụ AI khởi động lại**. Bản deploy mang bản
  sửa sẽ khởi động lại AI, nên số liệu của lượt ngày 2026-10-07 có thể đã mất. Đây là **mốc
  so sánh**, không phải bằng chứng lịch sử.
- Các mã lý do đang có: `stale_event`, `policy_snapshot_unavailable`,
  `ai_cost_policy_not_found`, `duplicate_source_event`, `transient_failure`,
  `policy_grant_constraint_violation`. Cả sáu đều nằm trong bản deploy này — 027 phát hành
  toàn bộ trong một tag, nên không còn chuyện "bản deploy chỉ gồm Phase 1–4".

Sau khi sửa xong, chụp lại bộ đếm và đối chiếu với mốc này. Không được xuất hiện thêm lần từ
chối nào cho các tài khoản vừa sửa: nếu có, xem lại mục "Nếu có gì đó hỏng".

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

> **Vì sao cấp lại đúng gói đang có lại sinh ra `EXTENDED`.** Dịch vụ Account chọn loại sự kiện
> theo việc gói có **đổi** hay không: cùng gói → `EXTENDED`. Gói `free` không có thời hạn
> (`duration_days = NULL`), nên payload mang `durationDays = 0`. Trước bản 027 này, parser dịch
> vụ AI từ chối mọi `EXTENDED` có `durationDays <= 0`, nên **mọi** sự kiện sửa cho tài khoản
> `free` bị chết trong hàng đợi — bảy trong tám tài khoản ở đây. Luật đó nay chỉ áp cho sự kiện
> **có** ngày kết thúc (FR-020), nên hình dạng này là hợp lệ.

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

# KHOÁ MỚI. Khoá cũ `027-grant-repair-2026-10-07` đã bị tiêu thụ ở lần chạy trước và sẽ
# trả `duplicate_request` — nó bảo vệ chống chạy trùng, nên đổi hậu tố cho mỗi lần chạy lại.
curl -s -X POST "https://api.closy.hycat.online/api/v1/admin/subscription-grants" \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Idempotency-Key: 027-grant-repair-2026-10-07-b" \
  -H "Content-Type: application/json" \
  --data-binary @/tmp/grant-027.json | python3 -m json.tool
```

### Cổng thật: đọc cơ sở dữ liệu, KHÔNG đọc `data.succeeded`

> **`data.succeeded` KHÔNG chứng minh bản ghi mức quyền đã được ghi.** Endpoint trả về sau khi
> cập nhật bản ghi subscription; consumer AI ghi bản ghi mức quyền **bất đồng bộ về sau** và có
> thể từ chối. Trong lần chạy ngày 2026-10-07, `succeeded` báo thành công trong suốt thời gian
> bảy sự kiện bị chết trong hàng đợi. Đây là FR-021 và SC-009.

Đọc `data.results[]` chỉ để bắt lỗi **đồng bộ** (sai UUID, sai mã gói) — nếu có `status: "error"`:
  - `user_not_found` → sai UUID
  - `plan_not_found` / `plan_inactive` → sai mã gói
  - `target_is_admin` → danh sách có tài khoản admin thật; tài khoản demo `admin` ở đây
    có `role_slug = user` nên không dính, nhưng nếu gặp thì bỏ tài khoản đó ra
  - `duplicate_request` → đã chạy với khoá này rồi, không cần chạy lại
  - `grant_failed` → xem log dịch vụ account

Chỉ đi tiếp khi **không** có dòng nào `status: "error"`. Nhưng điều đó **vẫn chưa** nghĩa là
đã sửa xong — nó chỉ nghĩa là dịch vụ Account không từ chối. Cổng thật là bước 6, và nó đọc
bản ghi mức quyền.

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

**Đây là cổng duy nhất chứng minh việc sửa đã xảy ra.** Consumer của dịch vụ AI xử lý bất
đồng bộ, nên **đợi 10–15 giây** trước khi đọc.

**6a — 8 tài khoản đều có bản ghi mức quyền đang hiệu lực, ảnh chụp đọc được:**

```bash
$PSQL -c "
SELECT g.user_id, g.status, g.effective_from, g.effective_to,
       (g.policy_snapshot ? 'envelope')     AS has_env,
       (g.policy_snapshot ? 'entitlements') AS has_ent
FROM aiexecutor.ai_policy_grants g
WHERE g.user_id IN (
  'ad11ad11-ad11-ad11-ad11-ad11ad11ad11','736445e9-5046-4c63-9611-8baa3f2dc657',
  '8523cbb9-7082-4177-99fd-3c6e26575df6','e8808dea-dd64-4f30-afa2-a4bcebf47a23',
  '1aaa85cf-a219-4e75-aad3-36475025a2ef','01b168be-f497-443c-8383-226e952b6d8e',
  'c24024c5-0905-4926-903d-9b8c7fb2cb27','ad11ad11-ad11-ad11-ad11-ad11ad11ad12')
  AND g.status IN ('active','future')
  AND g.effective_from <= now() AND (g.effective_to IS NULL OR g.effective_to > now())
ORDER BY g.user_id;"
```

Kỳ vọng: **8 dòng**, mỗi tài khoản đúng một dòng, `has_env` và `has_ent` đều **true**. Thiếu
dòng nào thì sự kiện của tài khoản đó chưa được áp — đọc 6b.

**6b — không có sự kiện nào bị từ chối cho 8 tài khoản đó:**

```bash
$COMPOSE logs --tail=500 ai | grep -E 'rejected subscription.plan.changed|plan-changed event could not be applied'
```

Kỳ vọng: **không có dòng nào**. Nếu có, `reason` trong dòng đó chỉ ra nguyên nhân; các mã
thường gặp: `stale_event`, `missing_required_field`, `policy_grant_constraint_violation`.

> **`user_ai_policy_grants_check` không còn là tín hiệu ở log dịch vụ AI.** Trước 027, một vi
> phạm ràng buộc đi qua đường `transient`, mà đường đó log nguyên lỗi driver nên tên ràng buộc
> hiện ra. Sau 027, `23514` là từ chối vĩnh viễn và đi qua `reject`, mà `reject` **bỏ lỗi gốc**
> (xem `deferred.md` D-07). Nên tín hiệu trong log AI nay là mã lý do
> `policy_grant_constraint_violation`; còn tên ràng buộc chỉ còn thấy trong **log PostgreSQL**.

**6c — kiểm tra toàn vẹn:**

```bash
$COMPOSE exec -T postgres psql -U closy -d closy_db -q -f database/scripts/07-verify_policy_grants.sql
```

Mục 1 và mục 4 phải bằng **0**. Mục 4 chỉ bằng 0 sau khi đã áp dụng FR-014 của spec 027 —
trước đó nó vẫn đếm cả bản ghi lịch sử đã đóng, và đó là hành vi cũ đã được sửa.

Mục **4b** thì ngược lại: **khác 0 là đúng**. Nó là dòng thông tin, không chặn, đếm những bản
ghi lịch sử có ảnh chụp hỏng mà cửa sổ đã hết hạn (FR-016). Nếu mục 4 về 0 **và** mục 4b cũng
về 0, đó không phải thành công — đó là dấu hiệu dữ liệu lịch sử đã bị xoá thay vì được đóng lại.

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