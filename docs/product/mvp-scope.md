# Phạm vi MVP & Danh sách tính năng (MVP Scope & Features)

Tài liệu này định nghĩa tầm nhìn, phạm vi và danh sách tính năng chính thức cho MVP của dự án Closy (Smart Wardrobe).

---

## 1. Danh sách tính năng chính (Feature List)

### A. Các tính năng cốt lõi cho Người dùng (B2C Core Features)
- **Số hóa tủ đồ**: Upload ảnh chụp quần áo, tự động tách nền và trích xuất đặc tính (loại, màu sắc, chất liệu) qua AI.
- **Gợi ý phối đồ AI**: Gợi ý phối hợp các món đồ có sẵn thành bộ trang phục dựa trên thời tiết và quy tắc màu sắc thời trang.
- **Quản lý bộ sưu tập OOTD**: Lưu và gắn thẻ các bộ trang phục yêu thích.
- **Gói dịch vụ Premium**: Nâng cấp hạn mức tủ đồ và tăng lượt gọi AI phối đồ hàng ngày.

### B. Các tính năng thương hiệu (B2B Brand Features)
- **Brand Portal**: Trang quản trị dành riêng cho nhãn hàng thời trang để quản lý chiến dịch và xem dữ liệu phân tích.
- **Hệ thống Loyalty & Membership**:
  - Tích điểm thành viên từ các hóa đơn mua sắm.
  - Quy đổi điểm nhận ưu đãi, voucher đặc quyền của thương hiệu.
- **Quản lý chiến dịch (Campaign Management)**:
  - Đăng tải các sự kiện, ưu đãi mùa mới.
  - Phân phối mã giảm giá cá nhân hóa dựa trên gu thời trang của người dùng.
- **Digital Sample Lab**:
  - Đăng tải mẫu thiết kế ảo (3D / Bản vẽ phác thảo).
  - Tích hợp tính năng thử đồ mẫu (Virtual Trial), bình chọn (Voting) và thu thập phản hồi (Feedback) trực tiếp từ người dùng.
  - Bảng thống kê Insight hành vi và đánh giá mức độ quan tâm của thị trường.

---

## 2. Giai đoạn triển khai (Roadmap Phases)

### Giai đoạn 1: B2C Core và Nền tảng Brand cơ bản
- **B2C**:
  - Tính năng số hóa tủ đồ cơ bản (upload ảnh, lưu danh mục).
  - Tính năng gợi ý phối đồ AI theo thời tiết địa phương.
  - Lưu lịch sử OOTD.
- **B2B**:
  - Tạo trang quản trị Brand cơ bản.
  - Cơ chế tích điểm thành viên và quy đổi voucher giảm giá đơn giản.
  - **Digital Sample Lab (v1)**: Cho phép Brand đăng tải hình ảnh thiết kế mới, hỗ trợ người dùng bình chọn (Vote) và gửi phản hồi dạng text.

### Giai đoạn 2: Tối ưu hóa AI và Mở rộng dịch vụ Brand
- Nâng cấp AI Styling Engine hỗ trợ Color Theory sâu hơn và cá nhân hóa theo phong cách riêng.
- Mở rộng Digital Sample Lab hỗ trợ thử đồ ảo bằng AI (Virtual Trial) trên cơ thể ảo của người dùng.
- Bảng báo cáo phân tích Insight chuyên sâu (Insight Dashboard) cho Brand.

---

## 3. Các mục tiêu phi cốt lõi (Non-Goals)

Để giữ sự tập trung cao độ vào mô hình B2B2C, Closy loại bỏ hoàn toàn các mục tiêu và tính năng sau khỏi định hướng Core MVP:

- **Không xây dựng mạng xã hội đại chúng**: Không phát triển hệ thống bảng tin (Social Feed), bài đăng của người dùng (User posts), lượt thích, bình luận công khai theo phong cách Facebook/Instagram.
- **Không làm sàn giao dịch đồ cũ (Resale Platform)**: Không tích hợp các chức năng định giá đồ cũ, giao dịch mua bán trực tiếp ngang hàng (P2P transaction) hay đăng ký gian hàng bán lẻ cá nhân.
- **Không quản lý vận chuyển/giao hàng**: Không phát triển hệ thống kết nối đơn vị vận chuyển hoặc giao nhận đồ ký gửi.
- **Không cạnh tranh với các sàn TMĐT**: Định vị Closy là kênh tương tác và nghiên cứu thị trường cho Brand, không phải là một mini Shopee hoặc sàn thương mại điện tử mua bán thời trang đại trà.
