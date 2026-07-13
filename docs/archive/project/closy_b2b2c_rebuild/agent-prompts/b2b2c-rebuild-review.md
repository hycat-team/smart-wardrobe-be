# Prompt cho agent đánh giá và lập kế hoạch refactor Closy

Bạn là senior backend architect kiêm code reviewer cho dự án Closy của team HYCAT. Hãy đọc toàn bộ codebase, migration, database schema, domain model, repository, service/usecase, handler/API, DTO, seed data, middleware, permission, test và tài liệu liên quan trước khi kết luận.

Hãy trả lời hoàn toàn bằng tiếng Việt.

## Bối cảnh sản phẩm mới

Closy đang chuyển từ mô hình B2C wardrobe app thuần tuý sang mô hình B2B2C.

Định vị mới:

> Closy là Fashion Loyalty & Co-creation Platform powered by AI wardrobe data.

Nghĩa là:

- Người dùng vẫn dùng Closy để số hoá tủ đồ, quản lý item, lưu outfit và nhận gợi ý phối đồ bằng AI.
- B2C là lớp tạo người dùng, engagement và dữ liệu tủ đồ.
- B2B là nguồn thu chính, nơi fashion brand trả tiền để quản lý khách hàng trung thành, chăm sóc sau bán hàng, tạo chiến dịch ưu đãi, và test digital sample trước khi sản xuất.
- Closy không nên trở thành mini Shopee.
- Closy không nên lấy hoa hồng giao dịch làm nguồn thu chính.
- Closy không nên tập trung vào social community feed hoặc C2C resale trong MVP mới.

## Trạng thái làm việc

Team đã backup dữ liệu local và đang làm trên một nhánh hoàn toàn mới. Vì vậy không cần đề xuất feature flag, compatibility layer phức tạp, hoặc phương án giữ song song schema cũ. Nếu nhánh mới không ổn thì team có thể quay lại `main`.

Hãy đánh giá mạnh tay những phần nên xoá, nên bỏ khỏi MVP, nên rewrite hoặc nên rebuild từ đầu.

## Nhiệm vụ của bạn

Hãy đánh giá codebase hiện tại và lập kế hoạch thay đổi để Closy phù hợp với mô hình mới.

Tập trung trả lời các nhóm câu hỏi sau.

## Phần cần giữ

Hãy xác định những phần nên giữ vì vẫn phù hợp với mô hình mới, ví dụ:

- Auth/user
- Wardrobe item
- Category
- Outfit
- Outfit item
- AI outfit recommendation
- AI chat/conversation
- AI quota/cost control
- B2C subscription/payment nếu vẫn cần làm nguồn thu phụ

Với mỗi phần, hãy nói rõ:

- File/package/module liên quan
- Bảng database liên quan
- API liên quan
- Lý do nên giữ
- Có cần chỉnh sửa gì không

## Phần cần xoá khỏi MVP mới

Hãy tìm và đánh giá tất cả phần đang kéo Closy sang hướng social marketplace, community feed, resale hoặc C2C transfer.

Đặc biệt kiểm tra các khái niệm như:

- Post cộng đồng tự do
- Like/comment kiểu mạng xã hội
- Hotness ranking
- Post item có giá bán
- Buyer user
- Transfer request
- Sold status
- Item resale
- Contact info để mua bán
- Feed không gắn với brand/campaign

Nếu tồn tại, hãy đề xuất rõ:

- Nên xoá hoàn toàn khỏi nhánh mới hay chỉ đưa vào backlog
- Bảng nào cần drop
- Model/entity nào cần xoá
- Repository/usecase/handler nào cần xoá
- Route/API nào cần xoá
- Test nào cần xoá/cập nhật
- Seed nào cần xoá/cập nhật
- Lý do xoá dưới góc nhìn mô hình kinh doanh mới

## Phần nên rebuild hoặc đổi nghĩa

Nếu có các module như `posts`, `comments`, `likes`, hãy đánh giá xem có nên rebuild thành:

- Brand post
- Campaign post
- Campaign interaction
- Brand announcement
- Sample announcement
- Outfit challenge post

Không được chỉ rename cơ học nếu semantic cũ vẫn là social/resale. Hãy chỉ ra field nào không còn phù hợp, ví dụ:

- total_price
- contact_info
- hotness_dirty_at
- buyer_user_id
- transfer_state
- sold_at

Với mỗi module rebuild, hãy đề xuất entity mới, field mới, API mới và luồng nghiệp vụ mới.

## Phần cần thêm cho mô hình mới

Hãy đề xuất thiết kế domain/database/API cho các module sau.

### Brand core

Cần có ít nhất:

- brands
- brand_members
- brand_customers

Hãy đề xuất field, constraint, index, relationship, role/permission và API.

### Loyalty

Cần có ít nhất:

- loyalty_programs
- loyalty_tiers
- loyalty_accounts
- loyalty_point_transactions
- brand_benefits
- benefit_redemptions

Hãy thiết kế cách cộng/trừ điểm từ:

- mua hàng offline do brand nhập
- tương tác campaign
- hoàn thành task
- thử digital sample
- manual adjustment
- redeem benefit

### Campaign engine

Cần có ít nhất:

- brand_campaigns
- campaign_posts
- campaign_participants
- campaign_interactions
- campaign_rewards

Hãy thiết kế campaign type, status, visibility, targeting rule, reward rule và flow user tham gia campaign.

### Customer service hub

Cần có ít nhất:

- support_tickets
- support_ticket_messages
- return_exchange_requests nếu phù hợp MVP

Hãy thiết kế flow:

- user tạo ticket với brand
- brand staff nhận và phản hồi
- ticket đổi/trả lưu external order reference vì Closy không phải marketplace xử lý đơn hàng

### Digital Sample Lab

Cần có ít nhất:

- digital_samples
- digital_sample_assets
- digital_sample_variants
- sample_test_participants
- sample_outfit_trials
- sample_votes
- sample_feedback

Hãy thiết kế flow:

- brand tạo digital sample
- brand mời khách hàng thân thiết tham gia test
- user thử phối sample với tủ đồ/outfit
- user vote hoặc feedback
- brand xem insight tổng hợp

### Consent/privacy

Hãy đánh giá cần thêm bảng hoặc logic nào để user kiểm soát việc chia sẻ dữ liệu với brand, ví dụ:

- user_brand_consents
- consent type
- revoke consent
- chỉ cung cấp insight tổng hợp, không định danh cá nhân nếu chưa có consent

## Yêu cầu về database

Hãy đề xuất schema mới theo hướng PostgreSQL.

Với mỗi bảng mới, hãy nêu:

- Mục đích bảng
- Field chính
- Primary key
- Foreign key
- Unique constraint
- Index quan trọng
- Enum/status đề xuất
- Quan hệ với bảng hiện có
- Bảng nào nên drop hoặc không migrate sang nhánh mới

Không cần viết full SQL migration ngay nếu chưa đủ thông tin, nhưng phải đủ chi tiết để backend developer có thể tạo migration.

## Yêu cầu về code architecture

Hãy đánh giá code hiện tại đang tổ chức theo package/layer nào và đề xuất thay đổi cụ thể.

Với mỗi domain mới, hãy đề xuất:

- Entity/model
- Repository interface
- Usecase/service
- Handler/controller
- DTO/request/response
- Validation
- Permission check
- Transaction boundary
- Event/outbox nếu cần
- Test cần viết

Nếu code hiện tại dùng Go/Gin/Wire/Goose/Postgres thì hãy bám theo kiến trúc đó, không đề xuất đổi stack nếu không cần thiết.

## Yêu cầu về API

Hãy đề xuất API route mới cho:

- User-facing brand/loyalty/campaign/sample/support
- Brand portal
- Admin nếu cần

Với mỗi nhóm API, hãy nói:

- Route
- Method
- Actor được phép gọi
- Request chính
- Response chính
- Usecase xử lý
- Bảng ảnh hưởng

## Yêu cầu về MVP

Hãy chia thành 3 mức:

### Must-have cho MVP mới

Chỉ gồm những thứ cần để chứng minh mô hình B2B2C:

- Brand profile
- Brand member
- Brand customer
- Loyalty account
- Manual point transaction
- Campaign cơ bản
- Support ticket cơ bản
- Digital sample cơ bản
- User thử sample và feedback
- Brand xem insight đơn giản

### Should-have

Các phần nên có nếu còn thời gian.

### Later/backlog

Các phần chưa nên làm ngay, ví dụ:

- Marketplace
- C2C resale
- Social feed tự do
- Hotness ranking
- Logistics
- Payment giữa brand và user
- Creator membership kiểu Patreon

## Yêu cầu về output

Hãy trả lời theo cấu trúc sau:

```md
# Đánh giá tổng quan

# Bản đồ module hiện tại

# Những phần nên giữ

# Những phần nên xoá khỏi MVP mới

# Những phần nên rebuild hoặc đổi nghĩa

# Thiết kế domain mới đề xuất

# Thiết kế database mới đề xuất

# API mới đề xuất

# Kế hoạch refactor theo phase

# Danh sách file/package cần kiểm tra hoặc thay đổi

# Rủi ro kỹ thuật và nghiệp vụ

# MVP scope cuối cùng

# Câu hỏi còn thiếu thông tin
```

## Quy tắc trả lời

- Không bịa đặt. Nếu chưa chắc vì chưa thấy file hoặc chưa đọc đủ code, hãy nói rõ là chưa chắc.
- Không trả lời chung chung. Phải gắn với module, bảng, file hoặc API cụ thể khi có thể.
- Không đề xuất feature flag vì team đang làm trên nhánh mới và đã backup dữ liệu.
- Không cố giữ schema cũ nếu nó làm sai hướng sản phẩm.
- Không biến Closy thành marketplace.
- Không biến Closy thành mạng xã hội thời trang trong MVP.
- Không xoá lõi wardrobe/AI nếu không có lý do kỹ thuật rất mạnh.
- Nếu có tạo code hoặc migration, comment trong code không được đánh số thứ tự, không dùng icon trong comment.
- Hãy ưu tiên câu trả lời thực dụng để backend developer có thể bắt tay vào refactor.
