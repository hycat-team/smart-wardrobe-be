# Feature Specification: Sổ đăng ký chiến dịch trong cơ sở dữ liệu (thay cho cấu hình)

**Feature Branch**: `026-campaign-registry-db`

**Created**: 2026-10-03

**Status**: Draft

**Input**: User description: "Tạo spec 026 — campaign registry trong CSDL, thay cho quản lý chiến dịch qua file cấu hình. Bốn mục tiêu bắt buộc: (1) mở chiến dịch mà không cần deploy; (2) sửa hạn mức/thời gian linh hoạt với bất biến quota/reserve của chiến dịch đang chạy được nói rõ; (3) đóng chiến dịch sớm; (4) giao diện quản lý trong trang admin qua API có nhật ký kiểm toán. Một nguồn sự thật duy nhất. Không sửa bất kỳ tệp nào trong `specs/025-`; ghi nhận sự thay thế trong 026."

> **Quan hệ với 025.** Feature 025 (`specs/025-signup-campaign-grant/`) đã triển khai và đang chạy
> trên production với chiến dịch `launch-2026-10` (quota 50, reserve 5) và khoảng 55 dòng
> `subscription.campaign_claims` thật. Tài liệu này **không sửa** bất kỳ tệp nào trong `specs/025-`.
> Mọi điểm mà 026 làm cho một yêu cầu của 025 trở nên sai đều được ghi lại ở mục
> **"Sự thay thế so với 025"** bên dưới, và 025 vẫn là bản ghi lịch sử của chính nó.

---

## 1. Thuật ngữ

Tài liệu dùng **tiếng Việt trong văn xuôi** nhưng **mã thật trong cơ sở dữ liệu** khi nói về giá
trị cụ thể.

| Cách viết trong văn xuôi | Mã thật trong cơ sở dữ liệu | Ở đâu | Ghi chú |
|---|---|---|---|
| sổ đăng ký chiến dịch | `subscription.campaigns` | bảng mới của 026 | Nguồn sự thật **duy nhất** về việc chiến dịch nào tồn tại |
| chiến dịch ra mắt | `launch-2026-10` | `subscription.campaigns.code` | Đang chạy trên production; được seed một lần khi migrate |
| lượt đã cấp | `subscription.campaign_claims` | bảng của 025 | **Không đổi lược đồ.** `campaign_code` tiếp tục là khoá nhận diện, không đặt lại |
| trạng thái vận hành quét bù | `subscription.campaign_sweep_runs` | bảng của 025 | Tiền lệ đưa trạng thái ra khỏi bộ nhớ; **không đổi lược đồ** |
| đóng cưỡng bức | `closed_at` | cột mới trên `subscription.campaigns` | `NULL` = chưa đóng; khác `NULL` = đã đóng bởi người vận hành |
| khoá ngân sách | `budgetLocked` | trường dẫn xuất trong response admin | `true` khi đã có ít nhất một lượt cấp; khoá `quota`/`reserve`/`plan_slug` |
| trạng thái chiến dịch | `status` | trường trong response admin | 5 giá trị của 025, **cộng** giá trị mới `closed` |
| lý do bỏ qua | `reason` | trường trong response admin | **Giữ nguyên đúng 12 mã** của 025, không thêm mã mới |

---

## 2. Bối cảnh

### 2.1 Vì sao cấu hình không còn đủ

025 đã làm đúng điều 025 hứa: chiến dịch là **dữ liệu, không phải mã**, và mở một chiến dịch mới là
một mục YAML mới. Nhưng "dữ liệu" trong 025 vẫn nằm trong **tệp cấu hình được biên dịch vào image**.
Hệ quả vận hành:

| Mong muốn vận hành | Cách làm hiện tại (025) | Chi phí thật |
|---|---|---|
| Mở chiến dịch lúc 00:00 | Sửa `config.production.yaml` → commit → tag → chờ CI → deploy | Có thể trượt đúng thời điểm ra mắt vì một quy trình, không vì một lỗi |
| Đổi hạn mức / thời gian | Sửa config → **deploy lại** | Một thay đổi nghiệp vụ biến thành một lần phát hành mã nguồn |
| Đóng chiến dịch sớm | **Không có đường nào** — FR-013 của 025 chỉ cho *tự ngủ* | Phải chờ `ends_at` hoặc chờ hết hạn mức |
| Biết ai đổi gì | `git blame` trên tệp YAML | Không có nhật lý kiểm toán, không biết ai/lúc nào/đổi gì |

Mục tiêu của 026 là đưa **việc chiến dịch nào tồn tại** ra khỏi image và vào cơ sở dữ liệu, để bốn
thao tác trên trở thành thao tác dữ liệu có kiểm toán, **không cần deploy**.

### 2.2 Bốn mục tiêu bắt buộc

1. **Mở chiến dịch mà không cần deploy** — thêm một chiến dịch chỉ bằng một thao tác API quản trị.
2. **Sửa hạn mức / thời gian linh hoạt** — kể cả `quota`, `reserve`, `starts_at`, `ends_at`, với
   quy tắc rõ ràng cho bất biến `quota`/`reserve` của chiến dịch đã bắt đầu cấp.
3. **Đóng chiến dịch sớm** — một đường đóng cưỡng bức, ngoài đường *tự ngủ* của FR-013.
4. **Giao diện quản lý trong trang admin** — qua API quản trị, có nhật ký **ai / lúc nào / đổi gì**.
   Không ai sửa bảng này bằng SQL trực tiếp.

### 2.3 Những gì đã có sẵn, không thiết kế lại

- `campaign_code` **đã là khoá** trong `subscription.campaign_claims` và
  `subscription.campaign_sweep_runs`. 026 **dùng lại**, không đặt lại khoá và **không thêm khoá
  ngoại** (lý do ở mục 3 câu 5).
- `campaign_sweep_runs` **đã là bảng trạng thái vận hành trong CSDL** — tiền lệ cho việc đưa trạng
  thái ra khỏi config, và nó **nằm ngoài phạm vi 025**, nên 026 không đổi nó.
- Cấu trúc trường chiến dịch đã có trong `shared/config/config.go` → `CampaignConfig`: `code`,
  `plan_slug`, `quota`, `reserve`, `starts_at`, `ends_at`. 026 chuyển **đúng sáu trường này** thành
  cột bảng, không phát minh trường nghiệp vụ mới.
- API tra cứu đã có ở Phase 7 của 025 (`/api/v1/admin/campaigns/...`), gồm 4 endpoint chỉ đọc. 026
  phải nói rõ chúng đọc CSDL thế nào mà **không phá hợp đồng đã lên production**.

### 2.4 Ràng buộc bắt buộc

- **Một nguồn sự thật duy nhất.** Không có kiến trúc hai nguồn.
- Nạp cấu hình **không được** chạm CSDL (hiến phán §I, §VIII) — đó là lý do `plan_slug` được kiểm
  lúc chạy, không lúc khởi động.
- `services/ai` **không** được sửa (hiến phán §I). AI chỉ nghe `subscription.plan.changed`.
- **Không thêm** loại sự kiện đổi gói mới (FR-092 của 025) — `subscriptioneventtype.All` đúng 7.
- Không lồng `uow.Execute`.
- Tài liệu trong `specs/` viết **tiếng Việt**. Comment trong mọi tệp `.env*` viết **tiếng Anh**.
- Dùng đường dẫn **tương đối**.

---

## Clarifications

### Session 2026-10-03

- Q: Khi một lượt cấp đang xử lý (đã qua điều kiện 1–7, chưa vào transaction), người vận hành đóng chiến dịch hoặc sửa ngân sách thì hàng chiến dịch có được đọc lại trong transaction cấp gói không? → A: **Có — đọc lại dưới khoá chia sẻ trong cùng transaction cấp gói.** Đóng/sửa ngân sách và cấp gói nối tiếp nhau, nên "chiến dịch đã đóng không cấp gói" đúng tuyệt đối kể cả khi tương tranh. Xem FR-045.
- Q: Đóng cưỡng bức ảnh hưởng tới tập `status` thế nào? → A: **Thêm giá trị thứ sáu `closed`** (ưu tiên cao nhất), là thay đổi cộng thêm vào enum; hình dạng response và 12 mã `reason` không đổi. Xem FR-032.
- Q: Hai quản trị viên cùng sửa/đóng một chiến dịch thì có chống ghi đè ngầm không? → A: **Có — kiểm tra phiên bản (If-Match).** PATCH/close mang phiên bản mong đợi; lệch thì từ chối `412`, không ghi đè ngầm. Xem FR-055.
- Q: Màn hình nhật ký chiến dịch đọc từ đâu khi mã chiến dịch là chuỗi mà bộ lọc `targetId` sẵn có nhận UUID? → A: **Thêm endpoint riêng `GET /api/v1/admin/campaigns/{code}/audit`.** Lưu `targetType = "campaign"`, `targetId = NULL`, mã nằm trong payload; endpoint lọc theo `payload.campaignCode`. Không đụng API nhật ký chung. Xem FR-063.

### Hiệu chỉnh sau `/speckit-analyze`

Bốn điểm dưới đây **sửa chính tài liệu này** sau khi đối chiếu với mã nguồn thật; chúng không phải
yêu cầu mới của nhà phát triển, mà là chỗ tài liệu đã mô tả sai hoặc mơ hồ.

- **Kiểm toán là bất đồng bộ (FR-060, SC-006, SC-011, US5).** Bản đầu nói "bản ghi kiểm toán nằm
  trong cùng giao dịch". Đối chiếu `services/account/contract/admin/audit_log_contract.go:27` cho
  thấy `Record` phát sự kiện qua **outbox**: **dòng outbox** nằm trong cùng transaction, còn **dòng
  `admin_audit_logs`** do consumer ghi sau. Spec sửa lại đúng cơ chế này và gắn độ trễ vào tiêu chí
  đo được.
- **Bảng ánh xạ mã lỗi là duy nhất (FR-054).** Bản đầu viết "`400`/`422`" chung cho cả sai hình dạng lẫn
  vi phạm bất biến, trong khi hợp đồng đã tách hai loại. Spec nay giữ **một** bảng ánh xạ và coi đó là
  chuẩn; `400` = hỏng hình dạng, `422` = sai nghĩa.
- **Phân công trách nhiệm chọn chiến dịch (Câu 6).** Bản đầu đặt vế `registered_at` và `LIMIT 1` trong
  SQL, trong khi `data-model.md`/`research.md` đặt ở bộ đánh giá thuần. Spec nay nói rõ **mỗi việc ở
  đúng một nơi** để hai tài liệu không mâu thuẫn.
- **Cột `version` và `FR-005` (FR-010, FR-051, FR-005).** `version` được thêm vào danh sách cột bắt
  buộc và vào danh sách trường bổ sung của API đọc; FR-005 được thu hẹp thành "trong vận hành" để seed
  một lần không vi phạm.

---

## 3. Quyết định thiết kế — trả lời 11 câu hỏi bắt buộc

Mỗi câu dưới đây nêu **quyết định**, **điều kiện bất biến chính xác**, và **vì sao phương án khác bị
loại**.

### Câu 1 — Nguồn sự thật: CSDL là **nguồn duy nhất**

**Quyết định.** `subscription.campaigns` là nguồn sự thật duy nhất về chiến dịch.
`configs/config.yaml` và `configs/config.production.yaml` **bỏ hẳn** khối `campaigns`. Kiểu
`CampaignConfig`, hàm `validateCampaigns` và `validateCampaign` (`shared/config/validation.go`) bị
xoá. Sau cut-over, **không dòng mã nào đọc chiến dịch từ cấu hình**.

**Cách nạp không chạm CSDL vẫn được giữ.** Cấu hình vẫn được nạp tĩnh lúc khởi động và **không** mở
kết nối để đọc chiến dịch. Chiến dịch được đọc **lúc chạy**: mỗi khi một thông điệp đăng ký tới, mỗi
lần quét bù, mỗi lần gọi API tra cứu. Khối `campaign_sweep` (nhịp quét) **ở lại cấu hình** vì nó là
núm của cơ chế, không phải dữ liệu chiến dịch.

**Seed một lần, không phải đồng bộ hai chiều.** `launch-2026-10` được chèn bằng **một migration dữ
liệu một lần**, idempotent, với đúng giá trị đang chạy ở production (`plan_slug = premium-monthly`,
`quota = 50`, `reserve = 5`, `starts_at = 2026-10-02T00:00:00+07:00`, `ends_at = NULL`,
`closed_at = NULL`). Seed chạy trên mọi môi trường; ở dev nó bổ sung hàng còn thiếu, không đụng
`campaign_claims`.

**Vì sao không giữ cả hai nguồn.** Hai nguồn sẽ buộc phải có một luật hoà giải khi chúng lệch, và
chính luật hoà giải đó là chỗ bất biến `quota`/`reserve` bị phá: một chiến dịch đọc từ config với
quota 50, đọc từ DB với quota 60, thì "đã cấp" đếm theo bản nào cũng sai. Một nguồn loại bỏ cả lớp
lỗi này.

### Câu 2 — Bảy quy tắc chặn khởi động của 025 (SC-014) đi đâu

025 kiểm 7 quy tắc lúc nạp cấu hình. Khi chiến dịch không còn ở cấu hình, "từ chối khởi động" trở
thành **"từ chối ghi"**. Bảy quy tắc tách thành hai loại, và **quyết định lúc chạy của 025 cho
`plan_slug` được giữ nguyên**.

| # | Quy tắc 025 | Loại | Nơi thực thi mới |
|---|---|---|---|
| 1 | `code` không rỗng | hình dạng dữ liệu | API quản trị (thông báo theo trường) + `CHECK` trên bảng |
| 2 | `code` khớp `^[a-z0-9][a-z0-9-]{2,63}$` | hình dạng dữ liệu | API + `CHECK` |
| 3 | `code` không trùng | hình dạng dữ liệu | `PRIMARY KEY` (chặn tuyệt đối ở tầng dữ liệu) |
| 4 | `plan_slug` không rỗng | hình dạng dữ liệu | API + `CHECK` |
| 5 | `quota >= 1` | hình dạng dữ liệu | API + `CHECK` |
| 6 | `reserve >= 0` | hình dạng dữ liệu | API + `CHECK` |
| 7 | `starts_at` có múi giờ; `ends_at` sau `starts_at` | hình dạng dữ liệu | Kiểu `TIMESTAMPTZ` + `CHECK (ends_at IS NULL OR ends_at > starts_at)` |
| — | `plan_slug` trỏ tới gói tồn tại và đang active | **chỉ biết được khi chạy** | **Giữ nguyên quyết định của 025**: không chặn gì cả; mỗi lượt bị từ chối ghi `plan_unavailable` |

**Hệ quả cần nói rõ.** Không còn "chiến dịch sai làm sập khởi động", vì không còn nạp chiến dịch lúc
khởi động. Thay vào đó: **dữ liệu sai hình dạng không thể được ghi vào bảng** (API từ chối kèm thông
báo đúng trường; ràng buộc CSDL là chốt chặn cuối), còn **`plan_slug` sai vẫn chỉ làm chiến dịch đó
không cấp được gì cho ai**, đúng như FR-010 của 025 đã quyết. Một ký tự sai trong dữ liệu vận hành
không bao giờ làm sập production.

### Câu 3 — Bất biến `quota`/`reserve`: thực thi ở **tầng ứng dụng + trigger CSDL làm chốt chặn**

**Điều kiện bất biến chính xác.** `quota`, `reserve` và `plan_slug` của một chiến dịch **bất biến
ngay khi tồn tại ít nhất một dòng `subscription.campaign_claims` có `campaign_code` bằng mã chiến
dịch đó**. Điều kiện "đang chạy" được định nghĩa bằng **"đã cấp ít nhất một lần"**, không phải bằng
`now >= starts_at`.

**Vì sao định nghĩa theo số lượt cấp, không theo đồng hồ.** Một chiến dịch chỉ có thể có lượt cấp
khi nó đã mở, nên "có lượt cấp" bao hàm "đã qua `starts_at`". Ngược lại, một chiến dịch đã qua
`starts_at` mà chưa cấp cho ai thì `quota`/`reserve` **vẫn sửa được an toàn**, vì chưa có dòng lịch
sử nào bị diễn giải lại. Dùng đồng hồ sẽ khoá oan trường hợp lành. Dùng số lượt cấp vừa chặt hơn
vừa đơn giản hơn.

**Nơi thực thi — hai lớp, mỗi lớp một việc:**

1. **Tầng ứng dụng (dùng case quản trị)** kiểm tra bên trong transaction cập nhật: nếu có lượt cấp và
   yêu cầu đổi `quota`/`reserve`/`plan_slug`, trả lỗi nghiệp vụ tiếng Việt nêu rõ "chiến dịch đã cấp,
   hãy đóng và mở mã mới". Đây là lớp cho **thông báo đúng và thân thiện**.
2. **Trigger `BEFORE UPDATE` trên `subscription.campaigns`** nâng ngoại lệ khi một trong ba cột đổi
   mà tồn tại lượt cấp. Đây là lớp **không thể vòng qua bằng SQL trực tiếp**, đối xứng với triết lý
   của 025 ở chỉ mục một phần `ux_campaign_claims_watermark`: bất biến sống ở tầng dữ liệu.

**Vì sao không chỉ dùng `CHECK`.** `CHECK` là ràng buộc **theo từng hàng**, nó không nhìn thấy bảng
`campaign_claims`, nên không thể diễn tả "chỉ bất biến khi có lượt cấp".

**Vì sao không chỉ dùng tầng ứng dụng.** Yêu cầu nói rõ "không ai sửa bảng này bằng SQL trực tiếp";
kỷ luật tầng ứng dụng không ràng buộc được một câu SQL gõ tay. Trigger là lớp duy nhất ràng buộc được.

**Đổi `quota`/`reserve` khi chưa có lượt cấp là hợp lệ** và là đường sửa hạn mức linh hoạt cho chiến
dịch chưa cấp.

### Câu 4 — Đóng cưỡng bức: cột `closed_at`, trạng thái thứ sáu, một chiều

**Thêm gì.** Cột `closed_at TIMESTAMPTZ NULL` trên `subscription.campaigns`. `closed_at IS NOT NULL`
nghĩa là chiến dịch đã bị **đóng cưỡng bức** bởi người vận hành. Cột `closed_at` lưu **thời điểm**;
ai đóng và lý do đóng nằm ở nhật ký kiểm toán (câu 8).

**Ảnh hưởng tới máy trạng thái 5 trạng thái (`data-model.md` §5 của 025).** Trạng thái dẫn xuất
`status` nhận thêm giá trị thứ sáu: `closed`. Nó **đứng trước mọi trạng thái khác** — một chiến dịch
đã đóng thì ngủ bất kể `now` nằm ở đâu và bất kể ngân sách còn hay hết. Máy trạng thái của 025 vì
thế giữ nguyên 5 trạng thái cũ, cộng một trạng thái kết thúc do con người kích hoạt, không xoá hay
gộp trạng thái nào.

| `status` | Nguồn | Ghi chú |
|---|---|---|
| `not_started`, `running`, `compensation`, `exhausted`, `expired` | 025, không đổi | |
| `closed` | `closed_at IS NOT NULL` | Mới; ưu tiên cao nhất |

**Ảnh hưởng tới `status` mà API tra cứu trả về.** Đây là **thay đổi cộng thêm** vào một enum: hình
dạng response JSON **không đổi**, nhưng tập giá trị của `status` từ 5 thành 6. FE phải chịu được giá
trị lạ (đúng như tài liệu FE của 025 đã viết cho `reason`), và 026 cập nhật `frontend-guide.md` để
khai báo `closed`.

**Ảnh hưởng tới tập mã `reason` (12 mã).** **Không thêm mã nào.** Một tài khoản gặp chiến dịch đã
đóng được trả `reason = campaign_sleeping` — đúng nghĩa "chiến dịch đã kết thúc" mà 025 đã dùng cho
trường hợp quá `ends_at`. Vì vậy hợp đồng 12 mã đã lên production được giữ nguyên vẹn.

**Đóng rồi có hoàn tác được không?** **Không, qua API.** Đóng là một chiều. Lý do: mở lại một chiến
dịch đã đóng có thể cấp gói cho các tài khoản tạo trong khoảng thời gian đã đóng, và làm mất nghĩa
của nhật ký kiểm toán. Muốn chạy lại thì **mở một mã chiến dịch mới** — đúng nguyên tắc "end it and
open a new code" mà 025 đã dùng cho bất biến `quota`.

**Ai được đóng.** Chỉ quản trị viên, qua cùng lớp `RolesAuthorize(roleslug.Admin)` và cùng giới hạn
tần suất của các endpoint quản trị khác. Đóng bắt buộc kèm **lý do** (chuỗi), lý do này vào nhật ký
kiểm toán.

### Câu 5 — Chuyển đổi `launch-2026-10` không đổi nghĩa

**Migrate.** Hai migration Goose:
1. Tạo bảng `subscription.campaigns` và các ràng buộc (câu 3, trigger đóng ở câu 4/8).
2. Migration dữ liệu chèn `launch-2026-10` bằng đúng giá trị cấu hình production, idempotent
   (`INSERT ... ON CONFLICT (code) DO NOTHING`).

**Vì sao không đổi nghĩa.** `campaign_claims` và `campaign_sweep_runs` **không bị đụng tới**.
`campaign_code` trong hai bảng này là văn bản tự do, **không có khoá ngoại**, nên việc tạo bảng mới
không diễn giải lại dòng nào. Bất biến `UNIQUE (user_id)`, `grant_bucket`, `fills_main_quota` và mốc
ưu tiên giữ nguyên byte-for-byte. Số liệu API tra cứu trước và sau migration phải **giống hệt**.

**Vì sao không thêm khoá ngoại từ `campaign_claims` sang `campaigns`.** Hai cột `campaign_code` là
khoá lịch sử bằng văn bản, và bảng lượt cấp có thể chứa mã của một chiến dịch đã bị xoá khỏi sổ.
Thêm FK sẽ buộc phải tái dựng thuộc tính cho mọi mã lịch sử — việc migration không làm trung thực
được — và có thể chặn migration trên dữ liệu thật. Thay vào đó, tính toàn vẹn của sổ được giữ bằng
việc **không có đường xoá chiến dịch** (câu 8): chỉ tạo, sửa, đóng. Điều này cũng đúng với yêu cầu
"dùng lại khoá, đừng đặt lại khoá".

**Rollback.** Migration down xoá hàng seed rồi xoá bảng (theo thứ tự ngược). Vì lượt cấp không có
FK, rollback không mất dữ liệu lượt cấp. Khôi phục khối `campaigns` trong YAML và binary 025 là
đường lùi vận hành.

**Có giữ cấu hình làm bản sao trong giai đoạn chuyển tiếp không?** **Không giữ lâu dài.** Trong
đúng **một cửa sổ triển khai** — từ lúc migration chạy tới khi mọi bản sao chạy binary mới — khối
`campaigns` có thể còn trong YAML để làm **bản sao dự phòng cho rollback**, nhưng binary mới
**không đọc nó**. Khi mọi bản sao đã lên binary mới, khối này bị xoá trong cùng chu kỳ phát hành.
Nó không bao giờ là nguồn hành vi thứ hai.

**Thứ tự triển khai bắt buộc:** chạy migration (tạo bảng + seed) **trước**, rồi mới thay binary. Nếu
làm ngược, binary mới đọc bảng chưa tồn tại và không cấp gói cho ai cho tới khi migration chạy.

### Câu 6 — Cơ chế chọn chiến dịch (FR-011) khi nguồn là CSDL

**Truy vấn.** CSDL trả về **tập chiến dịch còn thức** bằng **một** câu, không có `LIMIT` và không có vế
`registered_at`:
```sql
SELECT * FROM subscription.campaigns
WHERE closed_at IS NULL
  AND starts_at <= :now
  AND (ends_at IS NULL OR :now < ends_at)
ORDER BY starts_at DESC, code DESC;
```
Hai vế đầu dùng `now` — đó là câu hỏi "chiến dịch còn **thức** không" (`isAwake`).

**Ai làm gì — phân công trách nhiệm rõ ràng, mỗi việc đúng một nơi:**

| Việc | Nơi thực hiện |
|---|---|
| Lọc "còn thức" theo `now` | SQL (vế `closed_at` + hai vế cửa sổ trên `now`) |
| Lọc "tài khoản có nằm trong cửa sổ của chiến dịch không" theo `registered_at` | **bộ đánh giá thuần** |
| Chọn `starts_at` lớn nhất, tie-break `code DESC` | **bộ đánh giá thuần** |

**Vì sao tách vậy.** Vế `registered_at` nằm ở bộ đánh giá chứ không ở SQL, là để giữ nguyên tính
thuần của điều kiện 1–7 của 025: nó nhận danh sách hàng và **không** tự truy vấn, nên vẫn kiểm thử
được không cần CSDL. Sự tách hai instant (`now` cho "còn thức", `registered_at` cho "nằm trong cửa
sổ") giữ nguyên từ 025 §6.1 và là điều kiện để quy tắc thuần túy. `code DESC` xuất hiện trong SQL
để thứ tự trả về là ổn định, và bộ đánh giá áp dụng lại cùng tiêu chí đó khi so sánh — hai nơi
cùng một quy tắc, không phải hai quy tắc.

**Tie-break khi hai chiến dịch cùng `starts_at`.** 025 không có tie-break tường minh vì cấu hình
không thể có hai mục trùng mã nhưng có thể trùng `starts_at`. 026 thêm **`code DESC`** làm tiêu chí
phụ để kết quả **tất định**: cùng một tập chiến dịch và cùng một `registered_at` luôn cho cùng một
lựa chọn.

**Chỉ mục.** `ix_campaigns_starts_at ON subscription.campaigns (starts_at DESC)` phục vụ
`ORDER BY`. Bảng chỉ có vài hàng, nên chỉ mục này là đủ; không cần chỉ mục phức hợp cho các vế cửa
sổ.

**Tính thuần túy còn giữ được không?** **Còn, theo thứ tự xử lý.** Lựa chọn là một hàm tất định của
(tập hàng trong sổ, `registered_at`). Thông điệp tới trước hay sau, chạy lại bao nhiêu lần, đều cho
cùng kết quả. Cái không bảo đảm được là **sự bất biến của sổ giữa hai lần đọc**: nếu người vận hành
sửa `starts_at` hoặc thêm chiến dịch giữa hai thông điệp, một tài khoản chưa có lượt cấp có thể được
xếp vào chiến dịch khác. Đây **không phải phụ thuộc thứ tự**; đây là thay đổi dữ liệu, và nó được
chấp nhận. Một khi tài khoản **đã có lượt cấp**, `UNIQUE (user_id)` khiến mọi thay đổi sổ về sau
không thể cấp lại cho nó.

### Câu 7 — Cache: **không cache** trong 026

**Quyết định.** Không thêm cache cho sổ đăng ký. Mỗi thông điệp đăng ký, mỗi lần quét bù, mỗi lần
tra cứu đọc thẳng CSDL.

**Vì sao chấp nhận được.** Tần suất đăng ký thấp (một thông điệp cho mỗi tài khoản mới), và truy
vấn chọn chiến dịch chạy trên bảng vài hàng với chỉ mục `(starts_at DESC)`. Chi phí là **một truy
vấn có chỉ mục trên mỗi lần đăng ký** — không đáng kể so với toàn bộ giao dịch tạo tài khoản và cấp
gói.

**Vì sao cố ý không cache.** Một cache sẽ tạo ra vấn đề **vô hiệu hoá xuyên bản sao**: người vận
hành vừa đóng một chiến dịch, nhưng bản sao B còn giữ bản cũ trong bộ nhớ sẽ **vẫn cấp gói** cho
chiến dịch đã đóng — đúng thứ mà mục tiêu "đóng sớm" sinh ra để chặn. Đổi lại một truy vấn rẻ lấy
được tính đúng tức thời, thì cache là lựa chọn tồi.

**Nếu sau này cần.** Hướng đi là TTL ngắn (vài chục giây) **cộng** một khoá phiên bản bị tăng khi có
thao tác quản trị, để thao tác đóng có hiệu lực ngay. Nhưng đó là tối ưu **khi đo được** nhu cầu,
không phải thiết kế của 026.

**Hệ quả cho kiến trúc mã.** Bộ đánh giá điều kiện (pure evaluator của 025) vẫn thuần: use case đọc
sổ **một lần** rồi truyền các hàng vào bộ đánh giá, nên điều kiện 1–7 vẫn kiểm thử được không cần
CSDL.

### Câu 8 — RBAC + nhật ký kiểm toán

**Ai được tạo/sửa/đóng.** Chỉ `Admin` (`roleslug.Admin`), qua cùng middleware `RolesAuthorize` và
cùng giới hạn tần suất quản trị mà API tra cứu 025 đang dùng. Không có vai trò mới.

**Ba endpoint ghi mới** (đặt trong nhóm `/api/v1/admin/campaigns` để tránh bẫy định tuyến Gin của
025 R11):
- `POST /api/v1/admin/campaigns` — tạo chiến dịch.
- `PATCH /api/v1/admin/campaigns/{code}` — sửa `quota`/`reserve`/`plan_slug`/`starts_at`/`ends_at`,
  tuân bất biến câu 3.
- `POST /api/v1/admin/campaigns/{code}/close` — đóng cưỡng bức kèm lý do.

**Lưu gì.** Mỗi thao tác ghi **một** dòng **outbox** kiểm toán vào nhật ký hiện có qua
`IAuditLogContract.Record`, gồm: `actorUserId` (quản trị viên), thời điểm, `action` ∈
{`campaign.create`, `campaign.update`, `campaign.close`} — theo **quy ước chấm** mà
`shared/domain/constants/adminaudit/action.go` đang dùng (`user.create`, `subscription.plan_create`),
`targetType = "campaign"`, `targetId = NULL` (vì bộ lọc `targetId` sẵn có nhận UUID, còn mã chiến dịch là
chuỗi), `payload` chứa `campaignCode` cùng **giá trị cũ → giá trị mới** từng trường đổi và lý do đóng khi
có, và `requestIp`.

**Dòng outbox nằm trong cùng giao dịch** với thao tác ghi chiến dịch: không có thao tác nào thành công mà
không có dấu vết. **Dòng trong bảng `admin_audit_logs` là hệ quả bất đồng bộ** của dòng outbox đó, nên
xuất hiện sau khi bộ phát hàng đỗi và consumer kiểm toán chạy xong — chi tiết và hệ quả đo được ở
FR-060, SC-006 và SC-011. Nhật ký được đọc lại qua endpoint riêng ở FR-063.

**Idempotent không?**
- **Đóng là idempotent**: gọi lại trên chiến dịch đã đóng là no-op thành công, không ghi thêm bản ghi
  kiểm toán trùng.
- **Tạo không idempotent theo thiết kế**: mã đã tồn tại bị từ chối `409`, không âm thầm bỏ qua.
- **Sửa naturally idempotent**: cùng một giá trị đích cho cùng kết quả.
- Không cần cơ chế idempotency-key riêng.

**Không sửa bằng SQL trực tiếp.** Không có endpoint xoá. Bảng chỉ được ghi bởi dùng case quản trị.
Trigger câu 3 chặn đường SQL vòng qua bất biến. Việc gán quyền CSDL để chặn ghi tay hoàn toàn nằm ở
tầng vận hành, ngoài phạm vi mã.

### Câu 9 — Worker quét bù khi sổ ở CSDL

**Quét theo lịch ra sao.** Worker quét bù **đọc sổ mới mỗi lần chạy** (không giữ danh sách đã nạp),
lọc `closed_at IS NULL`, rồi với mỗi chiến dịch còn thức quét các tài khoản tạo trong cửa sổ
`[starts_at, ends_at)` chưa có lượt cấp. Nhịp quét (`campaign_sweep.interval_seconds`,
`run_on_start`) **ở lại cấu hình** — đây là núm của cơ chế, không phải dữ liệu chiến dịch, và giữ nó
ở cấu hình giữ được tính không-chạm-CSDL khi nạp.

**Chiến dịch vừa được tạo giữa chừng có bị bỏ sót không?** Không, quá một chu kỳ quét. Vì danh sách
được đọc lại mỗi lần chạy, lần quét kế tiếp đã thấy chiến dịch mới; các tài khoản tạo **giữa lúc tạo
chiến dịch và lần quét kế tiếp** vẫn nằm trong `[starts_at, now)` nên được quét ở lần đó.
`run_on_start = true` đóng khoảng hở khi khởi động lại. Đường thông điệp vẫn xử lý ngay các tài khoản
tới sau khi tạo. Vì vậy một chiến dịch mới **không bị bỏ sót quá một chu kỳ quét**.

**Bảng sức khoẻ.** `campaign_sweep_runs` vẫn ghi một hàng cho mỗi chiến dịch sau mỗi lần quét, bằng
upsert theo `campaign_code`. Chiến dịch mới chưa quét có `lastSweepAt = null`, `degraded = false`.
Chiến dịch đã đóng thôi được quét; hàng sức khoẻ và sổ lượt cấp của nó **không bị xoá**.

### Câu 10 — Biến môi trường: **giữ nguyên quyết định không có env override**

**Quyết định.** Không mở env override cho dữ liệu chiến dịch. Sau 026, khối `campaigns` không còn
trong cấu hình nữa, nên câu hỏi "override danh sách bằng env" trở nên vô nghĩa: **chiến dịch là dữ
liệu vận hành** quản lý qua API có kiểm toán, và **CSDL chính là biên môi trường** — mỗi môi trường
có CSDL riêng, nên khác biệt giữa các môi trường tự nhiên không cần biến môi trường nào.

**Không thêm biến môi trường mới.** Nhịp quét `campaign_sweep.interval_seconds` giữ nguyên là giá
trị YAML như 025 (không có env riêng), và `EMAIL_ENABLED` sẵn có vẫn là công tắc dùng khi kiểm
chứng. Mọi comment trong tệp `.env*` (nếu có sửa) viết tiếng Anh.

**Không phá hiến pháp §VIII.** §VIII cấm viết cứng giá trị phụ thuộc môi trường. Ở đây **không có**
giá trị phụ thuộc môi trường nào bị viết cứng: giá trị chiến dịch nằm trong CSDL của chính môi trường
đó, và việc chuyển môi trường vẫn chỉ cần cấu hình kết nối CSDL + DNS, không cần đổi mã.

### Câu 11 — Hệ quả với `frontend-guide.md` của 025

026 **có** thay đổi bề mặt client-facing, nên phải ship `specs/026-campaign-registry-db/frontend-guide.md`
(hiến phán Governance). Nội dung nó phải gồm:

**Trường bổ sung cho API đọc hiện có (cộng thêm, không phá hình dạng):**
- `closedAt` (thời điểm đóng, hoặc `null`) trên danh sách và chi tiết chiến dịch.
- `budgetLocked` (boolean, `true` khi đã có lượt cấp) để FE **khoá ô sửa** `quota`/`reserve`/`plan_slug`
  mà không phải tự đoán.
- Giá trị mới `status = "closed"` (giá trị thứ sáu).

**Bốn endpoint đọc giữ nguyên** hình dạng response và **đúng 12 mã `reason`**.

**Ba endpoint ghi mới** với hình dạng request/response, mã lỗi `400` (sai hình dạng), `404` (không
tìm thấy mã), `409` (mã đã tồn tại khi tạo), `412` (phiên bản lệch), `422` (vi phạm bất biến ngân
sách), `403`, `429`. Response đọc và ghi MUST trả `version`; FE MUST gửi lại phiên bản đó khi sửa/đóng
và MUST xử lý `412` bằng thông báo "dữ liệu đã thay đổi, tải lại".

**Một endpoint đọc mới** `GET /api/v1/admin/campaigns/{code}/audit` trả nhật ký của một chiến dịch
(phân trang), theo FR-063.

**Màn hình admin cần có cho vòng đời chiến dịch:**
1. **Danh sách chiến dịch** — trạng thái, ngân sách, nút hành động.
2. **Chi tiết chiến dịch** — toàn bộ trường + lịch sử thay đổi.
3. **Tạo chiến dịch** — form `code`, `plan_slug`, `quota`, `reserve`, `starts_at`, `ends_at`.
4. **Sửa chiến dịch** — ô `quota`/`reserve`/`plan_slug` **bị vô hiệu hoá kèm chú thích "chiến dịch đã
   cấp — hãy đóng và mở mã mới"** khi `budgetLocked = true`; thời gian sửa được.
5. **Đóng chiến dịch** — hộp xác nhận + ô lý do.
6. **Nhật ký kiểm toán chiến dịch** — ai / lúc nào / đổi gì.

FE phải chịu được giá trị `status` lạ (hiển thị mặc định thay vì so khớp chuỗi), và không được giả
định danh sách mã chiến dịch là tĩnh.

---

## 4. Sự thay thế so với 025

Ghi lại để 025 vẫn đúng với chính nó và không ai phải sửa `specs/025-`. Mỗi dòng nêu yêu cầu 025 trở
thành sai **theo cách nào** và 026 thay bằng gì.

| Yêu cầu 025 | Vấn đề khi 026 ra đời | 026 thay bằng |
|---|---|---|
| FR-011: "nhiều chiến dịch cùng tồn tại **trong cấu hình**" | Nguồn đã đổi sang CSDL | FR-020/FR-021: nhiều chiến dịch cùng tồn tại **trong `subscription.campaigns`** |
| FR-010 + SC-014: 7 quy tắc từ chối **khởi động** | Không còn nạp chiến dịch lúc khởi động | FR-010–FR-016: từ chối **ghi** + ràng buộc CSDL; `plan_slug` vẫn kiểm lúc chạy |
| FR-013: chiến dịch chỉ **tự ngủ** | Không có đường đóng cưỡng bức | FR-030–FR-033: thêm đóng cưỡng bức qua `closed_at` |
| FR-074: **không** cho đổi hạn mức/thời gian từ giao diện | Nay cần giao diện quản lý | FR-052: API quản trị cho phép sửa (có bất biến), mở khối US1/US4 |
| Key Entities: "Chiến dịch … Không lưu thành một bảng riêng — thuộc tính nằm trong cấu hình" | Sổ nằm ở CSDL | Key Entities của 026: **Chiến dịch** là bảng `subscription.campaigns` |
| `data-model.md` §4 "Cấu hình chiến dịch — hình dạng dữ liệu … Không phải bảng" | Như trên | Cột bảng (mục 5) |
| `research.md` R10: danh sách chiến dịch chỉ khai trong YAML | Khối bị bỏ | `campaign_sweep` vẫn ở YAML; `campaigns` không còn |
| `quickstart.md` §1.2: khai chiến dịch trong **cả hai** tệp cấu hình | Không còn khối để khai | Seed migration + API quản trị |
| `data-model.md` §1 "Bảng bị thay đổi lược đồ: đúng **3** bảng" | 026 thêm bảng thứ tư | 026: **4** bảng (`campaigns` mới; 3 bảng cũ không đổi) |

**Không bị thay thế** (giữ nguyên): 4 endpoint đọc của FR-065–FR-072, 12 mã `reason`, 6 giá trị
`eligibility`, máy trạng thái 5 trạng thái cũ, bảng `campaign_claims`, bảng `campaign_sweep_runs`,
bất biến `UNIQUE (user_id)`, `grant_bucket`, `fills_main_quota`, và các quyết định FR-092–FR-098 về
sự kiện và dịch vụ AI.

---

## 5. Mô hình dữ liệu

### 5.1 Bảng mới `subscription.campaigns`

| Cột | Kiểu | Ràng buộc |
|---|---|---|
| `code` | `VARCHAR(64)` | `PRIMARY KEY` (cho `NOT NULL` + duy nhất) và `CHECK` khớp `^[a-z0-9][a-z0-9-]{2,63}$` |
| `plan_slug` | `VARCHAR(100)` | `NOT NULL`, `CHECK` không rỗng; tồn tại/active kiểm lúc chạy |
| `quota` | `INTEGER` | `NOT NULL`, `CHECK (quota >= 1)` |
| `reserve` | `INTEGER` | `NOT NULL DEFAULT 0`, `CHECK (reserve >= 0)` |
| `starts_at` | `TIMESTAMPTZ` | `NOT NULL` |
| `ends_at` | `TIMESTAMPTZ` | `NULL`; `CHECK (ends_at IS NULL OR ends_at > starts_at)` |
| `closed_at` | `TIMESTAMPTZ` | `NULL`; khác `NULL` = đóng cưỡng bức |
| `version` | `INTEGER` | `NOT NULL DEFAULT 1`; tăng mỗi lần sửa/đóng, dùng làm If-Match |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` |

Chỉ mục: `ix_campaigns_starts_at (starts_at DESC)`. Bất biến ở câu 3 được cài bằng trigger
`BEFORE UPDATE`.

### 5.2 Các bảng đọc/không đổi

- `subscription.campaign_claims` — **không đổi lược đồ**. Tiếp tục là sổ lượt cấp, cơ chế chống trùng
  (`UNIQUE (user_id)`), và nơi lưu mốc ưu tiên (`fills_main_quota`), với `grant_bucket`.
- `subscription.campaign_sweep_runs` — **không đổi lược đồ**. Trạng thái sức khoẻ vận hành.
- `identity.users` — chỉ đọc `created_at`, `created_by_admin` qua hợp đồng.
- `subscription.subscription_plans`, `subscription.user_subscriptions` — chỉ đọc.
- `aiexecutor.*` — **không đụng tới**.

---

## 6. User Scenarios & Testing *(bắt buộc)*

### User Story 1 - Người vận hành mở chiến dịch mới ngay lập tức (Priority: P1)

Tôi là người vận hành. Tôi cần mở một chiến dịch tặng gói lúc 00:00 ngày mai, hoặc bất kỳ lúc nào
phát sinh. Tôi mở trang quản trị, điền mã chiến dịch, gói cước, hạn mức, thời gian, bấm lưu. Chiến
dịch **có hiệu lực ngay**, không có commit, không tag, không chờ CI, không deploy lại. Nếu tôi đặt
`starts_at` trong quá khứ, các tài khoản đăng ký sau đó được xét vào chiến dịch ngay.

**Why this priority**: Đây là mục tiêu số 1 và là lý do feature tồn tại. Không có nó thì 026 không
giải quyết được vấn đề deploy.

**Independent Test**: Gọi API tạo chiến dịch, rồi đăng ký một tài khoản mới trong cửa sổ chiến dịch
và kiểm tra tài khoản nhận đúng gói. Không sửa tệp mã nguồn nào, không khởi động lại dịch vụ.

**Acceptance Scenarios**:

1. **Given** chưa có chiến dịch `x`, **When** quản trị viên gọi API tạo `x` với `starts_at` trong quá
   khứ, **Then** API trả thành công, chiến dịch xuất hiện trong danh sách với `status` đúng, và một
   tài khoản đăng ký sau đó được cấp gói.
2. **Given** API tạo thành công, **When** người vận hành kiểm tra, **Then** **không** tệp mã nguồn
   nào bị sửa và **không** binary nào được triển khai lại.
3. **Given** quản trị viên tạo chiến dịch với `code` đã tồn tại, **When** gọi API, **Then** bị từ chối
   `409`, và không có chiến dịch thứ hai nào được tạo.
4. **Given** tài khoản người dùng thường, **When** gọi API tạo, **Then** bị từ chối `403`.

---

### User Story 2 - Người vận hành đóng chiến dịch sớm (Priority: P1)

Tôi là người vận hành. Chiến dịch đang chạy, nhưng tôi cần dừng nó trước `ends_at` — ví dụ hết ngân
sách khuyến mãi, hoặc phát hiện sai sót. Tôi bấm đóng, nhập lý do. Chiến dịch **ngừng cấp gói ngay**
cho mọi tài khoản mới, kể cả tài khoản tạo trước thời điểm đóng nhưng xử lý sau. Các tài khoản **đã
được cấp vẫn giữ nguyên** gói và thời hạn.

**Why this priority**: Là mục tiêu số 3; đồng thời là lá chắn vận hành bắt buộc khi một chiến dịch
chạy sai.

**Independent Test**: Đóng một chiến dịch đang chạy qua API, rồi đăng ký tài khoản mới và kiểm tra nó
**không** nhận gói và `reason` là `campaign_sleeping`. Không cần deploy.

**Acceptance Scenarios**:

1. **Given** chiến dịch đang chạy còn ngân sách, **When** quản trị viên gọi API đóng kèm lý do,
   **Then** `closed_at` được ghi, `status` trả về `closed`, và không có lượt cấp mới nào cho tài khoản
   tạo sau đó.
2. **Given** chiến dịch đã đóng, **When** một tài khoản đủ điều kiện khác đăng ký, **Then** tài khoản
   đó **không** được cấp gói.
3. **Given** các tài khoản đã được cấp trước khi đóng, **When** chiến dịch đóng, **Then** gói và thời
   hạn của họ **không đổi**.
4. **Given** chiến dịch đã đóng, **When** gọi API đóng lần nữa, **Then** thành công idempotent, không
   ghi thêm bản ghi kiểm toán trùng.

---

### User Story 3 - Bất biến ngân sách của chiến dịch đã cấp (Priority: P1)

Tôi là người vận hành. Tôi muốn sửa `quota`/`reserve` của một chiến dịch. Hệ thống cho tôi sửa nếu
chiến dịch **chưa cấp cho ai**, và **từ chối** nếu đã có lượt cấp — vì mọi lượt cấp đã ghi lại nó
lấy suất từ phần nào, nên đổi ngân sách sẽ làm sai số liệu và lệch mốc ưu tiên. Khi bị từ chối, tôi
được hướng dẫn đóng chiến dịch cũ và mở mã mới. Nếu tôi cố sửa thẳng bằng SQL, hệ thống cũng chặn.

**Why this priority**: Là mục tiêu số 2 và là bất biến dễ mất nhất khi trạng thái ra khỏi config.

**Independent Test**: Tạo một chiến dịch chưa cấp, sửa `quota` thành công. Cấp cho một tài khoản, sửa
`quota` lần nữa và kiểm tra bị từ chối; thử `UPDATE` thẳng trong CSDL và kiểm tra trigger chặn.

**Acceptance Scenarios**:

1. **Given** chiến dịch chưa có lượt cấp, **When** sửa `quota`/`reserve`/`plan_slug`, **Then** thành
   công và `budgetLocked` vẫn `false`.
2. **Given** chiến dịch đã có ít nhất một lượt cấp, **When** sửa `quota`/`reserve`/`plan_slug`,
   **Then** bị từ chối `422` kèm thông báo tiếng Việt hướng dẫn đóng và mở mã mới.
3. **Given** chiến dịch đã có lượt cấp, **When** ai đó chạy `UPDATE` thẳng trong CSDL, **Then**
   trigger nâng ngoại lệ và giá trị không đổi.
4. **Given** sửa thành công, **When** gọi lại API sửa với bất kỳ giá trị không đổi, **Then** kết quả
   không thay đổi (idempotent), không ghi thêm bản ghi kiểm toán trùng.

---

### User Story 4 - Sửa thời gian linh hoạt (Priority: P2)

Tôi là người vận hành. Tôi cần dời `starts_at` sớm hơn để chiến dịch bắt đầu ngay, hoặc dời
`ends_at` xa hơn để kéo dài, hoặc rút `ends_at` lại. Tôi sửa được các mốc thời gian bất kể chiến dịch
đã cấp hay chưa, miễn `ends_at` vẫn sau `starts_at`. Việc dời thời gian không làm hỏng các lượt cấp
đã ghi, vì mỗi lượt cấp đã tự chụp `registered_at` và phần ngân sách của nó.

**Why this priority**: Là phần "thời gian linh hoạt" của mục tiêu số 2; tách khỏi US3 vì luật khác.

**Independent Test**: Sửa `ends_at` của một chiến dịch đang chạy sang tương lai rồi kiểm tra nó vẫn
cấp; rồi sửa `ends_at` về quá khứ và kiểm tra nó ngừng cấp.

**Acceptance Scenarios**:

1. **Given** `ends_at` đang ở quá khứ, **When** sửa `ends_at` sang tương lai, **Then** chiến dịch cấp
   lại bình thường cho tài khoản tạo trong cửa sổ mới.
2. **Given** chiến dịch đang chạy, **When** sửa `ends_at` về quá khứ, **Then** `status` chuyển
   `expired` và không cấp thêm cho tài khoản tạo sau mốc mới.
3. **Given** tôi đặt `ends_at` không sau `starts_at`, **When** lưu, **Then** bị từ chối `422` (đúng hình
   dạng nhưng sai nghĩa — xem bảng ánh xạ ở FR-054).

---

### User Story 5 - Nhật ký kiểm toán (Priority: P2)

Tôi là quản trị viên. Sau khi có người mở/sửa/đóng chiến dịch, tôi cần biết **ai làm, lúc nào, đổi
gì**. Tôi mở lịch sử chiến dịch và thấy từng thao tác: tạo, sửa trường nào từ giá trị nào sang giá
trị nào, đóng với lý do gì.

**Why this priority**: Là mục tiêu số 4; là cơ sở để tin được các thao tác không cần deploy.

**Independent Test**: Thực hiện một chuỗi tạo/sửa/đóng, đợi bộ phát hàng đỗi và consumer kiểm toán
chạy xong (bản ghi trong `admin_audit_logs` là hệ quả bất đồng bộ của dòng outbox — FR-060), rồi đọc
nhật ký kiểm toán và đối chiếu từng bản ghi.

**Acceptance Scenarios**:

1. **Given** một thao tác tạo/sửa/đóng thành công, **When** đọc nhật ký, **Then** có đúng một bản ghi
   với `actorUserId`, thời điểm, `action`, `targetId`, và giá trị cũ/mới.
2. **Given** thao tác ghi thất bại, **Then** không có bản ghi kiểm toán nào được ghi cho nó.
3. **Given** nhật ký đã có dữ liệu, **Then** việc đọc nhật ký **không** thay đổi ngân sách hay trạng
   thái chiến dịch.

---

### User Story 6 - Chuyển `launch-2026-10` sang sổ mà không đổi nghĩa (Priority: P1)

Tôi là người vận hành. `launch-2026-10` đang chạy với khoảng 55 lượt cấp thật. Tôi chạy migration
để đưa nó vào sổ. Sau migration, con số đã cấp, mốc ưu tiên, và danh sách tài khoản đã cấp phải
**giống hệt** trước đó, và chiến dịch tiếp tục cấp cho tài khoản mới như cũ.

**Why this priority**: Là điều kiện an toàn để 026 có thể lên production mà không phá dữ liệu thật.

**Independent Test**: So sánh output của 4 API tra cứu trước và sau migration; chạy migration hai lần
để chứng minh idempotent; rollback rồi chạy lại.

**Acceptance Scenarios**:

1. **Given** production có ~55 lượt cấp cho `launch-2026-10`, **When** migration chạy, **Then** hàng
   chiến dịch được seed đúng sáu giá trị, và `campaign_claims` không đổi một dòng nào.
2. **Given** migration đã chạy, **When** chạy lại, **Then** không tạo hàng trùng và không lỗi.
3. **Given** đã migrate, **When** tra cứu, **Then** số đã cấp / còn lại / `watermarkAt` / `status`
   giống trước migration.
4. **Given** cần lùi, **When** chạy goose `down` **hai lần** (một lần cho migration seed, một lần cho
   migration lược đồ — `down` chỉ lùi **một** phiên bản mỗi lần), **Then** bảng `campaigns` biến mất
   và `campaign_claims` vẫn nguyên số dòng.

---

### Edge Cases

**Sổ và thao tác quản trị**
- Đóng một chiến dịch **chưa tới** `starts_at` ⇒ đóng được; `status` là `closed`, không phải
  `not_started`.
- Đóng một chiến dịch đã `exhausted`/`expired` ⇒ đóng được, idempotent nếu đã đóng.
- Sửa `quota` của chiến dịch **chưa cấp** ⇒ được. Sửa khi **đã cấp** ⇒ bị từ chối.
- Hai chiến dịch cùng `starts_at` cùng đủ điều kiện ⇒ tie-break `code DESC`, kết quả tất định.
- Tạo chiến dịch với `starts_at` trong quá khứ ⇒ hợp lệ và cấp ngay.
- Tạo/sửa để `ends_at <= starts_at` ⇒ bị từ chối.
- `plan_slug` trỏ tới gói không tồn tại ⇒ hàng vẫn được phép tồn tại; mỗi lượt bị từ chối ghi
  `plan_unavailable`.
- Sửa đúng giá trị hiện có ⇒ no-op idempotent.
- Cố sửa thẳng `quota` bằng SQL khi đã có lượt cấp ⇒ trigger chặn.
- Đóng chiến dịch đúng lúc một thông điệp đang xử lý (đã qua điều kiện 1–7) ⇒ lượt đó bị chặn tại
  transaction nhờ đọc lại hàng chiến dịch dưới khoá chia sẻ (FR-045); không lượt cấp nào vượt qua lệnh
  đóng.
- Sửa ngân sách đúng lúc một lượt cấp đang xử lý ⇒ hai thao tác nối tiếp nhau; lượt cấp hoặc thấy
  ngân sách cũ (khi cấp trước khi sửa) hoặc thấy ngân sách mới, không có kết quả nửa vời.
- Đổi `code` của một chiến dịch ⇒ bị từ chối; mã bất biến (FR-024).
- Hai quản trị viên cùng sửa một chiến dịch ⇒ người thứ hai nhận `412`, không ghi đè ngầm; nhật ký
  kiểm toán không có bản ghi của thao tác bị từ chối.
- Sửa trong lúc người khác đóng cùng chiến dịch ⇒ một bên thắng, bên kia nhận `412`; riêng thao tác
  đóng trên chiến dịch **đã đóng** vẫn idempotent thành công bất kể `version`.

**Nguồn sự thật**
- Sau cut-over, khối `campaigns` không còn trong YAML; `campaign_sweep` còn.
- Khởi động khi bảng `campaigns` **rỗng** ⇒ hệ thống chạy bình thường, không cấp cho ai, không cảnh
  báo giả (tương đương FR-004 của 025).
- Ai đó tạo chiến dịch bằng SQL trực tiếp (không qua API) với hình dạng hợp lệ ⇒ hệ thống vẫn đọc
  được, nhưng đây là đường không được hỗ trợ; thao tác qua API mới có kiểm toán.

**Quét bù**
- Chiến dịch được tạo sau lần quét trước ⇒ lần quét kế tiếp nhận nó; tài khoản tạo trong khoảng giữa
  vẫn được quét.
- Chiến dịch bị đóng giữa hai lần quét ⇒ thôi được quét; sổ và hàng sức khoẻ còn nguyên.
- Nhiều bản sao hệ thống quét song song ⇒ cùng bộ luật hạn mức của 025 vẫn giữ trần; không đổi.

**Tương thích hợp đồng đã lên production**
- Response của 4 API đọc chỉ **thêm** trường; không đổi kiểu của trường cũ.
- Tập `reason` giữ đúng 12 mã.
- `status` thêm `closed`; client cũ hiển thị mặc định cho giá trị lạ, không lỗi.

**Bảo mật**
- Người dùng thường gọi bất kỳ endpoint ghi nào ⇒ `403`, không lộ dữ liệu.
- Vượt giới hạn tần suất ⇒ `429`.

---

## 7. Requirements *(bắt buộc)*

### Nguồn sự thật và vòng đời dữ liệu

- **FR-001**: Việc chiến dịch nào tồn tại MUST được lưu trong bảng `subscription.campaigns`; bảng này
  là **nguồn sự thật duy nhất**.
- **FR-002**: Khối `campaigns` MUST bị bỏ khỏi `configs/config.yaml` và `configs/config.production.yaml`;
  kiểu `CampaignConfig` và các hàm kiểm tra chiến dịch lúc nạp cấu hình MUST bị xoá.
- **FR-003**: Nạp cấu hình MUST NOT chạm cơ sở dữ liệu; chiến dịch MUST được đọc lúc chạy (thông điệp,
  quét bù, tra cứu).
- **FR-004**: Khối `campaign_sweep` (nhịp quét, `run_on_start`) MUST vẫn nằm trong cấu hình.
- **FR-005**: **Trong vận hành**, chỉ API quản trị được ghi vào bảng; MUST NOT tồn tại đường xoá chiến
  dịch. *(Ngoại lệ một lần và có kiểm soát: migration seed chèn `launch-2026-10` bằng `INSERT` trực
  tiếp, đúng một lần, idempotent — xem FR-090.)*

### Hình dạng dữ liệu và kiểm tra

- **FR-010**: Bảng `subscription.campaigns` MUST có các cột `code`, `plan_slug`, `quota`, `reserve`,
  `starts_at`, `ends_at`, `closed_at`, `version`, `created_at`, `updated_at`.
- **FR-011**: `code` MUST NOT rỗng, MUST khớp `^[a-z0-9][a-z0-9-]{2,63}$`, và MUST là duy nhất.
- **FR-012**: `quota` MUST `>= 1`; `reserve` MUST `>= 0`.
- **FR-013**: `starts_at` MUST được lưu dưới dạng thời điểm có múi giờ; `ends_at` MUST là `NULL` hoặc
  **sau** `starts_at`.
- **FR-014**: `plan_slug` MUST NOT rỗng khi ghi. Việc nó trỏ tới gói **tồn tại và active** MUST được
  kiểm **lúc chạy**, không lúc ghi và không lúc khởi động; sai thì bỏ qua lượt với `plan_unavailable`
  (giữ nguyên FR-010 của 025).
- **FR-015**: Mọi phép so sánh thời gian MUST dùng múi giờ cấu hình, MUST NOT dùng giờ máy chủ.

### Bất biến ngân sách

- **FR-020**: `quota`, `reserve` và `plan_slug` của một chiến dịch MUST bất biến ngay khi tồn tại ít
  nhất một dòng `campaign_claims` với `campaign_code` tương ứng.
- **FR-021**: Việc bất biến MUST được thực thi ở **tầng ứng dụng** (thông báo thân thiện, `422`) và
  ở **tầng dữ liệu** bằng trigger `BEFORE UPDATE`, để SQL trực tiếp không vòng qua được.
- **FR-022**: Khi chưa có lượt cấp, `quota`/`reserve`/`plan_slug` MUST sửa được.
- **FR-023**: `starts_at` và `ends_at` MUST sửa được bất kể đã có lượt cấp hay chưa, miễn FR-013 đúng.
- **FR-024**: `code` MUST bất biến sau khi tạo. `campaign_claims` và `campaign_sweep_runs` tham chiếu
  nó bằng văn bản, nên đổi `code` sẽ làm mồ côi sổ lịch sử; MUST NOT có thao tác đổi mã.

### Đóng cưỡng bức

- **FR-030**: Hệ thống MUST có thao tác đóng cưỡng bức ghi `closed_at`, chỉ dành cho quản trị viên và
  bắt buộc kèm lý do.
- **FR-031**: Một chiến dịch có `closed_at` MUST NOT cấp gói cho bất kỳ tài khoản nào, bất kể cửa sổ
  thời gian và ngân sách.
- **FR-032**: Trạng thái dẫn xuất `status` MUST nhận thêm giá trị `closed`, với độ ưu tiên cao nhất.
- **FR-033**: Đóng MUST NOT thay đổi gói của các tài khoản đã được cấp.
- **FR-034**: Đóng MUST idempotent. Đóng MUST one-way qua API; MUST NOT có thao tác mở lại.

### Chọn chiến dịch và cấp gói

- **FR-040**: Việc chọn chiến dịch MUST là một truy vấn trên `subscription.campaigns` chọn chiến dịch
  có `starts_at` lớn nhất trong số các ứng viên thỏa: chưa đóng, còn thức theo `now`, và cửa sổ chứa
  `registered_at` của tài khoản; tie-break bằng `code DESC` để tất định.
- **FR-041**: Kết quả chọn MUST là hàm tất định của (tập hàng trong sổ, `registered_at`), MUST NOT phụ
  thuộc thứ tự xử lý thông điệp.
- **FR-042**: Các điều kiện đủ điều kiện và bộ luật hạn mức của 025 (`data-model.md` §6) MUST được giữ
  nguyên; chỉ nguồn danh sách chiến dịch đổi từ cấu hình sang CSDL.
- **FR-043**: Chống cấp trùng, `grant_bucket`, `fills_main_quota`, mốc ưu tiên và trần `quota + reserve`
  MUST giữ nguyên hành vi của 025.
- **FR-044**: Hệ thống MUST NOT thêm loại sự kiện đổi gói mới; `services/ai` MUST NOT bị sửa; MUST NOT
  lồng `uow.Execute`.
- **FR-045**: Trong **cùng transaction** cấp gói (đang giữ khoá tư vấn), hàng chiến dịch của chiến dịch
  được chọn MUST được **đọc lại dưới khoá chia sẻ** trước khi ghi lượt cấp. Nếu `closed_at` khác `NULL`
  hoặc ngân sách đã thay đổi khiến lượt không còn hợp lệ, lượt đó MUST bị bỏ qua và không ghi. Điều
  này làm cho bảo đảm "chiến dịch đã đóng không cấp gói" đúng cả khi việc đóng/sửa xảy ra song song với
  một lượt cấp đang xử lý.

### Hợp đồng API

- **FR-050**: Bốn endpoint đọc hiện có MUST giữ nguyên hình dạng response và **đúng 12 mã `reason`**.
- **FR-051**: Response đọc MUST được bổ sung (cộng thêm) ba trường `closedAt`, `budgetLocked` và
  `version`; `status` MUST có thêm giá trị `closed`.
- **FR-052**: Hệ thống MUST có ba endpoint ghi dưới `/api/v1/admin/campaigns`: tạo, sửa, đóng. Endpoint
  sửa và đóng MUST nhận **phiên bản mong đợi** của chiến dịch.
- **FR-053**: Các endpoint ghi MUST yêu cầu vai trò `Admin` và dùng cùng giới hạn tần suất quản trị
  hiện có.
- **FR-054**: Mỗi lỗi MUST có đúng một mã trong bảng ánh xạ dưới đây. **Đây là nguồn duy nhất** của bảng
  mã lỗi; hợp đồng ở `contracts/campaign-registry-api.md` chỉ **hiện thực** bảng này và phải khớp từng
  dòng:

  | Mã | Nghĩa | Ví dụ |
  |---|---|---|
  | `400` | Yêu cầu **hỏng hình dạng**: thiếu trường bắt buộc, `code` sai định dạng, `starts_at`/`endsAt` không phải RFC3339 có múi giờ | `code = "Launch 2026"`, `startsAt = "2026-10-02 00:00:00"` |
  | `422` | Yêu cầu **đúng hình dạng nhưng sai nghĩa**: `quota < 1`, `reserve < 0`, `endsAt` không sau `startsAt`, `planSlug` rỗng, đổi `quota`/`reserve`/`planSlug` khi `budgetLocked`, `reason` đóng rỗng | `quota = 0`, sửa `quota` của chiến dịch đã cấp |
  | `404` | `:code` không tồn tại | tra mã lạ |
  | `409` | `code` đã tồn tại khi **tạo** | tạo trùng mã |
  | `412` | Phiên bản lệch | `PATCH` với `version` cũ |
  | `403` | Không phải `Admin` | tài khoản người dùng thường |
  | `429` | Vượt giới hạn tần suất | vòng dò |

  `409` chỉ dành cho trùng mã lúc tạo; `422` **không** dùng cho lỗi cú pháp. Không mã nào ngoài bảng này
  được trả về từ ba endpoint ghi.
- **FR-055**: Endpoint sửa và đóng MUST dùng **kiểm tra phiên bản lạc hậu (optimistic concurrency)**:
  response đọc trả `version`; yêu cầu ghi mang phiên bản đó; nếu phiên bản hiện tại khác thì MUST từ
  chối `412` và MUST NOT ghi. `version` tăng mỗi lần sửa/đóng **có thay đổi ít nhất một trường** thành
  công. Đóng một chiến dịch **đã đóng** MUST vẫn idempotent thành công (kiểm tra idempotent chạy trước
  kiểm tra phiên bản). **Ngoại lệ no-op (US3 kịch bản 4):** một lần `PATCH` mà **không trường nào đổi**
  là no-op — MUST trả `200` với trạng thái hiện tại, MUST NOT tăng `version`, MUST NOT ghi kiểm toán.

### Nhật ký kiểm toán

- **FR-060**: Mỗi thao tác tạo/sửa/đóng **có thay đổi** MUST ghi **một** dòng **outbox** kiểm toán trong
  **cùng giao dịch** với thao tác ghi chiến dịch, gồm người thực hiện, thời điểm, hành động, mã chiến
  dịch, giá trị cũ→mới, và lý do đóng khi có. *(Hai trường hợp **không** ghi: lần `PATCH` no-op ở FR-055,
  và lần đóng trên chiến dịch đã đóng.)* Bản ghi trong bảng `admin_audit_logs` là **hệ quả bất đồng bộ** của
  dòng outbox đó (mẫu outbox của hiến phán §III), nên nó **MUST** xuất hiện sau khi bộ phát hàng đỗi và
  consumer ghi bảng chạy xong; hệ thống vận hành phải chịu được độ trễ đó.
- **FR-061**: Thao tác ghi thất bại MUST NOT để lại bản ghi kiểm toán.
- **FR-062**: Bảng chiến dịch MUST NOT bị sửa bằng SQL trực tiếp trong vận hành; trigger FR-021 là
  chốt chặn cho bất biến ngân sách.
- **FR-063**: Bản ghi kiểm toán chiến dịch MUST dùng `targetType = "campaign"`, `targetId = NULL`, và
  đặt mã chiến dịch trong `payload.campaignCode`. Hệ thống MUST có endpoint **chỉ đọc**
  `GET /api/v1/admin/campaigns/{code}/audit` trả về nhật ký của đúng mã đó (phân trang), dành cho
  quản trị viên. API nhật ký quản trị chung `GET /api/v1/admin/audit-logs` MUST NOT bị đổi.

### Quét bù

- **FR-070**: Worker quét bù MUST đọc sổ mới mỗi lần chạy và chỉ xét chiến dịch `closed_at IS NULL`.
- **FR-071**: Một chiến dịch được tạo giữa chừng MUST được quét trễ nhất trong một chu kỳ; tài khoản
  tạo giữa hai lần quét MUST NOT bị bỏ sót quá một chu kỳ.
- **FR-072**: `campaign_sweep_runs` MUST tiếp tục ghi sức khoẻ theo `campaign_code`; chiến dịch mới
  chưa quét có `lastSweepAt = null`.

### Cache và cấu hình

- **FR-080**: Hệ thống MUST NOT cache sổ chiến dịch trong 026; mỗi thông điệp/lần quét/lần tra cứu đọc
  CSDL.
- **FR-081**: Hệ thống MUST NOT thêm biến môi trường mới cho dữ liệu chiến dịch.
- **FR-082**: Comment trong mọi tệp `.env*` MUST viết tiếng Anh.

### Chuyển đổi và tài liệu

- **FR-090**: Migration tạo bảng MUST chạy trước; migration dữ liệu seed `launch-2026-10` MUST
  idempotent và dùng đúng giá trị production.
- **FR-091**: Migration MUST NOT thêm khoá ngoại từ `campaign_claims`/`campaign_sweep_runs` sang
  `campaigns`; MUST NOT đổi lược đồ hai bảng đó.
- **FR-092**: Migration down MUST khôi phục được mà không mất dữ liệu `campaign_claims`.
- **FR-093**: 026 MUST ship `specs/026-campaign-registry-db/frontend-guide.md` bao gồm ba endpoint ghi,
  các trường bổ sung, và màn hình admin cần có.

### Ngoài phạm vi

- **FR-100**: 026 MUST NOT thay đổi giá, quyền lợi hay định nghĩa của bất kỳ gói cước nào.
- **FR-101**: 026 MUST NOT thay đổi cách dịch vụ AI tính phí hay kiểm soát chi phí.
- **FR-102**: 026 MUST NOT cung cấp giao diện cho khách hàng cuối.
- **FR-103**: 026 MUST NOT sửa bất kỳ tệp nào trong `specs/025-`.

---

## 8. Key Entities

- **Chiến dịch (Campaign)**: một chương trình tặng quyền lợi trong một khoảng thời gian. Lưu tại
  `subscription.campaigns`. Thuộc tính: `code`, `plan_slug`, `quota`, `reserve`, `starts_at`,
  `ends_at`, `closed_at`. Đây là **bảng**, thay cho mục cấu hình của 025.
- **Lượt cấp (Campaign claim)**: giữ nguyên như 025; **không đổi lược đồ**.
- **Trạng thái quét bù (Campaign sweep run)**: giữ nguyên như 025; **không đổi lược đồ**.
- **Bản ghi kiểm toán (Audit log)**: dùng nhật ký kiểm toán quản trị sẵn có; ghi actor/thời điểm/hành
  động/giá trị cũ-mới cho mỗi thao tác vòng đời chiến dịch.
- **Gói cước (Subscription plan)**: chiến dịch **tham chiếu** `plan_slug`, không sao chép thuộc tính.
- **Tài khoản (Account)**: chỉ đọc `created_at` và `created_by_admin` qua hợp đồng.

---

## 9. Success Criteria *(bắt buộc)*

### Measurable Outcomes

- **SC-001**: Một chiến dịch mới được tạo qua API quản trị có hiệu lực cho tài khoản đăng ký sau đó
  **trong vòng dưới 1 phút**, với **0** tệp mã nguồn bị sửa và **0** lần triển khai lại.
- **SC-002**: Sau khi đóng qua API, **0** lượt cấp mới nào được ghi cho tài khoản tạo sau thời điểm
  đóng; các lượt cấp đã có **không đổi**.
- **SC-003**: **100%** nỗ lực đổi `quota`/`reserve`/`plan_slug` trên chiến dịch đã có lượt cấp bị từ
  chối, kể cả `UPDATE` trực tiếp trong CSDL.
- **SC-004**: Sau migration, **100%** lượt cấp của `launch-2026-10` còn nguyên, và output của 4 API
  tra cứu giống hệt trước migration.
- **SC-005**: Khác biệt về hình dạng response của 4 API đọc so với production chỉ là **các trường
  thêm vào**; tập `reason` vẫn **đúng 12 mã**.
- **SC-006**: **100%** thao tác tạo/sửa/đóng thành công có **đúng một** dòng outbox kiểm toán trong
  cùng transaction, chứa actor, thời điểm, và giá trị cũ→mới; và bản ghi trong `admin_audit_logs`
  xuất hiện **trong vòng 1 phút** kể từ lúc thao tác thành công (độ trễ bộ phát hàng đỗi + consumer).
- **SC-007**: Một chiến dịch tạo giữa chừng được quét trễ nhất trong **một chu kỳ**, và **0** tài
  khoản đủ điều kiện bị bỏ sót quá một chu kỳ.
- **SC-008**: Sau cut-over, **0** khối `campaigns` còn trong cấu hình, và hệ thống khởi động/chạy đầy
  đủ khi sổ rỗng.
- **SC-009**: Cấu hình sai hình dạng bị từ chối **tại lúc ghi** kèm thông báo đúng trường; **100%**
  trường hợp `plan_slug` trỏ tới gói không tồn tại ghi `plan_unavailable` lúc chạy.
- **SC-010**: **0** biến môi trường mới; nạp cấu hình **0** truy vấn CSDL.
- **SC-011**: Một người vận hành xác định được ai đổi gì, lúc nào cho một chiến dịch trong **dưới
  1 phút** kể từ khi thao tác ghi thành công, bằng màn hình nhật ký (đo từ lúc thao tác ghi, không từ
  lúc bản ghi audit xuất hiện — xem FR-060 và SC-006).

---

## 10. Assumptions

- `launch-2026-10` là chiến dịch duy nhất đang chạy trên production; seed migration chỉ cần chèn nó.
  Nếu có mã lịch sử khác trong `campaign_claims` mà không nằm trong sổ, chúng vẫn được tra cứu bình
  thường vì `campaign_claims` không có khoá ngoại.
- Bảng sổ chiến dịch chỉ có vài hàng trong suốt vòng đời dự kiến, nên một truy vấn có chỉ mục trên
  mỗi lần đăng ký là chấp nhận được và không cần cache.
- Hạ tầng đã có trigger PL/pgSQL của PostgreSQL thuần và nhật ký kiểm toán quản trị sẵn có; không cần
  hạ tầng mới.
- Giao diện quản trị dùng lại cơ chế xác thực, phân quyền `Admin` và giới hạn tần suất hiện có.
- Đóng cưỡng bức là one-way qua API; mở lại là ngoài phạm vi và được thực hiện bằng mã chiến dịch mới.
- Cơ chế ưu tiên, chống trùng và quét bù của 025 không đổi hành vi; 026 chỉ đổi nguồn danh sách chiến
  dịch và thêm vòng đời quản trị.
- Sau cut-over, không còn nguồn thứ hai; khối YAML cũ chỉ có thể tồn tại tạm trong một cửa sổ triển
  khai làm bản sao rollback và không được binary mới đọc.
- Chiến dịch mẫu `sample-2026-11` trong `configs/config.yaml` bị bỏ cùng khối `campaigns`. Từ nay
  chiến dịch mẫu ở dev được tạo qua API quản trị, không qua tệp cấu hình.
- Một chiến dịch tạo với `starts_at` lùi rất xa quá khứ sẽ khiến quét bù quét một cửa sổ lớn. Đây là
  trách nhiệm vận hành, chịu cùng giới hạn quy mô mà 025 đã ghi (quét phù hợp quy mô nhỏ; hàng trăm
  nghìn tài khoản cần cách khác).
- Hai kiểm thử cấu hình của 025 khẳng định sự **tồn tại** của khối `campaigns`, nên 026 phải xử lý cả
  hai, không chỉ một: `shared/config/campaign_validation_test.go` (bảng lỗi hình dạng lúc nạp — **xoá**)
  và `shared/config/campaign_config_files_test.go` (T106 — cả hai tệp YAML **phải có** khoá `campaigns`;
  **đảo chiều** thành phải **không** có, và vẫn phải có `campaign_sweep`). Đây là công việc kế hoạch, không
  phải thay đổi 025.

---

## 11. Kịch bản kiểm chứng bắt buộc

Kinh nghiệm từ 025: hai cổng an toàn ở `quickstart.md` đã phải sửa hai lần. 026 ghi thẳng quy tắc
phân biệt môi trường ngay trong đặc tả để không lặp lại.

### 11.1 Cổng an toàn môi trường (bắt buộc đọc trước mọi kịch bản chạm dữ liệu)

**Tín hiệu phân biệt duy nhất là tên cơ sở dữ liệu so với `.env.production`.** Không dùng địa chỉ
mạng, không dùng loopback: cả dev lẫn production đều chạy container với `DB_HOST=postgres` và
`inet_server_addr()` trả về địa chỉ bridge ở **cả hai** bên, nên địa chỉ không phân biệt được.

Quy tắc:
1. Đọc `DB_NAME` từ chính `.env.production` của môi trường đang xét (tệp thật, không phải file mẫu —
   giá trị trong `.env.production.example` chỉ là ví dụ và có thể khác).
2. So với `current_database()`. **Nếu trùng tên ⇒ dừng, tuyệt đối không chạy kịch bản chạm dữ liệu.**
3. Trong kho này, 025 ghi nhận production là `closy_db` và môi trường phát triển cục bộ là
   `smart_wardrobe_db`; `.env.production.example` ghi `DB_NAME=closy`. Vì ví dụ và tệp thật có thể
   khác, **tệp thật là nguồn quyết định**.

Phân loại kịch bản:

| Kịch bản | Chạm dữ liệu thật? | Ghi chú |
|---|---|---|
| V1 (mở/đóng không deploy) | CÓ — tạo tài khoản + cấp gói + email | Phải tắt `email.enabled` trừ tối đa 1 tài khoản |
| V2 (chiến dịch đã đóng không cấp) | CÓ — tạo tài khoản | Phải tắt `email.enabled` |
| V3 (bất biến ngân sách) | CÓ — ghi sổ + cố tình `UPDATE` | Chỉ trên CSDL kiểm thử |
| V4 (migrate launch-2026-10) | CÓ — migration | **Không chạy trên production**; trên bản sao/kiểm thử |
| V5 (hợp đồng API đọc) | KHÔNG — chỉ đọc | An toàn |
| V6 (quét bù nhận chiến dịch mới) | CÓ — tạo tài khoản | Phải tắt `email.enabled` |
| V7 (đọc nhật ký kiểm toán) | CÓ — chỉ ghi chiến dịch qua API | Không đăng ký tài khoản mới nên không sinh email; phải chờ consumer |

**Cổng email.** Mỗi lượt cấp gói sinh một email qua Brevo có trần theo ngày. Mọi kịch bản tạo từ hai
tài khoản trở lên MUST đặt `email.enabled: false` (hoặc `EMAIL_ENABLED=false`) trước khi chạy. Tối đa
**một** email được gửi trong toàn bộ lượt kiểm chứng, dành cho kịch bản kiểm nội dung nếu cần.

### 11.2 V1 — Mở và đóng chiến dịch **mà không cần deploy** (bắt buộc)

1. Trên CSDL kiểm thử đã qua cổng 11.1, gọi API tạo chiến dịch `verify-026` với `starts_at` trong quá
   khứ, `plan_slug = premium-monthly`, `quota = 2`, `reserve = 0`.
2. Đăng ký một tài khoản mới email. Kiểm tra trong vòng 5 phút tài khoản ở gói `premium-monthly`.
3. Gọi API đóng `verify-026` kèm lý do.
4. Đăng ký một tài khoản mới thứ hai.
5. **Điều kiện đạt**: bước 2 nhận gói; bước 4 **không** nhận gói và lý do quan sát được là
   `campaign_sleeping`; trong **suốt** quá trình **không** có commit/tag/deploy nào; `status` trả về
   `closed` ở bước 3 trở đi.

### 11.3 V2 — Chiến dịch đã đóng **không cấp gói cho tài khoản mới** (bắt buộc)

Là phần bước 3–5 của V1, tách riêng để chạy độc lập: trên một chiến dịch đã `closed_at IS NOT NULL`,
tạo một tài khoản mới và chứng minh **không** có dòng `campaign_claims` nào sinh ra cho tài khoản đó,
và API tra cứu trả `eligibility` là không đủ điều kiện/đã ngừng, `reason = campaign_sleeping`.

### 11.4 V3 — Bất biến ngân sách

1. Tạo chiến dịch `verify-026-lock`, sửa `quota` khi chưa cấp ⇒ thành công.
2. Cấp cho một tài khoản; gọi API sửa `quota` ⇒ nhận `422`.
3. Chạy `UPDATE subscription.campaigns SET quota = quota + 1 WHERE code = 'verify-026-lock';` ⇒ trigger
   nâng ngoại lệ; giá trị không đổi.

### 11.5 V4 — Trung thực của migration

Trên bản sao CSDL có dữ liệu `launch-2026-10`: chụp output 4 API tra cứu trước migration; chạy
migration hai lần; so output sau migration. Điều kiện đạt: output giống hệt và lần chạy thứ hai không
lỗi.

### 11.6 V5 — Hợp đồng API đọc không đổi

Gọi 4 endpoint đọc và so tập khoá JSON với production: chỉ được **thêm** ba trường
`closedAt`/`budgetLocked`/`version`; tập `reason` vẫn 12 mã; `status` chỉ thêm `closed`.

### 11.7 V6 — Quét bù nhận chiến dịch mới

Tắt đường thông điệp (hoặc chờ quá chu kỳ), tạo chiến dịch mới rồi tạo một tài khoản trong cửa sổ.
Chờ một chu kỳ quét: tài khoản phải được cấp gói. Điều kiện đạt: có lượt cấp cho tài khoản đó, và
`lastSweepAt` của chiến dịch được cập nhật. **Ngay sau khi tạo** (trước lần quét đầu tiên) `lastSweepAt`
phải là `null` và `degraded` là `false` — FR-072.

### 11.8 V7 — Đọc nhật ký kiểm toán của một chiến dịch

Trên một chiến dịch tạo mới: tạo, sửa một trường thời gian, rồi đóng — mỗi bước **ghi thời điểm thành
công**. Chờ bộ phát hàng đỗi và consumer kiểm toán chạy xong (FR-060: dòng `admin_audit_logs` là hệ quả
bất đồng bộ của dòng outbox), rồi gọi `GET /api/v1/admin/campaigns/{code}/audit`.

**Điều kiện đạt**:
- Có **đúng một** bản ghi cho mỗi thao tác, với `action` lần lượt là `campaign.create`,
  `campaign.update`, `campaign.close`.
- Mọi bản ghi có `targetType = "campaign"` và `targetId = null`; mã chiến dịch nằm ở
  `payload.campaignCode`.
- Bản ghi `campaign.update` có `payload.before` và `payload.after` mô tả đúng trường vừa đổi.
- Bản ghi `campaign.close` có `payload.reason` bằng lý do đã gửi.
- Mọi bản ghi có `actorUserId` là quản trị viên đã thao tác, và `createdAt` **sau** thời điểm thao tác.
- Mã chiến dịch không tồn tại ⇒ `404`.

---

**Changelog**

| Ngày | Thay đổi |
|------|----------|
| 2026-10-03 | Tạo đặc tả 026. Sổ đăng ký chiến dịch trong CSDL thay cho cấu hình. Trả lời 11 câu hỏi thiết kế; ghi nhận sự thay thế so với 025; giữ hợp đồng đọc và 12 mã `reason`; thêm `closed_at`/`status=closed`; no cache; không env mới; kịch bản kiểm chứng mở/đóng không deploy và chiến dịch đã đóng không cấp. |
| 2026-10-03 | Ba vòng `/speckit-analyze` đã sửa artifact (không sửa mã): (1) ngữ nghĩa outbox thật của kiểm toán; (2) một bảng mã lỗi duy nhất ở FR-054; (3) phân công trách nhiệm chọn chiến dịch giữa SQL và bộ đánh giá thuần; (4) `version` vào FR-010/FR-051, FR-005 thu hẹp thành "trong vận hành"; (5) bổ sung `campaign_config_files_test.go`, `grant_uc.go` và 13 tệp kiểm thử vào phạm vi; (6) rollback goose hai lần; (7) thêm kịch bản V7; (8) khoét ngoại lệ no-op cho `PATCH`; (9) DTO trả lời riêng `CampaignAuditLogRes` để `targetId` phát `null`. |
