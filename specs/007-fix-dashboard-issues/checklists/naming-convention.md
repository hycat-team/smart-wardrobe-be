# Checklist Quy Tắc Đặt Tên DTO Response: camelCase

**Mục đích**: Kiểm tra chất lượng yêu cầu về quy tắc đặt tên camelCase trong DTO **response trả về frontend** — phát hiện field nào không tuân thủ và đánh giá mức độ đầy đủ/rõ ràng/nhất quán của yêu cầu này.
**Ngày tạo**: 2026-07-24
**Phạm vi**: Chỉ DTO response trả về frontend trong `internal/**/application/dto/`. DTO nội bộ (event payload, search document, AI response) được phép dùng snake_case vì lý do tương thích.

## Tính Đầy Đủ Của Yêu Cầu

- [ ] CHK001 - Quy tắc camelCase cho DTO response trả về frontend có được ghi nhận rõ ràng trong tài liệu API guidelines không? [Completeness, docs/api/README.md §1 — ghi chung "JSON" chưa phân biệt request/response/internal]
- [ ] CHK002 - Tài liệu có phân biệt rõ **DTO response** (bắt buộc camelCase) vs **DTO nội bộ** (được phép snake_case) không? [Completeness, Gap]
- [ ] CHK003 - Quy tắc camelCase có được đưa vào Hiến pháp dự án trong mục "Code Style, Naming & Language Standards" không? [Completeness, Gap — constitution.md §73-75 không đề cập JSON naming]
- [ ] CHK004 - Quy ước camelCase có được tham chiếu nhất quán trong tất cả tài liệu API module (brand-api.md, wardrobe-api.md, ...) không? [Completeness]

## Tính Rõ Ràng

- [ ] CHK005 - Quy tắc camelCase có được định nghĩa rõ cho các trường hợp viết tắt (ID → `id`, URL → `url`, SKU → `sku`, VND → `vnd`) không? [Clarity, Gap]
- [ ] CHK006 - Có ví dụ cụ thể cho field response ghép nhiều từ (ví dụ: `wardrobeItemId`, `subscriptionRevenueVnd`) không? [Clarity, docs/api/README.md §1 chỉ có 1 ví dụ]
- [ ] CHK007 - Có rõ ràng rằng query parameters, path parameters và JSON response fields đều tu theo cùng quy tắc camelCase không? [Clarity, docs/api/README.md §1]
- [ ] CHK008 - Có định nghĩa rõ phạm vi áp dụng: DTO nào là "response trả về frontend" vs DTO nào là "nội bộ" không? [Clarity, Gap]

## Tính Nhất Quán

- [ ] CHK009 - Tất cả DTO response trả về frontend trên 8 module có nhất quán dùng camelCase cho JSON tags không? [Consistency, ✅ không phát hiện vi phạm — xem Kết quả quét]
- [ ] CHK010 - DTO dùng chung (pagination, media) có tuân theo camelCase như DTO module không? [Consistency, shared DTOs ✅ tuân thủ]
- [ ] CHK011 - Quy tắc đặt tên trong API Guidelines (`docs/api/README.md`) có khớp với quy tắc trong các module API docs không? [Consistency]
- [ ] CHK012 - Quy tắc đặt tên DTO response có nhất quán giữa JSON tags trong Go struct và Swagger annotations không? [Consistency, Gap]

## Ngoại Lệ & Tình Huống Biên

- [ ] CHK013 - Các trường hợp DTO nội bộ được phép dùng snake_case (event payload, search document, AI response) đã được ghi nhận chính thức trong tài liệu chưa? [Edge Cases, Gap]
- [ ] CHK014 - Có danh sách rõ ràng các DTO/ngữ cảnh được miễn trừ khỏi quy tắc camelCase không? [Edge Cases, Gap]
- [ ] CHK015 - Có quy tắc xử lý khi thay đổi tên field trong DTO response để tránh breaking change với frontend không? [Edge Cases, Gap]
- [ ] CHK016 - Có hướng dẫn khi một DTO nội bộ (snake_case) cần được expose ra response API không? (Ví dụ: cần tạo DTO response riêng với camelCase hay dùng json tag override) [Edge Cases, Gap]

## Yêu Cầu Phi Chức Năng

- [ ] CHK017 - Có cơ chế thực thi (Go linter, CI pipeline gate) để ngăn non-camelCase JSON tags trong DTO response không? [Non-Functional, Gap]
- [ ] CHK018 - Checklist review PR có quy định kiểm tra JSON tag naming convention cho response DTO không? [Non-Functional, Gap]
- [ ] CHK019 - Quy tắc đặt tên có được ghi ở nơi dễ tìm cho contributor mới và frontend developer không? [Non-Functional, docs/api/README.md §1]

## Bao Phủ Kịch Bản

- [ ] CHK020 - Quy tắc camelCase có được định nghĩa cho tất cả loại response DTO: single item, list, paginated, error response không? [Coverage, Gap]
- [ ] CHK021 - Có quy tắc đặt tên cho field chứa đơn vị tiền tệ (ví dụ: `subscriptionRevenueVnd`, `costPerWearVnd`) để đảm bảo suffix nhất quán không? [Coverage, Gap]
- [ ] CHK022 - Có quy tắc cho field kiểu boolean trong response (ví dụ: `isActive`, `isLocked`, `isDeleted`) nhất quán không? [Coverage, Gap]

## Kết Quả Quét Thực Tế

- [ ] CHK023 - **Cả 3 struct dùng snake_case đều là DTO nội bộ (không trả về frontend):**
    - `WardrobeEventPayload` — event messaging (RabbitMQ) ✅ hợp lệ
    - `SearchDocumentDTO` — Elasticsearch indexing ✅ hợp lệ
    - `FashionMetadataResult` — AI response parsing ✅ hợp lệ
    - → **Không phát hiện vi phạm camelCase nào trong DTO response trả về frontend.** [✅ Phát hiện, cần ghi nhận exceptions]
- [ ] CHK024 - Các trường hợp snake_case trong DTO nội bộ có cần mapper/convert khi response có chứa thông tin từ các DTO này không? [Gap]

## Ghi Chú

- Đánh dấu `[x]` khi mục đã được kiểm tra
- **Kết quả quét code**: 3 file dùng snake_case, tất cả đều là DTO nội bộ — không ảnh hưởng đến frontend
- **Cần thêm**: quy tắc JSON naming vào Hiến pháp (phân biệt response vs internal), CI linter cho response DTO, và tài liệu hóa exceptions
