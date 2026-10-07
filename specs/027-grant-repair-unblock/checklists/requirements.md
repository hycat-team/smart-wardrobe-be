# Specification Quality Checklist: grant-repair-unblock

**Purpose**: Kiểm tra tính đầy đủ và chất lượng đặc tả trước khi lập kế hoạch
**Created**: 2026-10-07
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] CHK001 Không có chi tiết triển khai (ngôn ngữ, framework, API)
- [x] CHK002 Tập trung vào giá trị và nhu cầu, không vào cách làm
- [x] CHK003 Viết cho người đọc hiểu nghiệp vụ, không chỉ cho lập trình viên
- [x] CHK004 Đủ các mục bắt buộc

**Ghi chú CHK001 — cần nói rõ phạm vi kiểm tra**: mô tả đặc tả có nhắc tên hàm, tên bảng
và mã lỗi ở phần "Quan hệ với các yêu cầu đang tồn tại" và phần Bối cảnh. Đây là
**cố ý** và không vi phạm: đặc tả sửa lỗi cho một cơ chế đã tồn tại không thể mô tả mà
không nói nó sửa cái gì, và việc ghi tên file giúp người đọc kiểm chứng lại luận điểm.
Các mục yêu cầu bắt buộc (FR-001..FR-013, AC-01..AC-03, SC-001..SC-007) đều viết ở
mức hành vi và ràng buộc, không chỉ định thuật toán hay cấu trúc dữ liệu.

## Requirement Completeness

- [x] CHK005 Không còn nhãn [NEEDS CLARIFICATION]
- [x] CHK006 Yêu cầu kiểm thử được và không mơ hồ
- [x] CHK007 Tiêu chí thành công đo lường được
- [x] CHK008 Tiêu chí thành công không phụ thuộc công nghệ
- [x] CHK009 Đủ kịch bản chấp nhận
- [x] CHK010 Đã nhận diện trường hợp biên
- [x] CHK011 Phạm vi được xác định rõ
- [x] CHK012 Đã nhận diện phụ thuộc và giả định

**Ghi chú CHK008 — điểm cần biết**: SC-002, SC-004, SC-006 và SC-007 là tiêu chí vận
hành, mô tả trạng thái hệ thống chứ không mô tả trải nghiệm người dùng cuối. Đặc tả
này không có giao diện người dùng, và đối tượng bị ảnh hưởng là quyền lợi của khách
hàng chứ không phải một màn hình. Các tiêu chí được viết theo kết quả quan sát được
("0 tài khoản", "đúng một lần", "0 lần thử lại") chứ không theo tên bảng hay tên hàm,
nên chúng vẫn kiểm chứng được mà không cần biết cách hiện thực.

## Feature Readiness

- [x] CHK013 Mọi yêu cầu chức năng có tiêu chí chấp nhận rõ ràng
- [x] CHK014 Kịch bản người dùng bao phủ các luồng chính
- [x] CHK015 Đặc tả đáp ứng các kết quả đo lường ở Tiêu chí thành công
- [x] CHK016 Không rò rỉ chi tiết triển khai vào yêu cầu

## Đối chiếu với hiến pháp và spec 024

- [x] CHK017 Không vi phạm nguyên tắc quyền sở hữu dữ liệu (§I, FR-027) — FR-013
- [x] CHK018 Không đảo ngược quyết định gỡ client subscription (024 T178) — AC-01
- [x] CHK019 Không thêm cờ đặc quyền — AC-02
- [x] CHK020 Không thay đổi hợp đồng sự kiện — AC-03
- [x] CHK021 Nêu rõ quan hệ với FR-033, kể cả tiền đề mà dữ liệu hiện tại vi phạm
- [x] CHK022 Không thêm ràng buộc duy nhất ở tầng dữ liệu (FR-034 của spec 024)
- [x] CHK023 Ghi nhận phát hiện ngoài phạm vi vào `deferred.md` chứ không để thành task
  chưa đánh dấu (hiến pháp §Governance)

## Tài liệu cần tạo ở giai đoạn kế hoạch

- [ ] CHK024 `plan.md` — chốt vị trí sửa và cơ chế tính lại thời điểm bắt đầu
- [ ] CHK025 ADR 014 — quyết định nới lỏng chốt thứ tự bằng đồng hồ thay thế. Chốt này
  có hệ quả bảo mật nên không được chỉ nằm trong `spec.md`.
- [ ] CHK026 `frontend-guide.md` — FR-011 làm phản hồi sức khoẻ dịch vụ có thể xuất hiện
  thêm một khoá mới trong `grant_rejected_by_reason`. Theo quy tắc hiến pháp, đây là thay
  đổi hình dạng phản hồi và cần hướng dẫn, hoặc phải nêu rõ trong `tasks.md` là không
  có thay đổi phía client. Quyết định này thuộc `plan.md`, không thuộc `spec.md`.
- [ ] CHK027 `deferred.md` — mục "Ngoài phạm vi" đã liệt kê bốn phát hiện
- [ ] CHK028 Đính chính dòng mô tả sai phạm vi `stale_event` trong
  `contracts/subscription-plan-changed-event.md`

## Notes

- Đặc tả này là **sửa lỗi**, không phải tính năng mới: không có endpoint mới, không có
  trường sự kiện mới, không đổi lược đồ.
- Nguyên nhân bốn lỗi và phép đo production đều đã được xác minh bằng log dịch vụ AI và
  truy vấn chỉ đọc trước khi viết đặc tả, nên không còn giả định nào đáng ngờ về dữ
  liệu thật.
- Điểm dễ đọc sai nhất của đặc tả này: mã lý do `stale_event` hiện mang **hai** nghĩa
  khác nhau, và FR-001 chỉ loại bỏ một nghĩa trong số đó. Người đọc `quickstart.md` của
  spec 024 và người đọc runbook vận hành cần biết điều này.