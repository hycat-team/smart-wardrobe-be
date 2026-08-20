# Architectural Decision Record: Out of Scope Features (2026-08)

**Date**: 2026-08-20 | **Status**: Accepted | **Context**: Spec 012 Brand RBAC & Admin Platform

Tài liệu này ghi nhận chính thức 9 hạng mục tính năng nằm ngoài phạm vi (Out of Scope) của hệ thống Backend Smart Wardrobe trong giai đoạn hiện tại (Q3/2026), kèm theo lý do kiến trúc và giải pháp thay thế.

---

## Danh sách 9 tính năng Out of Scope

### 1. Phân quyền RBAC toàn hệ thống phức tạp (Global Dynamic RBAC)
- **Quyết định**: Giữ phân quyền hệ thống đơn giản với 2 vai trò `roleslug.User = "user"` và `roleslug.Admin = "admin"`. Phân quyền theo tổ chức doanh nghiệp được cô lập trong module `brand` qua bảng `brand_members` (`owner`, `staff`).
- **Lý do**: Tránh quá phức tạp hóa bảng `roles`/`permissions` toàn cục khi nhu cầu người dùng tiêu dùng B2C chỉ cần phân biệt giữa user và admin.

### 2. Quản lý chiến dịch quảng cáo Brand (Campaign Management)
- **Quyết định**: Không xây dựng hệ thống chạy chiến dịch quảng cáo và phân bổ ngân sách cho Brand trong đợt này.
- **Lý do**: Chưa có nhu cầu nghiệp vụ thực tế; Brand chỉ cần quản lý sản phẩm, voucher/loyalty và trao đổi với khách hàng.

### 3. Hệ thống Support Ticket (Customer Support Ticketing)
- **Quyết định**: Hỗ trợ khách hàng được thực hiện trực tiếp qua module Chat (`brand_conversations`) hoặc các kênh liên lạc truyền thống.
- **Lý do**: Giảm tải độ phức tạp backend, tận dụng hệ thống Chat thời gian thực hiện có.

### 4. Phòng thử đồ ảo 3D/AR (Virtual Trial Room)
- **Quyết định**: Không xử lý mô hình hóa 3D hoặc tương tác AR thời gian thực trên Backend.
- **Lý do**: Tập trung vào AI Styling 2D (gợi ý phối đồ, trích xuất thuộc tính ảnh thời trang) với độ trễ thấp và chi phí tối ưu.

### 5. Quản trị thông báo đa kênh & tùy biến người dùng (Advanced Multi-channel Notification Management)
- **Quyết định**: Module `notification` chỉ tập trung vào gửi Email giao dịch qua RabbitMQ transactional outbox. Không xây dựng dashboard tùy biến template thông báo động cho admin.
- **Lý do**: Đảm bảo luồng xử lý email tinh gọn, ổn định trước khi mở rộng sang push notifications hoặc SMS.

### 6. Swap / Re-roll AI linh hoạt từng món trong Outfit (Granular Outfit Item Swap)
- **Quyết định**: AI outfit generation sinh gợi ý theo set nguyên bộ hoặc qua chat styling.
- **Lý do**: Tránh bùng nổ token AI không kiểm soát và hạn chế phát sinh logic quản lý state phức tạp.

### 7. AI OOTD Auto-Generator theo thời tiết tự động chạy nền (Background Scheduled OOTD)
- **Quyết định**: AI Styling chỉ được kích hoạt theo nhu cầu chủ động của người dùng (on-demand).
- **Lý do**: Tối ưu chi phí AI và tài nguyên tính toán định kỳ cho hàng triệu người dùng không hoạt động.

### 8. Xác thực số đo cơ thể chuyên sâu (Body-shape Validation & Anthropometry)
- **Quyết định**: Hồ sơ phong cách (`user_style_profiles`) chỉ lưu các thuộc tính cơ bản (sở thích, phong cách, tone da, màu sắc).
- **Lý do**: Giảm rào cản nhập liệu của người dùng mới; dữ liệu cơ bản đã đủ để AI embedding hoạt động hiệu quả.

### 9. Quản lý độ hao mòn quần áo (Garment Decay & Wear Lifecycle Tracking)
- **Quyết định**: Đồ trong tủ đồ (`wardrobe_items`) chỉ lưu trạng thái hiện tại (`itemcondition`), không theo dõi chu kỳ giặt ủi/hao mòn chi tiết theo thời gian thực.
- **Lý do**: Tính năng nâng cao chưa cần thiết cho trải nghiệm MVP của người dùng.
