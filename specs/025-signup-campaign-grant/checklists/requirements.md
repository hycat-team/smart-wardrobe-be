# Specification Quality Checklist: Cơ chế chiến dịch tặng gói cho tài khoản đăng ký mới

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-10-01
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

## Kết quả kiểm tra lần 2 (2026-10-01)

Đã xử lý xong 3 câu hỏi còn treo. Đặc tả đã được cập nhật theo 3 quyết định và kiểm tra lại bằng máy.

### Kiểm tra bằng máy

| Mục | Kết quả |
|---|---|
| Số marker `[NEEDS CLARIFICATION]` còn lại | **0** |
| Số yêu cầu chức năng | **77**, đánh số từ FR-001 tới FR-077, **không trùng, không thiếu số** |
| Số tiêu chí thành công | **19** (SC-001 tới SC-016, trong đó SC-001 có 4 nhánh a/b/c/d và có SC-001b, SC-001c, SC-001d tính riêng) |
| Số câu chuyện người dùng | **7**, mỗi câu chuyện đánh số thứ tự ưu tiên và có phần kiểm thử độc lập |
| Số chuyện cần hỏi về ranh giới (edge case) | **24**, chia thành 6 nhóm: tính đúng đắn hạn mức, thời điểm, điều kiện đủ điều kiện, nhiều chiến dịch, vận hành, định nghĩa "tài khoản đầu tiên" |

### Đạt

- **Câu chuyện người dùng độc lập được**: US1 (cấp gói), US2 (email), US3 (đúng hạn mức), US4 (bộ đếm lượt), US5 (tái sử dụng), US6 (loại trừ quản trị viên), US7 (tra cứu cho vận hành). Mỗi câu chuyện nêu rõ cách kiểm chứng mà không cần các câu chuyện khác.
- **Yêu cầu kiểm chứng được**: FR-053 nêu ra tình huống cụ thể (29 trên 30, ba thao tác cùng lúc, kết quả phải là 30) thay vì nói chung chung. FR-030 và FR-031 nêu rõ cơ chế của mốc ưu tiên và điều kiện so sánh.
- **Tiêu chí thành công đo được và trung lập công nghệ**: SC-001 (khoảng 50 tới 55), SC-001b (không lượt cấp dự phòng nào tạo sau mốc ưu tiên), SC-001c (dự phòng không dùng hết thì tổng ít hơn 55), SC-001d (thứ tự thuận cho 50, thứ tự ngược có thể 55, cả hai đều đúng), SC-004 (chậm hơn dưới 10%), SC-009 (1000 thao tác đồng thời), SC-010 (chỉ thêm cấu hình), SC-011 (dưới 1 phút). Không mục nào nhắc tới ngôn ngữ, khung làm việc, hay công cụ.
- **Phạm vi được chốt rõ**: Có mục `Ngoài phạm vi` (FR-073 tới FR-077) nói thẳng cái gì **không** làm, gồm cả việc không cho sửa hạn mức từ giao diện, và việc không sửa lõi kiểm soát chi phí AI.
- **Đánh đổi được nêu thẳng, không giấu**: Mục "Bốn quyết định đã chốt" nói rõ quyết định 1 **phá vỡ tính tuyệt đối của con số 50**. Mục Giả định ghi rõ hệ quả "chi phí cấp gói có thể vượt 50 tài khoản", và ghi rõ giá trị hạn mức dự phòng 5 là **con số quyết định theo phán đoán chứ chưa có số liệu thực tế**, có thể chỉnh bằng khai báo.

### Điểm cần lưu ý khi lập kế hoạch

1. **Đặc tả trộn 2 việc**: (a) sửa lỗi có sẵn về bộ đếm lượt dùng AI, và (b) xây cơ chế chiến dịch. Đây là **có chủ ý** do người dùng chốt, không phải trôi dạt. Lý do đã nêu ở mục "Cần sửa trước, rồi mới xây chiến dịch". Khi lập kế hoạch phải tách thành 2 nhóm giai đoạn rõ ràng, và **giữ nguyên ranh giới này trong danh sách công việc** để việc sửa lỗi không bị trộn lẫn với việc xây mới.

2. **Cơ chế của mốc ưu tiên (FR-030) là quyết định thiết kế, chưa phải kết luận duy nhất.** Đặc tả đã chốt **quy tắc kinh doanh** (ưu tiên tài khoản tạo sớm, trần cứng 55, kết quả không đổi theo thứ tự xử lý) nhưng chưa chốt **cách hiện thực**. Có ít nhất ba cách, mỗi cách đổi đổi lượng công việc và rủi ro khác nhau:
   - Cấp ngay theo ngân sách chính, chốt mốc ưu tiên tại lần cấp thứ 50, rồi bù từ dự phòng cho tài khoản tạo sớm hơn mốc đó. Đơn giản nhất, không cần thêm lịch trình. Đây là hướng mặc định của đặc tả.
   - Ghi nhận toàn bộ tài khoản đủ điều kiện thành ứng viên, đóng cửa sổ nhận rồi mới chọn ra N ứng viên sớm nhất. Công bằng tuyệt đối nhưng **phải chờ đóng cửa sổ** mới cấp gói, gây trải nghiệm xấu.
   - Cấp ngay rồi thu hồi và thay thế ở cuối cửa sổ. Công bằng nhưng người dùng bị lấy rồi lại lần nữa, không chấp nhận được.
   
   ⇒ Điểm này **đã được giải quyết trong phiên `/speckit-clarify` ngày 2026-10-01**: chọn phương án 1 (cấp ngay, bù sau bằng hạn mức dự phòng) và hạ yêu cầu công bằng xuống mức cơ chế thực sự bảo đảm được. Xem mục `## Clarifications` trong `spec.md`.

3. **Giao diện tra cứu là khối lượng lớn nhất còn lại** (User Story 7, FR-065 tới FR-072). Nó kéo theo DTO, định tuyến, phân quyền, phân trang, và kiểm thử. Khi lập kế hoạch cần cân nhắc tách thành một nhóm công việc riêng để có thể cắt bớt nếu cần.

4. **Có 1 vấn đề có sẵn được ghi nhận tường minh là không sửa** trong lần này (mục "Một vấn đề đã phát hiện nhưng quyết định không sửa" + FR-075). Đã kiểm tra lại rằng nó **không ảnh hưởng** tới chiến dịch: chiều từ gói miễn phí lên gói Premium không bị chặn, vì gói Premium không đọc bộ đếm đó.

## Notes

- Tất cả các mục đã đánh dấu đạt sau lần kiểm tra thứ 2.
- Checklist này đã hoàn tất. Bước tiếp theo là `/speckit-clarify` (nếu còn muốn làm rõ thêm) hoặc `/speckit-plan` (sinh tài liệu thiết kế).

## Lần kiểm tra sau `/speckit-clarify` (2026-10-01)

### Kết quả

**19/19 mục đạt, trước và sau đều 19/19. Không mục nào đổi trạng thái, không có hồi quy.**

Phiên làm rõ đã hỏi và tích hợp **4 câu hỏi** (xem mục `## Clarifications` trong `spec.md`):

| # | Câu hỏi | Kết quả |
|---|---|---|
| 1 | Cơ chế dự phòng + mốc ưu tiên có bảo đảm công bằng tuyệt đối bất kể thứ tự tin nhắn không | Không. Hạ yêu cầu xuống điều kiện thực thi được, ghi nhận giới hạn tường minh |
| 2 | Cam kết cấp gói trong bao lâu, có cơ chế đóng sót không | Hai tầng: 5 phút cho đường bình thường + quét bù mỗi 15 phút |
| 3 | Có nên chuyển sang phát hướng publisher để giảm ràng buộc không | Có. Sự kiện `identity.user.created` phát vô điều kiện, luồng tạo tài khoản không biết chiến dịch tồn tại |
| 4 | Phát sự kiện từ 2 hay 3 đường tạo tài khoản | Cả 3, kèm cờ `createdByAdmin`; bên chiến dịch tự bỏ qua |

### Mâu thuẫn đã phát hiện và sửa trong lúc làm rõ

Câu hỏi 1 phát hiện mâu thuẫn **trong chính đặc tả do câu trả lời ở phiên trước tạo ra**: tiêu chí đòi hỏi kết quả không đổi theo thứ tự tin nhắn, trong khi cơ chế dự phòng + mốc ưu tiên **không thể** thỏa được điều đó. Kiểm chứng bằng 55 tài khoản, `quota=50`, `dự phòng=5`:

| Thứ tự tin nhắn | Tổng cấp | Tập tài khoản được cấp |
|---|---|---|
| Đúng thứ tự đăng ký | 50 | tài khoản 1 tới 50 (đúng 50 người sớm nhất) |
| Ngược thứ tự | 55 | tài khoản 1 tới 55 — **toàn bộ 55 tài khoản**, không phải 50 người sớm nhất |

Ngoài ra còn sửa: bảng thuật ngữ ghi sai "mỗi chiến dịch 1 lượt" (mâu thuẫn với quyết định mỗi tài khoản 1 lượt cho toàn bộ cơ chế), một tham chiếu chéo chỉ sai số, và User Story 3 còn giữ tuyên bố tuyệt đối "không bao giờ có tài khoản thứ 51".

### Yêu cầu được bổ sung trong lúc làm rõ

| Khoảng | Nội dung |
|---|---|
| FR-017 | Tài khoản tạo **sau** thời điểm kết thúc chiến dịch cũng không đủ điều kiện |
| FR-050, FR-051 | Tách rõ: quản trị viên không tiêu suất, **vẫn** được cấp gói miễn phí, cấp thủ công vẫn chạy |
| FR-078 tới FR-083 | Cam kết 5 phút, cơ chế quét bù, chung khoá và chung bộ luật, ghi nhận độ trễ, thời gian quét khai báo được |
| FR-084 tới FR-091 | Sự kiện `identity.user.created`: phát vô điều kiện, cấm phụ thuộc cấu hình chiến dịch, phát từ cả 3 đường, cấm chứa dữ liệu nhận diện cá nhân, chỉ là đường tắt, cờ quản trị viên |
| SC-015, SC-016 | 95% trong 5 phút; 100% trong 1 giờ kể cả khi hệ thống gián đoạn |

### Kiểm tra bằng máy sau khi tích hợp

| Mục | Kết quả |
|---|---|
| Marker `[NEEDS CLARIFICATION]` | **0** |
| Yêu cầu chức năng | **91**, từ FR-001 tới FR-091, không trùng, không thiếu số |
| Tham chiếu chéo FR/SC chỉ tới mục không tồn tại | **0** |
| Tiêu chí thành công | **19** |
| Câu chuyện người dùng | **7** (User Story 6 nay có 4 kịch bản chấp nhận) |
| Bullet trong mục Clarifications | **4**, đúng bằng số câu hỏi đã hỏi |
| Tiêu đề mới được phép | Chỉ `## Clarifications` và `### Session 2026-10-01` |

### Điểm cần lưu ý cho `plan.md`

1. **FR-084 tới FR-091 thay đổi hình dạng phần phát sự kiện.** Chủ đề `identity.user.created` là **chủ đề dùng chung cho nhiều tính năng trong tương lai**, không phải chủ đề riêng của chiến dịch. Tên hàng đỗi phía nhận thông báo cũng vì thế mà đặt theo tính năng, không đặt theo chiến dịch.

2. **Sự kiện không còn là cơ chế bảo đảm tính đúng đắn** (FR-089). Kế hoạch phải coi cơ chế quét bù (FR-079) là nguồn bảo đảm duy nhất, và phải chứng minh được rằng quét bù tìm ra **đúng tập** tài khoản mà sự kiện sẽ tìm ra, cộng thêm các tài khoản do sự kiện bị thất lạc.

3. **Số chủ đề mới tăng lên 2** (`identity.user.created` và chủ đề của bên chiến dịch). Giả định "không cần thêm hạ tầng mới" vẫn đúng, nhưng kế hoạch phải khai báo cả hai.

4. **Đặc tả vẫn trộn 2 việc**: sửa lỗi bộ đếm lượt dùng AI (User Story 4, FR-052 tới FR-058) và cơ chế chiến dịch. Có chủ ý, nhưng phải tách thành 2 nhóm giai đoạn rõ ràng trong danh sách công việc.

## Lần kiểm tra sau phiên làm rõ thứ hai (2026-10-01)

Đề nghị của người dùng: kiểm tra lại xem sự kiện thay đổi gói có đúng là **không chỉ** được phát khi tạo tài khoản hay không.

### Kết quả kiểm chứng

**Xác nhận: không chỉ tạo tài khoản.** Đã rà toàn bộ mã nguồn module gói cước:

| Kiểm tra | Kết quả |
|---|---|
| Số đường ghi vào bảng gói của tài khoản | 12 đường |
| Số đường có phát sự kiện thay đổi gói | **12 / 12** |
| Câu lệnh SQL ghi thô nào ghi vòng qua không | 0 |

Bao gồm cả trường hợp gói **không đổi** (bật/tắt tự động gia hạn) — vẫn phát, vì bên nhận mới là bên quyết định có xử lý hay không.

⇒ Ba yêu cầu đã dựa trên giả định này là chính xác: FR-036, FR-037 và SC-008.

### Câu hỏi đã hỏi: 1/5

| Câu hỏi | Kết quả |
|---|---|
| Có thêm loại sự kiện thay đổi gói riêng cho lần cấp gói của chiến dịch không | **Không** — dùng chung loại hiện có, phân biệt nguồn gốc bằng nhật ký kiểm toán |

Lý do được chọn: tập loại sự kiện là **đóng** và dùng chung cho cả hai dịch vụ; bên nhận **từ chối vĩnh viễn** mọi loại ngoài tập và đưa thông báo vào hàng đỗi chết. Thêm loại mới không phải cải tiến mà là tạo một điểm hỏng mới: nếu bên nhận chưa kịp triển khai, mọi lần cấp gói của chiến dịch sẽ mất sạch.

### Yêu cầu được bổ sung

| Khoảng | Nội dung |
|---|---|
| FR-036 | Bổ sung trỏ tới nhóm ràng buộc mới |
| FR-092 | Cấm thêm loại thay đổi gói mới vào tập đóng dùng chung |
| FR-093 | Lần cấp gói của chiến dịch dùng **cùng** loại sự kiện với lần cấp gói của quản trị viên |
| FR-094 | Nguồn gốc "cấp bởi chiến dịch" chỉ ghi ở nhật ký kiểm toán, không ở lịch sử thay đổi gói |
| FR-095 | Kiểm thử phải chứng minh sự kiện được phát trên ít nhất 6 đường |

Ngoài ra ghi vào mục `Bối cảnh` một mục **"Đã kiểm chứng: mọi đường đổi gói đều phát sự kiện"** với bảng đối chiếu 12 đường, và thêm một dòng vào bảng thuật ngữ cho loại sự kiện `ACTIVATED`.

### Kết quả kiểm tra lại

**19/19 mục đạt, trước và sau đều 19/19. Không mục nào đổi trạng thái, không có hồi quy.**

| Kiểm tra bằng máy | Kết quả |
|---|---|
| Marker `[NEEDS CLARIFICATION]` | **0** |
| Yêu cầu chức năng | **95**, từ FR-001 tới FR-095, không trùng, không thiếu số |
| Tham chiếu chéo FR/SC chỉ tới mục không tồn tại | **0** |
| Tiêu chí thành công | **19** |
| Câu chuyện người dùng | **7** |
| Bullet trong mục Clarifications | **5** |
| Tiêu đề mới được phép | Chỉ `## Clarifications` và `### Session 2026-10-01` |
| Dung lượng `spec.md` | 537 dòng |

### Bổ sung cho danh sách việc dành cho `plan.md`

5. **Cấm thêm loại sự kiện mới** (FR-092) là ràng buộc kiến trúc, không phải sở thích. Trình biên dịch không bắt được: ai đó thêm một hằng số mới vào tập đóng cũng chạy được, và lỗi chỉ xuất hiện ở dịch vụ nhận, dưới dạng thông báo bị loại trong hàng đỗi chết. Cần một kiểm thử khẳng định danh sách loại sự kiện khớp giữa hai dịch vụ.

6. **Lần cấp gói của chiến dịch ghi loại sự kiện `ACTIVATED`, không phải `UPGRADED`**, vì điều kiện nâng cấp yêu cầu hạng cấp cũ lớn hơn 0 mà tài khoản gói miễn phí có hạng 0. Dịch vụ AI xử lý `ACTIVATED` đúng như cần, nên không có vấn đề về hành vi — nhưng `plan.md` phải biết để không viết kiểm thử đoán sai loại sự kiện.

## Lần kiểm tra sau phiên làm rõ thứ ba (2026-10-01)

Đề nghị của người dùng: chỉ phát `identity.user.created` lúc tạo tài khoản, **không** phát `subscription.plan.changed` — để mỗi lần tạo tài khoản chỉ bắn một chủ đề. Sau đó hỏi tiếp: nếu cho bộ nhận thông báo nghe cả hai chủ đề thì sao.

### Kết quả: **không khả thi**, đã chốt giữ cả hai

| Câu hỏi | Kết quả |
|---|---|
| Có thể bỏ `subscription.plan.changed` lúc tạo tài khoản không | **Không** |
| Có thể cho bộ nhận thông báo nghe cả hai chủ đề không | **Không giải quyết được** — vẫn phát hai chủ đề, hoặc buộc phải tra cứu gói cước |

Lý do cốt lõi: sự kiện đổi gói phát lúc cấp phát gói mặc định là **tín hiệu duy nhất** để dịch vụ AI biết tài khoản mới được cấp quyền chi phí. Bảng quyền chi phí có **đúng một nhà ghi**, là dịch vụ AI, và nó chỉ tới đó bằng cách tiêu thụ chủ đề đổi gói (kiến trúc §VII, mục kiến trúc dòng 414).

Dịch vụ AI có cơ chế dự phòng 2 tầng cho tài khoản không có bản ghi quyền, nên bỏ sự kiện **không làm hỏng hệ thống** — nhưng mọi tài khoản mới sẽ đi qua đường dự phòng, dùng chính sách **cấu hình** thay vì chính sách theo gói, và bị đếm suy giảm chất lượng ở mỗi lần gọi. Cơ chế dự phòng được viết ra cho trường hợp "quyền chưa tới" trong vài giây, không phải "quyền sẽ không bao giờ tới".

Một phát hiện kỹ thuật quan trọng trong lúc này: **cờ `createdByAdmin` không lấy được từ dữ liệu.** Bảng tài khoản không có trường nào ghi ai đã tạo; hàng do quản trị viên tạo và hàng tự đăng ký giống hệt nhau về cấu trúc. Cờ chỉ lấy được từ đường gọi, tức chỉ người phát ở module tài khoản mới biết. Đây là lý do người phát chủ đề tài khoản không thể chuyển sang module gói cước.

### Yêu cầu được bổ sung

| Khoảng | Nội dung |
|---|---|
| FR-096 | Lúc tạo tài khoản phải phát **cả hai** chủ đề, không được bỏ chủ đề nào |
| FR-097 | Cấm thay đổi cách dịch vụ AI nhận biết tài khoản mới; cấm cho dịch vụ AI nghe thêm chủ đề mới; cấm để dịch vụ AI tra cứu gói cước |
| FR-098 | Cấm module tạo tài khoản tra cứu gói cước hay chính sách chi phí để làm phong bị thông báo |

Kèm một bảng **ba phương án đã cân nhắc và loại bỏ** trong mục `Sự kiện tạo tài khoản`, để `plan.md` không tranh luận lại.

### Kết quả kiểm tra lại

**19/19 mục đạt, trước và sau đều 19/19. Không mục nào đổi trạng thái, không có hồi quy.**

| Kiểm tra bằng máy | Kết quả |
|---|---|
| Marker `[NEEDS CLARIFICATION]` | **0** |
| Yêu cầu chức năng | **98**, từ FR-001 tới FR-098, không trùng, không thiếu số |
| Tham chiếu chéo FR/SC chỉ tới mục không tồn tại | **0** |
| Tiêu chí thành công | **19** |
| Câu chuyện người dùng | **7** |
| Bullet trong mục Clarifications | **6** |
| Dung lượng `spec.md` | 541 dòng |

### Bổ sung cho danh sách việc dành cho `plan.md`

7. **FR-097 và FR-098 là ràng buộc phòng ngừa lỗi, và trình biên dịch không bắt được.** Không có gì trong hệ thống khiến việc "cho dịch vụ AI nghe thêm một chủ đề nữa" là hỏng — nó chỉ biến mọi tài khoản mới thành tài khoản không có quyền AI mà không ai báo lỗi. Nên có một kiểm thử khẳng định bộ nhận thông báo của dịch vụ AI chỉ tiêu thụ **đúng một** chủ đề.

8. **Cờ `createdByAdmin` là thông tin chỉ lấy được từ đường gọi.** Kế hoạch phải ghi rõ nó được truyền từ module tài khoản vào thông báo, và **không** được suy ra từ dữ liệu. Nếu sau này ai đó cố suy ra từ bảng tài khoản, kết quả sẽ là mọi tài khoản bị nhận nhầm là tự đăng ký.
