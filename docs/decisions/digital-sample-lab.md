# ADR 003: Triển khai tính năng Digital Sample Lab cho B2B

- **Trạng thái**: Accepted
- **Ngày quyết định**: 2026-06-28
- **Người quyết định**: Duck

## 1. Ngữ cảnh

Các thương hiệu thiết kế gặp khó khăn trong việc thử nghiệm phản hồi của khách hàng đối với các sản phẩm mẫu (vải thử mẫu thực tế tốn kém và mất nhiều thời gian chế tác). Việc số hóa mẫu thử 3D hoặc bản vẽ và phân phối đến tệp người dùng tiềm năng trên ứng dụng sẽ giải quyết triệt để painpoint này.

## 2. Quyết định

Phát triển module **Digital Sample Lab**:

- Cung cấp API cho nhãn hàng đẩy sản phẩm mẫu thử ảo (hình ảnh thiết kế 3D/concept).
- Xây dựng cơ chế bình chọn (Voting) và thu thập phản hồi (Feedback) của người dùng Closy.
- Tự động tổng hợp dữ liệu ẩn danh và trực quan hóa lên Insight Dashboard.

## 3. Hệ quả

- Tích cực: Tạo ra giá trị đặc biệt thu hút các nhãn hàng hợp tác lâu dài.
- Cần giải quyết: Cần tối ưu hóa trải nghiệm ướm thử ảo (Virtual Trial) trên điện thoại và thiết kế database hỗ trợ vote/feedback với hiệu năng cao.
