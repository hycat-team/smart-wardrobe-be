# Hằng số Nghiệp vụ AI Executor Module (AI Executor Constants)

Các hằng số dùng trong Admin AI APIs:

## 1. Trạng thái cấp quyền chính sách AI (AIPolicyGrantStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/subscription/aipolicygrantstatus`
- **Các giá trị hợp lệ:**
    - `active`: Chính sách AI hiện tại đang được áp dụng.
    - `future`: Chính sách AI được xếp hàng sẽ kích hoạt trong tương lai (khi gói cũ hết hạn).
    - `expired`: Chính sách AI đã hết hạn.

## 2. Chế độ thực thi kiểm soát chi phí AI (AIEnforcementMode)

- **Đường dẫn package:** `internal/shared/domain/constants/subscription/aienforcementmode`
- **Các giá trị hợp lệ:**
    - `strict`: Áp dụng chặt chẽ hạn mức chi phí AI tối đa của gói cước, chặn ngay khi vượt quá.
    - `observe_only`: Chỉ theo dõi chi phí AI phát sinh và không thực hiện chặn sử dụng.
    - `free_only`: Chỉ cho phép thực hiện các thao tác AI nằm trong luồng miễn phí (`free_route`).

## 3. Trạng thái log sự kiện AI (AIUsageEventStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/subscription/aiusageeventstatus`
- **Các giá trị hợp lệ:**
    - `reserved`: Số dư token đã được đặt trước trước khi gọi API AI (Preflight).
    - `in_flight`: Đang thực hiện xử lý gọi API AI.
    - `confirmed`: Xử lý AI thành công và đã ghi nhận lượng token tiêu thụ thực tế.
    - `released`: Hủy bỏ lượng token đặt trước (do xảy ra lỗi trước khi gọi AI).
    - `unknown_usage`: Lượng token tiêu thụ chưa được xác định rõ.
    - `expired_unverified`: Hết hạn đặt trước token mà không xác minh được kết quả.
