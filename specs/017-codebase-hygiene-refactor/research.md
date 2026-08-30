# Research: Chuẩn hóa Codebase — Scan & Refactor (017)

**Date**: 2026-08-30 | **Feature**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)

Mọi NEEDS CLARIFICATION trong Technical Context đã được resolve hoặc ghi ở đây dưới dạng quyết
định có rationale. Các câu hỏi spec-level đã resolve ở bước `/speckit-clarify` (Session
2026-08-30 trong spec).

## R1: Công cụ scan tĩnh viết bằng gì, đặt ở đâu?

**Decision**: Viết Go CLI tool tại `internal/tools/hygiene-scan`, chỉ dùng thư viện chuẩn
(`go/parser`, `go/ast`, `go/token`), chạy qua target Makefile mới `make hygiene-scan` (pattern
giống `internal/tools/swagger-split` đã có trong `make generate`).

**Rationale**:
- Cross-platform ngay lập tức (dev trên Windows, CI Linux) — script shell/PowerShell sẽ phát sinh
  nhánh code kép và lỗi path.
- Rule engine (ngưỡng dòng, phân tích AST, phân loại import) cần logic đủ phức tạp để unit test
  được; Go test phù hợp tự nhiên với repo này.
- Không thêm dependency vào runtime service: tool là `package main` riêng, không import module
  nghiệp vụ, không nằm trong Wire DI.

**Alternatives considered**:
- *Shell/PowerShell script* (`scripts/`): nhanh viết nhưng khó test, khó chạy cả hai OS, không
  phân tích được AST (đếm struct/hàm chính xác cần AST).
- *Custom golangci-lint plugin*: cho phép chặn ở CI, nhưng setup plugin riêng phức tạp hơn nhiều
  so với nhu cầu (báo cáo + snapshot + so sánh baseline mà linter không có sẵn). Có thể cân nhắc
  integrate rule đơn giản bằng depguard ở lần chuẩn hóa sau — out of scope.

## R2: Cách phát hiện vi phạm ranh giới module & phân lớp?

**Decision**: Dùng `go/parser` đọc block import của mọi file `.go` (trừ `_test.go` cho một số
rule), phân loại theo đường dẫn package, đối chiếu với bảng Allowed/Prohibited của Constitution
§II và layering §IV:

- **Cấm chéo module** (theo §II): `internal/modules/<A>/**` import
  `internal/modules/<B>/domain/entities`, `internal/modules/<B>/domain/repositories`,
  `internal/modules/<B>/infrastructure/**` với A ≠ B → fail.
- **Cho phép chéo module**: `presentation/contract/`, `application/dto`, `domain/constants/`,
  `internal/shared/**`, `internal/pkg/**` → hợp lệ.
- **Bỏ qua tầng** (theo §IV, trong cùng module): `presentation/**` import
  `infrastructure/**` → fail; `domain/**` import `application|infrastructure|presentation` → fail.
  Ngoại lệ được chấp nhận: file `provider.go` / thư mục `di`/`bootstrap` (nơi dựng Wire) — ghi
  nhận rõ trong rule để tránh false positive.

**Rationale**: Đây là phép đối chiếu path-pattern khách quan, không cần type-check đầy đủ;
`go/parser` đủ và nhanh. Độ chính xác ưu tiên cao (SC-003 ≥90%) nên rule thiên về phát hiện
chính xác nhóm vi phạm rõ ràng, tránh đoán mò các case mơ hồ.

**Alternatives considered**: `go/packages` + full type-check — chính xác hơn nhưng chậm hơn nhiều
và cần load toàn bộ type info; không cần thiết cho luật dựa trên import path.

## R3: Dead code phát hiện bằng gì?

**Decision**: Gọi `golang.org/x/tools/cmd/deadcode` qua `go run` (không pin vào go.mod) làm
**nguồn tham vấn**; kết quả được gắn nhãn `needs-confirmation` trong báo cáo — không bao giờ tự
động xóa (đúng FR-008). Với symbol exported trong `internal/shared/`, `internal/pkg/` (có thể
dùng qua reflection/annotation hoặc là API nội bộ có chủ đích), tool luôn gắn nhãn
cần xác nhận tay.

**Rationale**: `deadcode` là tool chính chủ x/tools, phân tích reachability từ entry points
chính xác hơn grep; nhưng không nhìn thấy dùng-qua-interface động → bắt buộc có bước xác nhận
tay. Scan tool chỉ tổng hợp output, không nhúng thư viện để tránh thêm go.mod dependency.

**Alternatives considered**: `go vet` (không có dead code check), grep tên symbol (false
positive/negative cao).

## R4: Duplication phát hiện bằng gì?

**Decision**: Không tự động hóa trong scan tool v1. Scan tool chỉ gợi ý các **cụm nghi vấn**
(file cùng tên / hàm cùng tên xuất hiện ở ≥2 module, ví dụ `helper`, `mapper`, `utils`). Việc
quyết định gộp hay giữ nguyên là audit thủ công có áp đặt ranh giới module (constitution ưu tiên
hơn de-dup — CHK013).

**Rationale**: De-dup tự động (jscpd/dupl) tạo nhiễu lớn trên codebase Go (DTO struct giống nhau
giữa module là có chủ đích theo §II — contract tách bạch). Quyết định gộp cần đánh giá kiến trúc
mà tool không thể tự quyết.

**Alternatives considered**: `dupl`/jscpd → nhiễu cao, chi phí cấu hình ngưỡng token lớn; để sau
nếu cần.

## R5: Error handling & naming kiểm tra thế nào?

**Decision**: Rule heuristic dựa trên AST/regex, mức **warning** (không fail):
- Error handling: file trong `application/usecase` hay `presentation/handler` trả `error` mà
  không wrap qua `apperror.` / `fmt.Errorf(... %w ...)` (heuristic); handler không theo pattern
  trả `error` được `WrapHandler` wrap (đối chiếu tên hàm `WrapHandler` xuất hiện trong provider).
- Naming: file có struct chính nhưng tên file không khớp (heuristic, warning); hàm query đặt
  `Find...` thay vì `Get...` (regex trên method receiver repo, warning).

**Rationale**: Đây là các quy ước thuộc `design-pattern.md` — mục tiêu của scan là liệt kê lệch
chuẩn để review, không phải chặn hard fail (tránh false positive làm mất uy tín báo cáo).
Ngưỡng fail chỉ dành cho các rule khách quan tuyệt đối (độ dài file, import cấm).

## R6: Định dạng báo cáo & so sánh baseline?

**Decision**: Mỗi lần scan ghi 2 file vào `specs/017-codebase-hygiene-refactor/reports/`:
- `snapshot-YYYYMMDD-HHMMSS.json`: snapshot máy đọc được (mọi violation: id rule, vị trí, mức
  severity, số liệu tổng hợp) — dùng so sánh baseline bằng diff có cấu trúc.
- `scan-report.md`: báo cáo người đọc — tổng kết theo rule + theo module, danh sách vi phạm
  sort theo severity, phần "so với lần trước" (nếu có baseline flag), phần loại trừ (generated,
  archived) và phần dead code needs-confirmation.

CLI flag: `-baseline <file>` để so sánh, mặc định lấy snapshot mới nhất trong thư mục reports.

**Rationale**: JSON cho máy, markdown cho người — đáp ứng FR-004 (re-runnable, so sánh trước/sau)
và CHK021 mà không cần thêm dependency report engine.

## R7: Thiết lập giai đoạn refactor thế nào để giữ nguyên hành vi?

**Decision**: Refactor thủ công theo phase trong tasks.md (mỗi commit ≈ 25% tasks). Mỗi phase:
1. Chạy scan trước phase (snapshot baseline).
2. Tách file / sửa import: chỉ di chuyển & tổ chức lại code, không sửa logic; tách file Go giữ
   cùng package (di chuyển trong package) hoặc extract package con theo chuẩn module hiện có.
3. Chạy `make check` + `make test-race` (theo git-workflow) — pass mới dừng phase.
4. Chạy lại scan ghi snapshot — vi phạm của phase phải về 0.

Vùng code test chưa phủ: đánh dấu rủi ro trong report (theo Edge Case spec) và ưu tiên tách
thuần cơ học (move code) cho vùng đó.

**Rationale**: Đúng FR-005/FR-006 và Decision clarify về batching; giữ hành vi được chứng minh
bằng test hiện có + build + race check thay vì review logic thủ công toàn bộ.

## R8: Các thư mục/file loại trừ khỏi scan?

**Decision**:
- Loại trừ hoàn toàn: `api/swagger/**` (generated), `bin/`, `deployments/`, `specs/`, `docs/`,
  `database/` (Goose SQL), file có marker `// Code generated ... DO NOT EDIT`.
- Scan nhưng ngưỡng nới: `*_test.go` (750/1200 dòng).
- Archived module (nếu có): scan phát hiện nhưng chỉ ghi nhận, không refactor — đối chiếu tên
  module với cấu hình DI hiện hành của `internal/di`/`bootstrap` để xác định module "live".

**Rationale**: Khớp Edge Cases trong spec; nguyên tắc xác định "live vs archived" dựa trên nguồn
sự thật duy nhất là cấu hình DI (Constitution governance: archived module bị gỡ khỏi Wire).
