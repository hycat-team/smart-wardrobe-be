# 1. Về dự án

Dự án hiện tại tạm gọi là **Closy**, đồng thời đang được hiện thực dưới dạng backend của nền tảng **SmartWardrobe**.

Closy là website và ứng dụng quản lý tủ đồ thông minh, giúp người dùng số hóa tủ quần áo cá nhân, nhận gợi ý phối đồ bằng AI và tận dụng tốt hơn những món đồ đang sở hữu.

Core hiện tại của dự án là:

- Quản lý tủ đồ số cá nhân.
- Gợi ý phối đồ bằng AI.
- Tận dụng lại quần áo sẵn có.

Phiên bản hiện tại của hệ thống đã mở rộng hơn so với mô tả MVP ban đầu và hiện có thêm:

- AI Fashion Chatbot theo phiên trò chuyện.
- Cộng đồng bài đăng và bài đăng thanh lý.
- Quản lý gói hội viên, quota ngày và ví nội bộ.
- Thanh toán mua gói, nạp ví và webhook thanh toán.
- Công cụ quản trị dành cho admin.

Về dài hạn, nếu có đủ người dùng, Closy có thể mở rộng sang các hợp tác nhẹ với brand như voucher, campaign, affiliate, private deal hoặc loyalty engagement.

---

# 2. Target dự án

Target chính là **Gen Z tại Việt Nam**.

Đây là nhóm:

- Quan tâm đến thời trang và phong cách cá nhân.
- Bị ảnh hưởng bởi TikTok, Instagram và các xu hướng thời trang.
- Có nhu cầu mặc đẹp và thể hiện bản thân qua outfit.
- Sở hữu nhiều quần áo nhưng vẫn thường gặp tình trạng “không có gì để mặc”.
- Muốn phối đồ nhanh hơn và tận dụng tốt hơn tủ đồ hiện có.

---

# 3. Pain point chính

## 3.1. “Hôm nay mặc gì?”

Người dùng có nhiều quần áo nhưng vẫn khó chọn outfit, khó phối đồ mới hoặc quên mất những món đồ đang có.

Closy giải quyết vấn đề này bằng cách giúp người dùng:

- Quản lý tủ đồ rõ ràng hơn.
- Tìm lại các item bị bỏ quên.
- Nhận gợi ý phối đồ từ quần áo đang sở hữu.
- Tiết kiệm thời gian khi lựa chọn trang phục.

## 3.2. Tận dụng tủ đồ tốt hơn

Người dùng thường mua thêm quần áo mới trong khi nhiều món đồ cũ vẫn chưa được sử dụng hiệu quả.

Closy hướng tới việc giúp người dùng:

- Phối lại đồ cũ theo cách mới.
- Giảm tình trạng mặc lặp lại một vài outfit quen thuộc.
- Tăng tần suất sử dụng các item đang có.
- Hạn chế việc mua sắm không cần thiết.

---

# 4. Nhóm chức năng chính

## 4.1. AI Outfit Recommendation

AI gợi ý outfit dựa trên:

- Tủ đồ hiện có.
- Dịp sử dụng.
- Phong cách mong muốn.
- Thời tiết hoặc ngữ cảnh nếu có.

Người dùng có thể lưu lại các outfit đã phối.

### Phiên bản hiện tại

Phiên bản hiện tại của backend đã có API gợi ý phối đồ, đồng thời có module lưu outfit riêng để người dùng lưu hoặc cập nhật outfit đã chọn.

Kỳ vọng đầu ra hiện tại của recommendation cũng đã rõ cấu trúc hơn so với cách hiểu cũ về một outfit dạng danh sách phẳng. Ở mức DTO, kết quả hiện tại gồm:

- `title` cho bộ gợi ý
- `explanation` để giải thích vì sao bộ đồ phù hợp
- `items` là các nhóm item theo vai trò
- mỗi nhóm có `role`, một item `primary` và các `alternatives`

Điều này có nghĩa là sản phẩm hiện hướng tới một bộ gợi ý có khả năng giải thích và thay thế theo từng vai trò, thay vì chỉ trả về một phương án duy nhất.

### Thiết kế mục tiêu

Thiết kế mục tiêu trong các tài liệu cũ còn đi xa hơn, bao gồm local swap, partial re-roll, OOTD hằng ngày và các thuật toán kết hợp màu hoặc suy giảm vòng đời item sâu hơn. Những mô tả đó vẫn được giữ trong bộ docs như định hướng mở rộng.

## 4.2. Digital Wardrobe

Người dùng có thể thêm quần áo, phụ kiện vào tủ đồ số.

AI hỗ trợ nhận diện item và tạo metadata như:

- Loại trang phục.
- Màu sắc.
- Chất liệu.
- Mục đích sử dụng.

### Phiên bản hiện tại

Hiện hệ thống đã có thêm các capability thực tế ngoài mô tả gốc:

- batch upload nhiều item và xử lý nền
- clone item
- khởi tạo tủ đồ từ danh mục hệ thống
- phân loại thủ công
- tìm kiếm item

### Thiết kế mục tiêu

Các hướng như batch matrix crop cho phụ kiện, phân tích ảnh cơ thể sâu hơn hoặc các workflow AI nâng cao vẫn nên được giữ là mô tả mục tiêu hoặc mở rộng tiếp theo.

## 4.3. AI Fashion Chatbot

Chatbot hỗ trợ người dùng hỏi về thời trang, phối đồ và phong cách cá nhân.

### Phiên bản hiện tại

Phiên bản hiện tại đã có:

- tạo phiên chat
- lấy danh sách phiên chat
- lấy lịch sử tin nhắn
- lưu trữ phiên
- gửi tin nhắn và nhận phản hồi theo luồng stream

### Thiết kế mục tiêu

Các mô tả cũ như ReAct agent loop, tóm tắt ngữ cảnh theo ngưỡng tin nhắn hay các guardrail nâng cao vẫn được giữ lại như định hướng nghiệp vụ và kiến trúc mục tiêu, dù mức độ triển khai hiện tại có thể chưa trùng hoàn toàn.

## 4.4. Outfit Inspiration

Người dùng có thể xem hoặc chia sẻ outfit để tìm cảm hứng phối đồ.

Tính năng cộng đồng chỉ đóng vai trò hỗ trợ engagement, không phải core chính của dự án.

### Phiên bản hiện tại

Hiện phần cộng đồng đã tiến xa hơn mô tả ban đầu:

- có feed bài đăng công khai
- có like và comment
- có xem chi tiết bài đăng, bình luận và lượt thích
- có quản trị bài đăng, bình luận và post item ở phía admin

## 4.5. Sell Post

Người dùng có thể đăng thanh lý quần áo không còn sử dụng theo mô hình C2C.

Closy không trực tiếp xử lý giao dịch và không thu hoa hồng trong giai đoạn đầu.

### Phiên bản hiện tại

Hiện hệ thống đã có thêm các bước nghiệp vụ thật:

- tạo yêu cầu mua item
- người bán xem luồng chuyển nhượng liên quan đến bài đăng của mình
- đánh dấu đã bán cho người mua cụ thể
- người mua chấp nhận hoặc từ chối item đang chờ chuyển

### Thiết kế mục tiêu

Các mô tả sâu hơn về uy tín giao dịch, tối ưu local swap sau thanh lý hoặc các vòng đời giao dịch nâng cao vẫn được giữ là định hướng mở rộng.

## 4.6. Subscription, Billing và Wallet

Đây là nhóm chức năng đã được bổ sung rõ trong phiên bản hiện tại của hệ thống.

Người dùng hiện có thể:

- xem danh sách gói hội viên
- xem gói hiện tại
- xem quota ngày
- bật hoặc tắt tự động gia hạn
- xem ví nội bộ
- xem lịch sử biến động ví
- nạp tiền vào ví
- mua gói trực tiếp
- mua gói bằng ví

Hệ thống cũng đã có webhook thanh toán để xử lý hoàn tất giao dịch từ cổng thanh toán.

---

# 5. Định hướng kinh doanh

## Giai đoạn MVP

Closy tập trung vào:

- Quản lý tủ đồ số.
- AI gợi ý phối đồ.
- Kiểm chứng nhu cầu sử dụng thật của người dùng.

Mục tiêu chính là có user, hiểu hành vi tủ đồ và kiểm chứng xem người dùng có sẵn sàng tạo tủ đồ số hay không.

## Mô hình freemium

Closy có thể có gói free và premium.

Tuy nhiên, premium không nên khóa các tính năng cốt lõi. Hai gói vẫn có cùng nhóm feature chính, nhưng khác nhau ở quota sử dụng AI.

Gói premium chủ yếu dùng để:

- Tăng số lượt AI gợi ý outfit.
- Tăng số lượt AI phân tích item.
- Giảm giới hạn sử dụng.
- Hỗ trợ chi phí vận hành AI.

### Phiên bản hiện tại

Hiện hệ thống đã có cấu trúc gói hội viên, quota ngày, auto-renew, ví nội bộ và các luồng mua gói, nên mô hình freemium không còn chỉ là ý tưởng kinh doanh mà đã có nền thực thi ở backend.

## Định hướng dài hạn

Sau khi có đủ user base, Closy có thể hợp tác với brand theo hướng nhẹ và ít vận hành như:

- Voucher.
- Private deal.
- Sponsored campaign.
- Affiliate.
- Styling challenge.
- Loyalty engagement.

Closy không nên bắt đầu bằng việc làm chăm sóc khách hàng, đổi trả, bảo hành hoặc vận chuyển thay brand, vì những phần đó dễ làm dự án lệch khỏi core là tủ đồ cá nhân.
