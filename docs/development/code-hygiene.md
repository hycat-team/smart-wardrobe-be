# Code Hygiene Standard

Chuẩn hygiene chính thức của project — nguồn sự thật duy nhất cho các quy tắc cấu trúc code,
ngưỡng định lượng và quy trình scan. Mọi lần scan (`make hygiene-scan`), code review và agent
thao tác trên repo đối chiếu về tài liệu này.

Phạm vi áp dụng: toàn bộ mã nguồn Go của ứng dụng (`cmd/`, `internal/`), loại trừ mã sinh tự động
và artifact (xem §3). Tài liệu này KHÔNG định nghĩa lại quy ước đã có — kiến trúc nằm ở
[constitution](../../.specify/memory/constitution.md), conventions implementation nằm ở
[design-pattern.md](../system-design/design-pattern.md); tài liệu này chỉ bổ sung ngưỡng hygiene
và quy trình scan. Được thiết lập bởi feature
[017-codebase-hygiene-refactor](../../specs/017-codebase-hygiene-refactor/spec.md).

---

## 1. Checklist 8 mục (H1–H8)

| ID | Mục | Ngưỡng / quy tắc | Mức độ |
|----|-----|------------------|--------|
| **H1** | Độ dài file & hàm | File production > 500 dòng = warning, > 800 dòng = fail. File test nới lỏng 50%: > 750 = warning, > 1200 = fail. Hàm > 80 dòng = warning, tách theo bước nghiệp vụ. | fail / warning |
| **H2** | Struct-per-file | File production mang hành vi (có method receiver) cho **tối đa 2 kiểu dữ liệu**. File chỉ chứa data-shape thuần (DTO, entity, request/response) không bị giới hạn số struct — đó là hình dạng hợp lệ của package dto/entities. | warning |
| **H3** | Tổ chức file/folder | Tên file khớp chủ thể chính của file; vị trí folder tuân theo canonical layout `architecture.md §IV` (handler/usecase/mapper/dto/entities/persistence ở đúng layer). Xác minh bằng SC-006: định vị được chức năng chỉ qua tên folder/file. | review |
| **H4** | Ranh giới module & phân lớp | Import chéo module BỊ CẤM: `domain/entities`, `domain/repositories`, `infrastructure/**` của module khác (constitution §II). Bỏ qua tầng BỊ CẤM (constitution §IV): presentation/application không import infrastructure cùng module; domain không import application/infrastructure/presentation. | fail |
| **H5** | Dead code | Hàm/kiểu unreachable (theo `deadcode`) = needs-confirmation. Bắt buộc xác nhận tay (grep tham chiếu, interface, reflection) trước khi xóa. Không bao giờ xóa tự động. | needs-confirmation |
| **H6** | Trùng lặp | File/hàm cùng tên xuất hiện ở ≥ 2 module = gợi ý audit. Quyết định gộp hay giữ nguyên là audit tay; ranh giới module ưu tiên hơn de-dup — nếu gộp vi phạm §II thì giữ nguyên với lý do. | warning |
| **H7** | Naming | Method repository dùng tiền tố `Get...` thay vì `Find...` (design-pattern.md); tên file khớp struct chính; helper dùng-1-lần đuôi `_helper.go`. | warning |
| **H8** | Error handling | Handler không tự tạo error nghiệp vụ (`errors.New`/`fmt.Errorf`) — delegate xuống use case. Use case không tạo `errors.New` trần — dùng `apperror` factory hoặc domain error constructor; ngoại lệ: sentinel package-level `var ErrXxx = errors.New(...)` (quy ước Go). `fmt.Errorf` có argument kiểu error phải wrap `%w`; `fmt.Errorf` message-only (không có error argument) được chấp nhận. | warning |

### Ngoại lệ H4 được phép (sanctioned)

- `application/constants/**` (apperror, eventconstants...) và `domain/constants/**`: package
  hằng số thuần không hành vi — mọi layer được import (constitution §II cho `domain/constants`;
  §V bắt buộc dùng `apperror` tập trung).
- File dựng Wire: `provider.go` ở module root, `internal/di/**`, `internal/bootstrap/**` được
  miễn rule bỏ qua tầng (chúng là điểm nối DI) — nhưng vẫn chịu rule import chéo module bị cấm.
- Mã sinh tự động và mã lưu trữ (archived) không thuộc phạm vi scan (§3).

## 2. Severity

- **fail** — vi phạm khách quan tuyệt đối, phải sửa (H1 vượt ngưỡng fail, H4).
- **warning** — heuristic cần review (H1 hàm dài, H2, H6, H7, H8). Có thể chấp nhận giữ nguyên
  khi có lý do ghi rõ trong báo cáo (`accepted-with-reason`).
- **needs-confirmation** — phát hiện cần con người xác nhận trước khi hành động (H5).

## 3. Phạm vi scan

**Loại trừ hoàn toàn**: `api/swagger/**` (generated), `bin/`, `deployments/`, `specs/`, `docs/`,
`database/`, file có header `// Code generated ... DO NOT EDIT.`.

**Ngưỡng nới lỏng**: `*_test.go` (+50% cho H1).

**Module lưu trữ (archived)**: scan vẫn phát hiện nhưng chỉ ghi nhận (`accepted-with-reason`),
không refactor — giữ nguyên cấu trúc thư mục theo governance.

## 3b. Keep-list (accepted-with-reason) — FR-011

Các finding advisory (H5/H8) đã được team review và quyết định giữ được ghi vào
`internal/tools/hygiene-scan/accepted.json` (rule + file + symbol + reason). Khi scan:

- Mục khớp keep-list được đánh dấu `accepted-with-reason` — **không tính** vào tổng vi phạm
  fail/warning/needs-confirmation, nhưng vẫn được in minh bạch trong bảng Violations kèm reason.
- Mục mới chưa có trong keep-list vẫn hiển thị bình thường — vi phạm thật luôn nổi bật.
- Quy trình review: xác minh tham chiếu thật (interface/test/reflection) → xóa nếu chết thật,
  hoặc thêm entry vào `accepted.json` kèm reason → commit cùng cụm thay đổi.

Sentinel `errors.New` package-level (`var ErrXxx = errors.New(...)`) và `fmt.Errorf`
message-only (không có argument kiểu error) được heuristic H8 miễn sẵn — không cần thêm vào
keep-list.

## 4. Quy trình scan

```bash
make hygiene-scan                                  # scan + so sánh snapshot mới nhất
go run ./internal/tools/hygiene-scan -baseline none   # bỏ so sánh
go run ./internal/tools/hygiene-scan -baseline specs/017-codebase-hygiene-refactor/reports/snapshot-<run_id>.json
```

Đầu ra (mỗi lần chạy, trong `specs/017-codebase-hygiene-refactor/reports/`):

- `snapshot-<YYYYMMDD-HHMMSS>.json` — snapshot máy đọc được (so sánh baseline).
- `scan-report.md` — báo cáo người đọc: Summary (so baseline), By Rule, By Module, Violations,
  Exclusions, Needs Confirmation.

Tool thuần đọc: không bao giờ sửa source; kết quả deterministic. Chi tiết contract CLI:
[contracts/hygiene-scan-cli.md](../../specs/017-codebase-hygiene-refactor/contracts/hygiene-scan-cli.md).

## 5. Ví dụ trước/sau

### God file (H1/H2) — TRƯỚC

`brand/presentation/handler/brand_loyalty_handler.go` — 958 dòng chứa mọi handler loyalty
(điểm, hạng, phần thưởng...): vượt ngưỡng fail 800.

### SAU

Tách cùng package theo nhóm use case, mỗi file một trách nhiệm, giữ nguyên method set mà route
đăng ký dùng (không đổi hành vi, không đổi API):

```text
brand/presentation/handler/
├── brand_loyalty_handler.go          # lifecycle + route-facing entry methods chung
├── brand_loyalty_points_handler.go   # nhóm handler điểm
└── brand_loyalty_tier_handler.go     # nhóm handler hạng/thành viên
```

### Import vi phạm (H4) — TRƯỚC

```go
// internal/modules/dashboard/application/usecase/...
import "smart-wardrobe-be/internal/modules/wardrobe/domain/entities" // ❌ cấm §II
```

### SAU

Gọi qua contract + DTO, tự enrich bằng `sliceutils.IndexBy`/`GroupBy`
(theo [contract-purity-pattern.md](../system-design/contract-purity-pattern.md)):

```go
import wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"

items := wardrobeContract.NewWardrobeContract(...).GetItemsByIDs(ctx, ids) // ✅ DTO only
byID := sliceutils.IndexBy(items, func(i *dto.WardrobeItemRes) string { return i.ID.String() })
```

### Handler tạo error trần (H8) — TRƯỚC / SAU

```go
// ❌ TRƯỚC — handler tự tạo error nghiệp vụ
func (h *Handler) DeleteItem(c *gin.Context) error {
    if !owned {
        return errors.New("not owner")
    }
    ...
}

// ✅ SAU — use case trả domain error, handler chỉ trả kết quả
func (h *Handler) DeleteItem(c *gin.Context) error {
    if err := h.uc.DeleteItem(c.Request.Context(), req.ID); err != nil {
        return err
    }
    shared_pres.Success(c, msgDeleteSuccess, nil)
    return nil
}
```

## 6. Tham chiếu

- Kiến trúc & ranh giới: [.specify/memory/constitution.md](../../.specify/memory/constitution.md) §II, §IV, §V
- Conventions implementation: [design-pattern.md](../system-design/design-pattern.md)
- Pattern tách contract: [contract-purity-pattern.md](../system-design/contract-purity-pattern.md)
- Feature gốc: [spec 017](../../specs/017-codebase-hygiene-refactor/spec.md)
