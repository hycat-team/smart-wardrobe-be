# Hằng số Nghiệp vụ Subscription & Billing (Subscription Constants)

Các hằng số dùng trong APIs liên quan đến Gói dịch vụ, Giao dịch ví, Thanh toán và Chính sách AI:

## 1. Trạng thái Giao dịch nạp tiền (DepositStatus)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/depositstatus`
*   **Các giá trị hợp lệ:**
    *   `creating`: Đang khởi tạo link thanh toán từ cổng trung gian (PayOS, Momo...).
    *   `pending`: Link thanh toán đã được tạo, đang chờ người dùng chuyển khoản.
    *   `success`: Nạp tiền thành công, số dư ví đã được cộng.
    *   `cancelled`: Giao dịch bị người dùng hủy bỏ.
    *   `expired`: Link thanh toán hết hạn mà chưa nhận được tiền.
    *   `failed`: Thanh toán bị lỗi.
    *   `creation_failed`: Khởi tạo link thanh toán không thành công.
    *   `reconciliation_required`: Giao dịch phát sinh lỗi đối soát cần kiểm tra thủ công.

## 2. Loại Giao dịch nạp tiền (DepositTransactionType)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/deposittransactiontype`
*   **Các giá trị hợp lệ:**
    *   `direct_purchase`: Nạp tiền và mua thẳng gói Premium.
    *   `wallet_topup`: Nạp tiền vào số dư ví cá nhân.

## 3. Phân loại gói cước (PlanKind)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/plankind`
*   **Các giá trị hợp lệ:**
    *   `free`: Gói miễn phí mặc định khi đăng ký tài khoản.
    *   `premium`: Gói trả phí nâng cấp giới hạn tủ đồ và lượt gọi gợi ý phối đồ AI.

## 4. Chế độ thực thi kiểm soát chi phí AI (AIEnforcementMode)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/aienforcementmode`
*   **Các giá trị hợp lệ:**
    *   `strict`: Áp dụng chặt chẽ hạn mức chi phí AI tối đa của gói cước, chặn ngay khi vượt quá.
    *   `observe_only`: Chỉ theo dõi chi phí AI phát sinh và không thực hiện chặn sử dụng.
    *   `free_only`: Chỉ cho phép thực hiện các thao tác AI nằm trong luồng miễn phí (`free_route`).

## 5. Trạng thái log sự kiện AI (AIUsageEventStatus)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/aiusageeventstatus`
*   **Các giá trị hợp lệ:**
    *   `reserved`: Số dư token đã được đặt trước trước khi gọi API AI (Preflight).
    *   `in_flight`: Đang thực hiện xử lý gọi API AI.
    *   `confirmed`: Xử lý AI thành công và đã ghi nhận lượng token tiêu thụ thực tế.
    *   `released`: Hủy bỏ lượng token đặt trước (do xảy ra lỗi trước khi gọi AI).
    *   `unknown_usage`: Lượng token tiêu thụ chưa được xác định rõ.
    *   `expired_unverified`: Hết hạn đặt trước token mà không xác minh được kết quả.

## 6. Trạng thái cấp quyền chính sách AI (AIPolicyGrantStatus)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/aipolicygrantstatus`
*   **Các giá trị hợp lệ:**
    *   `active`: Chính sách AI hiện tại đang được áp dụng.
    *   `future`: Chính sách AI được xếp hàng sẽ kích hoạt trong tương lai (khi gói cũ hết hạn).
    *   `closed`: Chính sách cũ đã hết hạn hoặc bị thay thế.

## 7. Trạng thái gia hạn tự động (SubscriptionRenewalStatus)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/subscriptionrenewalstatus`
*   **Các giá trị hợp lệ:**
    *   `processing`: Đang trong quá trình tự động trừ tiền ví để gia hạn.
    *   `succeeded`: Gia hạn thành công.
    *   `skipped`: Bỏ qua lượt gia hạn (ví dụ: tài khoản đã hủy đăng ký trước đó).
    *   `failed`: Gia hạn thất bại (thường do ví không đủ số dư, sau đó sẽ bị hạ cấp xuống gói Free).
    *   `renewed`: Đã gia hạn thành công chu kỳ mới.
    *   `downgraded`: Bị hạ cấp xuống gói thấp hơn do gia hạn thất bại.

## 8. Phân loại biến động ví (WalletStatementType)
*   **Đường dẫn package:** `internal/shared/domain/constants/subscription/walletstatementtype`
*   **Các giá trị hợp lệ:**
    *   `topup`: Biến động cộng tiền nạp ví thành công.
    *   `subscription_purchase`: Trừ tiền mua gói Premium lần đầu.
    *   `subscription_renewal`: Trừ tiền tự động gia hạn gói hàng tháng.
    *   `lower_tier_payment_credit`: Hoàn tiền chênh lệch khi hạ cấp gói cước.
    *   `same_lifetime_payment_credit`: Hoàn tiền nâng cấp gói đồng hạng.
