# Deferred — 026 Sổ Đăng Ký Chiến Dịch Trong Cơ Sở Dữ Liệu

Phát hiện trong lúc chốt feature 026. **Không mục nào là việc của 026**: hoặc thuộc hạ tầng
kiểm thử dùng chung, hoặc thuộc một tài liệu mà lần chạy này không được phép sửa.

Không mục nào chặn việc đóng 026. Chúng được ghi ở đây chứ không để lại thành ô
`- [ ]` trong `tasks.md`, vì một ô chưa tick khẳng định *"feature này chưa xong"* — điều vừa là
một khẳng định khác vừa là sai.

- **D-01 làm suy yếu bằng chứng của 10 task đã tick.** Đọc D-01 trước khi tin bất kỳ task nào
  trong danh sách đó.
- **D-02 thuộc chính 026** nhưng không sửa được trong lần chạy này (xem "Vì sao ngoài phạm vi").

---

## D-01 — Không có hạ tầng nào chạy được quickstart V1–V7 của 026 trong `go test` thường

- **Vấn đề**: toàn bộ bằng chứng 026 cần CSDL thật đều nằm trong các tệp kiểm thử `t.Skip` khi
  `TEST_DATABASE_URL` không được đặt, và `campaign_seed_integration_test.go` còn cần thêm
  `TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK=true` (rollback xoá bảng nên phải được bật tường minh):

  | Tệp kiểm thử | Phủ claim |
  |---|---|
  | `infrastructure/persistence/campaign_repo_integration_test.go` | T022 (trigger If-Match, `FOR SHARE`, `EXISTS`) |
  | `infrastructure/persistence/campaign_schema_guard_integration_test.go` | T027 (không FK, `campaign_claims` không đổi) |
  | `infrastructure/persistence/campaign_guard_integration_test.go` | T057 (trigger bất biến chặn SQL thô) |
  | `infrastructure/persistence/campaign_seed_integration_test.go` | T059 (seed idempotent, rollback) |
  | `usecase/campaign/campaign_db_grant_integration_test.go` | T038 |
  | `usecase/campaign/campaign_db_close_integration_test.go` | T049 |
  | `usecase/campaign/campaign_db_close_granted_integration_test.go` | T047 |
  | `usecase/campaign/campaign_db_time_integration_test.go` | T063 |
  | `usecase/campaign/campaign_db_audit_read_integration_test.go` | T072 |

  Ngoài CSDL, các kịch bản `quickstart.md` còn cần hai thứ mà không kho kiểm thử nào cung cấp:
  **đăng ký tài khoản thật** (để có `identity.users` và một lượt đăng ký đúng mốc thời gian) và
  **chờ consumer outbox chạy xong** trước khi đọc `GET .../audit` (FR-060 ghi nhật ký bất đồng
  bộ). Không có bước dựng nào làm ra một tài khoản qua cả ba cửa tạo tài khoản, nên V1–V7 chỉ
  chạy được bằng tay với một môi trường thật.

- **Vì sao ngoài phạm vi 026**: đây là **hạ tầng kiểm thử dùng chung**, không phải logic chiến
  dịch. 026 không sửa entity, không sửa migration, không sửa mã harness nào; nó chỉ là feature
  đầu tiên phải dựng lại đường cấp gói trên CSDL thật. Sửa `openWebhookTestDB` hay viết một
  `openCampaignTestDB` là việc của feature hạ tầng hoặc của feature sau, và đặt nó ở đây sẽ
  biến một khoản nợ của hạ tầng thành một ô chưa tick của 026.

  Lưu ý đây là **cùng họ** với D-01 và D-02 của `specs/025-signup-campaign-grant/deferred.md`:
  cả hai feature đầu tiên chạm CSDL thật đều phát hiện hạ tầng chưa sẵn sàng. Ba phát hiện,
  cùng một nguyên nhân gốc.

- **Gỡ bằng cách nào**: một `make test-integration` hoặc một profile CI có sẵn Postgres, chạy
  `make migrate-up svc=account` trước rồi `go test` với `TEST_DATABASE_URL` và
  `TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK=true`. Riêng quickstart thì cần thêm một bước dựng tài
  khoản dùng chung — nên nằm ở `shared/tools`, không ở gói campaign. Sau khi có, các tệp trên
  chạy **không đổi một dòng**, vì chúng đã được viết đúng từ đầu; chỉ là không ai bật chúng.

- **Ảnh hưởng tới task đã tick**: **`[X] T022`, `[X] T027`, `[X] T038`, `[X] T047`, `[X] T049`,
  `[X] T057`, `[X] T059`, `[X] T063`, `[X] T072`, `[X] T078`.**

  Tất cả mười task đều đã tick, và tất cả đều **có mã** nhưng **chưa từng chạy** trong một lần
  `go test` thường — chúng đã báo `SKIP`. Đọc `[X]` của chúng là đọc "đã viết và biên dịch đúng",
  không phải "đã chạy và xanh". Điều chưa từng được kiểm chứng bằng CSDL thật gồm: hành vi thực
  của trigger `trg_campaigns_immutable_budget`, phép `SELECT … FOR SHARE` của FR-045, phép
  `UPDATE … WHERE version = ?` trả 0 dòng, idempotent của seed, việc rollback giữ nguyên
  `campaign_claims`, và việc nhật ký thật sự xuất hiện sau khi consumer chạy.

  Riêng `[X] T078` còn yếu hơn nữa: nó yêu cầu chạy V5, V6 và nhánh `plan_unavailable` **trên
  CSDL kiểm thử**, và điều đó đã không thực hiện được. Phần T078 *đã* làm là bổ sung các kiểm thử
  đơn vị chứng minh đúng nội dung của V5/V6 bằng fake (`campaign_read_contract_shape_test.go`,
  `campaign_sweep_reads_registry_each_run_test.go`) — đó là bằng chứng thay thế, không phải bằng
  chứng gốc.

---

## D-02 — `contracts/campaign-registry-api.md` §6.3 vẫn chỉ đúng đường dẫn tệp cũ của `CampaignAuditLogRes`

- **Vấn đề**: `contracts/campaign-registry-api.md` §6.3 (đoạn bắt đầu bằng "**Endpoint này dùng
  DTO trả lời riêng `CampaignAuditLogRes`**") vẫn ghi:

  > (đặt trong `services/account/dto/subscription/campaign_dto.go`)

  Trong lúc chốt 026, `CampaignAuditLogRes` được chuyển sang
  `services/account/dto/subscription/campaign_audit_dto.go` cùng `campaign_registry_dto.go`, để
  `campaign_dto.go` từ 578 dòng về dưới ngưỡng cảnh báo 500 dòng của `code-hygiene.md` §H1. Khả
  năng chuyển DTO chính xác không đổi: cùng package, cùng thẻ `json`, cùng mapper, không đổi
  hình dạng response.

  Lỗi nằm ở **tài liệu**, không nằm ở mã. Nhưng hậu quả không nhỏ: đây là hợp đồng mà
  `frontend-guide.md` mới viết dựa trên đó, và một người đọc hợp đồng sẽ tìm nhầm tệp.

- **Vì sao ngoài phạm vi lần chạy này**: đây **là việc của 026** — nó sửa hợp đồng của chính nó.
  Nhưng `contracts/` nằm ngoài phạm vi ghi được của lần chạy này (orchestrator giữ quyền sửa
  `spec.md` / `plan.md` / `contracts/` / `quickstart.md`), và sửa tài liệu hợp đồng mà không
  được sửa đồng thời thì tệ hơn là không sửa. Vì vậy nó được **báo cáo**, không được âm thầm bỏ
  qua.

- **Gỡ bằng cách nào**: một sửa đổi một dòng trong
  `specs/026-campaign-registry-db/contracts/campaign-registry-api.md` §6.3, đổi
  `campaign_dto.go` thành `campaign_audit_dto.go` và thêm câu "tách khỏi `campaign_dto.go` ở
  bước chốt để đáp ứng H1". Nên làm cùng lúc với bất kỳ thay đổi nào khác của 026.

- **Ảnh hưởng tới task đã tick**: **`[X] T069`.**

  T069 yêu cầu khai báo `CampaignAuditLogRes` trong `campaign_dto.go` và mapper của nó trong
  `campaign_mapper.go`; chỗ đầu đã không còn đúng sau bước tách tệp. Phần còn lại của T069 —
  `TargetID *uuid.UUID` khai `json:"targetId"` **không** `omitempty`, DTO riêng chứ không tái
  dùng `dto/admin.AdminAuditLogDTO`, và mapper `MapCampaignAuditLogRes` vẫn đúng và vẫn ở
  `application/mapper/campaign_mapper.go` — được giữ nguyên và vẫn đúng.