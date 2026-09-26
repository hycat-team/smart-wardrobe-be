# Feature Specification: Xử lý kịch bản theo trạng thái phân tích trang phục AI

**Feature Branch**: `023-analyze-status-handling`

**Created**: 2026-09-26

**Status**: Draft

**Input**: User description: "Chuẩn hóa và bổ sung kịch bản xử lý cho từng trạng thái phân tích AI của trang phục (đang xử lý, hoàn tất, thất bại, cần rà soát) ở cả phía người dùng cá nhân (B2C) và nhãn hàng (B2B), gồm: xác nhận lại món đồ cần rà soát để đưa vào sử dụng, thông báo tiến trình realtime đầy đủ (kể cả trạng thái đang xử lý), chính sách thử lại an toàn (cooldown, giới hạn số lần, giữ chỉnh sửa thủ công), phân biệt lỗi và trạng thái nhãn hàng, khả năng phát lại sự kiện khi client kết nối lại, và tài liệu playbook vận hành."

## Clarifications

### Session 2026-09-26

- Q: Xác nhận món "cần rà soát" có bắt buộc phải có danh mục hợp lệ không? → A: Có — bắt buộc danh mục hợp lệ (thuộc nhóm có thể phối đồ) mới xác nhận được; nếu thiếu/không hợp lệ thì từ chối và yêu cầu chọn danh mục.
- Q: Quyền đặt/đổi trạng thái "lỗi" của sản phẩm nhãn hàng? → A: "Lỗi" là trạng thái cuối với thao tác thủ công — hệ thống từ chối mọi thay đổi trạng thái rời khỏi "lỗi"; chỉ thao tác thử lại (hoặc hệ thống) mới đưa sản phẩm ra khỏi "lỗi".
- Q: Cơ chế truyền sự kiện realtime giữa worker (phát sinh) và tầng HTTP (giữ kết nối client) khi khác tiến trình? → A: Dùng Redis Pub/Sub làm kênh trung gian worker→http, kết hợp buffer in-memory trong tiến trình HTTP để phát lại cho client; đặt sau lớp trừu tượng transport để có thể thay bằng kho chia sẻ sau này.
- Q: Có cần chặn thử lại theo số lần/thời gian (cooldown) không? → A: Không — chỉ cần kiểm tra món đồ đang ở trạng thái cho phép thử lại (thất bại hoặc cần rà soát); không thêm cột đếm/thời gian cho mục đích này.
- Q: Trạng thái nào cho phép thử lại phân tích ở sản phẩm nhãn hàng? → A: Cho phép khi sản phẩm ở trạng thái lỗi, hoặc ở trạng thái nháp nhưng có lý do cần rà soát (`reviewReason`); không cho phép với nháp thường.
- Q: Cơ chế xác định "danh mục người dùng đã sửa" để bảo toàn khi phân tích lại? → A: Bảo toàn khi fashion item đã có danh mục (không cần cờ riêng); nếu chưa có danh mục thì AI được phép điền.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Rà soát và xác nhận món đồ AI không chắc chắn (Priority: P1)

Khi người dùng tải ảnh trang phục lên, AI có thể không đủ tự tin để phân loại món đồ. Món đồ đó được đưa vào trạng thái "cần rà soát" và tạm thời không dùng được để phối đồ. Người dùng muốn xem lại món đồ, chỉnh sửa danh mục/giá nếu cần, rồi xác nhận để đưa món đồ vào sử dụng.

**Why this priority**: Đây là điểm kẹt nghiêm trọng nhất hiện tại — món đồ ở trạng thái cần rà soát không có bất kỳ đường nào để trở thành sử dụng được, khiến ảnh người dùng tải lên bị "treo" vĩnh viễn và không tham gia gợi ý phối đồ.

**Independent Test**: Tạo một món đồ ở trạng thái cần rà soát, gọi thao tác xác nhận, và kiểm tra món đồ chuyển sang trạng thái sử dụng được và xuất hiện trong kết quả gợi ý phối đồ.

**Acceptance Scenarios**:

1. **Given** một món đồ đang ở trạng thái cần rà soát thuộc sở hữu người dùng, **When** người dùng gọi thao tác xác nhận, **Then** món đồ chuyển sang trạng thái sử dụng được và tham gia gợi ý phối đồ.
2. **Given** một món đồ đang ở trạng thái cần rà soát, **When** người dùng đã chỉnh sửa danh mục trước đó rồi xác nhận, **Then** danh mục người dùng chọn được giữ nguyên.
3. **Given** một món đồ không ở trạng thái cần rà soát, **When** gọi thao tác xác nhận, **Then** hệ thống từ chối bằng lỗi nghiệp vụ rõ ràng.
4. **Given** một món đồ thuộc sở hữu người dùng khác, **When** gọi thao tác xác nhận, **Then** hệ thống từ chối.

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

### User Story 4 - Nhân viên nhãn hàng xử lý sản phẩm lỗi hoặc cần rà soát (Priority: P2)

Nhân viên nhãn hàng tải danh sách sản phẩm lên Brand Portal. Khi AI phân tích thất bại hoặc không chắc chắn, họ cần phân biệt rõ đó là lỗi hệ thống hay cần duyệt thủ công, xem lý do, và có thể thử lại hoặc chủ động xử lý.

**Why this priority**: Đảm bảo vận hành B2B thông suốt và không phụ thuộc vào đội kỹ thuật; hiện tại lỗi và cần-duyệt bị gộp chung nên khó xử lý.

**Independent Test**: Tạo sản phẩm nhãn hàng gây lỗi phân tích và sản phẩm AI không chắc, kiểm tra trạng thái/lý do hiển thị khác nhau và thao tác thử lại hoạt động.

**Acceptance Scenarios**:

1. **Given** một sản phẩm phân tích thất bại, **When** nhân viên xem, **Then** sản phẩm ở trạng thái lỗi và hiển thị lý do.
2. **Given** một sản phẩm AI không đủ tự tin, **When** nhân viên xem, **Then** sản phẩm ở trạng thái cần duyệt/nháp và hiển thị lý do.
3. **Given** một sản phẩm lỗi hoặc cần rà soát, **When** nhân viên gọi thử lại, **Then** sản phẩm được đưa lại hàng đợi và nhân viên theo dõi được tiến trình realtime.
4. **Given** một sản phẩm ở trạng thái lỗi, **When** nhân viên cố thay đổi trạng thái thủ công, **Then** hệ thống từ chối; chỉ thao tác thử lại (hoặc hệ thống) mới đưa sản phẩm ra khỏi trạng thái lỗi.
5. **Given** nhân viên không thuộc nhãn hàng sở hữu sản phẩm, **When** gọi thao tác, **Then** hệ thống từ chối.

---

### Edge Cases

- Client mở kết nối realtime trước khi có sự kiện nào, hoặc sau khi phiên đã kết thúc: hệ thống phát lại các sự kiện còn trong giới hạn lưu trữ (nếu có) và không gây lỗi. Buffer KHÔNG bị xóa ngay khi subscriber cuối cùng ngắt kết nối; nó chỉ bị loại khi vượt giới hạn số lượng hoặc hết thời gian lưu trữ (TTL) cấu hình.
- Người dùng gọi thử lại đúng lúc hệ thống đang xử lý món đồ đó (tranh chấp): hệ thống từ chối và không tạo phiên trùng.
- Nhiều client cùng theo dõi một phiên phân tích: mọi client đều nhận được sự kiện.
- Giới hạn lưu trữ sự kiện đầy: hệ thống loại bỏ sự kiện cũ nhất và vẫn phục vụ các sự kiện mới.
- Món đồ cần rà soát nhưng thiếu danh mục hợp lệ: thao tác xác nhận MUST bị từ chối và yêu cầu người dùng chọn danh mục trước.
- Sự kiện bị phát trùng do cơ chế gửi lại: hệ thống xử lý idempotent, không gây sai lệch trạng thái.
- Xác nhận/thử lại khi món đồ đang bị khóa do vượt hạn mức gói cước: hệ thống xử lý nhất quán với chính sách khóa hiện hành.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST cung cấp cách để người dùng xác nhận một món đồ đang ở trạng thái cần rà soát, đưa nó sang trạng thái sử dụng được.
- **FR-002**: Thao tác xác nhận MUST chỉ thành công khi món đồ đang ở trạng thái cần rà soát, thuộc sở hữu người gọi, và có danh mục hợp lệ (thuộc nhóm có thể phối đồ); ngược lại MUST trả lỗi nghiệp vụ rõ ràng.
- **FR-003**: Sau khi xác nhận, món đồ MUST tham gia gợi ý phối đồ và mọi truy vấn "sử dụng được".
- **FR-004**: Hệ thống MUST phát sự kiện tiến trình "đang xử lý" cho từng món ngay khi bắt đầu phân tích.
- **FR-005**: Hệ thống MUST phát sự kiện cho trạng thái hoàn tất, thất bại, cần rà soát với đầy đủ: định danh món, chỉ số tiến trình, dữ liệu tóm tắt (với hoàn tất/cần rà soát) và thông báo lỗi thân thiện (với thất bại).
- **FR-006**: Hệ thống MUST lưu trữ các sự kiện gần nhất theo từng phiên và phát lại cho client kết nối lại, trong giới hạn cấu hình được; việc lưu trữ MUST đặt sau một lớp trừu tượng transport để có thể chuyển từ bộ nhớ sang kho chia sẻ mà không đổi hành vi client.
- **FR-007**: Thao tác thử lại MUST chỉ được chấp nhận khi món đồ đang ở trạng thái cho phép thử lại (thất bại hoặc cần rà soát); khi đang xử lý hoặc ở trạng thái khác MUST bị từ chối. Hệ thống MUST dùng cơ chế chuyển trạng thái nguyên tử để tránh tạo phiên trùng khi có yêu cầu đồng thời.
- **FR-008**: Khi phân tích lại, hệ thống MUST bảo toàn danh mục hiện có của món đồ (không để AI ghi đè); nếu món chưa có danh mục thì AI được phép điền.
- **FR-009**: Phản hồi khởi tạo thử lại MUST trả về mã phiên (taskId) để client theo dõi realtime.
- **FR-010**: Sản phẩm nhãn hàng MUST phân biệt trạng thái lỗi hệ thống và trạng thái cần duyệt thủ công, kèm lý do tương ứng. Trạng thái lỗi MUST là trạng thái cuối đối với thao tác thủ công: hệ thống MUST từ chối mọi thay đổi trạng thái rời khỏi lỗi, ngoại trừ thao tác thử lại (hoặc do hệ thống thực hiện).
- **FR-011**: Nhân viên nhãn hàng MUST có khả năng thử lại phân tích cho sản phẩm ở trạng thái lỗi, hoặc sản phẩm nháp có lý do cần rà soát; MUST từ chối với sản phẩm nháp thường (không có lý do cần rà soát).
- **FR-012**: Mọi thao tác xác nhận/thử lại MUST kiểm tra quyền: người dùng sở hữu món đồ, hoặc nhân viên là thành viên hợp lệ của nhãn hàng sở hữu sản phẩm.
- **FR-013**: Hệ thống MUST xử lý idempotent khi nhận sự kiện phân tích trùng lặp (bao gồm cả sự kiện "đang xử lý" mới), không cập nhật dư thừa hay sai lệch trạng thái.
- **FR-014**: Hệ thống MUST cung cấp tài liệu playbook mô tả kịch bản xử lý cho từng trạng thái phân tích, áp dụng cho cả B2C và B2B.
- **FR-015**: Việc thêm trạng thái mới MUST tương thích ngược: client hiện tại không bị phá vỡ khi nhận trạng thái mới.
- **FR-016**: Sự kiện realtime MUST được truyền từ nơi phát sinh tới nơi giữ kết nối client qua một kênh chia sẻ giữa các tiến trình, đảm bảo hoạt động ở CẢ hai topology triển khai: (a) self-hosted/VPS gộp http + worker + consumer trong cùng một tiến trình (`APP_MODE=all`); (b) managed/AWS tách riêng tiến trình http (`APP_MODE=http`) và worker (`APP_MODE=worker`) trên các container khác nhau.
- **FR-017**: Feature MUST tạo một tài liệu tích hợp dành riêng cho frontend, đặt trong thư mục feature `specs/023-analyze-status-handling/` (tên gợi ý `frontend-guide.md`). Tài liệu MUST nêu rõ: (a) từng trạng thái phân tích và kịch bản hành động phía client; (b) các API có thay đổi hành vi/hợp đồng (bao gồm "trước → sau", request/response, trạng thái mới, lỗi mới) kèm ví dụ; (c) luồng realtime (cách đăng ký, payload theo từng trạng thái, kết nối lại/phát lại, mã phiên); (d) checklist việc frontend cần làm. Tài liệu MUST đủ để lập trình viên frontend triển khai mà KHÔNG cần đọc code backend hay tài liệu Swagger.
- **FR-018**: Mọi thay đổi hành vi của API hiện hữu (không chỉ API mới) MUST được liệt kê tường minh trong tài liệu frontend (FR-017) theo dạng "trước → sau" kèm ví dụ payload, để tránh phá vỡ tích hợp hiện tại.

### Key Entities *(include if feature involves data)*

- **WardrobeItem (món đồ cá nhân)**: đại diện một trang phục của người dùng; vòng đời phân tích gồm đang xử lý → sử dụng được / thất bại / cần rà soát.
- **BrandItem (sản phẩm nhãn hàng)**: đại diện sản phẩm/mẫu thử của nhãn hàng; trạng thái gồm nháp, hoạt động, lưu trữ, và (mới) lỗi.
- **AnalyzeTask (phiên phân tích)**: một phiên xử lý hàng loạt gắn nhiều món; có định danh phiên và chỉ số tiến trình.
- **AnalyzeTaskStatus (trạng thái phân tích)**: đang xử lý, hoàn tất, thất bại, cần rà soát.
- **Reason (lý do)**: lý do cần rà soát hoặc lý do lỗi hệ thống, phục vụ hiển thị cho người dùng.
- **RealtimeEvent (sự kiện realtime)**: bản tin tiến trình gửi tới client, có thể được lưu lại tạm thời để phát lại.
- **FrontendGuide (tài liệu tích hợp frontend)**: tài liệu trong thư mục feature, mô tả kịch bản hành động của client, các thay đổi API "trước → sau", và luồng realtime — đủ để frontend tự triển khai.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% món đồ ở trạng thái cần rà soát có thể chuyển sang trạng thái sử dụng được thông qua thao tác xác nhận hợp lệ.
- **SC-002**: Tối thiểu 99% món đồ trong một phiên phân tích phát sinh đầy đủ sự kiện "đang xử lý" và sự kiện kết thúc tới client đang theo dõi.
- **SC-003**: 100% sự kiện đã phát (trong giới hạn lưu trữ) được phát lại cho client kết nối lại, không cần tải lại toàn bộ dữ liệu.
- **SC-004**: 100% yêu cầu thử lại khi trạng thái không cho phép bị chặn; các yêu cầu đồng thời trên cùng một món không tạo phiên trùng.
- **SC-005**: 100% chỉnh sửa danh mục thủ công của người dùng được giữ nguyên sau khi hệ thống phân tích lại.
- **SC-006**: 100% sản phẩm nhãn hàng lỗi/cần rà soát hiển thị đúng trạng thái và lý do tương ứng.
- **SC-007**: Tài liệu playbook bao phủ 100% các trạng thái phân tích và các luồng xử lý tương ứng cho cả B2C và B2B.
- **SC-008**: Không có thay đổi phá vỡ hợp đồng hiện tại với client (tương thích ngược).
- **SC-009**: Ở cả hai topology (gộp tiến trình self-hosted/VPS và tách http/worker managed/AWS), 100% sự kiện realtime phát sinh trong quá trình xử lý nền tới được client đang kết nối.
- **SC-010**: Tồn tại một tài liệu frontend trong thư mục feature bao phủ 100% trạng thái, API đổi hành vi, và luồng realtime; một lập trình viên frontend có thể hoàn tất tích hợp chỉ bằng tài liệu đó mà không cần đọc code backend hay Swagger.

## Assumptions

- Có hai topology triển khai: self-hosted/VPS chạy 3 container dịch vụ với `APP_MODE=all` (http + worker + consumer chung một tiến trình); managed/AWS chạy 6 container tách riêng http (`APP_MODE=http`) và worker (`APP_MODE=worker`). Feature MUST hoạt động đúng ở cả hai, nên sự kiện realtime MUST đi qua kênh chia sẻ giữa tiến trình (Redis Pub/Sub); buffer phát lại nằm trong tiến trình http và đặt sau lớp trừu tượng transport.
- Redis là hạ tầng đã có sẵn ở mọi môi trường (self-hosted và managed); việc dùng Redis Pub/Sub không thêm nhà cung cấp mới.
- Thao tác thử lại không giới hạn theo số lần/thời gian; chỉ kiểm soát bằng trạng thái món đồ và cơ chế chuyển trạng thái nguyên tử. Chi phí AI được kiểm soát bởi chính sách quota AI hiện hữu.
- Giới hạn lưu trữ sự kiện realtime mặc định khoảng 100 sự kiện mỗi phiên, cấu hình được; buffer được giữ theo TTL cấu hình (mặc định vài phút) để hỗ trợ reconnect muộn, không xóa ngay khi hết subscriber.
- Thao tác xác nhận món đồ cần rà soát không chạy lại AI; chỉ chuyển trạng thái và đưa vào sử dụng.
- Việc mở rộng trạng thái nhãn hàng với "lỗi" không yêu cầu di trú dữ liệu phá vỡ; cần xác nhận ràng buộc dữ liệu hiện hành.
- Cơ chế phát sự kiện bất đồng bộ hiện hữu được tái sử dụng; feature này mở rộng theo pattern sẵn có.
- Người dùng và nhân viên nhãn hàng có kết nối mạng ổn định trong quá trình thao tác.
- Hệ thống chỉ có hai vai trò hệ thống (`admin`, `user`); không thêm vai trò mới (theo constitution).
- Các sự kiện realtime hiện được định danh theo phiên; feature giữ nguyên mô hình này và bổ sung lưu trữ/phát lại.
