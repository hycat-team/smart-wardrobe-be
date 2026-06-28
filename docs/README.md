# Tài liệu Dự án Closy (SmartWardrobe Backend)

## Nguyên tắc tổ chức tài liệu

- `docs/` chỉ chứa tài liệu viết tay do con người soạn thảo.
- `api/swagger/` là nơi chứa các file Swagger tự động sinh bởi `swaggo/swag` (không đưa vào `docs/` để tránh lẫn lộn).
- Các tài liệu cũ thuộc mô hình Community Marketplace hoặc Resale đã lỗi thời được di chuyển vào `docs/archive/` để lưu trữ, không sử dụng làm định hướng phát triển Core MVP hiện tại.

## Cấu trúc thư mục `docs/`

```text
docs/
├── project/             # Tầm nhìn, phạm vi, thuật ngữ và lịch sử thay đổi của Closy.
├── business/            # Mô hình kinh doanh B2B2C, luồng doanh thu và đề xuất giá trị.
├── product/             # Danh sách tính năng Core MVP, Personas và lộ trình sản phẩm.
├── domain/              # Nghiệp vụ chi tiết của từng mảng (Tủ đồ, AI Styling, Brand, Loyalty, Campaign, CS, Digital Sample Lab).
├── flows/               # Các luồng đi (User flow, Brand flow, Support flow, Sample flow...).
├── system-design/       # Kiến trúc hệ thống, cơ sở dữ liệu, phân quyền và AI Pipeline.
├── api/                 # Quy chuẩn thiết kế API viết tay.
├── development/         # Hướng dẫn thiết lập local, convention, git branch và chiến lược testing.
├── deployment/          # Hướng dẫn Docker, CI/CD, deploy VPS và giám sát (đã loại bỏ IP/thông tin nhạy cảm).
├── decisions/           # Nhật ký các quyết định kiến trúc và sản phẩm lớn (ADR).
├── research/            # Nghiên cứu thị trường và khảo sát người dùng.
└── archive/             # Nơi lưu trữ tài liệu cũ hoặc các hướng đi đã deprecated.
```

## Định vị mô hình mới: B2B2C

- **B2C**: Ứng dụng tủ đồ số cá nhân (Digital Wardrobe) và trợ lý phối đồ AI (AI Outfit Assistant).
- **B2B**: Nguồn thu chính của dự án thông qua cung cấp dịch vụ Brand Loyalty, Customer Service, Brand Campaign, Benefits, Insight Dashboard và **Digital Sample Lab** dành cho các nhãn hàng thời trang.
- **Mục tiêu phi cốt lõi (Non-Goals)**: Không tiếp tục định vị hoặc phát triển Closy như một mạng xã hội (Community Feed, Community Post) hay sàn giao dịch đồ cũ (Resale, Transfer Item, P2P transaction).
