# Specification Quality Checklist: Một chủ sở hữu mức quyền tính phí AI và công cụ cấp gói Premium cho quản trị viên

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-29
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

### Vòng kiểm tra 1 — 29/09/2026 (bản cấp gói đơn lẻ + hàng loạt)

**Kết quả: Đạt toàn bộ.** 8 mục ban đầu đều đạt sau khi rà soát.

#### Điều chỉnh đã thực hiện

| Vấn đề phát hiện | Xử lý |
|---|---|
| FR cộng dồn thời hạn chưa khớp hành vi gia hạn tự động hiện có | Bổ sung yêu cầu buộc dùng chung quy tắc tính thời hạn |
| "Ghi nhận quyền lợi" chưa phân biệt với việc chỉ đổi tên hiển thị | Tách thành hai yêu cầu: ghi nhận hạn mức AI và phát tín hiệu tới hệ thống con |
| Thiếu yêu cầu về bộ nhớ đệm trạng thái gói trong khi SC-002 đòi 60 giây | Bổ sung yêu cầu về độ trễ, về thời hạn lưu tạm, và về việc không phụ thuộc bộ nhớ đệm khi xác định gói |
| Thiếu xử lý mã định danh khớp nhiều tài khoản | Bổ sung yêu cầu từ chối kèm thông báo mơ hồ |
| Thiếu yêu cầu loại trừ tài khoản đã xoá mềm | Bổ sung |
| Thiếu giới hạn kích thước danh sách hàng loạt | Bổ sung giới hạn 500 phần tử và yêu cầu từ chối danh sách rỗng |
| Thiếu xử lý chạy trùng khi gửi lại yêu cầu do lỗi mạng | Bổ sung vào Edge Cases và yêu cầu tương ứng |

### Vòng kiểm tra 2 — 29/09/2026 (mở rộng phạm vi: gộp phần sửa tín hiệu đổi gói)

**Kết quả: Đạt toàn bộ.** 8 mục đều đạt. Spec được viết lại toàn bộ với 4 phần.

#### Vì sao mở rộng phạm vi

Khi kiểm tra lại codebase lần cuối, phát hiện hai lỗi kỹ thuật nền tảng khiến việc chỉ sửa phần cấp gói là không đủ:

- Bản ghi quyền tính phí AI tạo ra từ tín hiệu đều **thiếu mã định danh gói**, trong khi bộ dò so khớp trường này ở một bước kiểm tra. Sửa phần cấp gói mà không sửa lỗi này thì tính năng cấp gói vẫn không mang lại quyền lợi AI.
- Bộ dò còn **tự đóng và xoá bản ghi trong lúc đọc** khi thấy dữ liệu không khớp, biến sai lệch nhỏ thành mất dữ liệu vĩnh viễn.

Ngoài ra phát hiện thêm: tài khoản mới **không bao giờ nhận được bản ghi quyền tính phí AI** vì cơ chế cấp quyền chỉ chạy khi gói thay đổi, mà tài khoản mới không có bước thay đổi gói nào.

Vì vậy Phần 1 của spec gộp ba việc: sửa tín hiệu, sửa hai lỗi nền tảng, và bổ sung đường cấp quyền cho tài khoản mới.

#### Điều chỉnh đã thực hiện trong vòng này

| Vấn đề phát hiện | Xử lý |
|---|---|
| Bản ghi lịch sử thay đổi gói đã chứa loại thay đổi nhưng nơi phát tín hiệu viết tay chuỗi riêng, dễ lệch với bảng rẽ nhánh phía nhận | Yêu cầu lấy loại thay đổi từ bản ghi lịch sử, không viết tay |
| Nơi phát tín hiệu có thể tự quyết bỏ qua loại thay đổi "không cần thiết", gây thất lạc tín hiệu | Yêu cầu phát tín hiệu cho mọi loại thay đổi, để bộ xử lý phía nhận tự quyết định |
| Nhiều nơi tạo bản ghi quyền tính phí AI riêng lẻ, không bảo đảm trường mã định danh gói luôn đầy đủ | Yêu cầu một cửa tạo duy nhất nhận đủ mã gói và mã chính sách phí |
| Tín hiệu mới có thể tới bộ xử lý cũ khi triển khai không đúng thứ tự | Yêu cầu bộ xử lý chịu được tín hiệu thiếu trường, và chốt thứ tự triển khai phía nhận trước |
| Đóng mức quyền cũ rồi tạo mức mới thành hai bước có thể để lại khoảng thời gian tài khoản không có quyền tính phí | Yêu cầu hai bước là một giao dịch duy nhất, và tối đa một mức hiệu lực |
| Khắc phục tài khoản cũ thiếu quyền tính phí có thể dùng cách sửa dữ liệu trực tiếp, vượt ranh giới hệ thống | Yêu cầu công cụ phát lại tín hiệu, chạy lại được, không cần truy cập cơ sở dữ liệu |
| Tài khoản mới cần đường cấp quyền, nếu viết nhánh riêng sẽ lệch với đường xử lý sẵn có | Yêu cầu dùng lại đường phát lại tín hiệu kích hoạt với gói miễn phí |
| Tín hiệu phát đi khi hệ thống con tạm không liên lạc được có thể mất | Yêu cầu giữ lại và xử lý lại, không được mất |
| Thiếu xử lý tín hiệu trùng khi tài khoản bị tạo lại qua nhiều đường | Yêu cầu bộ xử lý bỏ qua tín hiệu trùng mà không gây tác dụng phụ |
| Chưa rõ điều gì xảy ra khi tài khoản bật/tắt gia hạn tự động | Thêm kịch bản chấn nhận cho cả bật và tắt |
| Chưa rõ hành vi khi hai quản trị viên cùng lúc cấp gói cho một tài khoản | Bổ sung yêu cầu và kịch bản chấn nhận |
| Chưa rõ trường hợp gói bị vô hiệu hoá giữa lúc chuẩn bị và lúc thực thi | Thêm vào Edge Cases |
| Chưa rõ trường hợp tín hiệu thay đổi gói chồng lấn nhau | Thêm vào Edge Cases |

#### Điểm đã cân nhắc và chủ động loại khỏi phạm vi

| Nội dung | Lý do loại |
|---|---|
| Giao diện quản trị trên web | Chưa có trong phạm vi; đã ghi ở Assumptions |
| Cơ chế trao quà 50 tài khoản tự động | Tính năng độc lập, chỉ dùng chung Phần 1 |
| Trả tiền, hoàn tiền, gia hạn tự động | Đã có luồng riêng; chỉ bổ sung tín hiệu cho chúng |
| Cấp nhiều gói khác nhau cùng lúc cho một tài khoản | Giới hạn ở một mã gói cho mỗi phần tử |
| Thay đổi kiến trúc hàng đợi sự kiện | Hạ tầng ghi và phân phối sự kiện đã bảo đảm không mất tín hiệu; không cần đổi |

#### Xác nhận về các hiểu lầm đã loại trừ

Đã đối chiếu lại yêu cầu với hành vi hệ thống hiện có trước khi viết spec:

- Cơ chế cộng dồn thời hạn cho cùng gói **đã tồn tại** và dùng cho gia hạn tự động → spec giữ nguyên, không đặt ra hành vi mới.
- So khớp tên đăng nhập và email không phân biệt hoa thường **đã là hành vi chuẩn** → spec giữ nguyên.
- Hạ tầng ghi và phân phối sự kiện **đã bảo đảm** tín hiệu không mất khi hệ thống con tạm thời không liên lạc được → spec yêu cầu giữ hành vi này, không xây mới.
- Cơ chế ghi nhận ký quản trị **đã có sẵn** cho hành động cấp gói → spec yêu cầu tái sử dụng, không đề xuất cơ chế mới.
- Cơ chế ghi lịch sử thay đổi gói **đã có sẵn** trường loại thay đổi → spec yêu cầu dùng lại, không tạo trường mới.
- Bộ nhớ đệm trạng thái gói **đã có** thời hạn 60 giây → spec chốt lại mức tối đa, không đặt giá trị mới.

Các vấn đề về dữ liệu thực tế phát hiện trong quá trình kiểm tra đã được phản ánh vào Bối cảnh của spec: một tài khoản Premium không có bản ghi quyền tính phí AI, và nguyên nhân là chỉ một trong tám điểm chuyển gói phát tín hiệu.

Vòng kiểm tra thứ hai đạt hoàn toàn, không cần chạy thêm vòng thứ ba.

### Phiên làm rõ — 29/09/2026 (5 câu hỏi, tất cả đã tích hợp)

**Kết quả: 16/16 mục vẫn đạt, không có mục nào bị hồi quy.**

Phát hiện 5 khoảng trống và đã đóng bằng câu trả lời của người phát triển:

| # | Khoảng trống | Đã chốt | Yêu cầu tác động |
|---|---|---|---|
| 1 | Phạm vi khôi phục quyền tính phí chỉ nói tài khoản có gói, chưa nói tới tài khoản gói miễn phí đã tồn tại | Quét **mọi** tài khoản thiếu quyền tính phí, không phân biệt gói miễn phí hay trả phí | FR-015, SC-013, Bối cảnh |
| 2 | Không có con số thời gian chờ cho thao tác cấp gói | Danh sách 1 phần tử: 5 giây; 500 phần tử: 60 giây; kết quả trả ngay trong yêu cầu | Phần 6 mới (FR-058 → FR-060), SC-014, SC-015 |
| 3 | Có bao nhiêu giao diện lập trình và cho phép bao nhiêu tài khoản | **Chỉ một giao diện lập trình**, nhận danh sách từ 1 đến N; cấp một tài khoản chính là danh sách một phần tử | FR-031, FR-033, US2, US3, Phần 2/3 |
| 4 | Thiếu giới hạn tần suất gọi thao tác cấp gói | 60 yêu cầu/phút, trong đó tối đa 6 yêu cầu/phút có nhiều hơn một phần tử | FR-061, FR-062, SC-016 |
| 5 | Tín hiệu không có quy ước phiên bản nội dung | Thêm trường số phiên bản; phía nhận từ chối phiên bản mới hơn mức hiểu, chấp nhận phiên bản cũ hơn | FR-002, SC-017 |
| 6 | Đường tín hiệu không có yêu cầu quan sát, nên tín hiệu bị từ chối sẽ im lặng và lặp lại lỗi đang sửa | Bắt buộc ghi nhận mỗi lần từ chối, có bộ đếm theo lý do, có cảnh báo khi bộ đếm tăng | FR-063 → FR-065, SC-018, SC-019 |

#### Ảnh hưởng lớn nhất: khoảng trống số 1 và 3

- **Khoảng trống 1** chỉ lộ ra khi đối chiếu lại câu truy vấn đã chạy trên dữ liệu thật: câu đó lọc theo nhóm tài khoản trả phí, nên không đo được nhóm tài khoản gói miễn phí. Vì cơ chế lỗi áp dụng cho **mọi** tài khoản tạo sau mốc thời gian dữ liệu quyền tính phí được đưa vào hệ thống, số tài khoản miễn phí bị ảnh hưởng có thể lớn hơn số tài khoản trả phí. Spec cũ có nguy cả chỉ khôi phục một phần nhỏ.
- **Khoảng trống 3** làm thay đổi cấu trúc: trước đó Phần 2 và Phần 3 mô tả hai thao tác riêng biệt. Sau khi chốt chỉ có một giao diện lập trình, Phần 2 trở thành quy tắc nhận diện và áp dụng cho một tài khoản, Phần 3 trở thành quy tắc cho danh sách, và US2 trở thành trường hợp biên danh sách một phần tử.

#### Đã chuẩn hoá thuật ngữ

Sau khi gộp về một thao tác, các cụm "yêu cầu đơn lẻ" và "yêu cầu hàng loạt" đã được thay bằng "danh sách một phần tử" và "danh sách" xuyên suốt Bối cảnh, User Story, Edge Cases, Functional Requirements và Success Criteria, tránh việc đọc nhầm là còn hai thao tác riêng.

#### Điểm nhất quán đã kiểm tra

- 67 yêu cầu chức năng, 19 tiêu chí thành công, 25 dòng làm rõ, không có mã yêu cầu hay tiêu chí nào trùng lặp.
- Không còn chỗ trống `[NEEDS CLARIFICATION]`.
- Cấu trúc tiêu đề hợp lệ; các tiêu đề mới chỉ là `## Clarifications` và `### Session 2026-09-29` đã có sẵn.
- Số câu hỏi đã hỏi: 5, không vượt hạn mức.

### Viết lại toàn bộ — 29/09/2026 (mở rộng phạm vi, thêm thứ tự thực thi)

**Kết quả: 16/16 mục vẫn đạt, không có mục nào bị hồi quy.**

#### Vì sao viết lại toàn bộ

Đối chiếu trực tiếp với dữ liệu đang chạy trong Docker phát hiện bốn điểm mô tả sai trong phiên bản trước, và phát hiện cơ chế thật hoàn toàn khác với giả định ban đầu:

| Nội dung | Bản trước | Thực tế đã kiểm chứng |
|---|---|---|
| Cơ chế cấp mức quyền tính phí | "Tín hiệu qua hàng đỗi tới hệ thống AI" | **Có hai đường**: hệ thống Account ghi thẳng vào bảng của hệ thống AI bằng lệnh SQL thô trong cùng giao dịch (**vi phạm hiến pháp**), và đường hàng đỗi bị **hỏng** |
| Số tài khoản bị ảnh hưởng | 1 tài khoản | **74/82**; riêng nhóm có bản ghi gói nhưng thiếu mức quyền tính phí là **6**, và kiểm chứng cho thấy **không hề có bản ghi** chứ không phải bản ghi bị đóng |
| Khách hàng dùng gói Premium | Không dùng được AI trả phí | **Có mức quyền tính phí hợp lệ**; luồng thanh toán cổng đang hoạt động nhờ đường ghi đồng bộ |
| Điểm cấp gói hỏng | Chưa nêu | Ba đường không sinh bản ghi lịch sử: cấp gói thủ công, thu hồi gói thủ công, cấp phát gói mặc định |

#### Điều chỉnh cấu trúc

| Thay đổi | Lý do |
|---|---|
| Thêm mục **Phạm vi và thứ tự thực thi** chia 9 giai đoạn GĐ0–GĐ8 | Yêu cầu sắp xếp thứ tự hợp lí để thực thi |
| Đánh số lại toàn bộ yêu cầu theo đúng thứ tự giai đoạn | FR-001→087 liên tục, không trùng, không lỗ hổng |
| Thêm US1 mới về suy giảm mềm, nâng lên P1 | Đây là điều kiện tiên quyết để đổi cấu trúc an toàn |
| Bổ sung 4 vấn đề nền tảng vào Bối cảnh | Thay cho 3 vấn đề cũ đã sai |
| Bổ sung bảng giá trị cấu hình đang áp dụng | Làm căn cứ cho yêu cầu cấu hình GĐ0 |
| Bổ sung nhóm yêu cầu GĐ0 (cấu hình) và GĐ8 (quan sát, khôi phục, kiểm chứng) | Những việc này trước đây nằm ngoài spec, nay đã thuộc 024 |

#### Những điều chỉnh quan trọng nhất về nội dung

- **FR-025**: bảng mức quyền tính phí chỉ có một chủ sở hữu duy nhất là hệ thống AI; hệ thống khác không được ghi dưới bất kỳ hình thức nào.
- **FR-006**: khi thiếu mức quyền tính phí thì áp dụng chính sách gói miễn phí thay vì trả lỗi; biến phụ thuộc cứng vào độ trễ thành suy giảm mềm.
- **FR-037**: quy tắc tính mốc bắt đầu bản ghi chờ tới khi gia hạn phải được chuyển sang hệ thống AI **trước khi** gỡ cơ chế cũ.
- **FR-010**: mọi thay đổi gói đều phải sinh bản ghi lịch sử — bỏ ngoại lệ; đây là nguyên nhân gốc của ba đường hỏng.
- **FR-021**: tín hiệu phải mang đủ 13 trường để bên nhận tự ghi được một bản ghi hoàn chỉnh mà không cần đọc bảng của bên gửi.
- **FR-027**: ảnh chụp chính sách dựng từ bảng chính sách của chính hệ thống AI, không gửi qua tín hiệu.
- **FR-005**: trần chi phí 30.000đ, hạn mức **30 lượt gợi ý phối đồ và 10 lượt trò chuyện** mỗi ngày. *(Ghi đè giá trị cũ 30/20 sau khi người phát triển chốt H1 — xem vòng thứ năm.)*
- **SC-021**: chốt trần rủi ro tài chính của chiến dịch 50 suất ở mức 3,4 triệu đ.

#### Điểm nhất quán đã kiểm tra

- 87 yêu cầu chức năng đánh số liên tục từ FR-001 đến FR-087, không trùng, không lỗ hổng.
- 26 tiêu chí thành công đánh số liên tục từ SC-001 đến SC-026.
- Tất cả tham chiếu chéo tới yêu cầu khác đều trỏ tới yêu cầu tồn tại.
- 26 dòng làm rõ, không có dòng nào trùng lặp.
- Không còn chỗ trống `[NEEDS CLARIFICATION]`.
- Cấu trúc tiêu đề hợp lệ; các tiêu đề mới chỉ là các tiêu đề `#### GĐx` nằm trong mục Functional Requirements, không thay đổi thứ tự các mục bắt buộc của mẫu.

### Phiên làm rõ lần 2 — 29/09/2026 (4 câu hỏi, tất cả đã tích hợp)

**Kết quả: 16/16 mục vẫn đạt, không có mục nào bị hồi quy.**

Lần này tập trung vào những điểm còn bỏ ngỏ sau khi viết lại toàn bộ spec: tính chặt của ranh giới chuyển đổi, hành vi khi thiếu mức quyền tính phí, và tính nhất quán giữa các yêu cầu với phần giả định.

| # | Khoảng trống | Đã chốt | Yêu cầu tác động |
|---|---|---|---|
| 1 | Gỡ đường ghi đồng bộ là bước không đảo ngược, chưa rõ có cần cờ bật/tắt tạm thời không | **Không dùng cờ bật/tắt.** Kiểm chứng trên môi trường phát triển rồi mới gỡ; cần hoàn tác thì dùng git | FR-036 → FR-040, bảng thứ tự GĐ5, Edge Cases |
| 2 | Khi thiếu mức quyền tính phí mà cả chính sách gói mặc định cũng không dùng được thì làm gì | **Suy giảm hai tầng:** tầng một dùng chính sách gói mặc định; tầng hai dùng chế độ chỉ quan sát, không trần, 0 lượt trả phí mỗi ngày, kèm cảnh báo | FR-006 → FR-008, bảng thứ tự GĐ1, US1 |
| 3 | Trước khi chuyển sang cơ chế một chủ sở hữu có cần dọn dữ liệu tồn tại không | **Có, một tệp kịch bản dọn chạy tay trước GĐ4:** chỉ xoá bản ghi đã đóng và không còn hiệu lực, giữ nguyên bản ghi đang hiệu lực và đang chờ tới, báo cáo số lượng sẽ xoá trước khi ghi, chạy lại được | FR-092 → FR-094, SC-028, **giai đoạn mới GĐ3b** |
| 4 | Assumptions nói tài khoản do quản trị viên tạo không cần mức quyền tính phí, mâu thuẫn với FR-013 | **Có, y hệt tài khoản đăng ký bình thường.** Không còn ngoại lệ nào trong FR-012 và FR-013 | FR-013, SC-006, Assumptions |

#### Ghi nhận về khoảng trống số 4

Đây là mâu thuẫn nội tại trong chính spec, phát hiện khi đối chiếu từng yêu cầu với phần giả định chứ không phát sinh từ cấu hình dữ liệu: FR-012 phát biểu tuyệt đối rằng không có trường hợp nào đổi gói mà không sinh bản ghi lịch sử, trong khi Assumptions lại mở ngoại lệ cho tài khoản do quản trị viên tạo. Vì cơ chế cấp phát mặc định đã bao phủ nhóm tài khoản này, sửa phần giả định là đủ để hai bên thống nhất — không cần thêm đường xử lý riêng.

#### Điểm nhất quán đã kiểm tra

- 94 yêu cầu chức năng đánh số liên tục từ FR-001 đến FR-094, không trùng, không lỗ hổng.
- 28 tiêu chí thành công đánh số liên tục từ SC-001 đếc SC-028.
- Tất cả tham chiếu chéo giữa các yêu cầu đều trỏ tới yêu cầu tồn tại; đã kiểm tra bằng script, bao gồm cả tham chiếu nằm trong dòng định nghĩa.
- 30 dòng làm rõ, không có dòng nào trùng lặp.
- Không còn chỗ trống `[NEEDS CLARIFICATION]`.
- Bảng thứ tự thực thi có 10 mục (GĐ0–GĐ8, trong đó GĐ3b nằm sau GĐ3 và trước GĐ4) và mọi ràng buộc phụ thuộc vẫn trỏ tới đúng giai đoạn.
- Số câu hỏi đã hỏi trong phiên này: 4, không vượt hạn mức.

**Spec đã sẵn sàng cho `/speckit-plan`.**

### Vòng kiểm tra sau phân tích — 29/09/2026 (2 lượt, chạy sau khi đã có plan và tasks)

**Kết quả: 16/16 mục vẫn đạt.** Lượt thứ nhất phát hiện 2 lỗi nghiêm trọng và 6 lỗi mức cao; lượt thứ hai sau khi sửa phát hiện 1 lỗi mức cao và 6 lỗi mức vừa/thấp. Tất cả đã đóng.

#### 2 lỗi nghiêm trọng đã đóng

| Vấn đề | Cách đóng |
|---|---|
| `plan.md` yêu cầu gỡ `CreateGrant` khỏi dịch vụ AI nhưng `tasks.md` không có nhiệm vụ nào — đường ghi thứ hai của bảng mức quyền tính phí sống sót sau GĐ5, vi phạm §I | Thêm nhiệm vụ gỡ cả hàm, DTO, route và interface, đặt **trước** bước gỡ đường ghi vi phạm. FR-035 viết lại để nêu rõ lý do |
| Suy giảm GĐ1 dùng bộ đếm Redis thay cho sổ tính phí `FOR UPDATE` — vi phạm §VI, cần ngoại lệ chưa được phê duyệt | **Thiết kế lại để tuân thủ**, không xin ngoại lệ: cho phép dòng sổ tính phí không có bản ghi, cộng chỉ mục duy nhất riêng cho dòng đó. Đã kiểm chứng trên PostgreSQL 17.10. `Complexity Tracking` nay trống |

#### Điều chỉnh nội dung

| Vấn đề | Xử lý |
|---|---|
| `services/ai/go.mod` khai báo `go 1.25.7`, lệch `go.work` | Thêm nhiệm vụ nâng lên `1.26.0` |
| FR-014 (cấp bản ghi gói hàng loạt trong một thao tác đọc) chưa có nhiệm vụ — đây là một trong ba đường hỏng gốc ở Vấn đề 2 | Thêm nhiệm vụ sửa `BulkCreate` và thêm dòng này vào bảng kiểm chứng |
| FR-070/FR-071 (ảnh chụp quyền lợi khi cấp gói) chưa có nhiệm vụ | Bổ sung vào nhiệm vụ cấp gói |
| SC-006 (tài khoản mới, kể cả do quản trị viên tạo) chưa có nhiệm vụ kiểm chứng | Thêm kịch bản kiểm chứng cả ba loại tài khoản |
| FR-061 (trộn nhiều mã gói trong một danh sách) chưa có phép kiểm | Sửa danh sách kiểm thử thành hỗn hợp |
| FR-034 nói "bảo đảm" mà không nói bảo đảm bằng cách nào | Nêu rõ: bảo đảm ở tầng ứng dụng, **không** thêm ràng buộc duy nhất ở cơ sở dữ liệu vì cần giữ nhiều bản ghi `active` theo thời gian |
| FR-023/026, FR-052/072, FR-081/082 bị hiểu là trùng nhau | Thêm câu phân biệt thay vì gộp mã, để giữ nguyên 94 FR và không phá tham chiếu chéo |
| FR-076 thiếu cơ sở cho việc nâng `write_timeout_seconds` | Bổ sung câu ràng buộc hạn chót ghi đáp lời |
| FR-042 dễ bị đọc sai thành "đường đọc không được ghi gì", mâu thuẫn với §VI | FR-042, SC-002 và kịch bản US1-5 đều làm rõ: cấm ghi **bản ghi mức quyền tính phí**, không cấm ghi **sổ tính phí** ở bước đặt trước |
| Mã lý do suy giảm trong quickstart (`fallback_tier_2`) lệch hằng số (`tier_2_observe_only`) | Đồng nhất — cài sai thì cảnh báo tầng 2 nổ dưới tên không ai kiểm tra |
| Không có phép kiểm dương cho §VI | Bổ sung: sổ tính phí phải **tăng** khi suy giảm, còn `ai_policy_grants` phải **không đổi** |
| FR-040 chưa được viện dẫn ở đâu | Ghi vào nhiệm vụ gỡ đường ghi: thoả mãn bằng cách không tạo cờ |
| Nhiệm vụ ngoài phạm vi (xoá tệp SQL cloudinary) | Gỡ khỏi phạm vi 024 |
| Cây mã nguồn trong `plan.md` lệch với `tasks.md` | Đồng bộ, gồm hai xung đột trực tiếp về nơi đặt mã |

#### Điểm nhất quán đã kiểm tra

- **105** yêu cầu chức năng: FR-001 → FR-094 liên tục, cộng tám yêu cầu hậu tố `FR-004b`, `FR-004c`, `FR-004d`, `FR-004e`, `FR-005b`, `FR-005c`, `FR-005d`, `FR-005e`, `FR-029b`, `FR-029c`, `FR-092b`. Hậu tố đánh dấu **trình đơn số yêu cầu vô mới** đã tách khỏi nhóm gốc nên đứng ngay cạnh nhau yêu cầu gốc; chúng **không** làm đổi thứ tự thực thi. Bốn hậu tố `FR-004b` → `FR-004e` là phần ghi nhận ký cổng 2, phát sinh từ việc đổi giao diện quản trị sang HTTP và bổ sung đường outbox.
- **29** tiêu chí thành công SC-001 → SC-029 liên tục, không trùng mã.
- Tất cả tham chiếu chéo giữa các yêu cầu đều trỏ tới yêu cầu tồn tại; đã kiểm tra bằng script, bao gồm hậu tố chữ.
- 37 dòng làm rõ, không trùng lặp. Không còn `[NEEDS CLARIFICATION]`.
- **119** nhiệm vụ, T001 → T119 liên tục, 119/119 đúng mẫu định dạng.
- Phủ yêu cầu tính trên **dòng nhiệm vụ**: FR 105/105, SC cần build 28/28.
- Mọi tham chiếu chéo `Txxx` trong phần văn xuôi của `tasks.md` đều trỏ tới nhiệm vụ tồn tại **và đúng nội dung**.
- **Một** tệp migration duy nhất cho cả ba thay đổi lược đồ, theo yêu cầu của người phát triển về việc tránh làm loãng.
- **Không còn yêu cầu nào hướng tới khách hàng cần làm ở kho khác.** Lựa chọn "cổng 2 là cơ chế nội bộ" đã gỡ bỏ cả yêu cầu hiển thị trần chi phí lẫn yêu cầu cảnh báo lượt thứ 5, nên không còn phần nào bị bỏ sót ở kho ứng dụng khách.
- **Đã chốt, không còn chờ**: giao diện quản lý cổng 2 là **HTTP** trong nhóm `/admin/ai` sẵn có của dịch vụ AI, kế thừa `r.auth.Handle()` và `RolesAuthorize(roleslug.Admin)`; bản ghi kiểm toán nằm ở bảng **của AI** chứ không phải `shared.admin_audit_logs`. FR-004b, SC-029
- SC-022 và SC-027 là chỉ số kinh doanh hậu ra, **không** cần nhiệm vụ build.
- Hiến pháp: **0 vi phạm, 0 ngoại lệ**. §VI được tuân thủ trọn vẹn.
- Thứ tự GĐ0 → GĐ8 giữ nguyên là bắt buộc; GĐ3b nằm sau GĐ3 và trước GĐ4.

#### Vòng thứ tư — phát hiện hai cổng hạn mức độc lập

> ⚠️ **Đoạn dưới đây ghi lại kết luận ĐÃ BỊ BÁC BỎ — đừng dùng làm căn cứ.** Phần "Giá trị Premium đang chạy: outfit 15 / 30, chat 20 / 10" và nhận định rằng cổng 2 chặn ở 10 chat là **sai**. Vòng thứ bảy phía dưới đã chứng minh bằng mã và dữ liệu: nhánh `strict` không đọc `max_paid_attempts_per_day`, và một tài khoản Premium đã thực hiện 50 lượt chat trả phí trong một ngày so với cấu hình `10`. Giữ nguyên đoạn này chỉ để thấy **quy trình phân tích đã đi qua đâu**; nhãn "cổng 2 — chi phí AI" cũng đã lỗi thời vì cổng 2 nay có **hai** hành vi tuỳ chế độ.

Lần theo **đường thi hành thực tế** (thay vì tin bảng cấu hình) cho thấy mỗi thao tác AI bị giới hạn bởi **hai cổng tách biệt**:

| | Cổng 1 — người dùng | Cổng 2 — chi phí AI |
|---|---|---|
| Cấu hình | `subscription_plans.ai_*_daily_quota` | `ai_cost_policy_operations.max_paid_attempts_per_day` |
| Bộ đếm | `user_daily_quotas` | `ai_usage_events` |
| Thi hành | Commerce (`workflow_helpers.go:49`, `chat/workflows.go:235`) | AI (`ai_cost_repo_impl.go:96`) |
| Vượt thì | **chặn**, trả lỗi | **hạ mô hình**, vẫn chạy |

Giá trị Premium đang chạy: outfit 15 / 30, chat **20 / 10**. Ba lượt phân tích trước không thấy vì cả ba đều đọc bảng cấu hình và coi đó là chỗ thi hành.

**Đã đóng H2–H5:**
- `FR-004` nêu rõ giao diện lập trình chỉ sửa cổng 1, cổng 2 phải sửa bằng tệp kịch bản, và bắt buộc phải có cách đối chiếu hai cổng.
- `FR-005b` chốt nguyên tắc: **cổng chặn phải là cổng người dùng nhìn thấy**, trừ khi sự khác biệt là cố ý và đã nói với khách.
- `FR-090` và `SC-011` thêm trường hợp kiểm chứng "hai cổng lệch nhau".
- `FR-029c` nêu rõ thứ tự ba bước: sửa bằng tệp kịch bản → kiểm chứng → phát lại tín hiệu, bao gồm cả bốn thao tác.
- `data-model.md` thêm mục "Hai cổng hạn mức AI" với sơ đồ quan hệ.
- Ba nhiệm vụ mới trong Phase 1: cấu hình cổng 2, đối chiếu hai cổng, dừng lại nếu lệch.

**Còn H1 chờ quyết định của người phát triển** — giá trị cụ thể của cổng 2 cho gói Premium. `FR-005` để trống phần này có chủ đích cho tới khi anh chốt.

#### Vòng thứ năm — người phát triển chốt H1

**Mục tiêu được chốt: 30 lượt gợi ý phối đồ và 10 lượt trò chuyện; trần cứng 30.000đ.**

Đối chiếu với dữ liệu đang chạy cho thấy kết quả thuận lợi: **cổng 2 đã đúng mục tiêu** (outfit 30, chat 10), chỉ cổng 1 sai cả hai số. Hệ quả trực tiếp:

| Việc | Trước | Sau |
|---|---|---|
| Nhiệm vụ cấu hình cổng 2 bằng tệp kịch bản | Có (T010 cũ) | **Bỏ** — cổng 2 không cần sửa |
| Nhiệm vụ cấu hình cổng 1 | 30 / 20 | **30 / 10** (T009) |
| Phát lại tín hiệu sau khi đổi cổng | Bắt buộc ở GĐ0 | **Không cần** — chỉ kiểm chứng |

**Điểm mới phát sinh từ chính lựa chọn này — FR-005c.** Với mục tiêu 30/10, chi phí ước tính là 1.820–2.260đ mỗi ngày, nên ngân sách chu kỳ cạn ở mức 27.600đ (92% của trần) sau **12–15 ngày**. Tức là con số theo ngày **không phải hạn mứn cam kết** mà là tốc độ tiêu thụ trong ngân sách cả tháng: phần còn lại của chu kỳ hệ thống dùng mô hình miễn phí.

Vì vậy `FR-005c` mới được thêm: màn hình hạn mức phải hiện **cả hai** — số lượt tối đa mỗi ngày **và** trần chi phí AI của chu kỳ — cùng thông báo khi ngân sách cạn thì chuyển sang mô hình miễn phí. Hiển thị mỗi mình số lượt là sai, vì người dùng sẽ hiểu là được dùng 30 lượt outfit **trong cả tháng**.

- Đã sửa sau phân tích: thêm phương thức trả trần chi phí theo chu kỳ vào giao diện lập trình của hệ thống AI (FR-004b, T010-T011), rồi đưa trản đích vào DTO ghi của Account. Giá trị **không** được nhận bằng vào bảng gói - nhận bản chính nguồn của việc hai cổng lệch nhau.
- Đã chốt sau phân tích: bản ghi kiểm toán cổng 2 **phải** được phát qua outbox để hiện ở màn hình kiểm toán tập trung, không còn là lựa chọn 'nếu cần' (FR-004c, FR-004d, FR-004e, SC-030; T013, T014, T015).

**Hệ quả tài chính đã được ghi nhận và không đề xuất mở lại**: chi phí tối đa không đổi vì trần 30.000đ vẫn chi phối, và 50 tài khoản của SC-022 ở mức tối đa là 1,38 triệuđ, thấp hơn hạn 3,4 triệuđ. Đổi bật, mỗi tài khoản trả 59.000đ nhưng tối đa chỉ tiêu 27.600đ vào AI.

#### Vòng thứ sáu — người phát triển chốt gói miễn phí

**Mục tiêu được chốt: 5 lượt gợi ý phối đồ và 3 lượt trò chuyện; không giới hạn trần cứng.**

Kiểm chứng trên mã cho thấy nhánh `observe_only` **không** bỏ qua kiểm tra số lượt — nó vẫn đếm lượt trả phí trong ngày và chuyển sang mô hình miễn phí khi vượt (`ai_cost_repo_impl.go:90-96`, `cost/usecase.go:124-125`). Nghĩa là lệch ở outfit của gói miễn phí là **lỗi thật**, không phải chuyện hình thức. Đã ghi cảnh báo này vào `spec.md` vì cái tên `observe_only` rất dễ khiến người đọc hiểu nhầm là không thi hành gì.

| Việc | Xử lý |
|---|---|
| Cổng 1 gói miễn phí (5 / 3) | Đã đúng, **không sửa** |
| Cổng 2 gói miễn phí — outfit `3` → `5` | Tệp kịch bản + cặp tệp kiểm tra trước (T010) |
| Phát lại tín hiệu cho gói miễn phí | **Bắt buộc** (T011) — số lượt nằm trong ảnh chụp đóng băng |
| Trần cứng gói miễn phí | `hard_cost_micro_vnd` đang là `NULL`, giữ nguyên |

Yêu cầu mới: **FR-005d**. Chi phí gói miễn phí tính lại theo mục tiêu 5/3: khoảng **13.400đ mỗi tháng mỗi tài khoản**, tăng so với 7.900đ tính trên 3/3. Vẫn nằm trong mức đã được chấp nhận.

Điểm đáng lưu ý cho phần triển khai: hai gói xử lý **ngược nhau**. Premium chỉ sửa cổng 1 bằng giao diện lập trình, không cần tệp kịch bản cũng không cần phát lại tín hiệu. Miễn phí thì ngược lại — cổng 1 đã đúng, cổng 2 phải sửa bằng tệp kịch bản và bắt buộc phải phát lại tín hiệu. Gộp hai gói vào một bước là nguồn gốc của việc bỏ sót bước phát lại tín hiệu.

#### Vòng thứ bảy — phát hiện `max_paid_attempts_per_day` là cấu hình chết với Premium

Câu hỏi của người phát triển ("tại sao cổng 2 lại cần biết số lần tối đa") dẫn tới phát hiện làm **đảo ngược** kết luận của ba lượt phân tích trước.

`MaxPaidAttemptsPerDay` được đọc ở **đúng một chỗ** trong toàn bộ dịch vụ AI: `ai_cost_repo_impl.go:96`, bên trong nhánh `observe_only`. Nhánh `strict` không tham chiếu trường này — nó chỉ so `actual_cost + reserved_cost` với ngưỡng trong sổ tính phí. Nơi quyết định tuyến (`route := plan.Route`) cũng không dùng nó.

Xác nhận bằng dữ liệu: tài khoản `premium-monthly` thực hiện **50 lượt chat trả phí trong một ngày**, trong khi cấu hình ghi `10`. Lượt thứ 11 vẫn trả phí vì tổng tiền khoảng 2.600đ chưa chạm trần 25.000đ.

| Kết luận cũ (sai) | Kết luận đúng |
|---|---|
| Premium: khách thấy 20 chat, dùng được 10 | Premium: khách thấy 20, dùng được đủ 20, chỉ bị chặn bởi ngân sách 30.000đ |
| Cổng 2 Premium "đã đúng mục tiêu nên không cần sửa" | Cổng 2 Premium là **cấu hình chết** — đúng hay sai cũng không ảnh hưởng hành vi |
| Hai cổng lệch nhau ở cả hai gói | Chỉ **gói miễn phí** còn lệch thật |

Cơ chế này hoá ra rất hợp lý khi nhìn theo chế độ thi hành: gói miễn phí **không có ngân sách** nên phải chặn bằng số lượt; Premium **có ngân sách** nên chặn bằng tiền. Một cột duy nhất phục vụ hai cơ chế khác nhau — đó là nguồn gây nhầm lẫn, không phải thiết kế có chủ ý.

Đã thêm **FR-005e** buộc hệ thống coi trường này là cấu hình chết với mọi chính sách `strict`, và buộc tệp kiểm chứng báo cáo hai cột tách biệt. Cũng đã ghi cảnh báo về `observe_only` vào `spec.md` vì cái tên này dễ khiến người đọc hiểu nhầm là không thi hành gì.

#### Vòng thứ tám — người phát triển chốt lần cuối: cổng 2 miễn phí = 4, Premium = NULL

| Quyết định | Giá trị | Ảnh hưởng |
|---|---|---|
| Cổng 2 gói miễn phí — `outfit` | `4` (thay vì `5`) | Tệp kịch bản + phát lại tín hiệu, như đã có |
| Cổng 2 gói Premium — cả bốn thao tác | **`NULL`** | **Cần migration mới** |

Hai việc này có tính chất khác nhau:

- **`4` thay vì `5` là lần đầu hai cổng cố ý lệch nhau.** FR-005b vốn đã có mệnh đề thoát hiểm "trừ khi sự khác biệt là cố ý và đã được nêu rõ với người dùng" — nay được dùng đến. Hệ quả bắt buộc ghi trong FR-005d: lượt thứ 5 gợi ý phối đồ của người dùng miễn phí bị hạ xuống mô hình miễn phí, và hệ thống **MUST** báo trước cho người dùng. Im lặng là vi phạm, vì đó đúng là dạng phàn nàn mà SC-027 đang cố đếm.
- **`NULL` cho Premium cần migration** vì cột đang khai báo `NOT NULL`. Đây là thay đổi lược đồ thứ ba của tính năng. Theo yêu cầu của người phát triển về việc tránh nhiều tệp làm loãng, **cả ba** thay đổi lược đồ được gộp vào **một** tệp Goose `..._admin_premium_grant_schema.sql`. Một tệp là đủ vì cả ba cùng thuộc một lần phát hành và cùng nằm trong cây Goose của Account (AI không có cây migration riêng). Phần `Down` phải làm ngược lại **đúng thứ tự**: điền lại giá trị cũ rồi mới `SET NOT NULL` cho `max_paid_attempts_per_day`; xoá dòng `grant_id IS NULL` rồi mới `SET NOT NULL` cho `grant_id`; cuối cùng mới `DROP INDEX`. Nhiệm vụ kiểm chứng có bắt buộc chạy `migrate-down` rồi `migrate-up` một vòng để bắt lỗi thứ tự này.

**Đánh đổi được chấp nhận**: gộp tệp khiến việc hoàn tác trở thành **all-or-nothing** — không thể quay lại riêng một thay đổi lược đồ mà giữ hai thay đổi kia. Với một tính năng phát hành cùng nhau thì đây là trường hợp bình thường, nhưng phải biết trước khi cần.

Ba nguyên tắc cho `NULL` đã chốt trong FR-005e: cho phép NULL; NULL nghĩa là "chính sách này không dùng giới hạn số lượt"; và NULL ở chế độ `observe_only` là **cấu hình sai** phải ghi cảnh báo rồi áp tầng suy giảm 2, không được âm thầm bỏ qua — vì bỏ qua sẽ để gói miễn phí chạy không giới hạn.

Chi phí gói miễn phí tính lại theo 4/3: khoảng **11.600đ mỗi tháng mỗi tài khoản**.

#### Vòng thứ ba — chốt ranh giới ảnh chụp và cấu hình sống

Edge Case của spec hỏi *"chính sách phí sửa trong lúc bản ghi còn hiệu lực — dùng ảnh chụp cũ hay cấu hình mới?"* và không nơi nào trả lời. Đã kiểm chứng trên mã: `ResolvePolicy` đọc cấu hình **sống**, còn `PolicySnapshot` **không nơi nào đọc** — nó là cột ghi một chiều.

Đã nghiên cứu hai hướng và cả hai cho cùng một công thức:

| Nguồn | Phía quyền lợi | Phía trần chi phí |
|---|---|---|
| Chargebee | Grandfathering: giữ giá cũ là lựa chọn **tường minh**, mặc định áp dụng cho tất cả | — |
| Stripe | Đổi giá phải thay thẳng subscription item, không âm thầm đổi | — |
| GCP Spend Cap (7/2026) | — | Ngưỡng luôn là giá trị **đang cấu hình**, sửa là có hiệu lực ngay |
| Firebase | — | *"You remain responsible for any overage costs"* — không giữ trần cũ |

**Đã chốt**: ảnh chụp cho **quyền lợi** (số lượt/ngày, giới hạn token), cấu hình **sống** cho **chống lỗ tài chính** và **định tuyến**. Lý do quyết định nằm ở dữ liệu thật: 5 tài khoản gói miễn phí có bản ghi `effective_to IS NULL` — nếu trần chi phí nằm trong ảnh chụp thì việc hạ trần **không có tác dụng lên họ, có thể vĩnh viễn**.

**Không cần migration DDL** — `policy_snapshot` đã là `jsonb NOT NULL` không lược đồ. Nhưng đã phát hiện 8 ảnh chụp hiện có **thiếu hẳn nhóm quyền lợi**, trộn hai kiểu ghi (25 khoá), và lệch cấu hình sống (`maxUnknownPaidRequestsPerDay: 0` so với cột thật `2`). Nên phạm vi GĐ3b nới từ 8 dòng rỗng lên **ghi đè toàn bộ**. Chi tiết ở research.md R13.
