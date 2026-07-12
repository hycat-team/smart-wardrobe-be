# Closy B2B2C Rebuild Plan

## Mục đích tài liệu

Tài liệu này dùng để định hướng lại Closy trên một nhánh phát triển mới sau khi nhóm đã backup dữ liệu local và không cần triển khai cơ chế bật/tắt tính năng cho schema cũ. Mục tiêu là giúp team HYCAT thống nhất lại sản phẩm, mô hình kinh doanh, phạm vi MVP, cấu trúc database và hướng refactor code để Closy phù hợp với mô hình mới.

Hướng mới không xem Closy là một ứng dụng phối đồ B2C thuần tuý, cũng không xem Closy là mini Shopee hoặc nền tảng mua bán lại đồ cũ. Hướng mới là:

> Closy là nền tảng B2B2C cho ngành thời trang, trong đó người dùng sử dụng tủ đồ số và AI phối đồ để tạo engagement, còn các fashion brand trả tiền để quản lý khách hàng trung thành, chăm sóc sau bán hàng, chạy chiến dịch ưu đãi và thử nghiệm digital sample trước khi sản xuất.

Câu chốt ngắn gọn:

> B2C creates engagement and wardrobe data. B2B captures value through loyalty, customer service, campaigns, and product insight.

---

# Tóm tắt quyết định chiến lược

## Closy không nên trở thành mini Shopee

Closy không nên lấy trọng tâm là đăng bán sản phẩm, giỏ hàng, thanh toán, vận chuyển, quản lý đơn hàng hoặc ăn hoa hồng theo giao dịch. Nếu đi theo hướng đó, Closy sẽ bị kéo thành marketplace và phải cạnh tranh trực tiếp với Shopee, TikTok Shop, Lazada hoặc các kênh bán hàng hiện có của brand.

Trong mô hình mới, Closy không kiếm tiền chính từ giao dịch mua bán. Closy kiếm tiền từ việc cung cấp nền tảng giúp brand tăng retention, quản lý khách hàng thân thiết, cá nhân hoá ưu đãi, xử lý yêu cầu khách hàng và thu thập insight sản phẩm.

## Closy không nên trở thành mạng xã hội thời trang ở MVP

Các tính năng post cộng đồng, like, comment, hotness, đăng bán item, transfer request hoặc resale có thể tạo cảm giác sản phẩm nhiều tính năng, nhưng dễ làm Closy bị loãng. Nếu nguồn thu chính là B2B loyalty, thì community feed không nên là trọng tâm MVP.

Nếu vẫn muốn giữ cơ chế bài đăng, nên chuyển nó thành:

- Brand post
- Campaign post
- Sample announcement
- Loyalty benefit announcement
- Outfit challenge post

Không nên để nó là social feed tự do giữa người dùng với người dùng trong giai đoạn MVP mới.

## Closy nên định vị là Fashion Loyalty & Co-creation Platform

Định vị đề xuất:

> Closy is a Fashion Loyalty & Co-creation Platform powered by AI wardrobe data.

Bản tiếng Việt:

> Closy là nền tảng loyalty và co-creation cho ngành thời trang, sử dụng dữ liệu tủ đồ số và AI phối đồ để giúp brand quản lý khách hàng thân thiết, chăm sóc sau bán hàng, tạo chiến dịch ưu đãi cá nhân hoá và thử nghiệm mẫu sản phẩm digital trước khi sản xuất.

---

# Bài toán của từng bên

## Người dùng cá nhân

Người dùng vẫn vào Closy vì các nhu cầu B2C ban đầu:

- Không biết hôm nay mặc gì
- Có nhiều quần áo nhưng khó phối
- Muốn quản lý tủ đồ cá nhân
- Muốn tận dụng quần áo đang có
- Muốn lưu outfit và nhận gợi ý AI
- Muốn nhận benefit từ brand yêu thích
- Muốn được xem trước hoặc thử phối sản phẩm mới của brand

Giá trị người dùng nhận được:

- Tủ đồ số cá nhân
- AI outfit recommendation
- AI fashion chat
- Lưu outfit
- Gợi ý theo thời tiết, dịp sử dụng, màu sắc, phong cách
- Ưu đãi cá nhân hoá từ brand
- Tích điểm loyalty
- Hỗ trợ đổi trả/chăm sóc sau mua hàng trong một nơi
- Thử digital sample với tủ đồ của chính mình

## Fashion brand

Brand tham gia Closy vì các vấn đề hiện tại:

- Chưa có hệ thống quản lý khách hàng trung thành riêng
- Dữ liệu khách hàng phân tán ở cửa hàng offline, Facebook, Instagram, TikTok Shop, Shopee, Zalo hoặc file Excel
- Việc tích điểm thường chỉ dựa trên số điện thoại hoặc lịch sử mua hàng
- Chăm sóc khách hàng và đổi trả thường xử lý thủ công qua inbox
- Khó cá nhân hoá ưu đãi theo phong cách thật của khách
- Khó biết khách đã sử dụng item đã mua như thế nào
- Khó test ý tưởng sản phẩm trước khi sản xuất
- Khó đo mức độ quan tâm của khách hàng thân thiết với sample mới

Giá trị brand nhận được:

- Brand portal để quản lý khách hàng thân thiết
- Loyalty point và membership tier
- Campaign ưu đãi
- Chat/support ticket
- Return/exchange request management
- Digital Sample Lab
- Insight dashboard tổng hợp
- Tín hiệu demand trước khi sản xuất sản phẩm mới

## Closy

Closy hưởng lợi vì:

- B2C giúp tạo người dùng và engagement
- Tủ đồ số tạo dữ liệu khác biệt so với CRM thông thường
- Brand là bên có động lực trả tiền rõ hơn user cá nhân
- Doanh thu có thể đến từ SaaS fee, campaign fee, sample lab fee và insight report
- Sản phẩm giữ được lõi AI wardrobe nhưng có câu chuyện kinh tế mạnh hơn

---

# Mô hình doanh thu đề xuất

## Brand Subscription Fee

Brand trả phí hằng tháng để sử dụng nền tảng Closy.

Có thể chia gói theo:

- Số lượng khách hàng được quản lý
- Số lượng nhân sự brand được cấp quyền
- Số lượng campaign mỗi tháng
- Quyền sử dụng customer support hub
- Quyền sử dụng Digital Sample Lab
- Quyền xem dashboard/insight

Ví dụ gói:

| Gói | Đối tượng | Tính năng chính |
|---|---|---|
| Brand Basic | Local brand nhỏ | Quản lý khách hàng, điểm loyalty, benefit cơ bản |
| Brand Pro | Brand đang tăng trưởng | Campaign, support ticket, dashboard, phân hạng khách |
| Brand Plus | Brand cần insight sản phẩm | Digital Sample Lab, sample testing, insight report nâng cao |

## Campaign Fee

Brand trả phí theo từng chiến dịch.

Ví dụ:

- Outfit challenge
- Member-only voucher
- New collection preview
- Birthday campaign
- Style this item campaign
- Try digital sample campaign

## Digital Sample Lab Fee

Brand trả phí để đưa sample digital lên Closy và mời khách hàng thân thiết thử phối.

Có thể tính theo:

- Số lượng sample
- Thời gian chạy test
- Số lượng khách được mời
- Số lượng feedback cần thu thập
- Báo cáo insight sau chiến dịch

## Insight Report Fee

Closy cung cấp báo cáo tổng hợp, không định danh cá nhân, cho brand.

Ví dụ insight:

- Item nào được phối nhiều nhất
- Màu nào được quan tâm nhất
- Sample nào có tín hiệu demand tốt nhất
- Khách hàng hay phối sản phẩm với loại item nào
- Occasion nào phù hợp với sample
- Segment khách hàng nào tương tác mạnh nhất

## B2C Premium Subscription

B2C Premium vẫn có thể giữ như nguồn thu phụ.

Premium user có thể được:

- Tăng giới hạn wardrobe item
- Tăng giới hạn outfit
- Tăng quota AI outfit recommendation
- Tăng quota AI chat
- Mở khoá tính năng nâng cao

Tuy nhiên, trong mô hình mới, Premium không nên là nguồn thu chính trong pitch.

---

# Đánh giá schema hiện tại

Dựa trên schema hiện tại, hệ thống có khoảng 32 bảng chính:

- `users`
- `refresh_tokens`
- `categories`
- `wardrobe_items`
- `outfits`
- `outfit_items`
- `user_style_profiles`
- `conversational_contexts`
- `messages`
- `subscription_plans`
- `user_subscriptions`
- `user_subscription_events`
- `deposit_transactions`
- `provider_webhook_inbox`
- `provider_payment_events`
- `user_wallets`
- `wallet_statements`
- `subscription_renewal_attempts`
- `user_daily_quotas`
- `ai_cost_policies`
- `ai_cost_policy_operations`
- `user_ai_policy_grants`
- `ai_usage_period_ledgers`
- `ai_usage_events`
- `posts`
- `post_media`
- `comments`
- `likes`
- `post_items`
- `transfer_requests`
- `post_score_snapshots`
- `goose_db_version`

## Phần nên giữ

Các nhóm bảng sau vẫn phù hợp với mô hình mới.

### User/Auth

- `users`
- `refresh_tokens`

Cần giữ. Tuy nhiên nên xem lại `role_slug` để hỗ trợ tốt hơn cho các vai trò mới như customer, brand owner, brand staff, admin.

### Wardrobe core

- `categories`
- `wardrobe_items`
- `outfits`
- `outfit_items`
- `user_style_profiles`

Cần giữ. Đây là lõi tạo khác biệt cho Closy. Brand loyalty của Closy không giống CRM thông thường vì nó dựa trên tủ đồ thật và hành vi phối đồ của user.

Nên bổ sung về sau:

- Brand source của item nếu item đến từ một brand cụ thể
- External product reference nếu item được mua ở brand đối tác
- Consent/visibility để user kiểm soát dữ liệu nào được dùng cho brand insight

### AI usage và AI cost control

- `ai_cost_policies`
- `ai_cost_policy_operations`
- `user_ai_policy_grants`
- `ai_usage_period_ledgers`
- `ai_usage_events`
- `user_daily_quotas`
- `conversational_contexts`
- `messages`

Nên giữ nếu hệ thống AI đã hoạt động. Đây là phần phục vụ cả B2C và B2B. Digital Sample Lab cũng có thể dùng AI để gợi ý sample phù hợp với tủ đồ người dùng.

### Subscription/payment B2C

- `subscription_plans`
- `user_subscriptions`
- `user_subscription_events`
- `deposit_transactions`
- `provider_webhook_inbox`
- `provider_payment_events`
- `subscription_renewal_attempts`
- `user_wallets`
- `wallet_statements`

Có thể giữ nếu vẫn cần B2C Premium. Tuy nhiên trong mô hình mới, cần bổ sung nhóm billing riêng cho brand hoặc tái sử dụng một phần payment hiện có sau khi thiết kế lại.

## Phần nên xoá khỏi MVP hoặc đưa ra khỏi core

Các bảng sau đang kéo sản phẩm sang hướng social/resale/marketplace:

- `posts`
- `post_media`
- `comments`
- `likes`
- `post_items`
- `transfer_requests`
- `post_score_snapshots`

Nếu đi theo mô hình B2B loyalty, không nên giữ nguyên ý nghĩa hiện tại của cụm này.

### Vì sao `post_items` và `transfer_requests` nên xoá khỏi MVP mới

`post_items` có các field như `price`, `item_condition`, `buyer_user_id`, `transfer_state`, `sold_at`, `declined_at`. `transfer_requests` có `buyer_id` và trạng thái request. Hai bảng này thể hiện rất rõ hướng mua bán/chuyển nhượng giữa người dùng.

Điều này làm Closy bị hiểu thành C2C resale hoặc marketplace. Đây không phải nguồn thu chính của mô hình mới và dễ kéo theo các câu hỏi khó:

- Ai chịu trách nhiệm giao dịch?
- Ai xử lý tranh chấp?
- Ai xử lý đổi trả?
- Ai quản lý chất lượng hàng?
- Closy khác gì sàn mua bán đồ cũ?

Khuyến nghị: xoá khỏi MVP mới hoặc chuyển vào backlog dài hạn, không pitch.

### Vì sao `posts`, `comments`, `likes` không nên giữ nguyên

Nếu giữ dưới dạng post cộng đồng, hệ thống sẽ giống social network. Social feed cần moderation, report, anti-spam, ranking, policy và tăng complexity không cần thiết.

Khuyến nghị:

- Không giữ community post tự do trong MVP mới
- Nếu cần bài đăng, đổi thành `brand_posts` hoặc `campaign_posts`
- Like/comment nên trở thành interaction signal cho campaign/loyalty, không phải mạng xã hội tự do
- `post_score_snapshots`/hotness nên bỏ hoặc chuyển thành campaign performance metric đơn giản

---

# Kiến trúc domain mới đề xuất

## Identity & Access

Mục tiêu: hỗ trợ cả user cá nhân và người dùng thuộc brand.

Các vai trò đề xuất:

- `CUSTOMER`: người dùng cá nhân
- `BRAND_OWNER`: chủ brand hoặc tài khoản quản trị cao nhất của brand
- `BRAND_STAFF`: nhân sự brand xử lý campaign/support
- `ADMIN`: quản trị hệ thống Closy

Không nên chỉ dựa vào `users.role_slug` cho toàn bộ phân quyền brand, vì một user có thể là staff của nhiều brand hoặc vừa là customer vừa là brand staff.

Nên có bảng membership riêng:

```sql
brands
brand_members
```

Trong đó `brand_members` quyết định user nào thuộc brand nào và có quyền gì.

## Wardrobe & Outfit

Giữ lõi cũ:

- Wardrobe item
- Category
- Outfit
- Outfit item
- User style profile
- AI recommendation

Bổ sung quan hệ với brand khi cần:

```sql
brand_products
user_brand_items
```

Tuy nhiên ở MVP đầu, không nhất thiết phải làm đầy đủ product catalog như e-commerce. Chỉ cần đủ để biết một item có liên quan đến brand nào hoặc campaign nào.

## Brand CRM

Đây là domain mới quan trọng nhất.

Mục tiêu: brand quản lý danh sách khách hàng thân thiết và lịch sử tương tác.

Các thực thể chính:

```sql
brands
brand_members
brand_customers
brand_customer_notes
brand_customer_segments
brand_customer_segment_members
```

Ý nghĩa:

- `brands`: thông tin brand
- `brand_members`: nhân sự của brand
- `brand_customers`: user nào là khách hàng của brand nào
- `brand_customer_notes`: ghi chú nội bộ của brand về khách hàng
- `brand_customer_segments`: nhóm khách hàng
- `brand_customer_segment_members`: user thuộc segment nào

## Loyalty

Mục tiêu: quản lý điểm, hạng thành viên và benefit của từng brand.

Các thực thể chính:

```sql
loyalty_programs
loyalty_tiers
loyalty_accounts
loyalty_point_transactions
brand_benefits
benefit_redemptions
```

Ý nghĩa:

- `loyalty_programs`: chương trình loyalty của brand
- `loyalty_tiers`: hạng thành viên như Silver, Gold, VIP
- `loyalty_accounts`: tài khoản điểm của một user trong một brand
- `loyalty_point_transactions`: lịch sử cộng/trừ điểm
- `brand_benefits`: quyền lợi hoặc ưu đãi
- `benefit_redemptions`: lịch sử user nhận/dùng benefit

Nguồn cộng điểm có thể là:

- Mua hàng offline được brand nhập vào
- Nhập số điện thoại tại cửa hàng
- Tương tác campaign
- Like/comment brand post
- Tham gia outfit challenge
- Thử digital sample
- Feedback sản phẩm
- Manual adjustment từ staff brand

## Campaign Engine

Mục tiêu: brand tạo chiến dịch ưu đãi hoặc tương tác.

Các thực thể chính:

```sql
brand_campaigns
campaign_posts
campaign_participants
campaign_tasks
campaign_task_completions
campaign_rewards
campaign_reward_redemptions
```

Ví dụ campaign:

- Phối 3 outfit với item của brand để nhận voucher
- Tương tác bài post để tích điểm
- Thử sample mới và vote màu
- Thành viên VIP xem trước collection
- Birthday voucher
- Seasonal style challenge

Campaign nên hỗ trợ các trạng thái:

- Draft
- Scheduled
- Active
- Paused
- Completed
- Cancelled

## Customer Service Hub

Mục tiêu: brand có nơi chat và xử lý yêu cầu đổi trả/hỗ trợ khách hàng.

Các thực thể chính:

```sql
support_tickets
support_ticket_messages
return_exchange_requests
return_exchange_events
```

Ý nghĩa:

- `support_tickets`: ticket hỗ trợ chung
- `support_ticket_messages`: nội dung trao đổi giữa user và brand
- `return_exchange_requests`: yêu cầu đổi/trả sản phẩm
- `return_exchange_events`: timeline xử lý yêu cầu

Return/exchange không cần biến Closy thành marketplace. Brand vẫn là bên xử lý nghiệp vụ, Closy chỉ cung cấp công cụ quản lý yêu cầu và lịch sử hỗ trợ.

## Digital Sample Lab

Mục tiêu: brand test ý tưởng sản phẩm trước khi sản xuất.

Các thực thể chính:

```sql
digital_samples
digital_sample_assets
digital_sample_variants
sample_test_campaigns
sample_test_participants
sample_outfit_trials
sample_votes
sample_feedback
sample_interest_signals
```

Ý nghĩa:

- `digital_samples`: ý tưởng sản phẩm hoặc sample digital
- `digital_sample_assets`: ảnh/render/asset của sample
- `digital_sample_variants`: biến thể màu, size, style
- `sample_test_campaigns`: chiến dịch test sample
- `sample_test_participants`: user được mời test
- `sample_outfit_trials`: user phối sample với outfit/tủ đồ
- `sample_votes`: vote màu, kiểu dáng, mức độ thích
- `sample_feedback`: comment/feedback định tính
- `sample_interest_signals`: tín hiệu quan tâm như save, notify me, pre-order intent

Đây là module khác biệt lớn của Closy. Nó biến dữ liệu tủ đồ thành insight sản phẩm cho brand.

## Analytics & Insight

Mục tiêu: cung cấp insight tổng hợp cho brand, không định danh cá nhân nếu chưa có sự đồng ý rõ ràng.

Các thực thể có thể cần:

```sql
brand_insight_snapshots
campaign_metric_snapshots
sample_metric_snapshots
user_brand_consents
```

Insight nên ưu tiên dạng tổng hợp:

- Sample nào được phối nhiều
- Variant màu nào được vote cao
- Segment nào tương tác mạnh
- Item brand thường được phối với loại đồ nào
- Occasion phổ biến
- Wardrobe gap phổ biến

## Brand Billing

Mục tiêu: thu tiền từ brand theo SaaS/campaign/sample lab, không phải hoa hồng giao dịch.

Các thực thể đề xuất:

```sql
brand_subscription_plans
brand_subscriptions
brand_billing_accounts
brand_invoices
brand_invoice_items
brand_payments
```

Có thể chưa cần triển khai đầy đủ ở MVP nếu bài toán hiện tại là prototype. Nhưng về schema định hướng, nên tách brand billing khỏi user subscription để tránh lẫn B2C và B2B.

---

# Schema đề xuất cho MVP mới

Phần này là thiết kế tối thiểu để mô hình B2B2C có thể đứng vững trong pitch và codebase.

## Nhóm bảng nên giữ từ schema hiện tại

Giữ trực tiếp hoặc chỉnh nhẹ:

```text
users
refresh_tokens
categories
wardrobe_items
outfits
outfit_items
user_style_profiles
conversational_contexts
messages
user_daily_quotas
ai_cost_policies
ai_cost_policy_operations
user_ai_policy_grants
ai_usage_period_ledgers
ai_usage_events
subscription_plans
user_subscriptions
user_subscription_events
deposit_transactions
provider_webhook_inbox
provider_payment_events
```

Có thể tạm bỏ nếu muốn đơn giản hoá:

```text
user_wallets
wallet_statements
subscription_renewal_attempts
```

Chỉ bỏ nếu code hiện tại chưa phụ thuộc nặng hoặc team chấp nhận giảm scope payment/renewal.

## Nhóm bảng nên xoá khỏi MVP mới

```text
post_items
transfer_requests
post_score_snapshots
```

Lý do:

- Thể hiện hướng resale/marketplace
- Không phục vụ trực tiếp B2B loyalty
- Tăng complexity không cần thiết
- Dễ làm lệch câu chuyện startup

## Nhóm bảng nên rename hoặc rebuild

```text
posts      -> brand_posts hoặc campaign_posts
post_media -> brand_post_media hoặc campaign_post_media
comments   -> post_comments hoặc campaign_comments
likes      -> post_reactions hoặc campaign_interactions
```

Khuyến nghị mạnh hơn cho nhánh mới: không rename cơ học, mà rebuild theo domain mới để tránh mang theo semantic cũ như `total_price`, `contact_info`, `hotness_dirty_at`.

## Nhóm bảng mới tối thiểu nên thêm

### Brand core

```text
brands
brand_members
brand_customers
```

### Loyalty

```text
loyalty_programs
loyalty_tiers
loyalty_accounts
loyalty_point_transactions
brand_benefits
benefit_redemptions
```

### Campaign

```text
brand_campaigns
campaign_posts
campaign_participants
campaign_rewards
campaign_interactions
```

### Support

```text
support_tickets
support_ticket_messages
return_exchange_requests
```

### Digital Sample Lab

```text
digital_samples
digital_sample_assets
digital_sample_variants
sample_test_participants
sample_outfit_trials
sample_votes
sample_feedback
```

### Consent/Privacy

```text
user_brand_consents
```

---

# Gợi ý field chi tiết cho các bảng mới

## `brands`

Mục đích: lưu thông tin brand.

Field đề xuất:

```text
id
slug
name
description
logo_url
logo_public_id
website_url
contact_email
contact_phone
status
created_at
updated_at
```

Ghi chú:

- `slug` unique để dùng trong URL
- `status` có thể là draft, active, suspended, archived

## `brand_members`

Mục đích: user nào thuộc brand nào và có quyền gì.

Field đề xuất:

```text
id
brand_id
user_id
role
status
invited_by_user_id
joined_at
created_at
updated_at
```

Role đề xuất:

```text
OWNER
MANAGER
SUPPORT_STAFF
MARKETING_STAFF
ANALYST
```

Unique constraint:

```text
brand_id + user_id
```

## `brand_customers`

Mục đích: brand quản lý khách hàng thân thiết.

Field đề xuất:

```text
id
brand_id
user_id
external_customer_code
phone_snapshot
email_snapshot
full_name_snapshot
source
status
first_seen_at
last_interacted_at
created_at
updated_at
```

Nguồn `source` có thể là:

```text
MANUAL_IMPORT
OFFLINE_STORE
PHONE_LOOKUP
CAMPAIGN_JOIN
USER_FOLLOW
DIGITAL_SAMPLE_TEST
```

Ghi chú:

- Có thể cho phép `user_id` nullable nếu brand import khách bằng số điện thoại/email nhưng người đó chưa có tài khoản Closy.
- Nếu muốn đơn giản MVP, bắt buộc khách phải có tài khoản Closy trước.

## `loyalty_programs`

Mục đích: mỗi brand có một hoặc nhiều chương trình loyalty.

Field đề xuất:

```text
id
brand_id
name
description
status
points_name
created_at
updated_at
```

## `loyalty_tiers`

Mục đích: hạng thành viên.

Field đề xuất:

```text
id
program_id
name
rank
min_points
benefit_summary
created_at
updated_at
```

Ví dụ tier:

- Member
- Silver
- Gold
- VIP

## `loyalty_accounts`

Mục đích: điểm và hạng hiện tại của user trong một brand.

Field đề xuất:

```text
id
program_id
brand_id
user_id
brand_customer_id
current_points
lifetime_points
current_tier_id
status
created_at
updated_at
```

Unique constraint:

```text
program_id + user_id
```

## `loyalty_point_transactions`

Mục đích: lịch sử cộng/trừ điểm.

Field đề xuất:

```text
id
loyalty_account_id
brand_id
user_id
points_delta
balance_after
reason_type
reason_ref_id
description
created_by_user_id
created_at
```

Reason type đề xuất:

```text
PURCHASE
MANUAL_ADJUSTMENT
CAMPAIGN_REWARD
POST_INTERACTION
SAMPLE_TEST
BENEFIT_REDEMPTION
REFUND_ADJUSTMENT
```

## `brand_benefits`

Mục đích: quyền lợi/ưu đãi của brand.

Field đề xuất:

```text
id
brand_id
tier_id
name
description
benefit_type
value_snapshot
start_at
end_at
status
created_at
updated_at
```

Benefit type đề xuất:

```text
VOUCHER
EARLY_ACCESS
FREE_SHIPPING
BIRTHDAY_GIFT
SAMPLE_LAB_ACCESS
PRIVATE_EVENT
```

## `benefit_redemptions`

Mục đích: user đã nhận/dùng benefit nào.

Field đề xuất:

```text
id
benefit_id
brand_id
user_id
status
redeemed_at
expires_at
metadata
created_at
updated_at
```

## `brand_campaigns`

Mục đích: chiến dịch ưu đãi hoặc tương tác của brand.

Field đề xuất:

```text
id
brand_id
created_by_user_id
campaign_type
title
description
status
start_at
end_at
visibility
targeting_rules
reward_rules
created_at
updated_at
```

Campaign type đề xuất:

```text
OUTFIT_CHALLENGE
VOUCHER_CAMPAIGN
POST_ENGAGEMENT
NEW_COLLECTION_PREVIEW
DIGITAL_SAMPLE_TEST
BIRTHDAY_OFFER
```

## `campaign_posts`

Mục đích: nội dung hiển thị trong campaign, thay cho community post cũ.

Field đề xuất:

```text
id
brand_id
campaign_id
author_user_id
title
content
status
published_at
created_at
updated_at
```

Không nên có các field kiểu marketplace như:

```text
total_price
contact_info
buyer_user_id
transfer_state
sold_at
```

## `campaign_interactions`

Mục đích: ghi nhận tương tác của user với campaign/post.

Field đề xuất:

```text
id
brand_id
campaign_id
post_id
user_id
interaction_type
metadata
created_at
```

Interaction type đề xuất:

```text
VIEW
LIKE
COMMENT
SAVE
SHARE
JOIN
COMPLETE_TASK
```

Dữ liệu này có thể dùng để cộng điểm loyalty hoặc đo hiệu quả campaign.

## `campaign_participants`

Mục đích: user tham gia campaign.

Field đề xuất:

```text
id
campaign_id
brand_id
user_id
status
joined_at
completed_at
created_at
updated_at
```

## `campaign_rewards`

Mục đích: reward của campaign.

Field đề xuất:

```text
id
campaign_id
brand_id
reward_type
reward_value_snapshot
max_redemptions
status
created_at
updated_at
```

Reward type đề xuất:

```text
POINTS
VOUCHER
BENEFIT
SAMPLE_ACCESS
```

## `support_tickets`

Mục đích: ticket hỗ trợ khách hàng giữa user và brand.

Field đề xuất:

```text
id
brand_id
user_id
brand_customer_id
subject
ticket_type
status
priority
assigned_to_user_id
created_at
updated_at
closed_at
```

Ticket type đề xuất:

```text
GENERAL_SUPPORT
RETURN_REQUEST
EXCHANGE_REQUEST
PRODUCT_QUESTION
LOYALTY_ISSUE
```

Status đề xuất:

```text
OPEN
PENDING_BRAND
PENDING_CUSTOMER
RESOLVED
CLOSED
CANCELLED
```

## `support_ticket_messages`

Mục đích: tin nhắn trong ticket.

Field đề xuất:

```text
id
ticket_id
sender_user_id
sender_type
message
attachments
created_at
```

Sender type đề xuất:

```text
CUSTOMER
BRAND_STAFF
SYSTEM
```

## `return_exchange_requests`

Mục đích: yêu cầu đổi/trả riêng nếu muốn tách khỏi ticket.

Field đề xuất:

```text
id
brand_id
user_id
support_ticket_id
request_type
status
reason
description
external_order_ref
external_product_ref
created_at
updated_at
resolved_at
```

Ghi chú:

- `external_order_ref` dùng vì Closy không phải nơi xử lý đơn hàng chính.
- Brand có thể nhập mã đơn từ Shopee/TikTok/cửa hàng nếu cần.

## `digital_samples`

Mục đích: sản phẩm ở dạng ý tưởng/sample digital.

Field đề xuất:

```text
id
brand_id
created_by_user_id
name
description
category_id
status
launch_intent
created_at
updated_at
```

Status đề xuất:

```text
DRAFT
TESTING
ARCHIVED
APPROVED_FOR_PRODUCTION
REJECTED
```

## `digital_sample_assets`

Mục đích: ảnh/render của sample.

Field đề xuất:

```text
id
sample_id
asset_type
asset_url
asset_public_id
sort_order
created_at
updated_at
```

## `digital_sample_variants`

Mục đích: biến thể sample.

Field đề xuất:

```text
id
sample_id
color_name
color_hex
size_label
material
pattern
variant_metadata
status
created_at
updated_at
```

## `sample_test_participants`

Mục đích: user nào được mời hoặc tham gia test sample.

Field đề xuất:

```text
id
sample_id
brand_id
user_id
invitation_source
status
invited_at
joined_at
created_at
updated_at
```

## `sample_outfit_trials`

Mục đích: user thử phối sample với outfit/tủ đồ.

Field đề xuất:

```text
id
sample_id
sample_variant_id
brand_id
user_id
outfit_id
trial_snapshot
created_at
updated_at
```

Ghi chú:

- `trial_snapshot` lưu trạng thái phối đồ tại thời điểm thử, tránh bị thay đổi nếu outfit/item sau này đổi.

## `sample_votes`

Mục đích: user vote sample/variant.

Field đề xuất:

```text
id
sample_id
sample_variant_id
brand_id
user_id
vote_type
rating
created_at
updated_at
```

Vote type đề xuất:

```text
LIKE
DISLIKE
COLOR_PREFERENCE
WOULD_BUY
NOT_MY_STYLE
```

## `sample_feedback`

Mục đích: feedback định tính.

Field đề xuất:

```text
id
sample_id
sample_variant_id
brand_id
user_id
content
sentiment
created_at
updated_at
```

## `user_brand_consents`

Mục đích: quản lý sự đồng ý của user khi chia sẻ dữ liệu với brand.

Field đề xuất:

```text
id
brand_id
user_id
consent_type
status
granted_at
revoked_at
created_at
updated_at
```

Consent type đề xuất:

```text
LOYALTY_JOIN
CAMPAIGN_PARTICIPATION
WARDROBE_INSIGHT_AGGREGATION
DIGITAL_SAMPLE_TESTING
DIRECT_SUPPORT_CHAT
```

---

# API scope đề xuất cho MVP mới

## User-facing APIs

### Wardrobe

```text
POST   /api/v1/wardrobe/items
GET    /api/v1/wardrobe/items
GET    /api/v1/wardrobe/items/{id}
PATCH  /api/v1/wardrobe/items/{id}
DELETE /api/v1/wardrobe/items/{id}
```

### Outfits

```text
POST   /api/v1/outfits
GET    /api/v1/outfits
GET    /api/v1/outfits/{id}
PATCH  /api/v1/outfits/{id}
DELETE /api/v1/outfits/{id}
```

### AI

```text
POST /api/v1/ai/outfit-recommendations
POST /api/v1/ai/chat
```

### Brand discovery and loyalty

```text
GET  /api/v1/brands
GET  /api/v1/brands/{brandId}
POST /api/v1/brands/{brandId}/join-loyalty
GET  /api/v1/me/brand-loyalties
GET  /api/v1/me/brand-loyalties/{brandId}
```

### Campaign participation

```text
GET  /api/v1/brands/{brandId}/campaigns
GET  /api/v1/campaigns/{campaignId}
POST /api/v1/campaigns/{campaignId}/join
POST /api/v1/campaigns/{campaignId}/interactions
```

### Support

```text
POST /api/v1/brands/{brandId}/support-tickets
GET  /api/v1/me/support-tickets
GET  /api/v1/support-tickets/{ticketId}
POST /api/v1/support-tickets/{ticketId}/messages
```

### Digital Sample Lab

```text
GET  /api/v1/brands/{brandId}/digital-samples
GET  /api/v1/digital-samples/{sampleId}
POST /api/v1/digital-samples/{sampleId}/join-test
POST /api/v1/digital-samples/{sampleId}/outfit-trials
POST /api/v1/digital-samples/{sampleId}/votes
POST /api/v1/digital-samples/{sampleId}/feedback
```

## Brand-facing APIs

### Brand management

```text
POST  /api/v1/brand-portal/brands
GET   /api/v1/brand-portal/brands/{brandId}
PATCH /api/v1/brand-portal/brands/{brandId}
POST  /api/v1/brand-portal/brands/{brandId}/members
GET   /api/v1/brand-portal/brands/{brandId}/members
```

### Brand customers

```text
GET   /api/v1/brand-portal/brands/{brandId}/customers
POST  /api/v1/brand-portal/brands/{brandId}/customers
GET   /api/v1/brand-portal/brands/{brandId}/customers/{customerId}
PATCH /api/v1/brand-portal/brands/{brandId}/customers/{customerId}
```

### Loyalty

```text
POST  /api/v1/brand-portal/brands/{brandId}/loyalty-program
GET   /api/v1/brand-portal/brands/{brandId}/loyalty-program
POST  /api/v1/brand-portal/brands/{brandId}/loyalty-tiers
POST  /api/v1/brand-portal/brands/{brandId}/loyalty/accounts/{accountId}/adjust-points
POST  /api/v1/brand-portal/brands/{brandId}/benefits
```

### Campaign

```text
POST  /api/v1/brand-portal/brands/{brandId}/campaigns
GET   /api/v1/brand-portal/brands/{brandId}/campaigns
GET   /api/v1/brand-portal/campaigns/{campaignId}
PATCH /api/v1/brand-portal/campaigns/{campaignId}
POST  /api/v1/brand-portal/campaigns/{campaignId}/publish
```

### Support

```text
GET   /api/v1/brand-portal/brands/{brandId}/support-tickets
GET   /api/v1/brand-portal/support-tickets/{ticketId}
POST  /api/v1/brand-portal/support-tickets/{ticketId}/messages
PATCH /api/v1/brand-portal/support-tickets/{ticketId}
```

### Digital Sample Lab

```text
POST  /api/v1/brand-portal/brands/{brandId}/digital-samples
GET   /api/v1/brand-portal/brands/{brandId}/digital-samples
GET   /api/v1/brand-portal/digital-samples/{sampleId}
PATCH /api/v1/brand-portal/digital-samples/{sampleId}
POST  /api/v1/brand-portal/digital-samples/{sampleId}/variants
POST  /api/v1/brand-portal/digital-samples/{sampleId}/publish-test
GET   /api/v1/brand-portal/digital-samples/{sampleId}/insights
```

---

# MVP nên làm gì trước

Vì nhánh mới cho phép rebuild mạnh tay, nhưng thời gian dự án có hạn, không nên làm toàn bộ cùng lúc. MVP nên chứng minh mô hình mới bằng một lát cắt rõ ràng.

## MVP đề xuất

### Phần B2C giữ lại

- User auth
- Tủ đồ số
- Outfit
- AI outfit recommendation cơ bản
- AI chat nếu đã có

### Phần B2B thêm vào

- Brand profile
- Brand member
- Brand customer
- Loyalty account
- Manual point adjustment
- Loyalty tier cơ bản
- Benefit cơ bản
- Campaign cơ bản
- Campaign interaction cộng điểm
- Support ticket cơ bản
- Digital sample cơ bản
- User thử phối sample với outfit hoặc item trong tủ đồ
- Brand xem insight đơn giản của sample

## MVP không nên làm

- Marketplace
- Giỏ hàng
- Thanh toán đơn hàng brand-user
- Giao hàng
- C2C resale
- Transfer item giữa user
- Social feed tự do
- Ranking hotness phức tạp
- Moderation system lớn
- Creator subscription kiểu Patreon
- Full CRM phức tạp như Salesforce

---

# Roadmap triển khai trên nhánh mới

## Phase 0: Chốt scope và đóng băng hướng cũ

Việc cần làm:

- Xác nhận với team rằng mô hình mới là B2B2C
- Xác nhận không làm marketplace trong MVP
- Xác nhận không lấy hoa hồng giao dịch làm nguồn thu chính
- Xác nhận community post không phải core MVP
- Xác nhận Digital Sample Lab là điểm khác biệt chính

Deliverable:

- Một file business model mới
- Một file target schema mới
- Một danh sách module giữ/xoá/sửa

## Phase 1: Dọn schema và domain

Việc cần làm:

- Giữ auth, wardrobe, outfit, AI, subscription nếu cần
- Xoá hoặc không migrate các bảng resale/social cũ
- Tạo migration baseline mới cho nhánh mới
- Thêm brand, loyalty, campaign, support, sample lab

Không cần feature flag vì đây là nhánh rebuild mới. Nếu không ổn có thể quay lại `main`.

## Phase 2: Brand core và permission

Việc cần làm:

- Implement `brands`
- Implement `brand_members`
- Middleware kiểm tra quyền brand
- API brand portal cơ bản
- Seed một brand demo

## Phase 3: Brand customer và loyalty

Việc cần làm:

- Implement `brand_customers`
- Implement `loyalty_programs`
- Implement `loyalty_tiers`
- Implement `loyalty_accounts`
- Implement `loyalty_point_transactions`
- API cộng/trừ điểm manual
- API user xem điểm của mình với từng brand

## Phase 4: Campaign và benefit

Việc cần làm:

- Implement `brand_campaigns`
- Implement `campaign_participants`
- Implement `campaign_interactions`
- Implement `brand_benefits`
- Implement `benefit_redemptions`
- Rule đơn giản: interaction hoặc complete task có thể cộng điểm

## Phase 5: Support ticket

Việc cần làm:

- Implement `support_tickets`
- Implement `support_ticket_messages`
- Implement `return_exchange_requests` nếu đủ thời gian
- Brand staff xử lý ticket
- User tạo ticket cho brand

## Phase 6: Digital Sample Lab

Việc cần làm:

- Implement `digital_samples`
- Implement `digital_sample_assets`
- Implement `digital_sample_variants`
- Implement `sample_test_participants`
- Implement `sample_outfit_trials`
- Implement `sample_votes`
- Implement `sample_feedback`
- Brand tạo sample
- User thử phối sample
- Brand xem insight cơ bản

## Phase 7: Polish cho pitch/demo

Việc cần làm:

- Demo flow user
- Demo flow brand
- Tạo data seed
- Dashboard insight đơn giản
- Tạo kịch bản demo rõ ràng

Demo nên thể hiện được:

- User có tủ đồ
- User là khách thân thiết của brand
- Brand tạo campaign/sample
- User thử phối sample với tủ đồ
- User vote/feedback
- Brand nhận insight
- User nhận điểm/benefit

---

# Luồng demo đề xuất

## Demo B2C

User đăng nhập vào Closy.

User upload một số item vào tủ đồ.

User tạo outfit hoặc nhận AI recommendation.

User theo dõi/tham gia loyalty của một brand.

User thấy brand có campaign hoặc digital sample mới.

User thử phối sample với tủ đồ cá nhân.

User vote màu hoặc gửi feedback.

User nhận điểm loyalty hoặc benefit.

## Demo B2B

Brand staff đăng nhập brand portal.

Brand xem danh sách khách hàng thân thiết.

Brand tạo campaign ưu đãi.

Brand tạo digital sample mới.

Brand mời nhóm khách VIP tham gia test.

Brand xem số liệu:

- Bao nhiêu user tham gia
- Bao nhiêu outfit trial được tạo
- Variant nào được thích nhất
- Feedback chính là gì
- User segment nào quan tâm nhiều nhất

Brand dùng insight đó để quyết định có nên sản xuất sample hay không.

---

# Các câu hỏi review team cần trả lời

Team nên trả lời các câu hỏi sau trước khi code sâu:

- Closy có chắc chắn không làm marketplace trong MVP không?
- B2C Premium còn giữ trong pitch hay chỉ là nguồn thu phụ?
- Brand nào là customer đầu tiên trong giả định demo?
- Brand có cần product catalog thật không, hay chỉ cần sample/campaign?
- Loyalty point đến từ mua hàng offline sẽ nhập bằng cách nào?
- User có cần consent trước khi brand xem insight không?
- Digital sample cần ảnh thật, ảnh render hay chỉ metadata/placeholder?
- Support ticket có cần đổi trả chi tiết không, hay chỉ cần chat/ticket cơ bản?
- MVP cần demo bằng UI đầy đủ hay API + dashboard đơn giản là đủ?

---

# Tiêu chí đánh giá code/schema mới có đúng hướng không

Schema/code mới được xem là đúng hướng nếu trả lời được các câu sau:

- Có brand là thực thể trung tâm chưa?
- Có brand member và phân quyền brand chưa?
- Có brand customer chưa?
- Có loyalty account, point transaction và tier chưa?
- Có campaign do brand tạo chưa?
- Có interaction của user với campaign chưa?
- Interaction có thể tạo loyalty value chưa?
- Có support ticket giữa user và brand chưa?
- Có digital sample do brand tạo chưa?
- User có thể thử phối sample với tủ đồ chưa?
- Brand có xem được insight từ sample/campaign chưa?
- Schema có còn bị kéo sang resale/marketplace không?
- Có còn các bảng như `transfer_requests`, `post_items` trong MVP không?
- `posts` nếu còn tồn tại thì đã chuyển nghĩa thành brand/campaign content chưa?

---

# Danh sách thay đổi đề xuất so với schema hiện tại

## Nên giữ

```text
users
refresh_tokens
categories
wardrobe_items
outfits
outfit_items
user_style_profiles
conversational_contexts
messages
user_daily_quotas
ai_cost_policies
ai_cost_policy_operations
user_ai_policy_grants
ai_usage_period_ledgers
ai_usage_events
subscription_plans
user_subscriptions
user_subscription_events
deposit_transactions
provider_webhook_inbox
provider_payment_events
goose_db_version
```

## Cân nhắc giữ nếu còn cần payment phức tạp

```text
user_wallets
wallet_statements
subscription_renewal_attempts
```

## Nên xoá khỏi MVP mới

```text
post_items
transfer_requests
post_score_snapshots
```

## Nên rebuild thay vì giữ nguyên

```text
posts
post_media
comments
likes
```

Lý do rebuild:

- Field hiện tại còn mùi marketplace/social
- `posts` có `contact_info`, `total_price`, `hotness_dirty_at`
- `likes/comments` hiện phục vụ social feed, chưa gắn với brand/campaign/loyalty

## Nên thêm

```text
brands
brand_members
brand_customers
loyalty_programs
loyalty_tiers
loyalty_accounts
loyalty_point_transactions
brand_benefits
benefit_redemptions
brand_campaigns
campaign_posts
campaign_participants
campaign_interactions
campaign_rewards
support_tickets
support_ticket_messages
return_exchange_requests
digital_samples
digital_sample_assets
digital_sample_variants
sample_test_participants
sample_outfit_trials
sample_votes
sample_feedback
user_brand_consents
```

---

# Kết luận

Hướng rebuild mới là hợp lý nếu team muốn Closy có câu chuyện kinh tế mạnh hơn B2C subscription đơn thuần. Lõi B2C wardrobe và AI vẫn cần giữ, vì đây là nguồn tạo engagement và dữ liệu khác biệt. Tuy nhiên, schema và code nên chuyển trọng tâm từ community/resale sang brand loyalty, campaign, support và digital sample lab.

Cách nói với team:

> Closy không bỏ B2C, nhưng B2C không còn là nguồn thu chính. B2C là lớp tạo người dùng, thói quen sử dụng và dữ liệu tủ đồ. B2B là lớp tạo doanh thu chính, nơi brand trả tiền để quản lý khách hàng trung thành, chăm sóc sau bán hàng, chạy chiến dịch cá nhân hoá và test digital sample trước khi sản xuất.

Cách nói với giảng viên/hội đồng:

> Closy is not a mini marketplace. Closy is a Fashion Loyalty & Co-creation Platform powered by AI wardrobe data. Users use Closy to manage wardrobes and style outfits, while brands use Closy to manage loyal customers, launch personalized campaigns, handle after-sales support, and validate digital product samples before production.
