# Hằng số Nghiệp vụ Admin Module (Admin Constants)

Các hằng số dùng trong Admin APIs:

## 1. Hành động Audit Log (AuditAction)

- **Các giá trị hợp lệ:**
    - `user.create`: Admin tạo tài khoản người dùng mới.
    - `user.role_change`: Admin thay đổi vai trò tài khoản.
    - `user.deactivate`: Admin vô hiệu hóa tài khoản.
    - `user.activate`: Admin kích hoạt lại tài khoản.
    - `user.sessions_revoke`: Admin thu hồi toàn bộ phiên đăng nhập.
    - `brand.member_add`: Admin thêm thành viên vào nhãn hàng.
    - `brand.member_role_change`: Admin thay đổi vai trò thành viên.
    - `brand.member_remove`: Admin gỡ thành viên khỏi nhãn hàng.
    - `brand.ownership_transfer`: Admin chuyển giao quyền sở hữu nhãn hàng.
    - `subscription.plan_create`: Admin tạo gói cước mới.
    - `subscription.plan_update`: Admin cập nhật gói cước.
    - `subscription.plan_deactivate`: Admin bật/tắt trạng thái gói cước.
    - `subscription.cancel`: Admin hủy gói hội viên của người dùng.
    - `subscription.grant`: Admin cấp gói hội viên thủ công.
    - `payment.retry_investigation`: Admin gửi lại yêu cầu đối soát giao dịch.
    - `payment.resolve_paid`: Admin xác nhận giao dịch đã thanh toán.
    - `payment.resolve_cancelled`: Admin xác nhận hủy giao dịch.
    - `payment.resolve_creation_failed`: Admin đánh dấu tạo link thanh toán thất bại.
    - `ai.pricing_update`: Admin cập nhật giá AI model.
    - `ai.grant_create`: Admin cấp chính sách hạn mức AI cho người dùng.

## 2. Loại đối tượng Audit Log (AuditTargetType)

- **Các giá trị hợp lệ:**
    - `user`: Người dùng.
    - `brand`: Nhãn hàng.
    - `brand_member`: Thành viên nhãn hàng.
    - `subscription_plan`: Gói cước đăng ký.
    - `user_subscription`: Đăng ký gói của người dùng.
    - `deposit_transaction`: Giao dịch nạp tiền.
    - `ai_model_config`: Cấu hình giá AI model.
    - `ai_policy_grant`: Chính sách hạn mức AI.
