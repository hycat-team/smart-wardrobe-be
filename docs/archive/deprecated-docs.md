# Tài liệu Lưu trữ / Hết hạn (Archived & Deprecated Docs)

Tài liệu này lưu trữ danh sách các tài liệu thuộc mô hình cũ (Community Marketplace, Resale, P2P Transactions) đã bị loại bỏ khỏi Core MVP của Closy:

## 1. Thư mục `docs/archive/community-model/`
*   `business-module-main-old.md`: Chứa thông tin về thiết kế vai trò người dùng, trong đó có phần feed cộng đồng và chia sẻ bài viết công khai cũ.
*   `core-feature-flows-old.md`: Chứa mô tả chi tiết về các luồng nghiệp vụ bài viết cộng đồng và giao lưu trước đây.

## 2. Thư mục `docs/archive/resale-flow/`
*   `transfer-flow.md`: Đặc tả chi tiết luồng chuyển nhượng trang phục (Item Transfer Flow) giữa người mua và người bán sau khi giao dịch ngoài đời từ bài đăng thanh lý.

## 3. Các tài liệu khác
*   `short-description-old.md`: Mô tả tóm tắt định vị cũ của dự án SmartWardrobe/Closy.

## 4. Workflow đã lưu trữ
*   `deploy-production.yml`: Workflow rollback monolith khẩn cấp (build từ tag frozen `monolith-legacy`). Đã ngừng dùng theo [ADR 012](../decisions/012-retire-monolith-rollback.md); tag `monolith-legacy` vẫn được giữ theo constitution §II. Muốn dùng lại: chuyển file về `.github/workflows/`.
