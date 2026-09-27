# Feature Specification: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature Branch**: `023-analyze-status-handling`

**Created**: 2026-09-26

**Status**: Draft

**Input**: User description: "Chuẩn hóa và bổ sung kịch bản xử lý cho từng trạng thái phân tích AI của trang phục (đang xử lý, hoàn tất, thất bại, cần rà soát) ở cả phía người dùng cá nhân (B2C) và nhãn hàng (B2B), gồm: xác nhận lại món đồ cần rà soát để đưa vào sử dụng, thông báo tiến trình realtime đầy đủ (kể cả trạng thái đang xử lý), chính sách thử lại an toàn (cooldown, giới hạn số lần, giữ chỉnh sửa thủ công), phân biệt lỗi và trạng thái nhãn hàng, khả năng phát lại sự kiện khi client kết nối lại, và tài liệu playbook vận hành."

## Clarifications

### Session 2026-09-26

- Q: Xử lý món "cần rà soát" có bắt buộc phải có danh mục hợp lệ không? → A: Có — bắt buộc danh mục hợp lệ (thuộc nhóm có thể phối đồ) mới xử lý được; nếu thiếu/không hợp lệ thì từ chối và yêu cầu chọn danh mục.
- Q: Quyền đặt/đổi trạng thái "lỗi" của sản phẩm nhãn hàng? → A: "Lỗi" là trạng thái cuối với thao tác thủ công — hệ thống từ chối mọi thay đổi trạng thái rời khỏi "lỗi"; chỉ thao tác thử lại (hoặc hệ thống) mới đưa sản phẩm ra khỏi "lỗi".
- Q: Cơ chế truyền sự kiện realtime giữa worker (phát sinh) và tầng HTTP (giữ kết nối client) khi khác tiến trình? → A: Dùng Redis Pub/Sub làm kênh trung gian worker→http, kết hợp buffer in-memory trong tiến trình HTTP để phát lại cho client; đặt sau lớp trừu tượng transport để có thể thay bằng kho chia sẻ sau này.
- Q: Có cần chặn thử lại theo số lần/thời gian (cooldown) không? → A: Không — chỉ cần kiểm tra món đồ đang ở trạng thái cho phép thử lại (thất bại hoặc cần rà soát); không thêm cột đếm/thời gian cho mục đích này.
- Q: Trạng thái nào cho phép thử lại phân tích ở sản phẩm nhãn hàng? → A: Cho phép khi sản phẩm ở trạng thái lỗi, hoặc ở trạng thái nháp nhưng có lý do cần duyệt (`reviewReason`); không cho phép với nháp thường.
- Q: Cơ chế xác định "danh mục người dùng đã sửa" để bảo toàn khi phân tích lại? → A: Bảo toàn khi fashion item đã có danh mục (không cần cờ riêng); nếu chưa có danh mục thì AI được phép điền.

### Session 2026-09-27

- Q: Ghi bug lọc danh mục rỗng vào spec nào? → A: Bổ sung vào chính spec `023-analyze-status-handling` (US5) để cùng thực thi.
- Q: Phạm vi sửa anti-pattern `len(fashionItemIDs) > 0`? → A: Sửa cả 3 đường dùng chung: `GetWardrobeItems`, `GetSystemCatalogWardrobeItems` (fallback DB search) và `catalog_uc.go`.
- Q: Cách phân biệt "không lọc" vs "lọc nhưng rỗng" ở tầng repository? → A: Thêm tham số boolean tường minh `filterByFashionItems`; khi bật thì luôn áp bộ lọc kể cả danh sách rỗng.
- Q: Hành vi khi lọc nhưng rỗng và khi không lọc? → A: (a) Lọc nhưng rỗng ⇒ short-circuit trả rỗng, không truy vấn DB; (b) Không lọc ⇒ trả tất cả món của người dùng, kể cả món chưa gắn fashion item (`fashion_item_id` NULL).
- Q: Có yêu cầu test cho các case này không? → A: Có — test đơn vị bao phủ cả case danh mục rỗng (→ 0 món) và không lọc (→ toàn bộ).
- Q: Khi Redis Pub/Sub (kênh realtime worker→http) tạm thời không khả dụng thì xử lý thế nào? → A: Best-effort — lỗi phát sự kiện chỉ được ghi log cảnh báo và KHÔNG chặn/không làm thất bại nghiệp vụ; nguồn sự thật vẫn là DB và client tự làm mới (refetch/polling) khi không nhận được sự kiện.
- Q: Cần tín hiệu quan sát nào cho luồng realtime best-effort? → A: Log cảnh báo khi phát/relay lỗi + metric đếm sự kiện publish/relay thành công–thất bại và số subscriber realtime.

### Session 2026-09-27 (bổ sung)

- Q: Ảnh không hợp lệ (đa món/toàn thân) nên ở trạng thái nào, có cho confirm/retry không? → A: Chuyển sang `failed` kèm lý do mã (`multiple_items_detected`/`full_body_outfit_detected`); KHÔNG cho confirm và KHÔNG cho retry (ảnh không đổi thì retry vô nghĩa); hướng người dùng tải ảnh khác.
- Q: `needs_review` còn dùng cho trường hợp nào? → A: Chỉ khi ảnh hợp lệ (đúng một món) nhưng AI chưa chắc danh mục; lý do `uncertain_category`; xử lý bằng `retry-analysis` kèm danh mục (không còn `confirm-review`).
- Q: Có đổi contract AI để tách "ảnh hợp lệ" khỏi "độ chắc chắn danh mục" không? → A: Có — tách bằng cờ riêng (ví dụ `is_single_item` + `category_confident`) để định tuyến failed vs needs_review.
- Q: Lý do (reason) nên là mã hay câu thông báo? → A: Là **mã** (kể cả trường SSE `error` của sự kiện thất bại); frontend tự đặt text hiển thị.
- Q: Reason là thuộc tính của ai và hiển thị ở đâu? → A: Của fashion item; chỉ hiển thị trong đối tượng `fashionItem` lồng nhau; KHÔNG lặp lại ở cấp gốc wardrobe/brand item.
- Q: Chuẩn hóa casing của contract DTO? → A: `FashionItemContractDTO` (và `CategoryContractDTO`) dùng camelCase khi serialize API.
- Q: Brand khi hết retry tự động thì sao? → A: Brand cũng phải nhận `processing-exhausted` và chuyển sang `failed` kèm lý do (không kẹt ở `draft`); cơ chế phát hiện xử lý quá hạn quét theo fashion item cho cả wardrobe và brand.
- Q: Khi AI trả `is_single_item=true` nhưng thiếu/null `category_confident` thì mặc định thế nào? → A: Mặc định `category_confident=false` (coi là chưa chắc category) ⇒ `needs_review` với `uncertain_category`; an toàn dữ liệu hơn là tự động `completed` với category có thể sai.
- Q: Khi AI trả mâu thuẫn `is_single_item=false` nhưng `review_reason=uncertain_category` thì định tuyến thế nào? → A: Ưu tiên tính hợp lệ của ảnh — coi là ảnh không hợp lệ ⇒ `failed` + `multiple_items_detected`; `uncertain_category` chỉ hợp lệ khi `is_single_item=true`.
- Q: Khi thử lại thất bại ở bước publish job (sau khi đã claim), có ghi lý do không? → A: Có — ghi `processingErrorReason=analysis_temporary_error` cho món/sản phẩm (qua fashion item) cùng lúc set `failed`, để FE luôn có mã lý do (FR-036).
- Q: Có giữ API `confirm-review` không? → A: Bỏ hẳn — vì không lưu metadata ở `needs_review`, confirm chỉ set danh mục + trạng thái mà rỗng metadata/embedding; gộp toàn bộ vào `retry-analysis` (tránh món "usable nhưng nghèo dữ liệu").
- Q: `needs_review` được xử lý thế nào sau khi bỏ confirm? → A: `retry-analysis` nhận `categoryId`; với `needs_review` (`uncertain_category`) bắt buộc danh mục hợp lệ (từ request hoặc danh mục có sẵn), rồi chạy lại AI theo danh mục đó để tạo metadata đầy đủ.
- Q: Làm sao tránh `needs_review` tái sinh khi thử lại kèm danh mục? → A: Dùng chế độ "danh mục cố định" khi phân tích lại: chỉ trích xuất metadata theo danh mục đã biết, KHÔNG phân loại lại category và KHÔNG áp gate `is_single_item`/`category_confident`; danh mục người dùng chọn được giữ nguyên.
- Q: Prompt phân tích ảnh có nên "fail-fast" để tiết kiệm token? → A: Có — khi ảnh không hợp lệ hoặc `category_confident=false`, AI trả JSON tối giản; analyze giữ một lượt gọi KHÔNG bật `ResponseSchema` (schema strict buộc đủ field, vô hiệu fail-fast).
- Q: Brand đặt danh mục ở đâu (brand chưa có API set category)? → A: Thêm `categoryId` vào chính request `retry-analysis` của brand; áp cùng cơ chế danh mục cố định.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Rà soát món AI không chắc danh mục bằng thử lại kèm danh mục (Priority: P1)

Khi người dùng tải ảnh trang phục lên, AI có thể không đủ tự tin để phân loại danh mục. Món đồ được đưa vào trạng thái "cần rà soát" và tạm thời không dùng được để phối đồ. Người dùng chọn danh mục cho món đồ rồi gọi thao tác thử lại kèm danh mục đó; hệ thống chạy lại phân tích theo danh mục đã chọn để tạo đầy đủ metadata (màu/chất liệu/…/embedding), giữ nguyên danh mục, và đưa món vào sử dụng.

**Why this priority**: Đây là điểm kẹt nghiêm trọng nhất — món đồ ở trạng thái cần rà soát không có đường nào để trở thành sử dụng được, khiến ảnh người dùng tải lên bị "treo" vĩnh viễn và không tham gia gợi ý phối đồ. Đồng thời món khi vào sử dụng phải có metadata đầy đủ để phối đồ chất lượng.

**Independent Test**: Tạo món ở trạng thái cần rà soát, gọi `retry-analysis` kèm `categoryId` hợp lệ, kiểm tra món chuyển `processing` rồi `inWardrobe`, có metadata và xuất hiện trong gợi ý phối đồ; gọi mà thiếu danh mục thì bị từ chối.

**Acceptance Scenarios**:

1. **Given** một món đồ ở trạng thái cần rà soát thuộc sở hữu người dùng, **When** người dùng gọi thử lại kèm danh mục hợp lệ, **Then** món chuyển sang "đang xử lý" và khi phân tích xong trở thành sử dụng được với metadata đầy đủ, tham gia gợi ý phối đồ.
2. **Given** một món đồ ở trạng thái cần rà soát không có danh mục sẵn, **When** gọi thử lại mà không truyền danh mục, **Then** hệ thống từ chối và yêu cầu chọn danh mục.
3. **Given** một món đồ ở trạng thái cần rà soát, **When** gọi thử lại với danh mục không thể phối đồ, **Then** hệ thống từ chối.
4. **Given** một món đồ không ở trạng thái cho phép thử lại, **When** gọi thử lại, **Then** hệ thống từ chối bằng lỗi nghiệp vụ rõ ràng.
5. **Given** một món đồ thuộc sở hữu người dùng khác, **When** gọi thử lại, **Then** hệ thống từ chối.
6. **Given** người dùng gọi thử lại kèm danh mục, **When** hệ thống phân tích lại, **Then** danh mục người dùng chọn được giữ nguyên (AI không ghi đè).

---

### User Story 2 - Theo dõi tiến trình realtime đầy đủ và không mất sự kiện (Priority: P1)

Người dùng mở màn hình tải lên/phân tích lại và cần thấy trạng thái của từng món theo thời gian thực: đang xử lý, hoàn tất, cần rà soát hay thất bại. Nếu đường truyền bị ngắt rồi kết nối lại giữa chừng, người dùng vẫn muốn nhận được các cập nhật đã phát mà không phải tải lại toàn bộ.

**Why this priority**: Trải nghiệm realtime là cốt lõi của tính năng số hóa tủ đồ. Việc thiếu trạng thái "đang xử lý" và mất sự kiện khi reconnect khiến giao diện hiển thị sai/đứng, ảnh hưởng trực tiếp đến tỉ lệ hoàn tất tải lên.

**Independent Test**: Bắt đầu một phiên phân tích nhiều món, mở kết nối realtime, ngắt kết nối giữa chừng rồi kết nối lại, và kiểm tra client nhận đủ sự kiện "đang xử lý" và các sự kiện kết thúc.

**Acceptance Scenarios**:

1. **Given** một phiên phân tích đang chạy, **When** một món bắt đầu được xử lý, **Then** client nhận được sự kiện trạng thái "đang xử lý" cho món đó.
2. **Given** client đang theo dõi một phiên, **When** client mất kết nối rồi kết nối lại trong lúc phiên chưa kết thúc, **Then** client nhận lại các sự kiện đã phát (trong giới hạn lưu trữ) mà không cần tải lại toàn bộ tủ đồ.
3. **Given** một món hoàn tất phân tích, **When** client nhận sự kiện, **Then** dữ liệu tóm tắt món đồ (danh mục, ảnh, màu sắc, phong cách) đầy đủ để hiển thị ngay.
4. **Given** một món thất bại, **When** client nhận sự kiện, **Then** thông báo lỗi thân thiện với người dùng được kèm theo.

---

### User Story 3 - Thử phân tích lại an toàn và không mất chỉnh sửa thủ công (Priority: P2)

Người dùng muốn thử phân tích lại đối với món đồ thất bại hoặc cần rà soát. Hệ thống cần đảm bảo chỉ thử lại khi trạng thái cho phép, tránh tạo phiên trùng, và bảo toàn những chỉnh sửa thủ công mà người dùng đã thực hiện.

**Why this priority**: Bảo vệ công sức chỉnh sửa của người dùng và tránh tạo phiên trùng, nhưng không phải luồng chính nên xếp sau P1.

**Independent Test**: Gọi thử lại khi món đồ đang xử lý hoặc ở trạng thái không hợp lệ, kiểm tra bị chặn; sửa danh mục rồi thử lại, kiểm tra danh mục được giữ.

**Acceptance Scenarios**:

1. **Given** một món đồ vừa được thử phân tích lại (đang xử lý), **When** người dùng gọi thử lại lần nữa, **Then** hệ thống từ chối vì trạng thái hiện tại không cho phép thử lại.
2. **Given** một món đồ đang ở trạng thái sử dụng được, **When** người dùng gọi thử lại, **Then** hệ thống từ chối.
3. **Given** người dùng đã chỉnh sửa danh mục thủ công, **When** hệ thống phân tích lại món đồ đó, **Then** danh mục thủ công được giữ nguyên (không bị ghi đè bởi AI).
4. **Given** một yêu cầu thử lại hợp lệ được khởi tạo, **When** hệ thống phản hồi, **Then** client nhận được mã phiên để theo dõi tiến trình realtime.

---

### User Story 4 - Nhân viên nhãn hàng xử lý sản phẩm lỗi hoặc cần duyệt (Priority: P2)

Nhân viên nhãn hàng tải danh sách sản phẩm lên Brand Portal. Khi AI phân tích thất bại hoặc không chắc chắn, họ cần phân biệt rõ đó là lỗi hệ thống hay cần duyệt thủ công, xem lý do, và có thể thử lại hoặc chủ động xử lý.

**Why this priority**: Đảm bảo vận hành B2B thông suốt và không phụ thuộc vào đội kỹ thuật; hiện tại lỗi và cần-duyệt bị gộp chung nên khó xử lý.

**Independent Test**: Tạo sản phẩm nhãn hàng gây lỗi phân tích và sản phẩm AI không chắc, kiểm tra trạng thái/lý do hiển thị khác nhau và thao tác thử lại hoạt động.

**Acceptance Scenarios**:

1. **Given** một sản phẩm phân tích thất bại, **When** nhân viên xem, **Then** sản phẩm ở trạng thái lỗi và hiển thị lý do.
2. **Given** một sản phẩm AI không đủ tự tin, **When** nhân viên xem, **Then** sản phẩm ở trạng thái nháp kèm lý do cần duyệt và hiển thị lý do.
3. **Given** một sản phẩm lỗi hoặc cần duyệt, **When** nhân viên gọi thử lại, **Then** sản phẩm được đưa lại hàng đợi và nhân viên theo dõi được tiến trình realtime.
4. **Given** một sản phẩm ở trạng thái lỗi, **When** nhân viên cố thay đổi trạng thái thủ công, **Then** hệ thống từ chối; chỉ thao tác thử lại (hoặc hệ thống) mới đưa sản phẩm ra khỏi trạng thái lỗi.
5. **Given** nhân viên không thuộc nhãn hàng sở hữu sản phẩm, **When** gọi thao tác, **Then** hệ thống từ chối.

---

### User Story 5 - Lọc theo danh mục phải tôn trọng kết quả rỗng (Priority: P2)

Khi người dùng lọc tủ đồ hoặc catalog theo một danh mục mà không có món nào thuộc danh mục đó, hệ thống phải trả về danh sách rỗng. Hiện tại hệ thống diễn giải sai kết quả "rỗng" thành "không lọc" và trả về toàn bộ món — một hồi quy (regression) được tạo ra khi chuyển việc lọc chéo module sang contract.

**Why this priority**: Đây là lỗi đúng đắn dữ liệu đã tồn tại trên môi trường (nhánh `023`): người dùng thấy các món không thuộc danh mục mình chọn. Cần sửa cùng đợt để tránh phát hành sai.

**Independent Test**: Chọn một danh mục không có món nào, gọi lọc danh mục đó, và kiểm tra kết quả trả về rỗng (0 món).

**Acceptance Scenarios**:

1. **Given** người dùng có nhiều món nhưng không món nào thuộc danh mục X, **When** lọc tủ đồ theo X, **Then** kết quả rỗng và tổng số bằng 0.
2. **Given** danh mục Y có món thuộc người dùng, **When** lọc theo Y, **Then** chỉ trả các món thuộc Y.
3. **Given** người dùng không truyền danh mục (và không có từ khóa), **When** lấy danh sách tủ đồ, **Then** trả về tất cả món của người dùng, kể cả món chưa gắn hạng mục trang phục.
4. **Given** tìm kiếm catalog với từ khóa không khớp fashion item nào, **When** tìm, **Then** kết quả rỗng (không trả toàn bộ catalog).

---

### User Story 6 - Phân loại ảnh không hợp lệ và mã hóa lý do (Priority: P1)

Ảnh không hợp lệ (chứa nhiều món, hoặc ảnh toàn thân/người) không thể sửa bằng cách chọn danh mục hay chạy lại AI trên cùng ảnh. Hiện tại các trường hợp này bị gộp vào `needs_review` và vẫn cho xử lý, khiến ảnh xấu lọt vào tủ đồ và hiển thị trong gợi ý phối đồ. Đồng thời lý do đang là câu thông báo thay vì mã, và brand không nhận được thông báo hết retry tự động.

**Why this priority**: Bảo vệ chất lượng dữ liệu tủ đồ/phối đồ (không cho ảnh xấu vào trạng thái sử dụng) và đảm bảo frontend có mã lý do ổn định để dựng UI — ảnh hưởng trực tiếp trải nghiệm và chi phí AI (tránh retry vô ích).

**Independent Test**: (a) Ảnh nhiều món → `failed` + lý do `multiple_items_detected`, không thể xử lý/thử lại. (b) Ảnh một món nhưng AI chưa chắc danh mục → `needs_review` + `uncertain_category`, xử lý bằng thử lại kèm danh mục. (c) Brand hết retry tự động → `failed` kèm lý do, không kẹt `draft`.

**Acceptance Scenarios**:

1. **Given** ảnh chứa nhiều món hoặc ảnh toàn thân, **When** phân tích xong, **Then** món/sản phẩm ở trạng thái `failed` với lý do mã tương ứng và KHÔNG thể xử lý/thử lại.
2. **Given** món/sản phẩm thất bại do ảnh không hợp lệ, **When** gọi thử lại, **Then** hệ thống từ chối và hướng dẫn tải ảnh khác.
3. **Given** ảnh đúng một món nhưng AI chưa chắc danh mục, **When** phân tích xong, **Then** món ở `needs_review` với `uncertain_category`, và người dùng xử lý bằng cách chọn danh mục rồi thử lại kèm danh mục.
4. **Given** một món/sản phẩm có lý do (reason), **When** client đọc payload, **Then** lý do là mã và chỉ nằm trong đối tượng `fashionItem` lồng nhau (không lặp ở gốc).
5. **Given** sản phẩm nhãn hàng xử lý quá hạn và hết retry tự động, **When** hệ thống phát hiện, **Then** sản phẩm chuyển sang `failed` kèm lý do và có thể thử lại.

---

### Edge Cases

- Client mở kết nối realtime trước khi có sự kiện nào, hoặc sau khi phiên đã kết thúc: hệ thống phát lại các sự kiện còn trong giới hạn lưu trữ (nếu có) và không gây lỗi. Buffer KHÔNG bị xóa ngay khi subscriber cuối cùng ngắt kết nối; nó chỉ bị loại khi vượt giới hạn số lượng hoặc hết thời gian lưu trữ (TTL) cấu hình.
- Người dùng gọi thử lại đúng lúc hệ thống đang xử lý món đồ đó (tranh chấp): hệ thống từ chối và không tạo phiên trùng.
- Nhiều client cùng theo dõi một phiên phân tích: mọi client đều nhận được sự kiện.
- Giới hạn lưu trữ sự kiện đầy: hệ thống loại bỏ sự kiện cũ nhất và vẫn phục vụ các sự kiện mới.
- Món đồ cần rà soát nhưng thiếu danh mục hợp lệ: thao tác xử lý (thử lại kèm danh mục) MUST bị từ chối và yêu cầu người dùng chọn danh mục trước.
- Sự kiện bị phát trùng do cơ chế gửi lại: hệ thống xử lý idempotent, không gây sai lệch trạng thái.
- Xử lý/thử lại khi món đồ đang bị khóa do vượt hạn mức gói cước: hệ thống xử lý nhất quán với chính sách khóa hiện hành.
- Lọc theo danh mục tồn tại nhưng không có món nào (kết quả rỗng): trả 0 món, tuyệt đối không diễn giải thành "không lọc".
- Từ khóa tìm kiếm catalog không khớp fashion item nào: trả 0 kết quả, không trả toàn bộ catalog.
- Danh sách món của tủ đồ có bản ghi chưa gắn hạng mục trang phục: khi KHÔNG lọc, các bản ghi này vẫn phải xuất hiện trong kết quả.
- Kênh chia sẻ realtime (Redis Pub/Sub) tạm thời không khả dụng: thao tác nghiệp vụ vẫn thành công bình thường; sự kiện realtime có thể không tới client nhưng trạng thái vẫn đúng ở DB và client tự làm mới khi cần.
- Ảnh chứa nhiều món hoặc ảnh toàn thân/người: chuyển `failed` với mã lý do; KHÔNG cho thử lại (retry cùng ảnh là vô nghĩa); FE hướng dẫn tải ảnh khác.
- Ảnh đúng một món nhưng AI chưa chắc danh mục: `needs_review` với `uncertain_category`; xử lý bằng thử lại kèm danh mục hợp lệ (chế độ danh mục cố định).
- Thử lại `needs_review` kèm danh mục: phân tích ở chế độ danh mục cố định không thể tái sinh `needs_review`; nếu AI lỗi tạm thời thì về `failed` (có thể thử lại), không quay lại `needs_review`.
- Client cũ gọi endpoint `confirm-review` (đã bị gỡ): nhận lỗi 404; FE phải chuyển sang `retry-analysis` kèm danh mục.
- AI trả thiếu trường phân loại: nếu thiếu/không xác định `is_single_item` ⇒ coi là ảnh không hợp lệ (`failed`); nếu `is_single_item=true` nhưng thiếu/null `category_confident` ⇒ mặc định coi là chưa chắc category (`needs_review` + `uncertain_category`).
- AI trả mâu thuẫn (`is_single_item=false` nhưng `review_reason=uncertain_category`): ưu tiên tính hợp lệ ảnh ⇒ `failed` + `multiple_items_detected`; không cho confirm/retry.
- Hết retry tự động ở sản phẩm nhãn hàng: chuyển `failed` kèm lý do, KHÔNG kẹt ở `draft` (nháp thường không retry được).
- Lý do trả về là mã ngoài catalog đã biết: frontend vẫn hiển thị được thông báo mặc định (không hardcode so khớp chuỗi).
- Sản phẩm nhãn hàng có ảnh không hợp lệ: là trạng thái cuối (`failed`) và KHÔNG thể thử lại (retry cùng ảnh vô nghĩa); nhân viên cần tạo sản phẩm mới với ảnh khác.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST cung cấp cách để người dùng xử lý món `needs_review` (`uncertain_category`) bằng thử lại kèm danh mục: gọi `retry-analysis` với danh mục đã chọn; hệ thống chạy lại phân tích theo danh mục đó (chế độ danh mục cố định) và đưa món sang trạng thái sử dụng được.
- **FR-002**: Điều kiện thành công của thử lại kèm danh mục MUST là: món đang ở trạng thái cần rà soát, thuộc sở hữu người gọi, và danh mục hợp lệ (thuộc nhóm có thể phối đồ); ngược lại MUST trả lỗi nghiệp vụ rõ ràng.
- **FR-003**: Sau khi phân tích lại hoàn tất, món đồ MUST tham gia gợi ý phối đồ và mọi truy vấn "sử dụng được" với **metadata đầy đủ** (màu/chất liệu/phong cách/…/embedding) và **danh mục người dùng chọn được giữ nguyên**.
- **FR-004**: Hệ thống MUST phát sự kiện tiến trình "đang xử lý" cho từng món ngay khi bắt đầu phân tích.
- **FR-005**: Hệ thống MUST phát sự kiện cho trạng thái hoàn tất, thất bại, cần rà soát với đầy đủ: định danh món, chỉ số tiến trình, dữ liệu tóm tắt (với hoàn tất/cần rà soát) và **mã lỗi** (với thất bại; FE tự ánh xạ mã sang text hiển thị — xem FR-030).
- **FR-006**: Hệ thống MUST lưu trữ các sự kiện gần nhất theo từng phiên và phát lại cho client kết nối lại, trong giới hạn cấu hình được; việc lưu trữ MUST đặt sau một lớp trừu tượng transport để có thể chuyển từ bộ nhớ sang kho chia sẻ mà không đổi hành vi client.
- **FR-007**: Thao tác thử lại MUST chỉ được chấp nhận khi món đồ đang ở trạng thái cho phép thử lại (thất bại hoặc cần rà soát); khi đang xử lý hoặc ở trạng thái khác MUST bị từ chối. Hệ thống MUST dùng cơ chế chuyển trạng thái nguyên tử để tránh tạo phiên trùng khi có yêu cầu đồng thời.
- **FR-008**: Khi phân tích lại, hệ thống MUST bảo toàn danh mục hiện có của món đồ (không để AI ghi đè); nếu món chưa có danh mục thì AI được phép điền.
- **FR-009**: Phản hồi khởi tạo thử lại MUST trả về mã phiên (taskId) để client theo dõi realtime.
- **FR-010**: Sản phẩm nhãn hàng MUST phân biệt trạng thái lỗi hệ thống và trạng thái cần duyệt thủ công, kèm lý do tương ứng. Trạng thái lỗi MUST là trạng thái cuối đối với thao tác thủ công: hệ thống MUST từ chối mọi thay đổi trạng thái rời khỏi lỗi, ngoại trừ thao tác thử lại (hoặc do hệ thống thực hiện).
- **FR-011**: Nhân viên nhãn hàng MUST có khả năng thử lại phân tích cho sản phẩm ở trạng thái lỗi, hoặc sản phẩm nháp có lý do cần duyệt; MUST từ chối với sản phẩm nháp thường (không có lý do cần duyệt).
- **FR-012**: Mọi thao tác xử lý cần-rà-soát/thử lại MUST kiểm tra quyền: người dùng sở hữu món đồ, hoặc nhân viên là thành viên hợp lệ của nhãn hàng sở hữu sản phẩm.
- **FR-013**: Hệ thống MUST xử lý idempotent khi nhận sự kiện phân tích trùng lặp (bao gồm cả sự kiện "đang xử lý" mới), không cập nhật dư thừa hay sai lệch trạng thái.
- **FR-014**: Hệ thống MUST cung cấp tài liệu playbook mô tả kịch bản xử lý cho từng trạng thái phân tích, áp dụng cho cả B2C và B2B.
- **FR-015**: Việc thêm trạng thái mới MUST tương thích ngược: client hiện tại không bị phá vỡ khi nhận trạng thái mới.
- **FR-016**: Sự kiện realtime MUST được truyền từ nơi phát sinh tới nơi giữ kết nối client qua một kênh chia sẻ giữa các tiến trình, đảm bảo hoạt động ở CẢ hai topology triển khai: (a) self-hosted/VPS gộp http + worker + consumer trong cùng một tiến trình (`APP_MODE=all`); (b) managed/AWS tách riêng tiến trình http (`APP_MODE=http`) và worker (`APP_MODE=worker`) trên các container khác nhau.
- **FR-017**: Feature MUST tạo một tài liệu tích hợp dành riêng cho frontend, đặt trong thư mục feature `specs/023-analyze-status-handling/` (tên gợi ý `frontend-guide.md`). Tài liệu MUST nêu rõ: (a) từng trạng thái phân tích và kịch bản hành động phía client; (b) các API có thay đổi hành vi/hợp đồng (bao gồm "trước → sau", request/response, trạng thái mới, lỗi mới) kèm ví dụ; (c) luồng realtime (cách đăng ký, payload theo từng trạng thái, kết nối lại/phát lại, mã phiên); (d) checklist việc frontend cần làm. Tài liệu MUST đủ để lập trình viên frontend triển khai mà KHÔNG cần đọc code backend hay tài liệu Swagger.
- **FR-018**: Mọi thay đổi hành vi của API hiện hữu (không chỉ API mới) MUST được liệt kê tường minh trong tài liệu frontend (FR-017) theo dạng "trước → sau" kèm ví dụ payload, để tránh phá vỡ tích hợp hiện tại.
- **FR-019**: Khi một danh mục hoặc từ khóa tìm kiếm được chỉ định nhưng không có fashion item nào khớp, hệ thống MUST trả về danh sách rỗng (tổng số bằng 0); MUST NOT trả về toàn bộ món.
- **FR-020**: Hệ thống MUST phân biệt tường minh "không lọc" với "lọc nhưng rỗng": khi có yêu cầu lọc, bộ lọc MUST luôn được áp dụng kể cả khi danh sách món ứng viên rỗng.
- **FR-021**: Khi có lọc nhưng không có món ứng viên nào, hệ thống MUST trả kết quả rỗng ngay mà KHÔNG truy vấn dữ liệu thừa.
- **FR-022**: Khi KHÔNG lọc, hệ thống MUST trả về tất cả món của người dùng, bao gồm cả món chưa gắn hạng mục trang phục (không có fashion item).
- **FR-023**: Hành vi lọc đúng (FR-019..FR-022) MUST được áp dụng nhất quán cho mọi đường trả danh sách tủ đồ và catalog (kể cả đường tìm kiếm dự phòng khi tìm kiếm nâng cao không khả dụng), không chỉ một luồng đơn lẻ.
- **FR-024**: MUST có test đơn vị bao phủ cho từng đường lọc, gồm tối thiểu: (a) danh mục/từ khóa rỗng ⇒ 0 kết quả; (b) không lọc ⇒ trả toàn bộ.
- **FR-025**: Việc phát sự kiện realtime MUST là best-effort: khi kênh chia sẻ (Redis Pub/Sub) tạm thời không khả dụng, lỗi phát sự kiện MUST chỉ được ghi log cảnh báo và MUST NOT chặn hoặc làm thất bại bất kỳ thao tác nghiệp vụ nào. Trạng thái món đồ MUST vẫn được cập nhật đúng ở DB, và client MUST có khả năng tự làm mới (refetch/polling) để đồng bộ khi không nhận được sự kiện — tối thiểu khi (a) người dùng mở lại màn hình liên quan, hoặc (b) kết nối lại mà không nhận được sự kiện nào trong khoảng chờ hợp lý.
- **FR-026**: Hệ thống MUST cung cấp tín hiệu quan sát cho luồng realtime: ghi log cảnh báo kèm mã phiên khi phát/relay sự kiện lỗi, và số liệu đếm sự kiện phát/relay thành công–thất bại cùng số subscriber đang kết nối theo phiên.
- **FR-027**: Ảnh không hợp lệ (không phải đúng một món: đa món hoặc toàn thân/người) MUST được đưa về trạng thái `failed` kèm lý do mã (`multiple_items_detected` hoặc `full_body_outfit_detected`); với các món/sản phẩm này hệ thống MUST NOT cung cấp thao tác xử lý (không còn `confirm-review`; retry bị từ chối — xem FR-033).
- **FR-028**: `needs_review` MUST chỉ dành cho trường hợp ảnh hợp lệ (đúng một món) nhưng AI chưa chắc danh mục; lý do MUST là `uncertain_category`; hệ thống MUST xử lý bằng `retry-analysis` kèm danh mục hợp lệ (không còn `confirm-review`) theo FR-001/FR-037.
- **FR-029**: Hợp đồng phân tích của AI MUST tách tín hiệu "ảnh có đúng một món" khỏi "độ chắc chắn danh mục" (ví dụ `is_single_item` và `category_confident`) để định tuyến: ảnh không hợp lệ ⇒ `failed`; một món nhưng chưa chắc danh mục ⇒ `needs_review`.
- **FR-030**: Mọi lý do MUST là **mã** (không phải câu thông báo), gồm cả trường `error` trong sự kiện realtime thất bại. Danh mục mã chuẩn: review `uncertain_category`; error `multiple_items_detected`, `full_body_outfit_detected`, `analysis_temporary_error`, `auto_retry_exceeded`.
- **FR-031**: Lý do MUST là thuộc tính của fashion item và MUST chỉ xuất hiện trong đối tượng `fashionItem` lồng nhau của payload API; hệ thống MUST NOT lặp lại lý do ở cấp gốc của wardrobe item hoặc brand item.
- **FR-032**: Các trường của contract DTO fashion item (và category contract DTO) MUST dùng camelCase khi serialize ra API.
- **FR-033**: Thao tác thử lại MUST bị từ chối với món/sản phẩm thất bại do ảnh không hợp lệ; chỉ cho phép với lỗi tạm thời/hết retry tự động hoặc với `needs_review` (`uncertain_category`).
- **FR-034**: Sản phẩm nhãn hàng MUST cũng nhận thông báo hết retry tự động và chuyển sang `failed` kèm lý do, để không bị kẹt ở `draft`.
- **FR-035**: Cơ chế phát hiện món/sản phẩm xử lý quá hạn MUST dựa trên trạng thái xử lý do fashion item sở hữu (quét theo fashion item) và MUST áp dụng cho cả wardrobe lẫn brand.
- **FR-036**: Khi phân tích thành công hoặc khi bắt đầu phân tích lại, lý do cũ MUST được xóa; khi thất bại, lý do MUST được ghi đúng mã tương ứng — bao gồm cả trường hợp thử lại thất bại ở bước publish job (MUST ghi `analysis_temporary_error`).
- **FR-037**: `retry-analysis` MUST nhận danh mục tùy chọn; với món `needs_review` (`uncertain_category`) MUST yêu cầu danh mục hợp lệ (từ request, hoặc danh mục có sẵn trên fashion item) và MUST validate danh mục thuộc nhóm có thể phối đồ.
- **FR-038**: Khi thử lại kèm danh mục, hệ thống MUST chạy phân tích ở chế độ **danh mục cố định**: chỉ trích xuất metadata, KHÔNG phân loại lại danh mục, KHÔNG áp gate `is_single_item`/`category_confident`, và MUST giữ nguyên danh mục đã chọn.
- **FR-039**: Hệ thống MUST NOT còn cung cấp API `confirm-review`; việc bỏ endpoint này là thay đổi phá vỡ hợp đồng cũ và MUST được ghi rõ trong tài liệu frontend (FR-017/FR-018).
- **FR-040**: Prompt phân tích ảnh MUST "fail-fast": khi ảnh không hợp lệ hoặc danh mục chưa chắc, AI MUST trả JSON tối giản (không sinh metadata dài); lượt phân tích ảnh ban đầu MUST giữ một lần gọi KHÔNG bật `ResponseSchema` để fail-fast phát huy tác dụng.
- **FR-041**: Sản phẩm nhãn hàng MUST nhận danh mục qua chính request `retry-analysis` (do chưa có API đặt danh mục khác) và áp dụng cùng cơ chế danh mục cố định.

### Key Entities *(include if feature involves data)*

- **WardrobeItem (món đồ cá nhân)**: đại diện một trang phục của người dùng; vòng đời phân tích gồm đang xử lý → sử dụng được / thất bại / cần rà soát.
- **BrandItem (sản phẩm nhãn hàng)**: đại diện sản phẩm/mẫu thử của nhãn hàng; trạng thái gồm nháp, hoạt động, lưu trữ, và (mới) lỗi.
- **AnalyzeTask (phiên phân tích)**: một phiên xử lý hàng loạt gắn nhiều món; có định danh phiên và chỉ số tiến trình.
- **AnalyzeTaskStatus (trạng thái phân tích)**: đang xử lý, hoàn tất, thất bại, cần rà soát.
- **Reason (lý do)**: **mã** thuộc catalog chuẩn (review/error), thuộc sở hữu fashion item; chỉ hiển thị trong đối tượng `fashionItem` lồng nhau, không lặp ở cấp gốc.
- **RealtimeEvent (sự kiện realtime)**: bản tin tiến trình gửi tới client, có thể được lưu lại tạm thời để phát lại.
- **FrontendGuide (tài liệu tích hợp frontend)**: tài liệu trong thư mục feature, mô tả kịch bản hành động của client, các thay đổi API "trước → sau", và luồng realtime — đủ để frontend tự triển khai.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% món đồ ở trạng thái cần rà soát có thể chuyển sang trạng thái sử dụng được thông qua thử lại kèm danh mục hợp lệ.
- **SC-002**: Tối thiểu 99% món đồ trong một phiên phân tích phát sinh đầy đủ sự kiện "đang xử lý" và sự kiện kết thúc tới client đang theo dõi.
- **SC-003**: 100% sự kiện đã phát (trong giới hạn lưu trữ) được phát lại cho client kết nối lại, không cần tải lại toàn bộ dữ liệu.
- **SC-004**: 100% yêu cầu thử lại khi trạng thái không cho phép bị chặn; các yêu cầu đồng thời trên cùng một món không tạo phiên trùng.
- **SC-005**: 100% chỉnh sửa danh mục thủ công của người dùng được giữ nguyên sau khi hệ thống phân tích lại.
- **SC-006**: 100% sản phẩm nhãn hàng lỗi/cần duyệt hiển thị đúng trạng thái và lý do tương ứng.
- **SC-007**: Tài liệu playbook bao phủ 100% các trạng thái phân tích và các luồng xử lý tương ứng cho cả B2C và B2B.
- **SC-008**: Không có thay đổi phá vỡ hợp đồng hiện tại với client (tương thích ngược).
- **SC-009**: Ở cả hai topology (gộp tiến trình self-hosted/VPS và tách http/worker managed/AWS), 100% sự kiện realtime phát sinh trong quá trình xử lý nền tới được client đang kết nối.
- **SC-010**: Tồn tại một tài liệu frontend trong thư mục feature bao phủ 100% trạng thái, API đổi hành vi, và luồng realtime; một lập trình viên frontend có thể hoàn tất tích hợp chỉ bằng tài liệu đó mà không cần đọc code backend hay Swagger.
- **SC-011**: 100% truy vấn lọc theo danh mục/từ khóa không khớp món nào trả về 0 kết quả (không bao giờ trả toàn bộ).
- **SC-012**: Test đơn vị cho mọi đường lọc bao phủ 100% hai case: kết quả rỗng ⇒ 0, và không lọc ⇒ toàn bộ (kể cả món chưa gắn hạng mục trang phục).
- **SC-013**: Khi kênh realtime gián đoạn, 100% thao tác nghiệp vụ (tải lên, thử lại) vẫn thành công và trạng thái vẫn được ghi đúng ở DB; tỉ lệ thất bại nghiệp vụ do lỗi phát sự kiện bằng 0%.
- **SC-014**: 100% lỗi phát/relay sự kiện realtime có bản ghi log cảnh báo kèm mã phiên; tồn tại số liệu đo được tỉ lệ phát/relay thành công–thất bại và số subscriber theo phiên để vận hành phát hiện mất sự kiện.
- **SC-015**: 100% ảnh không hợp lệ (đa món/toàn thân) ở trạng thái `failed` với mã lý do tương ứng; 0% ảnh không hợp lệ vào được trạng thái sử dụng.
- **SC-016**: 100% `needs_review` có lý do `uncertain_category` và xử lý được bằng thử lại kèm danh mục; 0% `needs_review` phát sinh từ ảnh không hợp lệ.
- **SC-019**: 100% món `needs_review` xử lý bằng thử lại kèm danh mục hợp lệ chuyển sang sử dụng được với **metadata đầy đủ** và giữ nguyên danh mục đã chọn.
- **SC-020**: 0% trường hợp thử lại ở chế độ danh mục cố định tái sinh `needs_review` (không vòng lặp).
- **SC-021**: Endpoint `confirm-review` không còn tồn tại (0 endpoint); tài liệu frontend nêu rõ thay đổi phá vỡ này.
- **SC-017**: 100% trường lý do trong payload API và sự kiện realtime (`reviewReason`/`processingErrorReason`/SSE `error`) là mã thuộc catalog chuẩn; 0 trường hợp trả câu thông báo tự do (không áp dụng cho trường `message` của `APIResponse`).
- **SC-018**: 100% món/sản phẩm hết retry tự động (kể cả brand) chuyển sang `failed` kèm lý do, không kẹt ở trạng thái xử lý/nháp.

## Assumptions

- Có hai topology triển khai: self-hosted/VPS chạy 3 container dịch vụ với `APP_MODE=all` (http + worker + consumer chung một tiến trình); managed/AWS chạy 6 container tách riêng http (`APP_MODE=http`) và worker (`APP_MODE=worker`). Feature MUST hoạt động đúng ở cả hai, nên sự kiện realtime MUST đi qua kênh chia sẻ giữa tiến trình (Redis Pub/Sub); buffer phát lại nằm trong tiến trình http và đặt sau lớp trừu tượng transport.
- Redis là hạ tầng đã có sẵn ở mọi môi trường (self-hosted và managed); việc dùng Redis Pub/Sub không thêm nhà cung cấp mới.
- Thao tác thử lại không giới hạn theo số lần/thời gian, nhưng MUST bị chặn theo lý do: không cho thử lại khi thất bại do ảnh không hợp lệ (đa món/toàn thân) vì retry cùng ảnh là vô nghĩa; `needs_review` chỉ cho thử lại khi kèm danh mục hợp lệ; lỗi tạm thời/hết retry tự động cho thử lại tự do. Chi phí AI được kiểm soát bởi chính sách quota AI hiện hữu.
- Lý do (reason) được chuẩn hóa thành mã thuộc catalog cố định và do fashion item sở hữu; frontend tự ánh xạ mã sang text hiển thị. Không thêm trạng thái mới cho ảnh không hợp lệ (dùng `failed` + mã lý do).
- Giới hạn lưu trữ sự kiện realtime mặc định khoảng 100 sự kiện mỗi phiên, cấu hình được; buffer được giữ theo TTL cấu hình (mặc định vài phút) để hỗ trợ reconnect muộn, không xóa ngay khi hết subscriber.
- Thao tác xử lý món đồ cần rà soát KHÔNG còn dùng API `confirm-review`; thay bằng `retry-analysis` kèm danh mục (có chạy lại AI ở chế độ danh mục cố định, bất đồng bộ, trả `taskId`).
- Việc mở rộng trạng thái nhãn hàng với "lỗi" không yêu cầu di trú dữ liệu phá vỡ; cần xác nhận ràng buộc dữ liệu hiện hành.
- Cơ chế phát sự kiện bất đồng bộ hiện hữu được tái sử dụng; feature này mở rộng theo pattern sẵn có.
- Người dùng và nhân viên nhãn hàng có kết nối mạng ổn định trong quá trình thao tác.
- Hệ thống chỉ có hai vai trò hệ thống (`admin`, `user`); không thêm vai trò mới (theo constitution).
- Các sự kiện realtime hiện được định danh theo phiên; feature giữ nguyên mô hình này và bổ sung lưu trữ/phát lại.
