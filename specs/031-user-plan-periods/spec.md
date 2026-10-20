# Feature Specification: Lịch sử gói theo khoảng hiệu lực

**Feature Branch**: `031-user-plan-periods`

**Created**: 2026-10-20

**Status**: Draft

**Input**: User description: "Lịch sử gói theo khoảng hiệu lực để phân bố chính xác theo kỳ (đóng nợ 029/D-06)"

## Bối cảnh

Báo cáo phân bố gói của admin (`GET /api/v1/admin/dashboard/subscriptions/distribution`) hiện trả lời
sai câu hỏi "trong kỳ [fromDate, toDate], mỗi gói có bao nhiêu người dùng". Nguyên nhân là hệ thống chỉ
lưu **trạng thái gói hiện tại của mỗi người dùng**; khi người dùng đổi hoặc hạ gói, thông tin về gói
trước đó bị ghi đè và mất hoàn toàn.

Hệ quả quan sát được:

- Người dùng hạ gói **sau** kỳ báo cáo bị loại khỏi mọi nhóm thay vì được tính theo gói họ đã dùng
  trong kỳ. Ví dụ: dùng gói trả phí suốt tháng 9, hạ gói ngày 01/10 thì báo cáo tháng 9 trả về 0 cho gói
  trả phí và người dùng đó không xuất hiện ở nhóm nào.
- Người dùng đổi gói **giữa** kỳ bị quy toàn bộ kỳ về gói mới; thời gian dùng gói cũ biến mất.
- Các số liệu phụ thuộc (tổng người dùng, tỷ lệ phần trăm) vì thế cũng sai.
- Không thể tái dựng lịch sử từ dữ liệu sự kiện hiện có, vì sự kiện gia hạn không ghi lại thời điểm
  hết hạn kết quả và dữ liệu giao dịch chỉ chứng minh việc mua, không chứng minh thời gian giữ gói.

Nợ này đã được ghi nhận và chấp nhận ở thời điểm làm báo cáo admin (mục D-06 của feature 029). Feature
này đóng nợ đó bằng cách lưu **lịch sử khoảng thời gian giữ gói** một cách tường minh.

## Clarifications

### Session 2026-10-20

- Q: Một người dùng giữ 2 gói trong cùng kỳ (đổi gói giữa kỳ) thì totalUsers và tỷ lệ phần trăm nên
  tính thế nào? → A: Tổng theo nhóm (tổng của các nhóm, mỗi nhóm đếm theo gói), tỷ lệ phần trăm luôn
  cộng 100%; người đổi gói giữa kỳ được tính ở cả hai gói nên tổng lớn hơn số người thực, và điều này
  được ghi rõ trong tài liệu.
- Q: Khi client không truyền fromDate/toDate thì báo cáo phân bố phải trả về gì? → A: Ảnh chụp trạng
  thái hiện tại — gói đang có hiệu lực tại thời điểm gọi, giống hành vi hiện nay.
- Q: User story 2 (admin tra cứu lịch sử gói của một người dùng) có cần một endpoint mới trong feature
  này không? → A: Không. Lịch sử là dữ liệu nền, tra cứu qua công cụ vận hành/backoffice hiện có;
  feature chỉ đóng nợ báo cáo phân bố.
- Q: Khi kỳ báo cáo bắt đầu trước mốc lịch sử nhưng kết thúc sau mốc thì hệ thống nên làm gì? → A: Tính
  số cho phần có dữ liệu và đánh dấu kỳ là "một phần" để admin không đọc nhầm là số đầy đủ.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Admin đọc báo cáo phân bố đúng theo kỳ (Priority: P1)

Là quản trị viên, tôi chọn một kỳ đã có lịch sử đầy đủ và muốn biết mỗi gói có bao nhiêu người dùng trong kỳ đó,
kể cả những người đã đổi hoặc hạ gói sau kỳ.

**Why this priority**: Đây là mục đích tồn tại của báo cáo. Số liệu sai khiến mọi quyết định dựa trên
báo cáo (đánh giá hiệu quả gói, doanh thu theo gói) không đáng tin.

**Independent Test**: Đặt mốc lịch sử lùi về trước kỳ kiểm thử, dựng ba người dùng: (a) dùng gói trả phí trong kỳ rồi hạ gói sau kỳ,
(b) đổi từ gói A sang gói B giữa kỳ, (c) dùng gói miễn phí suốt kỳ. Gọi báo cáo cho kỳ đó
và so với bảng tính chuẩn. Test độc lập với phần còn lại của hệ thống vì chỉ cần dựng dữ liệu và đọc
một endpoint.

**Acceptance Scenarios**:

1. **Given** người dùng dùng gói trả phí hết kỳ và hạ gói sau khi kỳ kết thúc, **When** admin xem báo cáo kỳ đó, **Then** người dùng được tính cho gói trả phí trong kỳ đó và không bị mất khỏi tổng.
2. **Given** người dùng đổi từ gói A sang gói B giữa kỳ, **When** admin xem báo cáo kỳ đó,
   **Then** người dùng xuất hiện ở cả gói A và gói B (vì đã giữ cả hai trong kỳ).
3. **Given** người dùng chỉ dùng gói miễn phí, **When** admin xem bất kỳ kỳ nào sau khi họ tham gia,
   **Then** người dùng được tính cho gói miễn phí trong kỳ đó, miễn là kỳ nằm sau mốc lịch sử.
4. **Given** không có ai dùng một gói trong kỳ, **When** admin xem báo cáo, **Then** gói đó không xuất
   hiện trong kết quả và tỷ lệ phần trăm vẫn cộng đúng 100%.
5. **Given** kỳ báo cáo nằm hoàn toàn trước mốc lịch sử, **When** admin xem báo cáo, **Then** phản hồi báo "không đủ dữ liệu" thay vì biểu đồ 0%.

---

### User Story 2 - Đối chiếu lịch sử gói của một người dùng (Priority: P2)

Là kỹ sư vận hành đang xử lý khiếu nại, tôi muốn đối chiếu một người dùng đã giữ những gói nào, từ ngày
nào đến ngày nào, với giao dịch thanh toán của họ.

**Why this priority**: Giải quyết tranh chấp và hoàn tiền cần lịch sử chính xác; đây là giá trị thứ hai
của cùng nguồn dữ liệu, không cần thêm hạ tầng mới. Feature này **không mở endpoint mới**; lịch sử được
tra cứu qua SQL chỉ đọc trên psql (đã tài liệu hoá ở `docs/development/technical-notes.md`).

**Independent Test**: Dựng một người dùng trải qua bốn bước: kích hoạt gói trả phí, gia hạn, nâng cấp,
hạ gói. Đọc lịch sử bằng SQL vận hành đã tài liệu hoá (psql, chỉ đọc) và kiểm tra số khoảng, mốc bắt đầu/kết thúc, không có khoảng chồng lấn; khi chuyển gói liên tục thì khoảng sau bắt đầu đúng mốc khoảng trước kết thúc; khi gói hết hạn rồi mua lại thì có khoảng hở đúng như thực tế.

**Acceptance Scenarios**:

1. **Given** người dùng đã đổi gói nhiều lần mà không có khoảng hết hạn ở giữa, **When** kỹ sư vận hành đọc lịch sử gói, **Then** các
   khoảng xuất hiện theo thứ tự thời gian, khoảng sau bắt đầu đúng mốc khoảng trước kết thúc.
2. **Given** người dùng gia hạn cùng một gói, **When** đọc lịch sử, **Then** không sinh thêm khoảng
   trùng lặp; khoảng hiện có chỉ được kéo dài thời điểm kết thúc.
3. **Given** người dùng dùng gói vô hạn (miễn phí hoặc trọn đời), **When** đọc lịch sử, **Then** khoảng
   đó không có thời điểm kết thúc và không bị coi là đã hết hạn.

---

### User Story 3 - Hệ thống tự ghi lịch sử, không cần thao tác vận hành (Priority: P3)

Là kỹ sư vận hành, tôi muốn lịch sử gói được ghi tự động mỗi khi gói của người dùng thay đổi, để không
phải chạy job đối soát thủ công và không có ngày nào bị thiếu dữ liệu.

**Why this priority**: Nếu lịch sử được ghi bởi job định kỳ, mọi thay đổi trong ngày sẽ bị mất hoặc sai
lệch. Ghi ngay tại thời điểm thay đổi loại bỏ cả lớp lỗi này.

**Independent Test**: Thực hiện một thay đổi gói và kiểm tra lịch sử có thêm đúng một khoảng, đồng thời
kiểm tra rằng nếu thao tác nghiệp vụ thất bại thì lịch sử không có bản ghi mồ côi.

**Acceptance Scenarios**:

1. **Given** một thao tác đổi gói thành công, **When** kiểm tra lịch sử, **Then** có đúng một khoảng mới
   và khoảng cũ được đóng tại cùng mốc thời gian.
2. **Given** thao tác đổi gói bị lỗi và được hoàn tác, **When** kiểm tra lịch sử, **Then** không có
   khoảng nào được ghi thêm.
3. **Given** cùng một thao tác được thực hiện lại do retry, **When** kiểm tra lịch sử, **Then** không
   sinh khoảng trùng.

---

### Edge Cases

- **Kỳ chỉ có ngày bắt đầu hoặc chỉ có ngày kết thúc**: báo cáo vẫn phải trả lời nhất quán — chỉ có
  ngày bắt đầu nghĩa là "từ ngày đó đến hiện tại", chỉ có ngày kết thúc nghĩa là "từ trước đến hết
  ngày đó" — thay vì bỏ qua một phía điều kiện như hiện nay.
- **Gói đã hết hạn nhưng chưa được hạ xuống gói miễn phí**: người dùng vẫn phải xuất hiện trong gói
  trả phí cho đến thời điểm hết hạn, và không bị mất khỏi tổng.
- **Hai thay đổi gói xảy ra đồng thời cho cùng một người dùng**: lịch sử không được có hai khoảng
  chồng lấn; một thay đổi phải thắng và thay đổi còn lại phải phản ánh đúng trạng thái sau cùng.
- **Thay đổi gói xảy ra đúng mốc khoảng đang mở bắt đầu**: khoảng mới thay thế khoảng cũ thay vì tạo
  khoảng có độ dài bằng 0; không có khoảng rỗng trong lịch sử.
- **Admin restore lại đúng gói đang giữ**: không tạo và không sửa khoảng nào; lịch sử giữ nguyên nguồn
  ghi nhận ban đầu.
- **Kỳ báo cáo nằm hoàn toàn trước mốc có lịch sử đầy đủ**: báo cáo phải nêu rõ "không đủ dữ liệu cho
  giai đoạn này" thay vì trả về số 0 gây hiểu sai.
- **Kỳ báo cáo cắt ngang mốc lịch sử**: hệ thống tính số cho phần sau mốc và đánh dấu kỳ là "một phần",
  để không ai đọc nhầm là số đầy đủ.
- **Kỳ có ranh giới trùng đúng thời điểm chuyển gói**: người dùng chỉ được tính một lần cho mỗi gói
  tại mỗi thời điểm, không nhân đôi và không rơi vào khe hở.
- **Gói bị đổi tên hoặc ngừng bán**: lịch sử vẫn hiển thị đúng tên gói tại thời điểm người dùng dùng,
  không bị gộp sai hoặc biến mất.
- **Người dùng chưa từng có gói nào**: không xuất hiện trong báo cáo và không làm sai tỷ lệ phần trăm
  của các gói khác.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST lưu lịch sử gói của người dùng dưới dạng các **khoảng hiệu lực** liên tiếp,
  mỗi khoảng gồm: người dùng, gói, tên gói tại thời điểm đó, mốc bắt đầu, mốc kết thúc (cho phép không
  có mốc kết thúc với gói vô hạn).
- **FR-002**: Với mỗi người dùng, các khoảng MUST không chồng lấn và không có khoảng hở giữa hai khoảng
  liên tiếp khi việc chuyển gói xảy ra liên tục.
- **FR-003**: Mỗi khi gói của người dùng thay đổi (kích hoạt, nâng cấp, ghi đè trọn đời, phục hồi gói
  dự phòng, hạ xuống miễn phí, phục hồi bởi admin), hệ thống MUST đóng khoảng đang mở tại đúng mốc hiệu
  lực và mở khoảng mới, trong **cùng một giao dịch** với thay đổi gói. Nếu thay đổi xảy ra đúng tại mốc
  khoảng đang mở bắt đầu (khoảng mới có độ dài bằng 0, ví dụ cấp gói ngay sau khi tạo tài khoản), hệ
  thống MUST thay thế khoảng đó thay vì tạo khoảng rỗng. Nếu sự kiện không làm đổi gói (admin restore cùng gói, `AUTO_RENEW_UPDATED`), hệ thống MUST NOT tạo hoặc sửa khoảng; sự kiện admin restore (metadata source=admin_restore) MUST NOT đụng lịch sử kể cả khi chạy đồng thời với một giao dịch mua.
- **FR-004**: Khi người dùng gia hạn cùng một gói, hệ thống MUST chỉ cập nhật mốc kết thúc của khoảng
  đang mở, MUST NOT tạo khoảng mới.
- **FR-005**: Khi gói hết hạn và người dùng được hạ xuống gói miễn phí, hệ thống MUST đóng khoảng trả
  phí tại đúng thời điểm hết hạn và mở khoảng miễn phí tại cùng mốc đó.
- **FR-006**: Việc ghi lịch sử MUST idempotent: thực hiện lại cùng một thay đổi (retry, sự kiện trùng)
  MUST NOT tạo khoảng trùng hoặc sửa khoảng đã có.
- **FR-007**: Báo cáo phân bố MUST đếm người dùng theo gói dựa trên các khoảng giao với kỳ yêu cầu, và
  MUST tính cả người dùng đã đổi/hạ gói sau kỳ.
- **FR-008**: Báo cáo MUST xử lý nhất quán kỳ một phía: thiếu ngày kết thúc nghĩa là kéo đến hiện tại
  và MUST NOT tính khoảng bắt đầu trong tương lai; thiếu ngày bắt đầu nghĩa là tính từ đầu lịch sử có
  dữ liệu. Khi **không truyền cả hai**, báo cáo MUST trả về ảnh chụp trạng thái tại thời điểm gọi (gói
  đang có hiệu lực), giữ nguyên hành vi hiện nay.
- **FR-009**: Báo cáo MUST bao gồm người dùng có gói đã hết hạn trong kỳ nhưng chưa được hạ gói, tính
  đến đúng thời điểm hết hạn.
- **FR-010**: Mỗi người dùng MUST được đếm tối đa một lần cho mỗi gói trong một kỳ, kể cả khi họ dùng
  cùng gói ở nhiều khoảng rời nhau.
- **FR-011**: Tổng số người dùng MUST là tổng của các nhóm trả về (một người đổi gói trong kỳ được
  tính ở nhiều nhóm), và tỷ lệ phần trăm MUST được tính trên tổng đó để luôn cộng đúng 100% khi có ít
  nhất một người dùng. Tài liệu MUST nêu rõ tổng này không phải số người duy nhất.
- **FR-012**: Báo cáo MUST kèm mốc thời gian "lịch sử đầy đủ bắt đầu từ" và MUST nêu rõ mức độ đầy đủ
  của kỳ: kỳ nằm hoàn toàn trước mốc MUST được báo là "không đủ dữ liệu" (danh sách rỗng và tổng bằng
  0, nhưng client MUST NOT hiển thị như một biểu đồ 0% thật); kỳ cắt ngang mốc MUST chỉ tính cho phần
  sau mốc và báo là "một phần". Giá trị ngày trống (client không truyền) MUST được hiểu là vô cực, không
  phải mốc 0. Mốc và trạng thái này được gửi qua các trường tuỳ chọn để không phá vỡ client cũ.
- **FR-013**: Hệ thống MUST khởi tạo lịch sử cho trạng thái gói hiện tại của tất cả người dùng tại thời
  điểm triển khai; MUST NOT cố tái dựng lịch sử trước đó.
- **FR-014**: Lịch sử gói MUST được giữ lại theo vòng đời tài khoản người dùng (không tự động xoá theo
  thời gian), để báo cáo của các kỳ cũ vẫn trả lời được.
- **FR-015**: Hình dạng phản hồi hiện tại của báo cáo MUST được giữ nguyên; mọi bổ sung thông tin MUST
  là trường tuỳ chọn và không làm hỏng client hiện có.
- **FR-016**: Tài liệu dành cho client (hướng dẫn tích hợp và tài liệu API) MUST được cập nhật để mô tả
  ngữ nghĩa mới và mốc lịch sử, thay cho cảnh báo "không thể tái dựng lịch sử" hiện tại.

### Key Entities

- **Khoảng hiệu lực gói**: Một giai đoạn liên tục mà người dùng giữ một gói. Thuộc tính: người dùng,
  gói, tên gói và hạng gói tại thời điểm đó, mốc bắt đầu, mốc kết thúc (có thể trống), nguồn ghi nhận
  (để chống trùng), thời điểm tạo. Quan hệ: nhiều khoảng thuộc một người dùng; mỗi khoảng tham chiếu
  một gói.
- **Mốc lịch sử đầy đủ**: Thời điểm hệ thống bắt đầu ghi lịch sử một cách đầy đủ; dùng để giải thích
  giới hạn của các kỳ trước đó.
- **Báo cáo phân bố**: Tập hợp các nhóm (gói, số người dùng, tỷ lệ phần trăm) cho một kỳ, kèm tổng số
  người dùng.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% kịch bản kiểm thử chuẩn (hạ gói sau kỳ, đổi gói giữa kỳ, gia hạn, gói trọn đời, gói
  miễn phí, kỳ một phía) cho số liệu khớp chính xác với bảng tính đối chiếu.
- **SC-002**: 0 trường hợp người dùng đang giữ gói hiệu lực trong kỳ bị thiếu khỏi cả nhóm gói và tổng
  số, trên toàn bộ bộ kiểm thử hồi quy.
- **SC-003**: Với kỳ nằm trước hoặc cắt ngang mốc lịch sử, 100% phản hồi nêu rõ giới hạn dữ liệu
  ("không đủ dữ liệu" hoặc "một phần") thay vì trả về số 0 hoặc số đầy đủ gây hiểu sai.
- **SC-004**: Với 200.000 khoảng hiệu lực, 95% yêu cầu báo cáo trả về trong dưới 1 giây.
- **SC-005**: 100% thao tác đổi gói thất bại không để lại bản ghi lịch sử mồ côi; 100% thao tác retry
  không tạo khoảng trùng.
- **SC-006**: Không có thay đổi phá vỡ client: phản hồi cũ vẫn đọc được với cùng tên trường và kiểu
  dữ liệu.

## Assumptions

- **Ngữ nghĩa đếm giữ nguyên**: người dùng được tính cho một gói nếu họ giữ gói đó tại bất kỳ thời điểm
  nào trong kỳ (giao cửa sổ), đúng như hành vi đã công bố của báo cáo hiện tại. Yêu cầu "tại thời điểm
  cuối kỳ" không nằm trong phạm vi lần này; nếu cần, sẽ là thay đổi riêng.
- **Bổ sung tối thiểu vào phản hồi**: thêm hai trường tuỳ chọn — mốc lịch sử (`dataAvailableFrom`, ngày
  VN `YYYY-MM-DD`) và trạng thái đầy đủ (`dataCompleteness`) — để báo cáo nêu rõ giới hạn dữ liệu theo
  FR-012; không đổi tên, kiểu hay ý nghĩa của bất kỳ trường cũ nào. Client hiện tại bỏ qua hai trường này
  mà không bị ảnh hưởng.
- **Khởi tạo chỉ từ trạng thái hiện tại**: lịch sử trước ngày triển khai được chấp nhận là không có;
  không tái dựng từ sự kiện hay giao dịch vì không đủ độ chính xác.
- **Không thay đổi ý nghĩa "gói hiện tại"**: các màn hình và endpoint khác vẫn đọc trạng thái gói hiện
  tại như trước; feature này chỉ thêm nguồn dữ liệu lịch sử và đổi cách tính báo cáo phân bố.
- **Quy mô dữ liệu**: một người dùng có trung bình dưới 5 lần đổi gói mỗi năm; tổng số khoảng tăng
  tuyến tính theo số lần đổi gói, không theo số ngày.
- **Vận hành**: triển khai cần một lần khởi tạo dữ liệu cho trạng thái hiện tại, chạy trong migration.
  Bản sao đầu tiên chạy migration có thể mất thêm thời gian khởi động; các bản sao khác chờ khoá tư vấn
  của Goose và không phục vụ request cho tới khi migration xong. Không cần downtime có kế hoạch, nhưng
  độ trễ khởi động này là chấp nhận được và phải được đo trước khi triển khai.
- **Tra cứu lịch sử (US2)**: công cụ vận hành hiện có là truy cập SQL chỉ đọc qua `psql`; câu truy vấn
  chuẩn được tài liệu hoá trong `docs/development/technical-notes.md`. Không cần CLI hay endpoint mới.
- **Phụ thuộc**: dùng chung cơ sở dữ liệu hiện tại của dịch vụ tài khoản; không cần hệ thống lưu trữ
  hay hàng đợi mới.

## Out of Scope

- Tái dựng lịch sử gói trước ngày triển khai (kể cả best-effort từ sự kiện hoặc giao dịch).
- Chụp ảnh trạng thái định kỳ theo ngày/tháng thay cho khoảng hiệu lực.
- Thay đổi giao diện quản trị; chỉ cập nhật tài liệu và hợp đồng phía máy chủ.
- Thay đổi các endpoint báo cáo khác của feature 029.
- Mở endpoint mới để tra cứu lịch sử gói của một người dùng (dùng SQL chỉ đọc trên psql đã tài liệu hoá).
- Ngữ nghĩa "phân bố tại thời điểm cuối kỳ".
