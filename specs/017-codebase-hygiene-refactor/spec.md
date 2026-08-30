# Feature Specification: Chuẩn hóa Codebase — Scan & Refactor theo Checklist Hygiene

**Feature Branch**: `017-codebase-hygiene-refactor`

**Created**: 2026-08-30

**Status**: Draft

**Input**: User description: "Chuẩn hóa project: scan và refactor codebase theo checklist hygiene — giới hạn độ dài file, số struct trên file, tổ chức file/folder, ranh giới module, dead code, duplication, naming consistency, error handling. Git phải tuân thủ tuyệt đối AGENTS.md (no autonomous git mutations)."

## Clarifications

### Session 2026-08-30

- Q: Ngưỡng định lượng chính thức cho hygiene checklist (FR-003) chốt mức nào? → A: File mã
  nguồn ứng dụng > 500 dòng = warning, > 800 dòng = fail; hàm > 80 dòng = xem xét tách; file test
  nới lỏng thêm 50% (warning > 750 dòng, fail > 1200 dòng).
- Q: Văn bản chuẩn hygiene (checklist + ngưỡng) lưu ở đâu làm nguồn duy nhất? → A: Tạo doc mới
  `docs/development/code-hygiene.md`; `design-pattern.md` và `AGENTS.md` chỉ thêm tham chiếu tới.
- Q: Báo cáo scan và snapshot từng lần scan lưu ở đâu? → A: Trong
  `specs/017-codebase-hygiene-refactor/reports/` (một file báo cáo tổng hợp + snapshot theo từng
  lần scan để đối chiếu trước/sau refactor), commit cùng nhánh feature.
- Q: Ranh giới "cụm refactor nhỏ" (FR-006) chốt theo tiêu chí nào? → A: Theo phase của tasks.md;
  mỗi commit tương đương khoảng 25% số tasks trong tasks.md (tỉ lệ commit/phase sẽ cân nhắc lại
  chi tiết khi tasks.md được tạo).
- Q: Chỉ tiêu "≥90% kết quả scan đúng" (SC-003) xác minh bằng cách nào? → A: Developer rà mẫu
  ngẫu nhiên 20 vi phạm (hoặc toàn bộ nếu báo cáo có ít hơn 20), đối chiếu code thực tế; tỉ lệ
  đúng = số vi phạm xác thực được / tổng số mẫu; kết quả ghi vào báo cáo scan.
- Q: Khi nào commit được thực hiện? → A: (Cập nhật yêu cầu 2026-08-30) Chỉ commit SAU KHI toàn
  bộ tasks.md hoàn tất — các cụm refactor vẫn chia nhỏ để tổ chức công việc và dễ review, nhưng
  git mutation chỉ thực hiện một lần ở cuối feature, khi developer chỉ định.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scan toàn codebase và xuất báo cáo vi phạm (Priority: P1)

Như một developer của team, tôi muốn chạy một lần scan toàn bộ codebase theo checklist hygiene
đã thống nhất và nhận được một báo cáo liệt kê từng vi phạm (file nào, vị trí nào, vi phạm mục
nào, mức độ nghiêm trọng ra sao), để tôi có bức tranh toàn cảnh về chất lượng cấu trúc code trước
khi quyết định refactor cái gì.

Báo cáo scan là hoạt động **chỉ đọc** — không thay đổi bất kỳ dòng code nào, không ảnh hưởng hành
vi hệ thống.

**Why this priority**: Báo cáo scan là đầu vào bắt buộc cho mọi refactor phía sau. Không có scan
thì không biết vi phạm gì, ở đâu, nghiêm trọng cỡ nào, và không thể ưu tiên công việc.

**Independent Test**: Có thể kiểm tra độc lập bằng cách chạy scan trên codebase hiện tại và đối
chiếu báo cáo với thực tế code (kiểm tra ngẫu nhiên vài kết quả). Giá trị: danh sách vi phạm đầy
đủ, có định lượng, có phân loại mức độ.

**Acceptance Scenarios**:

1. **Given** codebase hiện tại, **When** developer chạy scan, **Then** báo cáo được tạo ra liệt
   kê toàn bộ vi phạm theo từng mục của checklist (độ dài file, số struct trên file, tổ chức
   folder, ranh giới module, dead code, duplication, naming, error handling), mỗi vi phạm có vị
   trí cụ thể và mức độ (warning / fail).
2. **Given** codebase hiện tại, **When** developer chạy scan, **Then** các mã nguồn được sinh tự
   động (generated code, ví dụ tài liệu API sinh máy) được loại trừ khỏi kết quả và được ghi chú
   rõ trong báo cáo.
3. **Given** báo cáo scan, **When** developer kiểm tra ngẫu nhiên ít nhất 5 vi phạm được liệt kê,
   **Then** mỗi vi phạm đều xác thực được trong code thực tế (không có kết quả bịa hoặc false
   positive nghiêm trọng).
4. **Given** báo cáo scan, **When** đọc phần tổng kết, **Then** thấy số lượng vi phạm theo từng
   mục checklist và theo từng module, để xếp hạng ưu tiên refactor.

---

### User Story 2 - Refactor các vi phạm nghiêm trọng về kích thước file và trách nhiệm (Priority: P2)

Như một developer, tôi muốn các file quá dài hoặc chứa quá nhiều kiểu dữ liệu không liên quan
được tách thành các file nhỏ hơn theo đúng trách nhiệm (responsibility), để tôi mở một file là
ngay lập tức hiểu file đó làm gì và sửa code mà không phải cuộn hàng trăm dòng.

Việc tách file phải **giữ nguyên hành vi hệ thống** (behavior-preserving): không có API nào thay
đổi, không có logic nào bị sửa ngoài việc di chuyển/tổ chức lại code.

**Why this priority**: Đây là nhóm vi phạm ảnh hưởng trực tiếp đến tốc độ đọc hiểu và bảo trì
hằng ngày của dev, và dễ xác nhận nhất (đếm số dòng, đếm kiểu dữ liệu trên file).

**Independent Test**: Có thể kiểm tra độc lập bằng cách chạy lại scan sau refactor (các vi phạm
kích thước file phải giảm về 0 hoặc về mức đã thống nhất) đồng thời toàn bộ bộ test hiện có phải
vẫn pass mà không cần sửa assertion nào (trừ import path nếu ngôn ngữ yêu cầu).

**Acceptance Scenarios**:

1. **Given** một file bị đánh dấu fail vì vượt ngưỡng độ dài, **When** refactor xong, **Then**
   không còn file nào (không tính mã sinh tự động) vượt ngưỡng fail, và mỗi file kết quả có một
   trách nhiệm rõ ràng đúng như tên file.
2. **Given** một file chứa nhiều kiểu dữ liệu không cùng mối quan tâm, **When** refactor xong,
   **Then** các kiểu dữ liệu được nhóm theo mối quan tâm: các kiểu nhỏ phục vụ chung một hành vi
   có thể ở chung file, còn các kiểu độc lập tách ra file riêng.
3. **Given** việc tách file trong một module, **When** refactor xong, **Then** toàn bộ bộ test
   pass, build thành công, và linter không báo lỗi mới.

---

### User Story 3 - Kiểm tra và sửa vi phạm ranh giới module và phân lớp (Priority: P2)

Như một developer, tôi muốn scan phát hiện mọi vi phạm ranh giới kiến trúc — module này nhúng tay
vào module khác, tầng presentation gọi thẳng tầng hạ tầng, entity của module A bị import bừa bãi
bởi module B — và các vi phạm này được sửa đúng theo quy tắc kiến trúc hiện hành của dự án, để
codebase không bị mục rữa theo thời gian.

**Why this priority**: Vi phạm ranh giới là loại vi phạm nguy hiểm nhất (làm kiến trúc modular
monolith mất ý nghĩa) dù khó nhận thấy hơn bằng mắt thường so với file dài.

**Independent Test**: Có thể kiểm tra độc lập bằng cách rà soát danh sách import chéo giữa các
module và giữa các tầng trong báo cáo: sau refactor, không còn import nào thuộc nhóm bị cấm.

**Acceptance Scenarios**:

1. **Given** codebase, **When** scan, **Then** báo cáo liệt kê mọi import chéo bị cấm giữa các
   module (entity, repository, persistence của module khác) và mọi việc bỏ qua tầng (presentation
   gọi thẳng infrastructure).
2. **Given** danh sách vi phạm ranh giới, **When** refactor xong, **Then** mỗi vi phạm được sửa
   theo đúng quy tắc kiến trúc hiện hành (chuyển qua contract/DTO hợp lệ) mà không thay đổi hành
   vi người dùng cuối.
3. **Given** sau refactor ranh giới, **When** developer thêm một import mới vi phạm sau này,
   **Then** bộ quy tắc scan có thể chạy lại để phát hiện (scan phải lặp lại được, không phải
   thao tác one-off).

---

### User Story 4 - Dọn dead code, trùng lặp và thống nhất naming (Priority: P3)

Như một developer, tôi muốn các đoạn code không còn ai dùng, các helper bị copy-paste giữa các
module, và các cách đặt tên không thống nhất được phát hiện và xử lý, để khi tôi tìm một hàm hoặc
đặt tên một thứ mới, tôi không bị nhầm lẫn bởi các bản sao lỗi thời và các quy ước lệch nhau.

**Why this priority**: Ít cấp bách hơn vì không gây sai kiến trúc, nhưng tích tụ lâu ngày làm
tăng chi phí tìm kiếm và bảo trì.

**Independent Test**: Có thể kiểm tra độc lập bằng cách: sau cleanup, mọi loại bỏ dead code đều
kèm bằng chứng "không còn chỗ nào tham chiếu" và toàn bộ test vẫn pass; các quy ước naming được
liệt kê và đối chiếu nhất quán.

**Acceptance Scenarios**:

1. **Given** một hàm/kiểu dữ liệu/file không còn tham chiếu, **When** cleanup xong, **Then** nó
   bị xóa kèm ghi chú bằng chứng trong báo cáo (tại sao chắc chắn không dùng), và test vẫn pass.
2. **Given** các helper trùng lặp giữa các module, **When** cleanup xong, **Then** mỗi logic
   trùng lặp được gộp về đúng vị trí dùng chung (nếu hợp lệ về ranh giới) hoặc được ghi nhận lý
   do giữ nguyên nếu việc gộp vi phạm ranh giới module.
3. **Given** các lệch chuẩn naming và error handling, **When** cleanup xong, **Then** tồn tại
   một danh sách quy ước thống nhất đã áp dụng và số vi phạm còn lại được ghi nhận.

---

### User Story 5 - Chốt checklist và ngưỡng định lượng thành chuẩn chính thức (Priority: P1)

Như một developer, tôi muốn các ngưỡng và quy tắc hygiene được chốt thành văn bản chuẩn chính thức
của dự án (số liệu cụ thể cho "file quá dài", "quá nhiều struct trên file", v.v.), để mọi lần scan
sau này — cũng như mọi code review và agent thao tác trên repo — đều đối chiếu về cùng một chuẩn
duy nhất thay vì cảm tính.

**Why this priority**: Không chốt ngưỡng thì scan không thể ra kết luận khách quan ("dài hay
không" sẽ thành tranh cãi), nên việc này phải xong trước khi chấm vi phạm.

**Independent Test**: Có thể kiểm tra độc lập bằng cách đọc văn bản chuẩn: mỗi mục checklist đều
có ngưỡng định lượng hoặc quy tắc kiểm định được, và mọi vi phạm trong báo cáo scan đều quy chiếu
được về đúng một điều trong văn bản này.

**Acceptance Scenarios**:

1. **Given** checklist hygiene, **When** văn bản chuẩn được chốt, **Then** mỗi mục có ngưỡng định
   lượng rõ ràng (hoặc quy tắc có thể kiểm định khách quan) và được cập nhật vào tài liệu quy ước
   phát triển hiện hành của dự án.
2. **Given** báo cáo scan với nhiều vi phạm, **When** đối chiếu từng vi phạm với văn bản chuẩn,
   **Then** mỗi vi phạm đều trỏ về đúng mục điều luật cụ thể mà nó vi phạm.

---

### Edge Cases

- **Mã nguồn sinh tự động** (generated code): loại trừ khỏi mọi ngưỡng, nhưng phải được liệt kê
  minh bạch trong báo cáo để không lọt vi phạm thật đằng sau.
- **File test**: có thể áp ngưỡng nới lỏng hơn file production (test thường dài hơn do dữ liệu
  fixtures) — ngưỡng cụ thể chốt trong User Story 5.
- **Module đã lưu trữ (archived)**: theo governance hiện hành, giữ nguyên cấu trúc thư mục, chỉ
  ghi nhận trong báo cáo, không refactor.
- **Refactor khi test chưa phủ vùng đó**: nếu tách file mà không có test bảo vệ, phải chứng minh
  không đổi hành vi bằng biên dịch + lint + so sánh logic, và đánh dấu vùng rủi ro trong báo cáo
  thay vì "hy vọng" là an toàn.
- **Vi phạm số lượng quá lớn**: nếu một mục có hàng chục vi phạm, refactor theo cụm nhỏ (từng
  module hoặc từng nhóm vi phạm) để mỗi lần thay đổi đều dễ review và dễ revert — không gom một
  khối khổng lồ.
- **Dead code không chắc chắn** (dùng qua reflection/annotation sinh máy): mặc định KHÔNG xóa,
  đưa vào danh sách "cần xác nhận tay" thay vì xóa tự động.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống quy trình MUST thực hiện scan toàn bộ mã nguồn ứng dụng (loại trừ mã sinh
  tự động, mã lưu trữ) theo checklist hygiene đã chốt, và xuất báo cáo vi phạm có: vị trí vi phạm,
  mục checklist bị vi phạm, mức độ (warning/fail), và tổng hợp theo mục + theo module.
- **FR-002**: Checklist hygiene MUST bao gồm tối thiểu 8 mục: (1) độ dài file, (2) số kiểu dữ liệu
  trên một file kèm quy tắc "cùng mối quan tâm", (3) tổ chức file/folder hợp lý để định vị,
  (4) ranh giới module & phân lớp, (5) dead code, (6) trùng lặp, (7) naming consistency,
  (8) error handling consistency.
- **FR-003**: Mỗi mục checklist MUST có ngưỡng định lượng hoặc quy tắc kiểm định khách quan,
  được chốt trong văn bản quy ước chính thức trước khi chấm vi phạm.
- **FR-004**: Scan MUST chạy lặp lại được (re-runnable) với cùng một chuẩn để đối chiếu trước/sau
  refactor, không phải thao tác thủ công one-off. Báo cáo và snapshot từng lần scan MUST được lưu
  trong `specs/017-codebase-hygiene-refactor/reports/` để so sánh trực quan giữa các lần scan.
- **FR-005**: Mọi refactor MUST giữ nguyên hành vi hệ thống: không thay đổi hợp đồng API, không
  sửa logic nghiệp vụ; mỗi cụm refactor phải được xác nhận bằng build + lint + toàn bộ bộ test
  (kèm race check theo quy trình hiện hành) trước khi kết thúc.
- **FR-006**: Refactor MUST được chia thành các cụm nhỏ có ranh giới rõ theo phase của tasks.md
  để tổ chức công việc, review và xác minh từng bước; tuy nhiên git mutation (commit) chỉ thực
  hiện MỘT LẦN sau khi toàn bộ tasks.md hoàn tất và `make check` + `make test-race` pass toàn
  repo — theo yêu cầu cập nhật của developer (2026-08-30).
- **FR-007**: Vi phạm ranh giới module (import bị cấm, bỏ qua tầng) MUST được sửa theo đúng quy
  tắc kiến trúc hiện hành của dự án, không được "né" bằng cách tạo exception ad-hoc.
- **FR-008**: Loại bỏ dead code MUST kèm bằng chứng không tham chiếu; các trường hợp không chắc
  chắn (sinh máy, reflection) MUST đưa vào danh sách xác nhận tay thay vì xóa tự động.
- **FR-009**: Toàn bộ hoạt động git trong feature này MUST tuân thủ tuyệt đối quy trình git hiện
  hành của dự án: agent KHÔNG tự thực hiện bất kỳ git mutation nào (commit, push, branch, merge,
  v.v.) nếu không có chỉ định rõ ràng từ developer; trước khi đề xuất commit, build + lint +
  test-race phải pass.
- **FR-010**: Sau khi hoàn tất, chuẩn hygiene MUST được chốt thành doc mới
  `docs/development/code-hygiene.md` (checklist + ngưỡng định lượng), với `design-pattern.md` và
  tài liệu quy tắc làm việc của agent chỉ thêm tham chiếu — để các scan lần sau và các agent làm
  việc trên repo đối chiếu về cùng một nguồn duy nhất.

### Key Entities *(include if feature involves data)*

- **Vi phạm (Violation)**: một điểm không tuân thủ checklist — có thuộc tính: vị trí (file/vị
  trí trong file), mục checklist, mức độ (warning/fail), quy chiếu điều luật, trạng thái xử lý
  (chưa xử lý / đã refactor / giữ nguyên có lý do).
- **Báo cáo scan (Scan Report)**: tổng hợp các vi phạm của một lần scan — có ngày scan, phạm vi
  (những gì bị loại trừ), số liệu theo mục và theo module, danh sách ưu tiên refactor.
- **Checklist hygiene**: tập 8+ mục quy tắc, mỗi mục có ngưỡng định lượng và quy chiếu tài liệu
  kiến trúc/quy ước hiện hành của dự án.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Sau refactor, số file mã nguồn ứng dụng (không tính mã sinh tự động) vượt ngưỡng
  fail về độ dài file là **0**.
- **SC-002**: Sau refactor, số import chéo bị cấm giữa các module và số lần bỏ qua tầng phân lớp
  là **0** theo danh sách quy tắc kiến trúc hiện hành.
- **SC-003**: Báo cáo scan cho phép xác thực với tỉ lệ kết quả đúng ≥ **90%**, đo bằng cách:
  developer rà mẫu ngẫu nhiên 20 vi phạm (hoặc toàn bộ nếu báo cáo có ít hơn 20), đối chiếu code
  thực tế; tỉ lệ đúng = số vi phạm xác thực được / tổng số mẫu; kết quả audit ghi vào báo cáo scan.
- **SC-004**: Mọi cụm refactor hoàn tất đều có bằng chứng build + lint + toàn bộ bộ test pass;
  số cụm refactor bị phát hiện làm thay đổi hành vi người dùng cuối là **0**.
- **SC-005**: 100% vi phạm trong báo cáo đều quy chiếu được về đúng một điều luật cụ thể trong
  văn bản chuẩn hygiene đã chốt.
- **SC-006 (định hướng gián tiếp)**: Sau hoàn tất, một dev mới (hoặc agent) có thể định vị bất
  kỳ chức năng nào chỉ qua tên folder/file mà không cần tra cứu ngoài — xác nhận bằng việc giải
  thích vị trí của ít nhất 5 chức năng đại diện mà không mở code.

## Assumptions

- "Người dùng" của feature này là developer của team (bao gồm agent thao tác trên repo), không
  phải người dùng cuối sản phẩm — do đó mọi tiêu chí đều xét từ góc độ bảo trì mã nguồn.
- Ngưỡng định lượng đã chốt chính thức (Session 2026-08-30): file mã nguồn ứng dụng > 500 dòng =
  warning, > 800 dòng = fail; hàm > 80 dòng = xem xét tách; file test nới lỏng thêm 50%
  (warning > 750 dòng, fail > 1200 dòng). File nhiều kiểu dữ liệu: chấp nhận nếu các kiểu cùng
  phục vụ một mối quan tâm.
- Mã sinh tự động (tài liệu API sinh máy, file lớn sinh máy) được loại trừ toàn bộ khỏi ngưỡng.
- Scan được thực hiện bằng phân tích tĩnh trên mã nguồn; không yêu cầu công cụ thương mại.
- Phạm vi scan là mã nguồn ứng dụng (thư mục mã nguồn chính); tài liệu, script triển khai, mã
  lưu trữ (archived) không thuộc phạm vi refactor.
- Việc refactor ưu tiên các module có vi phạm nghiêm trọng nhất trước (theo số liệu báo cáo),
  bắt đầu từ các vi phạm fail trước vi phạm warning.
- Quy trình git hiện hành (không git mutation tự động, commit tiếng Anh theo conventional
  commits, cụm thay đổi đủ lớn để tránh noise) được giữ nguyên như quy định trong tài liệu quy
  tắc làm việc của dự án.
