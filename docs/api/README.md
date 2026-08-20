# Quy chuẩn Thiết kế API (API Guidelines)

Tất cả các API được phát triển trong dự án Closy tuân thủ chuẩn RESTful:

## 1. Định dạng JSON
*   Mọi yêu cầu gửi lên và phản hồi trả về sử dụng định dạng JSON chuẩn.
*   Đặt tên thuộc tính theo quy tắc camelCase (ví dụ: `wardrobeItemId`).

## 2. Mã trạng thái HTTP (HTTP Status Codes)
*   `200 OK`: Yêu cầu xử lý thành công.
*   `201 Created`: Tạo thực thể mới thành công.
*   `400 Bad Request`: Lỗi định dạng dữ liệu đầu vào.
*   `401 Unauthorized`: Chưa đăng nhập hoặc token hết hạn.
*   `403 Forbidden`: Không có quyền truy cập tài nguyên.
*   `404 Not Found`: Đường dẫn hoặc thực thể không tồn tại.
*   `500 Internal Server Error`: Lỗi phát sinh từ hệ thống máy chủ.

## 3. Tiền tố đường dẫn (Endpoints Prefix)
*   API công khai/người dùng: `/api/v1/...`
*   API đối tác thương hiệu: `/api/v1/brand-portal/...` (đối với Brand Portal) hoặc `/api/v1/brands/...` (đối với phía User tương tác với Brand)

## 4. Quản lý Hằng số (Constants)
*   Mọi API có sử dụng các giá trị hằng số (Status, Type, Role, v.v.) trong Request Body hoặc Response Body phải được đặc tả giá trị rõ ràng trong tài liệu hằng số tương ứng.
*   Bắt buộc phải dẫn liên kết (link) đến tài liệu hằng số tại thuộc tính đó để Frontend có thể đối chiếu giá trị hợp lệ.

## 5. Phân nhóm theo Luồng Nghiệp vụ (User Flows)
*   Các API không đứng đơn lẻ mà phục vụ một chuỗi nghiệp vụ (ví dụ: Quy trình tham gia thành viên và quy đổi đặc quyền) phải được nhóm lại theo từng Luồng sử dụng (User Flows).
*   Mỗi luồng cần thể hiện rõ thứ tự gọi các API và các bên liên quan (Actors).

## 6. Tính Hiện Hữu
*   Tài liệu API chỉ ghi nhận các API thực tế đã được triển khai (implemented) trong codebase. 
*   Các tính năng chưa phát triển (Future backlog) không được liệt kê để tránh gây nhầm lẫn khi tích hợp.

---

## Cấu trúc Thư mục API Docs

```
docs/api/
├── README.md                    ← Tài liệu này
├── admin/                       ← Admin Audit Log module
│   ├── constants.md             ← Hằng số Audit Action, Target Type
│   └── audit-log-api.md         ← API tra cứu Nhật ký Quản trị
├── aiexecutor/                  ← AI Executor module
│   ├── constants.md             ← Hằng số AI Policy Grant, Enforcement Mode
│   └── admin-ai-api.md          ← API quản lý giá AI, hạn mức, usage
├── brand/                       ← Brand module (B2B2C)
│   ├── constants.md             ← Hằng số Brand, Member, Loyalty, Benefit
│   ├── brand-portal-api.md      ← API Brand Portal, Customer-facing
│   ├── admin-brand-api.md       ← API quản trị Brand (Admin)
│   ├── brand-item-api.md        ← API Digital Sample Lab & Brand Items
│   ├── brand-chat-api.md        ← API Chat Customer ↔ Brand Staff
│   └── brand-loyalty-api.md     ← API Loyalty, Points, Benefits, Vouchers
├── dashboard/                   ← Dashboard & Analytics module
│   ├── constants.md             ← Hằng số Dashboard (Granularity, Metric Key)
│   ├── dashboard-admin-api.md   ← API Dashboard Admin
│   ├── dashboard-brand-api.md   ← API Dashboard Brand (B2B)
│   └── dashboard-analytics-api.md ← API Dynamic Query Engine & Analytics
├── fashion/                     ← Fashion / AI Styling module
│   ├── constants.md             ← Hằng số Outfit Item Context, AI Request
│   └── ai-styling-api.md        ← API gợi ý phối đồ AI
├── identity/                    ← Identity & Auth module
│   ├── constants.md             ← Hằng số Role, User Status, Gender
│   ├── auth-api.md              ← API Đăng ký, Đăng nhập, Quên mật khẩu
│   ├── me-api.md                ← API Hồ sơ của tôi (Profile)
│   └── admin-user-api.md        ← API quản trị Người dùng (Admin)
├── subscription/                ← Subscription & Billing module
│   ├── constants.md             ← Hằng số Deposit, Plan, AI Policy
│   ├── subscription-api.md      ← API xem gói, quản lý quota
│   ├── billing-api.md           ← API ví nội bộ, mua gói, PayOS
│   ├── admin-subscription-api.md ← API quản trị gói cước (Admin)
│   └── admin-billing-api.md     ← API quản trị thanh toán (Admin)
└── wardrobe/                    ← Wardrobe & Outfit module
    ├── constants.md             ← Hằng số Wardrobe, Outfit, Task
    ├── wardrobe-api.md          ← API quản lý tủ đồ cá nhân
    ├── outfits-api.md           ← API thiết kế bộ phối đồ
    ├── category-api.md          ← API danh mục thời trang
    └── admin-wardrobe-api.md    ← API quản trị tủ đồ (Admin)
```

## Mapping Swagger Modules → API Docs

| Swagger Module | API Docs Folder | Tags |
|----------------|-----------------|------|
| `admin` | `admin/` | Admin Audit |
| `aiexecutor` | `aiexecutor/` | Admin AI |
| `brand` | `brand/` | Admin Brand, Brand, Brand Portal, Brand Chat, Brand Loyalty, Brand Item, Brand Benefit |
| `dashboard` | `dashboard/` | Dashboard Admin, Dashboard Brand, Dashboard Analytics, Dashboard User |
| `fashion` | `fashion/` | Wardrobe AI |
| `identity` | `identity/` | Admin User, Auth, Me |
| `subscription` | `subscription/` | Admin Subscription, Admin Billing, Subscription, Billing |
| `wardrobe` | `wardrobe/` | Admin Wardrobe, Category, Wardrobe, Outfits |
