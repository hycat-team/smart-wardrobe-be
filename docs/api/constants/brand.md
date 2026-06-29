# Hằng số Nghiệp vụ Brand (Brand Constants)

Các hằng số dùng trong APIs liên quan đến Thương hiệu, Thành viên, Loyalty và Đặc quyền:

## 1. Trạng thái của Brand (BrandStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/brandstatus`
- **Các giá trị hợp lệ:**
    - `pending_review`: Đang chờ quản trị viên duyệt yêu cầu tạo brand.
    - `active`: Thương hiệu đang hoạt động bình thường trên Closy.
    - `suspended`: Thương hiệu bị tạm đình chỉ hoạt động.
    - `archived`: Thương hiệu đã bị đưa vào lưu trữ (ngừng hoạt động).

## 2. Vai trò thành viên trong Brand (BrandMemberRole)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/brandmemberrole`
- **Các giá trị hợp lệ:**
    - `owner`: Chủ sở hữu nhãn hàng (quyền cao nhất).
    - `staff`: Nhân viên truyền thông.

## 3. Trạng thái thành viên trong Brand (BrandMemberStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/brandmemberstatus`
- **Các giá trị hợp lệ:**
    - `active`: Thành viên đang hoạt động.
    - `invited`: Được mời nhưng chưa kích hoạt tài khoản thành viên.
    - `disabled`: Thành viên đã bị vô hiệu hóa quyền truy cập brand-portal.

## 4. Trạng thái khách hàng thân thiết (BrandCustomerStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/brandcustomerstatus`
- **Các giá trị hợp lệ:**
    - `active`: Khách hàng đang tham gia chương trình loyalty và tích lũy điểm bình thường.
    - `blocked`: Khách hàng bị thương hiệu chặn tham gia loyalty.
    - `left`: Khách hàng đã hủy tham gia chương trình loyalty của thương hiệu.

## 5. Nguồn gốc tham gia loyalty (BrandCustomerJoinedSource)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/brandcustomerjoinedsource`
- **Các giá trị hợp lệ:**
    - `self_join`: Người dùng tự tham gia qua ứng dụng.
    - `offline_purchase`: Tham gia tự động khi mua sắm offline và tích điểm lần đầu.
    - `import`: Nhập danh sách khách hàng từ hệ thống cũ của thương hiệu.

## 6. Loại đặc quyền của Brand (BenefitType)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/benefit/benefittype`
- **Các giá trị hợp lệ:**
    - `voucher`: Mã giảm giá liên kết.
    - `discount`: Chiết khấu trực tiếp.
    - `gift`: Quà tặng hiện vật.
    - `free_shipping`: Miễn phí vận chuyển.
    - `early_access`: Được mua sớm bộ sưu tập mới.
    - `feature_access`: Được mở khóa tính năng đặc quyền trên ứng dụng.

## 7. Phương thức mở khóa đặc quyền (BenefitUnlockType)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/benefit/benefitunlocktype`
- **Các giá trị hợp lệ:**
    - `tier_privilege`: Đặc quyền tự động mở khóa theo hạng thành viên (Loyalty Tiers).
    - `point_redemption`: Cần dùng điểm loyalty tích lũy để quy đổi đặc quyền.
    - `manual_grant`: Được thương hiệu trao tặng thủ công.

## 8. Mã đặc quyền hệ thống (BenefitFeatureCode)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/benefit/benefitfeaturecode`
- **Các giá trị hợp lệ:**
    - `sample_mix_access`: Cho phép thử đồ mẫu thiết kế ảo trong Digital Sample Lab.
    - `brand_item_recommendation`: Nhận gợi ý phối đồ AI ưu tiên sản phẩm của nhãn hàng.
    - `priority_brand_chat`: Kênh chat hỗ trợ được ưu tiên phản hồi sớm.

## 9. Trạng thái quy đổi đặc quyền (BenefitRedemptionStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/benefit/benefitredemptionstatus`
- **Các giá trị hợp lệ:**
    - `pending`: Đang chờ xử lý quy đổi (nhận quà hiện vật).
    - `redeemed`: Đã đổi quà thành công nhưng chưa sử dụng.
    - `used`: Đã sử dụng quà tặng/đặc quyền.
    - `cancelled`: Lượt đổi đặc quyền bị hủy.
    - `expired`: Đặc quyền hết hạn sử dụng.

## 10. Trạng thái vật phẩm trong Digital Sample Lab (BrandItemStatus)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/branditem/branditemstatus`
- **Các giá trị hợp lệ:**
    - `draft`: Bản nháp thiết kế, chưa hiển thị cho người dùng.
    - `active`: Đang công khai hiển thị để thử đồ và nhận vote/feedback.
    - `archived`: Đã lưu trữ (kết thúc đợt khảo sát ý kiến).

## 11. Phân loại vật phẩm trong Digital Sample Lab (BrandItemType)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/branditem/branditemtype`
- **Các giá trị hợp lệ:**
    - `product`: Sản phẩm thương mại có sẵn trên thị trường.
    - `sample`: Bản vẽ phác thảo / Mẫu thiết kế ảo 3D cần khảo sát ý kiến.

## 12. Loại vote sản phẩm mẫu (VoteType)

- **Đường dẫn package:** `internal/shared/domain/constants/brand/branditem/votetype`
- **Các giá trị hợp lệ:**
    - `like`: Thích mẫu thiết kế.
    - `dislike`: Không thích mẫu thiết kế.
    - `would_buy`: Sẵn sàng mua nếu sản phẩm được sản xuất.
    - `not_interested`: Không quan tâm.
