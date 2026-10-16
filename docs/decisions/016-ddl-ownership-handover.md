# ADR 016 - Bàn giao quyền sở hữu DDL theo schema (Account → AI/Commerce)

- **Status**: Accepted
- **Decision Date**: 2026-10-16
- **Decision Maker**: Duck
- **Resolves**: `specs/027-grant-repair-unblock/deferred.md` D-06

## 1. Context

DDL của các bảng `aiexecutor.*` (dịch vụ AI) và `dashboard.*` (dịch vụ Commerce)
nằm rải rác trong lịch sử migration của dịch vụ Account, trong khi cây
migration của AI rỗng và Commerce chỉ giữ bảng của mình. Điển hình là D-06 của
spec 027: bảng `aiexecutor.ai_policy_grants` do migration Account
(`20260622131631_add_ai_cost_policy.sql`) tạo với `CHECK` viết nội tuyến không
đặt tên, nên PostgreSQL tự sinh tên `user_ai_policy_grants_check` theo tên bảng
cũ; hai lần đổi tên/chuyển schema sau đó giữ nguyên tên ràng buộc mồ côi này.
Hệ quả là không service nào tự đứng được về DDL: Account chạy trước nhưng tạo
bảng của AI, AI boot trắng không có bảng của mình — vi phạm trực tiếp nguyên
tắc §I (mỗi service tự triển khai độc lập, kể cả tầng migration theo FR-035).

Hai tiền đề đã được xác minh bằng thực nghiệm trước quyết định này:

- **Goose chỉ track theo tập version, không checksum nội dung** (research R1,
  đã chứng minh trên database tạm ở T004): viết lại nội dung file cũ mà giữ
  nguyên tên + version là vô hình với database đã applied; cấm duy nhất là xóa
  file (database trắng replay sẽ mất DDL, `Down` sẽ `ErrVersionNotFound`).
- **Parity đã đo**: toàn bộ cột/constraint/index của 7+3 bảng trên dev khớp
  với nội dung baseline mới, delta duy nhất có chủ ý là tên constraint mới
  (xem verification-log mục parity check trước T040).

Các phương án đã loại, ghi lại để không ai phải phân tích lại:

- *Chỉ gỡ file mới, giữ lịch sử cũ*: loại vì để lại DDL ngoài chủ trong lịch
  sử, gate sau này không thể phân biệt cũ/mới nếu không có danh sách miễn trừ
  phình to vô hạn.
- *Baseline dạng delta so với nội dung cũ đã gỡ*: loại vì database trắng sau
  rewrite không còn "nội dung cũ" để delta lên — trắng sẽ thiếu vĩnh viễn
  trong khi database cũ vẫn đủ (lệch hai đường, vỡ SC-002).
- *Phẫu thuật bảng version (INSERT/DELETE tay)*: loại vì không cần thiết khi
  content-edit đã vô hình với database cũ.

## 2. Decision

**Viết lại nội dung 11 file migration Account** (gỡ toàn bộ DDL/data
`aiexecutor.*`/`dashboard.*`, giữ nguyên tên file + version + `Down` đảo ngược
tương ứng), và **dựng chain baseline độc lập đầy đủ ở đúng cây chủ**:

- `services/ai/database/migrations/20261016213000_adopt_aiexecutor_ownership.sql`
  (m = 1): đủ 7 bảng `aiexecutor` ở trạng thái cuối, tên constraint tường minh
  ngay từ đầu — gồm `ai_policy_grants_effective_window_check` thay cho tên mồ
  côi `user_ai_policy_grants_check` — kèm khối rename một lần duy nhất cho
  database cũ (chạy chỉ khi tên cũ tồn tại và tên mới chưa có). Tên legacy của
  bảng/index/FK còn lại (`user_ai_policy_grants_pkey`, …) được giữ nguyên để
  khớp database cũ từng byte.
- `services/commerce/database/migrations/20261016213001_adopt_dashboard_ownership.sql`
  (k = 1): đủ 3 bảng `dashboard` (đã gồm 3 cột payos) ở trạng thái cuối.
- Mọi baseline viết idempotent (`IF NOT EXISTS`, seed guard) để database cũ
  boot lên là hội tụ no-op, database trắng replay ra đúng schema cuối.
  Convergence audit (diff trắng-mới vs restore-cũ-hội-tụ) là bằng chứng duy
  nhất chứng minh baseline đủ, không phải gate.

**Luật ông nội (grandfather rule)**: đúng 13 version trên (11 nội dung viết
lại + 2 baseline) được miễn kiểm tra chéo; gate chỉ chặn vi phạm MỚI phát sinh
về sau. Danh sách miễn trừ nằm trong gate script dưới dạng tên file chính xác,
không phải pattern cả cây.

**Gate máy lâu dài**: script `shared/tools/migration-ownership-check` đọc bảng
sở hữu ở `contracts/ownership-contract.md` §1 làm single source of truth (ba
luật G-ACCT/G-COMM/G-AI ở §2, ngoại lệ `REFERENCES identity.users(` cho FK
user xuyên schema), quét `services/*/database/migrations/*.sql`, đỏ kèm tên
luật + file + dòng và exit non-zero. Target Makefile `migration-check` đấu vào
`make check` ngay sau `vet` để dev thấy đỏ local trước khi push. Mọi sửa map ở
§1 về sau phải qua spec riêng có ADR (hợp đồng §4), không sửa lặng lẽ.

**Rollback duy nhất = restore backup**: cấm `down -v`, cấm xóa volume, cấm
target production cho mọi verify. Thứ tự deploy Account → AI → Commerce trong
cửa sổ full-stop do chủ hệ thống thao tác tay theo cutover-runbook.

## 3. Consequences

**Tích cực**

- Mỗi service tự chạy DDL của mình khi boot (FR-035), khôi phục §I cho tầng
  migration; prod giữ boot song song, dev giữ Account-trước qua `depends_on`.
- Tên constraint mới khớp bảng chứa nó; mọi tham chiếu tên cũ đã cập nhật,
  còn lại chỉ là dòng lịch sử có chủ ý (ghi trong báo cáo T021).
- Gate chặn được cả ba mẫu vi phạm (đã chứng minh đỏ/xanh ở quickstart V3);
  lệnh ngừng bắn thủ công FR-011 hết hiệu lực khi spec đóng, ba luật gate có
  hiệu lực vĩnh viễn.

**Tiêu cực**

- Lịch sử migration Account vĩnh viễn mang 13 version "đã viết lại": ai đọc
  lịch sử mà không đọc hợp đồng sẽ không hiểu vì sao file cũ không còn DDL
  ngoài chủ — đó là lý do danh sách miễn trừ trỏ thẳng vào hợp đồng §3.
- Baseline một file duy nhất mỗi cây: review một lần là thấy hết, nhưng mọi
  thay đổi schema sau này phải là migration mới chịu gate đầy đủ.

**Rủi ro được chấp nhận**

- Baseline seed theo giá trị chain (premium `25000000000`, 6 model configs):
  drift dữ liệu dev do sửa tay không qua migration (premium `30000000000`,
  model `gemini-3.5-flash-lite`) được ghi nhận nhưng không ghi đè database cũ;
  cần đối chiếu trước giờ G (không ảnh hưởng schema, SC-002 ở mức schema).
