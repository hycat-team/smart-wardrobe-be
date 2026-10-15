# Specification Quality Checklist: admin-observability

**Purpose**: Kiểm tra tính đầy đủ và chất lượng đặc tả trước khi lập kế hoạch
**Created**: 2026-10-15
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] CHK001 Không có chi tiết triển khai (ngôn ngữ, framework, API)
- [x] CHK002 Tập trung vào giá trị và nhu cầu, không vào cách làm
- [x] CHK003 Viết cho người đọc hiểu nghiệp vụ, không chỉ cho lập trình viên
- [x] CHK004 Đủ các mục bắt buộc

**Ghi chú CHK001 — phạm vi kiểm tra**: đặc tả có nêu đường dẫn endpoint, tên DTO và tập giá trị trạng thái (FR-001, FR-007, FR-008). Đây là **cố ý** theo quy ước các spec gần đây của dự án (023, 025, 026, 028): tính năng sửa contract đã tồn tại không thể mô tả mà không nói rõ nó sửa cái gì, và đường dẫn/trường cụ thể giúp dev kiểm chứng lại luận điểm. Các mục FR đều phát biểu ở mức hành vi và ràng buộc (cái gì phải đúng, khứ hồi, tương thích ngược), không chỉ định thuật toán hay cấu trúc code.

## Requirement Completeness

- [x] CHK005 Không còn nhãn [NEEDS CLARIFICATION]
- [x] CHK006 Yêu cầu kiểm thử được và không mơ hồ
- [x] CHK007 Tiêu chí thành công đo lường được
- [x] CHK008 Tiêu chí thành công không phụ thuộc công nghệ
- [x] CHK009 Đủ kịch bản chấp nhận
- [x] CHK010 Đã nhận diện trường hợp biên
- [x] CHK011 Phạm vi được xác định rõ
- [x] CHK012 Đã nhận diện phụ thuộc và giả định

**Ghi chú CHK006 — điểm cần biết**: FR-013 từng là một diễn giải mở ("phân bổ theo gói khi có kỳ"); đã chốt tại Clarifications 2026-10-15 — đếm theo phép giao của cửa sổ gói hiện tại với kỳ lọc, giới hạn lịch sử đổi gói đã ghi trong Assumptions. Sau phiên clarify, không còn điểm mơ hồ mở nào trong spec.

**Ghi chú CHK008 — điểm cần biết**: SC-002 có nhắc "trường JSON" và "tài liệu bàn giao" — đây là mô tả sản phẩm bàn giao (hợp đồng hướng client), không phải công nghệ triển khai; cách đo (đếm hai chiều giữa mã nguồn và tài liệu) không cần biết cách hiện thực. Các SC còn lại đo bằng hành vi quan sát được ("một lần gọi API", "sai lệch 0 đồng", "không vượt 100%").

## Feature Readiness

- [x] CHK013 Mọi yêu cầu chức năng có tiêu chí chấp nhận rõ ràng
- [x] CHK014 Kịch bản người dùng bao phủ các luồng chính
- [x] CHK015 Đặc tả đáp ứng các kết quả đo lường ở Tiêu chí thành công
- [x] CHK016 Không rò rỉ chi tiết triển khai vào yêu cầu

## Đối chiếu với hiến pháp

- [x] CHK017 Tuân thủ quyền sở hữu dữ liệu (§I/§II): API nhật ký hoạt động thuộc account service (chủ sở hữu bảng `user_activity_events`); không có join chéo module nào được yêu cầu (FR-001)
- [x] CHK018 Tuân thủ chuẩn JSON camelCase cho DTO hướng frontend (§Code Style): FR-004, FR-007, FR-008
- [x] CHK019 Frontend integration guide bắt buộc (§Governance): FR-018, FR-019, FR-020
- [x] CHK020 Technical debt/defer ghi vào `deferred.md` chứ không để task chưa đánh dấu (§Governance): FR-021
- [x] CHK021 Tương thích ngược: các thay đổi contract giữ hành vi nghiệp vụ cũ, trừ các thay đổi chủ đích đã ghi rõ (FR-009; FR-010 kỳ lọc ngày của danh sách giao dịch; FR-012 ngoại lệ mặc định; SC-005)

## Ghi chú chung

- Phiên clarify 2026-10-15 đã chốt 5 quyết định (xem mục Clarifications trong spec): ngữ nghĩa phân bổ theo kỳ; phạm vi payOS gồm cả nạp ví; panel payOS nhận kỳ; sửa kỳ lọc ngày của danh sách giao dịch; trạng thái legacy `failed_legacy`.
- T048 (re-validate cuối, 2026-10-16): spec.md không đổi trong suốt implement (chỉ guide/tasks/deferred đổi) → giữ nguyên 21/21.
- FR-019 liệt kê tối thiểu các phát hiện UX từ khảo sát repo FE ngày 2026-10-15 (chi tiết file:line nằm trong kế hoạch/báo cáo khảo sát, không thuộc spec).
- Ranh giới phạm vi: chỉ repo BE; code FE không đổi trong tính năng này (Assumptions).
