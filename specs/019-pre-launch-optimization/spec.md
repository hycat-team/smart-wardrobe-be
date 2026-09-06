# Đặc tả Tính Năng: Tối Ưu Hóa Pre-Launch — Hardening, Hiệu Năng và Độ Bảo Trì

**Feature Branch**: `019-pre-launch-optimization`

**Created**: 2026-09-06

**Status**: Draft

**Input**: User description: "Tối ưu dự án backend trước khi launching: tốc độ app + coding standard để tối ưu khả năng bảo trì. Gộp toàn bộ hạng mục audit vào một spec duy nhất: (1) HTTP server timeouts + request body size limit, (2) test suite cho subscription billing + quota enforcement + auth RBAC, (3) cache subscription overview trong AI hot path với invalidation qua event, (4) gzip compression + chính sách migration khi chạy nhiều instance, (5) hygiene pass: tách 2 file ~500 dòng + dọn fmt.Println."

---

## Bối Cảnh & Vấn Đề

Kết quả audit pre-launch (2026-09-06) cho thấy nền tảng có nhiều điểm mạnh sẵn có (rate limiting, global timeout middleware, CORS, panic recovery, outbox LISTEN/NOTIFY + health-check backlog, index composite chuẩn, AI client timeout riêng từng purpose, 0 TODO debt), nhưng tồn tại 5 nhóm vấn đề cần xử lý trước khi mở cửa đón người dùng thật:

1. **HTTP server chưa được harden**: server HTTP hiện không cấu hình bất kỳ timeout kết nối nào (đọc/ghi/idle) — nguy cơ slowloris và kết nối treo tích tụ khi ra ngoài internet; đồng thời không có giới hạn kích thước request body, khiến endpoint nhận JSON lớn (batch upload tủ đồ, chat message) có thể bị lạm dụng bộ nhớ.
2. **Lỗ hổng test ở luồng rủi ro nhất**: 3 module (admin, aiexecutor, notification) không có test nào; module subscription (chịu trách nhiệm billing/payment) chỉ có 4 file test; luồng kiểm soát hạn mức AI (quota) và phân quyền (auth/RBAC) cũng thiếu kiểm thử tự động. Đây là rủi ro tiền bạc và lạm dụng tài nguyên khi launch.
3. **AI hot path truy vấn CSDL dư thừa**: mỗi request phối đồ/chat AI gọi xuyên module 2 lần (lấy hạn mức + lấy tổng quan gói cước) trước khi gọi AI provider, làm tăng độ trễ và tải DB khi người dùng tăng.
4. **Thiếu nén phản hồi và chính sách migration đa instance**: phản hồi JSON lớn (dashboard, danh sách) chưa được nén; cơ chế tự chạy migration khi khởi động app tiềm ẩn race condition nếu deploy nhiều replica đồng thời.
5. **Nợ vệ sinh mã nguồn**: 2 file production chạm ngưỡng cảnh báo 500 dòng của code-hygiene.md; một hàm vượt 80 dòng; còn sót vài lệnh in thô (fmt.Println) chưa đi qua hệ thống logger tập trung.

Mục tiêu: xử lý trọn gói 5 nhóm trên trong một đợt duy nhất, KHÔNG thay đổi bất kỳ cấu trúc JSON API công khai nào (zero breaking change), KHÔNG vi phạm Hiến pháp (§II ranh giới module, §III event-driven, §IV phân tầng, §VI pipeline AI).

## Clarifications

### Session 2026-09-06

- Q: Mức kiểm thử cho các luồng billing/webhook/quota (US2) là gì? → A: Unit test cho toàn bộ use case (mock repo/contract) chạy trong test suite chuẩn + integration test với DB thật bắt buộc cho riêng webhook reconciliation và renewal workflow.
- Q: Mức observability cho các cơ chế mới là gì? → A: Log có cấu trúc cho các sự kiện vận hành mới (body-limit, migration lock, cache invalidation) tái sử dụng logger/metrics hiện có; không xây hệ thống metrics mới.
- Q: Có route nào cần miễn ngưỡng body 2 MB không? → A: Áp mô hình 3 tầng: 2 MB toàn cục; 4 nhóm route (chat AI, phối đồ AI, chat user↔brand member, brand mutation + route chứa field description) áp ngưỡng nâng 10 MB; ngoài ra phải setup sẵn cơ chế registry khai báo route "free" (bỏ qua hoàn toàn ngưỡng) — hiện đang TRỐNG — để thêm route free về sau mà không đụng code middleware.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Harden HTTP Server: Timeouts và Giới Hạn Body (Priority: P1)

Là một vận hành hệ thống, tôi muốn server HTTP từ chối kết nối/đọc treo vô hạn và từ chối request body vượt ngưỡng, để kẻ xấu hoặc client lỗi không thể cạn kiệt tài nguyên của hệ thống khi mở ra public internet.

**Why this priority**: Đây là lỗ hổng ổn định/bảo mật mức launch-critical — không xử lý thì mọi tối ưu khác đều đứng trên nền không vững; chi phí thực hiện thấp, tác động cao nhất.

**Independent Test**: Có thể kiểm thử độc lập bằng cách gửi một kết nối mở nhưng không gửi dữ liệu (slowloris-style) và một request có body vượt ngưỡng — quan sát kết nối bị ngắt theo timeout cấu hình và nhận phản hồi lỗi chuẩn 413, trong khi các API khác hoạt động bình thường.

**Acceptance Scenarios**:

1. **Given** một client mở kết nối nhưng không gửi dữ liệu, **When** thời gian vượt quá ngưỡng đọc được cấu hình, **Then** hệ thống đóng kết nối và ghi nhận sự kiện, không để kết nối treo vô hạn.
2. **Given** một client gửi request JSON lớn hơn ngưỡng cho phép, **When** server nhận request, **Then** server trả về lỗi 413 theo đúng định dạng lỗi chuẩn của hệ thống (tiếng Việt, cấu trúc thống nhất) và không xử lý tiếp business logic.
3. **Given** người dùng bình thường gửi request hợp lệ, **When** request đi qua hệ thống, **Then** hành vi và cấu trúc phản hồi không thay đổi so với trước (zero breaking change).
4. **Given** một endpoint streaming (phân tích ảnh thời trang theo thời gian thực), **When** stream đang truyền dữ liệu liên tục, **Then** stream không bị cắt oan do ngưỡng ghi kết nối (cơ chế timeout phải tương thích với streaming).

---

### User Story 2 - Bộ Test Bảo Đảm Luồng Tiền và Phân Quyền (Priority: P1)

Là một leader kỹ thuật, tôi muốn các luồng billing/payment, kiểm soát hạn mức AI và phân quyền được bao phủ bởi bộ test tự động, để những thay đổi trong tương lai không âm thầm làm hỏng tiền của người dùng hoặc mở hố phân quyền khi hệ thống đã live.

**Why this priority**: Rủi ro trực tiếp tới doanh thu (subscription purchase/renewal/webhook reconciliation) và lạm dụng tài nguyên (quota AI); bộ test là mạng an toàn bắt buộc trước khi chấp nhận tiền thật từ người dùng.

**Independent Test**: Có thể kiểm thử độc lập bằng cách chạy toàn bộ test suite mới (`make test-race`) — các test mô phỏng happy path và edge case của mua gói, gia hạn, đối soát webhook, vượt hạn mức và các cặp vai trò truy cập admin/brand portal đều pass mà không cần sửa code production.

**Acceptance Scenarios**:

1. **Given** một người dùng thanh toán mua gói thành công qua cổng thanh toán, **When** webhook đối soát được xử lý, **Then** test xác nhận trạng thái giao dịch và quyền lợi gói được cập nhật đúng, giao dịch trùng lặp bị loại bỏ an toàn.
2. **Given** một gói finite sắp hết hạn, **When** worker gia hạn chạy, **Then** test xác nhận luồng gia hạn thành công, gia hạn thất bại và fallback về gói mặc định đều có kết quả đúng.
3. **Given** một người dùng free đã dùng hết hạn mức phối đồ trong ngày, **When** họ gọi API phối đồ AI, **Then** test xác nhận bị từ chối với lỗi hạn mức; khi reset hạn mức theo ngày, quyền sử dụng được khôi phục đúng.
4. **Given** các vai trò hệ thống (admin/user) và vai trò B2B (brand owner/staff/guest), **When** họ truy cập các nhóm route admin và brand portal, **Then** test xác nhận ma trận cho/phê duyệt khớp với ma trận phân quyền đã tài liệu hóa; truy cập trái phép bị chặn thống nhất.

---

### User Story 3 - Cache Tổng Quan Gói Cước Trong AI Hot Path (Priority: P2)

Là một người dùng premium đang dùng tính năng phối đồ AI, tôi muốn thời gian chờ phản hồi không bị cộng thêm chi phí truy vấn dữ liệu gói cước lặp lại mỗi lần gọi, để trải nghiệm AI nhanh hơn đặc biệt vào giờ cao điểm.

**Why this priority**: Giảm trực tiếp 1 trong 2 lượt truy vấn chéo module trên mỗi request AI — hiệu năng rõ rệt với rủi ro thấp, nhưng phải bảo đảm tuyệt đối tính đúng đắn của hạn mức (không được cache số đếm hạn mức) nên đặt sau 2 story P1.

**Independent Test**: Có thể kiểm thử độc lập bằng cách đo số lượt truy vấn gói cước trên N request AI liên tiếp — khi cache còn hạn, chỉ có 1 lượt truy vấn quota được thực hiện mỗi request; sau khi mua/gia hạn gói, dữ liệu gói mới có hiệu lực trong khoảng thời gian TTL ngắn hoặc ngay lập tức qua sự kiện.

**Acceptance Scenarios**:

1. **Given** một người dùng gọi 5 request phối đồ liên tiếp trong 1 phút, **When** tổng quan gói của họ không đổi, **Then** hệ thống chỉ truy vấn tổng quan gói tối đa 1 lần (cache hit cho các lần sau), số đếm hạn mức vẫn được ghi chính xác từng request.
2. **Given** một người dùng vừa mua/gia hạn thành công gói mới, **When** sự kiện thay đổi gói được phát ra, **Then** bản cache tổng quan gói của người dùng đó bị làm mới, quyền lợi mới có hiệu lực trong thời gian tối đa bằng TTL cấu hình (mặc định ≤ 60 giây).
3. **Given** cache bị lỗi hoặc trống, **When** request AI đi qua, **Then** hệ thống fallback đọc trực tiếp từ module subscription (không bao giờ fail request vì cache).
4. **Given** hai người dùng khác nhau, **When** cùng lúc gọi AI, **Then** cache phải phân tách theo người dùng, không rò rỉ quyền lợi giữa các tài khoản.

---

### User Story 4 - Nén Phản Hồi và Chính Sách Migration Đa Instance (Priority: P2)

Là một vận hành hệ thống, tôi muốn phản hồi JSON lớn được nén khi client hỗ trợ, và việc chạy nhiều bản thể phục vụ đồng thời không gây tranh chấp migration, để băng thông tiết kiệm và quy trình deploy an toàn khi scale ngang.

**Why this priority**: Lợi ích hiệu năng/băng thông rõ ràng nhưng phụ thuộc cấu hình hạ tầng thực tế (reverse proxy); migration lock là bảo hiểm cho đợt scale sắp tới, chưa chặn launch nếu deploy đơn instance.

**Independent Test**: Có thể kiểm thử độc lập bằng cách gọi một API danh sách với header chấp nhận nén — phản hồi nén đúng định dạng và giảm mạnh kích thước; và chạy hai tiến trình server khởi động cùng lúc — chỉ một tiến trình thực thi migration, tiến trình còn lại chờ hoặc bỏ qua an toàn.

**Acceptance Scenarios**:

1. **Given** client gửi header chấp nhận nén, **When** nhận phản hồi JSON từ API dashboard/danh sách, **Then** kích thước truyền giảm ít nhất 50% so với không nén và nội dung giải nén giữ nguyên 100% cấu trúc.
2. **Given** client không hỗ trợ nén hoặc phản hồi là dữ liệu stream thời gian thực, **When** phản hồi đi qua, **Then** hệ thống trả về dữ liệu không nén/stream nguyên vẹn (không phá vỡ SSE).
3. **Given** hai bản thể server khởi động đồng thời trên cùng CSDL, **When** migration chạy, **Then** chỉ một bản thể thực thi, bản thể kia chờ đến lượt hoặc bỏ qua an toàn; không có lỗi xung đột schema.
4. **Given** một bản thể bị chết giữa chừng khi giữ quyền migration, **When** bản thể khác tiếp quản, **Then** khóa migration tự động được nhả sau thời gian chờ cấu hình và việc deploy tiếp diễn bình thường.

---

### User Story 5 - Vệ Sinh Mã Nguồn: Tách File Lớn và Thống Nhất Logger (Priority: P3)

Là một lập trình viên bảo trì dự án, tôi muốn các file chạm ngưỡng cấu trúc được tách theo nghiệp vụ và các lệnh in thô được thay bằng logger tập trung, để những thay đổi tiếp theo sau launch dễ đọc, dễ review và log vận hành hiển thị thống nhất một nơi.

**Why this priority**: Không ảnh hưởng hành vi người dùng, thuần chất lượng bảo trì — làm cuối để không cản trở các hạng mục P1/P2, nhưng cần gọn trước khi code base phình thêm tính năng mới.

**Independent Test**: Có thể kiểm thử độc lập bằng cách chạy kiểm tra cấu trúc (hygiene scan) — 2 file đích nằm dưới ngưỡng cảnh báo 500 dòng, các hàm tách đều dưới 80 dòng, và toàn bộ test vẫn pass sau khi tách (pure refactor, không đổi hành vi).

**Acceptance Scenarios**:

1. **Given** handler tủ đồ (501 dòng) và workflow gia hạn subscription (495 dòng) hiện tại, **When** tách theo nhóm nghiệp vụ, **Then** mỗi file kết quả dưới 500 dòng, các hàm dưới 80 dòng, hành vi API và business logic không đổi, toàn bộ test pass.
2. **Given** các lệnh in thô còn sót ở bootstrap và migration, **When** thay bằng logger tập trung, **Then** log vận hành xuất hiện ở cùng định dạng/đích với các log khác, không còn dòng in nào bypass logger trong code production.

---

### Edge Cases

- **Streaming SSE bị cắt bởi timeout ghi kết nối**: ngưỡng ghi server phải không cắt các stream hợp lệ đang truyền dữ liệu liên tục (phân tích ảnh); nếu xung đột, endpoint streaming cần được miễn hoặc dùng cơ chế riêng.
- **Request body sát ngưỡng**: hệ thống phải từ chối rõ ràng (413) thay vì âm thầm cắt cụt JSON giữa chừng gây lỗi parse khó hiểu.
- **Route thuộc tầng nâng 10 MB và registry free**: các route tầng nâng vẫn có giới hạn (10 MB) nhưng payload lớn hơn bình thường — rào chắn chính là validator cấp field (giới hạn ký tự/token) và phải được test phủ trong phạm vi FR-005/FR-006 khi liên quan; registry route "free" hiện trống — khi thêm route free sau này, phải review riêng vì route đó bỏ hẳn rào chắn body.
- **Cache tổng quan gói stale khi người dùng nâng cấp giữa chừng request AI**: hạn mức ngày vẫn đọc trực tiếp (không cache), chỉ tổng quan gói được cache TTL ngắn — sai lệch tối đa 1 chu kỳ TTL và tự phục hồi qua event invalidation.
- **Cache tầng lỗi**: backend cache không khả dụng → fallback đọc trực tiếp, không panic, không tăng latency đáng kể.
- **Migration lock bị kẹt do instance chết**: khóa phải có thời hạn tự nhả (advisory lock có session lifetime), không bế tắc vĩnh viễn quá trình deploy.
- **Nén với dữ liệu đã nhỏ hoặc đã nén sẵn (ảnh base64)**: hệ thống bỏ qua nén khi không có lợi (dưới ngưỡng tối thiểu), tránh tốn CPU vô ích.
- **Tách file thuần túy**: quá trình tách không được đổi tên bất kỳ DTO/API nào đã chuẩn hóa theo spec 018 (nếu trùng phạm vi, giữ nguyên ranh giới file đã cam kết).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Server HTTP MUST áp dụng ngưỡng thời gian đọc/ghi/idle cho mọi kết nối, giá trị lấy từ cấu hình và có mặc định hợp lý cho môi trường production (không cho phép giá trị 0 = vô hạn ở production).
- **FR-002**: Server MUST áp dụng mô hình giới hạn body 3 tầng, mỗi tầng trả về mã lỗi 413 theo định dạng lỗi chuẩn của hệ thống khi vượt ngưỡng: (i) ngưỡng mặc định toàn cục 2 MB; (ii) ngưỡng nâng 10 MB cho 4 nhóm route: user chat với chatbot AI, user phối đồ AI, chat giữa user và brand member (cả hai chiều), các route brand mutation và route chứa field nhập description; (iii) registry khai báo route "free" (bỏ qua hoàn toàn ngưỡng) — cơ chế này MUST được setup sẵn, hiện TRỐNG, và cho phép thêm route mới chỉ bằng khai báo mà không sửa logic middleware. Plan MUST liệt kê pattern route cụ thể cho từng nhóm ở tầng nâng; các route ở tầng nâng bù trừ rủi ro payload lớn bằng cơ chế giới hạn riêng cấp field sẵn có (validator binding, giới hạn token).
- **FR-003**: Cơ chế timeout kết nối MUST không phá vỡ các luồng dữ liệu streaming hiện có (phân tích ảnh SSE); endpoint streaming phải tiếp tục hoạt động nguyên vẹn sau thay đổi.
- **FR-004**: Module subscription MUST có bộ test tự động bao phủ: mua gói trực tiếp, gia hạn (thành công/thất bại/fallback), đối soát webhook thanh toán (kể cả giao dịch trùng/tái phát), và apply quyền lợi gói.
- **FR-005**: Luồng kiểm soát hạn mức AI MUST có bộ test tự động bao phủ: từ chối khi vượt hạn mức, reset theo ngày, và ghi nhận sử dụng sau mỗi lần gọi.
- **FR-006**: Các nhóm route admin và brand portal MUST có bộ test phân quyền xác nhận ma trận cho/phê theo vai trò (system admin/user; brand owner/staff; guest) khớp tài liệu phân quyền hiện hành.
- **FR-007**: Tổng quan gói cước phục vụ luồng AI MUST được phục vụ từ cache phân tách theo người dùng với TTL mặc định ≤ 60 giây (cấu hình được); số đếm hạn mức hàng ngày TUYỆT ĐỐI KHÔNG được cache và vẫn ghi chính xác từng request.
- **FR-008**: Cache tổng quan gói MUST bị làm mới NGAY khi có mutation làm thay đổi nội dung overview — thực hiện bằng lời gọi xóa cache trực tiếp (intra-module) tại các điểm mutation trong module subscription (apply sau purchase, renewal, admin set plan active, toggle auto-renew, webhook reconciliation apply); TTL ngắn đóng vai trò backstop cho mọi đường mutation bị sót. Khi cache lỗi/trống hệ thống MUST fallback đọc trực tiếp, không làm fail request AI.
- **FR-009**: Phản hồi HTTP dạng JSON MUST được nén khi client khai báo hỗ trợ, với ngưỡng kích thước tối thiểu để tránh nén vô ích; dữ liệu stream thời gian thực không được nén; hành vi khi client không hỗ trợ giữ nguyên 100%.
- **FR-010**: Việc thực thi migration khi khởi động MUST an toàn trong kịch bản nhiều bản thể đồng thời (khóa phân tán hoặc cơ chế tương đương có thời hạn tự nhả), cấu hình được và mặc định bật ở production.
- **FR-011**: File production `wardrobe_item_handler` và `subscription_renewal_workflow` MUST được tách theo nhóm nghiệp vụ để mọi file kết quả dưới 500 dòng và mọi hàm dưới 80 dòng, giữ nguyên 100% hành vi (pure refactor).
- **FR-012**: Toàn bộ lệnh in thô trong code production MUST được thay bằng logger tập trung của hệ thống.
- **FR-013**: Toàn bộ thay đổi MUST không làm thay đổi cấu trúc JSON của bất kỳ API công khai nào (swagger chỉ được thêm/cải tiến nội bộ, không được xóa/đổi field).
- **FR-014**: Chính sách cấp độ test: các use case trong phạm vi FR-004..FR-006 MUST có unit test (mock repository/contract) chạy trong test suite chuẩn; webhook reconciliation và renewal workflow BẮT BUỘC có thêm integration test chạy trên CSDL thật để chứng minh hành vi transaction/lock.
- **FR-015**: Các cơ chế mới trong feature này MUST phát log có cấu trúc cho sự kiện vận hành quan trọng (từ chối body-limit, acquire/release khóa migration, invalidation cache tổng quan gói) thông qua logger hiện có; KHÔNG xây dựng hệ thống metrics mới.

### Key Entities *(include if feature involves data)*

- **Cấu hình Server Hardening**: bộ ngưỡng thời gian đọc/ghi/idle và kích thước body tối đa, cấu hình theo môi trường, có mặc định an toàn cho production.
- **Cache Tổng Quan Gói Cước**: bản ghi theo người dùng chứa quyền lợi gói hiện hành (không chứa số đếm hạn mức), có TTL ngắn và bị làm mới bởi sự kiện thay đổi gói; nguồn chân lý vẫn là module subscription.
- **Khóa Migration**: cơ chế đảm bảo chỉ một bản thể chạy migration tại một thời điểm, có thời hạn tự nhả khi bản thể giữ khóa chết.
- **Bộ Test Billing/Quota/RBAC**: các kịch bản kiểm thử tự động ánh xạ 1-1 vào user story 2, chạy được trong lệnh test chuẩn của dự án.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% kết nối treo/không gửi dữ liệu bị đóng tự động trong thời gian bằng đúng ngưỡng cấu hình (kiểm chứng bằng test slowloris mô phỏng); không còn kết nối giữ vô hạn.
- **SC-002**: Request vượt ngưỡng body nhận 413 chuẩn trong 100% trường hợp và không tiêu tốn tài nguyên xử lý business logic; các request hợp lệ không bị ảnh hưởng (0 false-positive trên test suite hiện hữu).
- **SC-003**: Trên chuỗi N request AI liên tiếp của cùng người dùng trong cửa sổ TTL, lượt truy vấn tổng quan gói giảm từ N xuống ≤ 1; tính đúng đắn hạn mức giữ nguyên 100% (0 case vượt hạn mức sai).
- **SC-004**: Các luồng mua gói, gia hạn, webhook, hạn mức và phân quyền đều có test tự động pass trong `make test-race`; mỗi luồng phủ ít nhất 1 happy path + 2 edge case nêu trong acceptance scenarios.
- **SC-005**: Kích thước truyền phản hồi JSON của API dashboard/danh sách giảm ≥ 50% khi client hỗ trợ nén; nội dung giải nén bất biến (diff 100% khớp).
- **SC-006**: Hai bản thể khởi động đồng thời không gây xung đột migration trong 100% lần thử mô phỏng; thời gian deploy không tăng quá 10% so với hiện tại.
- **SC-007**: Kiểm tra cấu trúc sau refactor: 0 file production trong phạm vi tách vượt 500 dòng, 0 hàm vượt 80 dòng, 0 lệnh in thô còn lại trong production code; toàn bộ test hiện hữu + mới pass.
- **SC-008**: Swagger diff sau toàn bộ feature chỉ chứa thay đổi phi phá vỡ (không xóa/đổi field nào của API công khai).

## Assumptions

- API hiện chỉ nhận JSON qua body; ảnh/phương tiện được client upload trực tiếp lên dịch vụ lưu trữ bên ngoài (Cloudinary) qua chữ ký server cấp — nên ngưỡng body mặc định 2 MB là đủ dư địa và không ảnh hưởng luồng upload.
- Hệ thống sẽ tiếp tục deploy phía sau một gateway/reverse proxy trong tương lai; do đó nén được triển khai ở tầng app nhưng cấu hình bật/tắt được, và route stream thời gian thực được loại trừ.
- Giá trị mặc định timeouts đề xuất: đọc 30s / ghi 60s (đủ cho SSE dài) / idle 120s; mọi giá trị đều cấu hình qua hệ thống config hiện hành và được tài liệu hóa trong spec kế hoạch (plan).
- Cache tổng quan gói đặt ở tầng application của module subscription (cấp qua contract hiện hành), tuân thủ Hiến pháp §II (không import chéo ngoài danh mục cho phép) và §IV (application → domain).
- Invalidation cache tổng quan gói thực hiện bằng lời gọi trực tiếp trong cùng module (không đi qua RabbitMQ) — quyết định thiết kế chi tiết tại research.md D4; sự kiện `PlanChanged` hiện hành giữ nguyên vai trò riêng của nó (grant AI) và không bị đụng tới.
- Phạm vi test của story 2 giới hạn ở các luồng nêu trong FR-004..FR-006; không yêu cầu coverage % theo dòng — yêu cầu là phủ các kịch bản nghiệp vụ liệt kê.
- Việc tách file ở story 5 là pure refactor theo code-hygiene.md, không kèm tối ưu logic; các file khác ngoài 2 file đích không bắt buộc xử lý trong spec này (tránh scope creep).
