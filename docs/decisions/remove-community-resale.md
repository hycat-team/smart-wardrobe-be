# ADR 002: Loại bỏ tính năng Community Posts & Resale làm core

*   **Trạng thái**: Accepted
*   **Ngày quyết định**: 2026-06-28
*   **Người quyết định**: Ban quản trị dự án Closy

## 1. Ngữ cảnh
Tính năng đăng bài cộng đồng (community post), bán đồ cũ (resale) và bàn giao đồ cũ giữa người dùng (transfer item) phát sinh nhiều friction point trong trải nghiệm thực tế và làm loãng định vị tủ đồ cá nhân thông minh của ứng dụng.

## 2. Quyết định
Loại bỏ hoàn toàn các luồng P2P transaction và social feed khỏi lõi MVP mới. Lưu trữ toàn bộ tài liệu đặc tả cũ vào thư mục `docs/archive/` để tham khảo khi cần, không mở rộng hoặc duy trì code liên quan đến phần này trong tương lai gần.

## 3. Hệ quả
*   Tích cực: Giảm thiểu độ phức tạp của codebase, tập trung nguồn lực phát triển mảng B2B.
*   Tiêu cực: Một số phần code liên quan đến community và resale sẽ được dọn dẹp hoặc đóng băng (deprecated).
