# Nghiên cứu & Quyết định Thiết kế — 026 Sổ đăng ký chiến dịch trong CSDL

**Đầu vào**: [spec.md](./spec.md) (kèm 4 quyết định làm rõ) · **Hiến pháp**:
`.specify/memory/constitution.md` v3.3.0
**Ngày**: 2026-10-03

Mỗi quyết định đều kèm **bằng chứng đọc từ mã nguồn hiện có**. Những mục đánh dấu **BẰNG CHỨNG** là
kết quả kiểm chứng, không phải suy luận.

---

## R1 — Nguồn sự thật duy nhất: bảng `subscription.campaigns`, gỡ hẳn khối `campaigns` khỏi cấu hình

**Quyết định**: bảng `subscription.campaigns` là nguồn duy nhất. Xoá `Config.Campaigns`,
`CampaignConfig` (`shared/config/config.go:39,130-172`), `validateCampaigns`/`validateCampaign`/
`campaignCodePattern` (`shared/config/validation.go:174-227`), và phần phân tích/ghi đè khối
`campaigns` trong `loader.go`. Giữ `CampaignSweepConfig` (`config.go:174-188`).

**BẰNG CHỨNG — vì sao cấu hình không còn chỗ đứng**: khối `campaigns` hiện nằm trong
`configs/config.yaml:245` và `configs/config.production.yaml:238`, được biên dịch vào image. Mọi thay
đổi chiến dịch = commit + tag + deploy. Cấu hình vẫn được nạp tĩnh, **không** chạm CSDL
(`loadConfig` trong `loader.go`), nên bảng không thể được nạp lúc khởi động — và không cần: sổ được
đọc lúc chạy ở ba chỗ (`campaign_uc.go:123`, `campaign_query_uc.go:110,305`, `sweep_uc.go:140,154,
356,450`).

**Vì sao không giữ hai nguồn**: `quota`/`reserve` quyết định trần cứng; hai nguồn với hai giá trị
khác nhau sẽ làm mọi phép đếm sai theo cả hai bản. Luật hoà giải giữa hai nguồn chính là chỗ bất biến
bị phá. Một nguồn xoá cả lớp lỗi.

**Phương án đã xét**:
- Giữ YAML làm nguồn chính, DB làm "bản sao" — bị loại: hai nguồn, và cấu hình vẫn buộc deploy.
- Đọc YAML lúc khởi động rồi upsert vào DB — bị loại: nạp cấu hình chạm CSDL (trái §I/§VIII) và vẫn
  cần deploy để đổi.

---

## R2 — Kiểm tra hình dạng: chuyển từ "chặn khởi động" sang "chặn ghi + ràng buộc CSDL"

**Quyết định**: 7 quy tắc của SC-014 (`campaign_validation_test.go`) tách thành:
- **Hình dạng dữ liệu** → API quản trị từ chối (thông báo theo trường) + `CHECK`/`PRIMARY KEY` trên
  bảng làm chốt chặn cuối.
- **Chỉ biết lúc chạy** (`plan_slug` trỏ tới gói tồn tại và active) → **giữ nguyên quyết định của
  025**: không chặn gì; mỗi lượt bị từ chối ghi `plan_unavailable`.

**BẰNG CHỨNG**: `validateCampaign` (`validation.go:184-227`) hiện kiểm mã rỗng/định dạng/trùng,
`plan_slug` rỗng, `quota >= 1`, `reserve >= 0`, `starts_at` có múi giờ, `ends_at > starts_at`. Phần
`plan_slug` tồn tại/active **đã** nằm ngoài hàm này (`validation.go:171-173` ghi rõ), và được kiểm ở
`campaign_uc.go:136-143` — nơi mã `SkipPlanUnavailable` được ghi.

**Hệ quả**: không còn "chiến dịch sai làm sập khởi động" vì không nạp chiến dịch lúc khởi động. Bảo
đảm tương đương: dữ liệu sai hình dạng **không thể được ghi**.

---

## R3 — Bất biến `quota`/`reserve`/`plan_slug`: tầng ứng dụng + trigger CSDL

**Điều kiện chính xác**: `quota`, `reserve`, `plan_slug` bất biến **ngay khi tồn tại ít nhất một dòng
`subscription.campaign_claims` với `campaign_code` tương ứng**. Không dùng `now >= starts_at`.

**BẰNG CHỨNG — vì sao số lượt cấp, không phải đồng hồ**: `grant_bucket` đóng băng phần ngân sách tại
lúc ghi (`entities/campaign_claim.go:58-62`), và mốc ưu tiên dựa trên `quota` tại lúc chốt
(`quota_uc.go:160-176`). Một chiến dịch chưa cấp dòng nào thì chưa có dòng lịch sử nào để diễn giải
lại — sửa ngân sách an toàn. "Có lượt cấp" bao hàm "đã qua `starts_at`", nên điều kiện này chặt hơn
và đơn giản hơn điều kiện đồng hồ.

**Nơi thực thi**:
1. **Tầng ứng dụng** — use case `registry_uc.go` đếm lượt cấp trong transaction cập nhật; nếu > 0 và
   có đổi một trong ba cột, trả lỗi tiếng Việt `422`.
2. **Trigger `BEFORE UPDATE`** trên `subscription.campaigns` — nâng ngoại lệ khi một trong ba cột đổi
   mà tồn tại lượt cấp. Đây là lớp không vòng qua được bằng SQL gõ tay.

**Vì sao không `CHECK`**: `CHECK` là ràng buộc theo từng hàng, không nhìn thấy `campaign_claims`.

**Vì sao không chỉ tầng ứng dụng**: yêu cầu "không ai sửa bảng này bằng SQL trực tiếp" (FR-062);
kỷ luật ứng dụng không ràng buộc được câu SQL gõ tay.

**Phương án đã xét**: trigger `AFTER UPDATE` — bị loại vì quá muộn (giá trị đã ghi); `RULE` — bị loại
vì khó quan sát và gây bất ngờ. `BEFORE UPDATE` nâng ngoại lệ là lựa chọn rõ ràng nhất.

---

## R4 — Đóng cưỡng bức: cột `closed_at`, trạng thái `closed`, one-way

**Quyết định**: thêm `closed_at TIMESTAMPTZ NULL`. `closed_at IS NOT NULL` = đã đóng. Trạng thái dẫn
xuất `status` nhận giá trị thứ sáu `closed`, **ưu tiên cao nhất**. Tập `reason` **không đổi**: tài
khoản gặp chiến dịch đã đóng nhận `campaign_sleeping` (đã dùng cho "quá `ends_at`").

**BẰNG CHỨNG**: máy trạng thái 5 giá trị ở `campaign_query_uc.go:267-297`, hằng số ở
`campaign_dto.go:242-259`. `deriveCampaignStatus` đã kiểm `expired` trước hết; thêm `closed` kiểm
trước cả `expired` là thay đổi một dòng, không xáo trộn các nhánh cũ. `campaign_sleeping` đã là mã
của điều kiện 2 (`admission_uc.go:76-78,152-159`) cho chiến dịch ngoài cửa sổ, nên dùng lại đúng
nghĩa.

**Ảnh hưởng hợp đồng**: giá trị `status` là **cộng thêm** vào enum; hình dạng JSON không đổi; tập 12
mã `reason` giữ nguyên. FE phải chịu giá trị lạ (đã có khuyến nghị tương tự cho `reason` trong
`frontend-guide.md` của 025).

**Một chiều**: không có thao tác mở lại qua API. Mở lại có thể cấp cho tài khoản tạo trong khoảng đã
đóng và làm mất nghĩa nhật ký. Muốn chạy lại → mở mã mới (nguyên tắc "end it and open a new code"
của 025 R2b).

---

## R5 — Chuyển `launch-2026-10`: seed một lần, không FK, rollback không mất dữ liệu

**Quyết định**: hai migration: (1) tạo bảng + trigger; (2) seed `launch-2026-10` idempotent
(`INSERT ... ON CONFLICT (code) DO NOTHING`) với đúng sáu giá trị production. Không `ALTER` bảng
`campaign_claims`/`campaign_sweep_runs`, không thêm khoá ngoại.

**BẰNG CHỨNG**: `launch-2026-10` có thật trong `configs/config.production.yaml:239-247` (quota 50,
reserve 5, `starts_at` `2026-10-02T00:00:00+07:00`). `campaign_claims.campaign_code` là `VARCHAR(64)`
không FK (`20261001125026_add_campaign_claims.sql:61`), `campaign_sweep_runs.campaign_code` là khoá
chính không FK (`20261002050721_add_campaign_sweep_runs.sql:47`).

**Vì sao không FK**: hai cột là khoá lịch sử bằng văn bản; thêm FK buộc phải tái dựng thuộc tính cho
mọi mã lịch sử (migration không làm trung thực được) và có thể chặn migration trên dữ liệu thật. Bù
lại, không có đường xoá chiến dịch, nên tính toàn vẹn do "chỉ tạo/sửa/đóng" giữ.

**Rollback**: down xoá hàng seed rồi xoá bảng; `campaign_claims` nguyên vẹn (không FK). Khôi phục
khối YAML + binary 025 là đường lùi vận hành.

**Thứ tự triển khai**: migration trước, binary sau. Trong một cửa sổ triển khai, khối YAML có thể còn
làm bản sao rollback, nhưng binary mới **không đọc** nó.

---

## R6 — Chọn chiến dịch: một truy vấn "còn thức", bộ chọn thuần, tie-break tất định

**Quyết định**: repository đọc một truy vấn các chiến dịch **còn thức**:
```sql
SELECT * FROM subscription.campaigns
WHERE closed_at IS NULL
  AND starts_at <= :now
  AND (ends_at IS NULL OR :now < ends_at)
ORDER BY starts_at DESC, code DESC;
```
rồi **bộ đánh giá thuần** (`admission_uc.go`) lọc tiếp theo `registered_at` và chọn `starts_at` lớn
nhất. Tie-break `code DESC` làm kết quả **tất định** khi hai chiến dịch cùng `starts_at`.

**BẰNG CHỨNG**: `selectCampaign` (`admission_uc.go:191-215`) đã tách hai instant: cửa sổ so
`registered_at`, "còn thức" so `now`. Giữ nguyên hai vế đó nhưng nguồn danh sách đổi thành repository.
`isAwake` là **một** định nghĩa dùng cho cả điều kiện 2 và bộ chọn (`admission_uc.go:166-174`), nên
truy vấn "còn thức" và bộ lọc `registered_at` không thể lệch nhau.

**Chỉ mục**: `ix_campaigns_starts_at (starts_at DESC)` phục vụ `ORDER BY`. Bảng vài hàng.

**Vì sao bộ chọn vẫn thuần**: bộ đánh giá nhận `[]*entities.Campaign` (đọc sẵn) chứ không tự truy
vấn, nên các điều kiện 1–7 vẫn kiểm thử không cần CSDL. Đây là lý do tách `campaignEvaluator` khỏi
`CampaignUseCase` từ 025 (`admission_uc.go:15-31`).

---

## R7 — Không cache sổ trong 026

**Quyết định**: không thêm cache; mỗi thông điệp/lần quét/lần tra cứu đọc CSDL.

**Lý do**: một cache tạo vấn đề **vô hiệu hoá xuyên bản sao** — chiến dịch vừa đóng vẫn được cấp bởi
một bản sao còn giữ bản cũ, đúng thứ mục tiêu "đóng sớm" chặn. Đổi lại, một truy vấn có chỉ mục trên
bảng vài hàng là rẻ. Hiện tại bộ đọc là bộ nhớ cấu hình (`cfg.Campaigns`), nhưng tần suất đăng ký
thấp và giao dịch tạo tài khoản đã nặng hơn nhiều.

**Nếu sau này cần**: TTL ngắn + khoá phiên bản tăng khi có thao tác quản trị; ghi vào `deferred.md`,
không làm trong 026.

---

## R8 — RBAC + nhật ký kiểm toán: qua outbox, mã trong payload, đọc bằng hợp đồng hẹp

**Quyết định**:
- Endpoint ghi/đọc nhật ký nằm trong nhóm `/admin/campaigns`, dùng `RolesAuthorize(roleslug.Admin)` +
  `adminGrantRateLimit()` sẵn có (`router.go:144-161`).
- Ghi kiểm toán qua `IAuditLogContract.Record` — **BẰNG CHỨNG**: hợp đồng này phát qua outbox
  (`contract/admin/audit_log_contract.go:25-28`, `publisher.Publish(ctx, TopicAuditLog, input)`), nên
  **dòng outbox nằm trong cùng transaction** nghiệp vụ, còn bảng `shared.admin_audit_logs` được ghi
  bất đồng bộ bởi `audit_log_consumer`. Đây đúng mẫu 025 đã dùng cho lần cấp gói
  (`grant_uc.go:176-203`).
- `targetType = "campaign"` (thêm hằng `adminaudit.Campaign`), `targetId = NULL` vì mã là chuỗi còn
  `targetId` là `*uuid.UUID` (`dto/admin/audit.go:15`), mã nằm trong `payload.campaignCode`.
- Thêm hằng hành động `CampaignCreate/CampaignUpdate/CampaignClose` vào
  `shared/domain/constants/adminaudit/action.go` theo quy ước chấm (`subscription.plan_create`…).
- **Đọc nhật ký** bằng **hợp đồng hẹp** `ICampaignAuditLookup` trong `services/account/contract/admin/`,
  cài bằng một bộ chuyển đổi trong module `admin`, trả `dto/admin.AdminAuditLogDTO`.

**Vì sao không mở rộng `IAuditLogContract`**: hợp đồng đó được phục vụ qua gRPC với adapter client và
server (`contracts/admin/v1/client/audit_log_client.go`, `admin/grpc/server.go`), nên thêm phương
thức bắt buộc sửa proto — thay đổi hợp đồng xuyên dịch vụ. Tiền lệ: `AdminGrantLookup` (024) và
`CampaignAccountLister` (R5 của 025) đều là hợp đồng hẹp riêng.

**Vì sao cần endpoint riêng**: bộ lọc sẵn có `targetId` là UUID
(`admin_audit_handler.go:67-71`), không lọc được mã chuỗi; thêm `ListByCampaignCode` lọc
`target_type = 'campaign' AND payload->>'campaignCode' = ?`.

---

## R9 — Quét bù đọc sổ mới mỗi lần chạy, nhịp vẫn ở cấu hình

**Quyết định**: `sweep_uc.go` thay `uc.cfg.Campaigns` (`sweep_uc.go:140,154,356-357,449-452`) bằng một
truy vấn repository `ListOpen()` (chỉ `closed_at IS NULL`). Nhịp quét (`cfg.CampaignSweep`) **ở lại
cấu hình** — nó là núm của cơ chế, không phải dữ liệu chiến dịch, và giữ nó ở cấu hình giữ được tính
không-chạm-CSDL khi nạp.

**Vì sao chiến dịch tạo giữa chừng không bị bỏ sót**: danh sách được đọc lại mỗi lần chạy, nên lần
quét kế tiếp thấy nó; tài khoản tạo giữa lúc tạo chiến dịch và lần quét kế tiếp nằm trong
`[starts_at, now)` nên được quét ở lần đó; `run_on_start` đóng khoảng hở khi khởi động lại. Trễ nhất
một chu kỳ.

**BẰNG CHỨNG**: `earliestCampaignStart` và cửa sổ quét tính từ danh sách cấu hình hiện tại; đổi sang
danh sách repository giữ nguyên thuật toán, chỉ đổi nguồn.

---

## R10 — Không thêm biến môi trường

**Quyết định**: không env cho dữ liệu chiến dịch. Sau 026, khối `campaigns` không còn; chiến dịch là
dữ liệu vận hành quản lý qua API, và CSDL là biên môi trường (mỗi môi trường một DB). Không phá
§VIII: không có giá trị phụ thuộc môi trường nào bị viết cứng. `campaign_sweep.interval_seconds` giữ
là YAML như 025 (`config.go:181-188`). Comment `.env*` (nếu sửa) viết tiếng Anh.

---

## R11 — Chống ghi đè ngầm: cột `version` + If-Match

**Quyết định**: `version INTEGER NOT NULL DEFAULT 1`; response trả `version`; PATCH/close mang phiên
bản mong đợi; `UPDATE ... WHERE code = ? AND version = ?`; 0 dòng → `412`. Đóng một chiến dịch **đã
đóng** idempotent thành công (kiểm idempotent trước kiểm phiên bản).

**BẰNG CHỨNG**: tiền lệ `user_subscriptions.version`. Cần thiết vì hai quản trị viên cùng sửa có thể
ghi đè ngầm, và nhật ký kiểm toán sẽ ghi một chuỗi thay đổi không khớp trạng thái cuối.

---

## R12 — FR-045: đọc lại hàng chiến dịch dưới khoá chia sẻ trong transaction cấp gói

**Quyết định**: trong `uow.Execute` của `campaign_uc.go` (sau khi đã lấy khoá tư vấn qua
`quotaDecider.decideAndClaim`), đọc lại hàng chiến dịch bằng `SELECT ... FOR SHARE` và kiểm
`closed_at IS NULL`; nếu đã đóng, bỏ qua không ghi.

**BẰNG CHỨNG**: hiện điều kiện 1–7 chạy **ngoài** transaction (`campaign_uc.go:116-143`), dùng danh
sách cấu hình bất biến trong bộ nhớ. Với nguồn CSDL, một lệnh đóng có thể commit sau khi điều kiện 7
đã chọn chiến dịch nhưng trước khi lượt cấp ghi. `FOR SHARE` (không phải `FOR UPDATE`) đủ để chặn
`UPDATE`/`DELETE` của lệnh đóng/sửa trong khi nhiều lượt cấp đọc song song.

**Không deadlock**: mọi lượt cấp đã được tuần tự hoá bởi khoá tư vấn theo `campaign_code`
(`campaign_claim_repo_impl.go:52-56`), nên thứ tự "advisory → row share" là cố định; đường đóng/sửa
chỉ lấy row lock, không lấy advisory, nên không tạo chu trình.

---

## R13 — Vị trí mã: không tách module mới; handler đọc và ghi tách riêng

**Quyết định**: sổ ở module `subscription`; handler tra cứu (chỉ đọc) giữ nguyên, thêm handler đăng
ký (ghi) riêng. Use case ghi tách khỏi use case đọc.

**BẰNG CHỨNG**: 025 R14 đã lập luận không tách module mới; lý do còn nguyên (dùng chung bảng gói,
chung đường đổi gói, chung hợp đồng tài khoản). Tách handler để bảo đảm chỉ-đọc của FR-069 vẫn là
ràng buộc kiểu (handler tra cứu giữ interface không có phương thức ghi).

---

## R14 — Bán kính gỡ cấu hình

**Quyết định**: xoá `campaign_validation_test.go`; và `campaign_config_files_test.go` (025 T106) phải
**được đảo chiều**, vì nó khẳng định cả hai tệp YAML **phải có** khoá `campaigns` và phải khai đúng 2
chiến dịch ở dev / 1 ở production — sau khi gỡ khối đó, những khẳng định đó sai và tệp gọi
`validateCampaign` mà T011 xoá nên `shared/config` sẽ không biên dịch. Chiến dịch mẫu ở dev được tạo
qua API quản trị, không qua tệp cấu hình.

**BẰNG CHỨNG**: `campaign_config_files_test.go:12` ("025 T106 … BOTH shipped config files declare a
`campaigns` block"), `:52-69` (gọi `validateCampaign` cho từng mục), `:98-113` (khẳng định cả hai tệp có
khoá `campaigns` và `campaign_sweep`), `:123-139` (khẳng định "2 ở dev, đúng 1 ở production"). Cần sửa
**cả hai** tệp kiểm thử, không chỉ một.

---

## R15 — Trigger: chi tiết cài đặt

**Quyết định**: một hàm `subscription.campaigns_guard_immutable_budget()` +
`CREATE TRIGGER trg_campaigns_immutable_budget BEFORE UPDATE ON subscription.campaigns FOR EACH ROW`.
Hàm nâng ngoại lệ khi `(NEW.quota, NEW.reserve, NEW.plan_slug) IS DISTINCT FROM (OLD.quota,
OLD.reserve, OLD.plan_slug)` **và** `EXISTS (SELECT 1 FROM subscription.campaign_claims WHERE
campaign_code = OLD.code)`. Dùng `IS DISTINCT FROM` để bắt cả trường hợp NULL (dù ba cột đều
`NOT NULL`, cách viết vẫn đúng nếu sau này có cột cho phép NULL). `plan_slug` **cũng** bất biến vì
`campaign_claims.plan_slug` đã chụp gói đã cấp (`campaign_claim.go:47`) — đổi gói giữa chiến dịch sẽ
khiến sổ mâu thuẫn với cấu hình.

**BẰNG CHỨNG**: `campaign_claims.plan_slug` là bản sao tại lúc cấp; mốc ưu tiên và số đếm đều dựa
trên `quota`/`reserve`. PostgreSQL thuần, không khoá nhà cung cấp.

**Phương án đã xét**: `EXCLUDE`/`CHECK` — không diễn tả được điều kiện xuyên bảng. Chỉ tầng ứng dụng
— vòng qua được bằng SQL.

---

## R16 — `budgetLocked` dẫn xuất từ hai đếm sẵn có

**Quyết định**: `budgetLocked = (grantedMainCount + grantedCompensationCount) > 0`. Không thêm cột,
không thêm truy vấn: `summarise` đã đọc hai đếm (`campaign_query_uc.go:184-191`). FE dùng nó để khoá ô
sửa `quota`/`reserve`/`plan_slug` mà không phải tự đoán.

**BẰNG CHỨNG**: hai đếm là hai số độc lập (R2b của 025); tổng của chúng chính là "đã cấp ít nhất một
lượt" vì mỗi lượt cấp thuộc đúng một bucket (`chk_campaign_claims_bucket`).

---

## R17 — Chọn gói tin khi đọc: `status` và `reason` độc lập

**Quyết định**: giữ nguyên quan hệ đã có: `eligibility` là thùng thô, `reason` là mã chẩn đoán. Đóng
cưỡng bức chỉ ảnh hưởng `status` (thêm `closed`) và làm điều kiện 2 trả `campaign_sleeping`. Không
đổi 6 giá trị `eligibility`, không đổi 12 mã `reason`.

**BẰNG CHỨNG**: `campaign_dto.go:196-236` (6 giá trị) và `contracts/campaign-query-api.md §3.5` (12
mã) của 025; 026 giữ cả hai.
