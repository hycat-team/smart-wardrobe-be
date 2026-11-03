# Feature Specification: Cá nhân hoá phối đồ và gợi ý bổ sung

**Feature Branch**: `033-personalized-styling`

**Created**: 2026-10-27

**Status**: Draft

**Input**: User description: "033" (phạm vi được chốt trong phiên thảo luận ngày 2026-10-27)

## Bối cảnh

Người dùng gói trả phí hiện nhận được cùng một chất lượng gợi ý như người dùng miễn phí. Hệ thống chỉ
hiểu phong cách ở mức **một yêu cầu đơn lẻ** (phong cách nhập trong từng lần gọi), không ghi nhớ gu
của người dùng giữa các phiên. Mọi tín hiệu học hỏi từ hành vi người dùng đều bị bỏ phí.

Ba khoảng trống cụ thể:

1. **Không có hồ sơ gu bền vững.** Người dùng không có nơi khai báo phong cách yêu thích, phong cách
   muốn tránh, cũng không có cách nào để hệ thống dùng gu đó khi phối đồ.
2. **Không có cơ chế học từ hành vi.** Việc người dùng lưu một bộ đồ là tín hiệu mạnh nhất về gu của
   họ, nhưng hiện tại tín hiệu này chỉ dùng để cập nhật "lần dùng gần nhất" của món đồ, không đóng góp
   gì vào gu của người dùng.
3. **Tủ đồ hệ thống chưa được khai thác.** Hệ thống đã có bộ sưu tập đồ do quản trị viên tạo, nhưng
   chỉ dùng được khi người dùng chủ động khởi tạo tủ đồ từ bộ sưu tập. Gợi ý phối đồ không bao giờ đề
   xuất món còn thiếu, và thao tác lưu outfit không chấp nhận món ngoài tủ đồ của người dùng.

Ngoài ra, cơ chế **khuyến khích mặc lại đồ lâu không dùng** đã được thiết kế nhưng không hiệu quả
trong thực tế: món đồ vẫn được gợi ý lặp lại, còn món lâu không mặc không có cơ hội quay lại. Nguyên
nhân đã được phân tích và ghi nhận trong phần Clarifications.

Feature này bổ sung lớp cá nhân hoá cho người dùng trả phí, biến hành vi lưu outfit thành tín hiệu
học gu, khai thác tủ đồ hệ thống để gợi ý món còn thiếu, và sửa cơ chế xoay vòng đồ lâu không dùng.

## Clarifications

### Session 2026-10-27

- Q: Hồ sơ gu nên thuộc service nào? → A: Thuộc miền thời trang (commerce), vì chỉ phục vụ pipeline
  phối đồ và dùng chung cấu hình biểu diễn ngữ nghĩa với món đồ; không dùng lại bảng hồ sơ phong cách
  đã bị xoá ở tài khoản (ADR 015).
- Q: Gói trả phí được xác định bằng gì? → A: Một cờ quyền lợi `personalizationEnabled` trên gói cước,
  mặc định tắt. Số phong cách tối đa là cấu hình của hệ thống (mặc định 4), không phải thuộc tính
  của từng gói.
- Q: Gợi ý món ngoài tủ dành cho ai? → A: Tất cả người dùng. Chỉ phần gu phong cách cá nhân là quyền
  lợi trả phí.
- Q: Khi lưu outfit có món hệ thống và việc sao chép làm vượt hạn mức tủ đồ thì xử lý thế nào? → A:
  Từ chối toàn bộ lần lưu và nêu rõ món nào gây vượt hạn mức; không sao chép một phần.
- Q: Hệ thống học gu từ outfit đã lưu như thế nào? → A: Ghi tín hiệu và cập nhật gu do hệ thống suy ra
  một cách bất đồng bộ qua sự kiện. Gu do người dùng tự khai không bao giờ bị ghi đè.
- Q: Sửa outfit có tính là một lần học gu không? → A: Chỉ tính khi tập món đồ thay đổi, và chỉ các món
  mới được thêm mới tạo tín hiệu.
- Q: Làm sao xác định "đã mặc"? → A: Trong phạm vi feature này, lưu outfit được dùng làm tín hiệu đại
  diện cho việc sử dụng. Không mở thêm thao tác "đánh dấu đã mặc".
- Q: Cơ chế xoay vòng đồ lâu không dùng cần sửa ở mức nào? → A: Sửa cả ba tầng: cửa sổ phạt và tín hiệu
  đúng, dành chỗ trong tập ứng viên cho đồ lâu không dùng, và ưu tiên theo vai trò khi món đầu tiên
  đang trong vùng làm mát. Đồng thời ghi nhớ cả lần gợi ý gần nhất, không chỉ lần sử dụng.
- Q: Kết quả gợi ý món ngoài tủ nằm ở đâu trong phản hồi? → A: Một mảng riêng, không trộn vào danh
  sách món của bộ đồ và không trộn vào danh sách thay thế.
- Q: Feature mang số 033 và `.specify/feature.json` là file cục bộ, không được version hoá. → A: Giữ
  đúng số 033; bổ sung file cấu hình cục bộ vào danh sách bỏ qua của kho mã và ngừng theo dõi file này.
- Q: Khi nào hệ thống nên trả gợi ý bổ sung cho một vai trò? → A: Khi vai trò thiếu hoàn toàn, hoặc
  khi vai trò chỉ có ứng viên với chất lượng dưới ngưỡng cấu hình của hệ thống.
- Q: Một phản hồi nên có bao nhiêu gợi ý bổ sung? → A: Tối đa một món mỗi vai trò và tối đa ba món
  mỗi phản hồi.
- Q: Vùng làm mát nên dài bao lâu? → A: Ba ngày cho món vừa được gợi ý; bảy ngày cho món vừa được
  sử dụng.
- Q: Làm sao đo hiệu quả xoay vòng? → A: Ghi log có cấu trúc và bộ đếm nội bộ cho mỗi lần gợi ý, đo
  SC-003/SC-004 bằng truy vấn vận hành; không cần dashboard mới.
- Q: Gợi ý bổ sung phải khớp ngữ cảnh và gu cá nhân ở mức nào? → A: Khớp ngữ cảnh của yêu cầu và gu
  cá nhân đang bật (nếu có), với gu là ưu tiên mềm giống bộ đồ chính.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Người dùng trả phí khai báo gu và nhận gợi ý đúng gu (Priority: P1)

Là người dùng gói trả phí, tôi muốn chọn những phong cách tôi thích và muốn tránh, để mỗi lần hệ thống
phối đồ, bộ đồ được đề xuất phản ánh gu của tôi thay vì chỉ dựa vào vài từ khóa tôi gõ trong lần đó.

**Why this priority**: Đây là lý do tồn tại của feature và là quyền lợi chính của gói trả phí. Không có
phần này, các phần còn lại chỉ là cải tiến chất lượng chung.

**Independent Test**: Thiết lập hồ sơ gu gồm hai phong cách yêu thích và một phong cách muốn tránh,
gọi gợi ý phối đồ hai lần (một lần không nhập phong cách, một lần nhập phong cách mâu thuẫn với hồ sơ),
rồi kiểm tra bộ đồ và phần giải thích phản ánh đúng thứ tự ưu tiên. Test độc lập vì chỉ cần một tài
khoản trả phí có tủ đồ tối thiểu.

**Acceptance Scenarios**:

1. **Given** người dùng trả phí chưa có hồ sơ gu, **When** họ lưu danh sách phong cách yêu thích và danh sách phong cách muốn tránh, **Then** hệ thống lưu thành công và trả lại đúng dữ liệu đã lưu.
2. **Given** người dùng đã có hồ sơ gu, **When** họ gọi gợi ý phối đồ mà không nhập phong cách, **Then** các món phù hợp gu được ưu tiên và phần giải thích đề cập đến gu đó.
3. **Given** người dùng đã có hồ sơ gu, **When** họ gọi gợi ý với một phong cách khác trong yêu cầu, **Then** phong cách trong yêu cầu được ưu tiên hơn hồ sơ, và bộ đồ vẫn hợp lệ.
4. **Given** người dùng gói miễn phí, **When** họ cố lưu hồ sơ gu, **Then** hệ thống từ chối với thông báo nêu rõ đây là quyền lợi của gói trả phí.
5. **Given** người dùng trả phí đã tắt cá nhân hoá, **When** họ gọi gợi ý, **Then** hệ thống không áp dụng gu cá nhân nhưng vẫn giữ nguyên hành vi gợi ý hiện có.

---

### User Story 2 - Gợi ý món còn thiếu từ tủ đồ hệ thống và tự đưa vào tủ khi lưu (Priority: P1)

Là người dùng, tôi muốn được giới thiệu những món đồ còn thiếu để hoàn thiện bộ đồ, và khi tôi đồng ý
lưu bộ đồ đó, hệ thống tự đưa món mới vào tủ đồ của tôi để tôi thực sự sở hữu nó.

**Why this priority**: Đây là giá trị mới rõ ràng nhất ngoài phần gu: biến gợi ý thành hành động mua
hoặc bổ sung tủ đồ, đồng thời bảo đảm quyền sở hữu dữ liệu nhất quán.

**Independent Test**: Dựng tủ đồ thiếu một vai trò (ví dụ không có giày), gọi gợi ý phối đồ và kiểm tra
mảng gợi ý bổ sung; sau đó lưu bộ đồ có món hệ thống và kiểm tra tủ đồ của người dùng có bản sao mới,
bộ đồ tham chiếu bản sao đó, và không có tình trạng món hệ thống bị coi là tài sản của người dùng.

**Acceptance Scenarios**:

1. **Given** tủ đồ của người dùng thiếu một vai trò cần thiết, **When** họ gọi gợi ý phối đồ, **Then** phản hồi có mảng gợi ý bổ sung cho vai trò còn thiếu, tách biệt khỏi bộ đồ chính.
2. **Given** người dùng đã sở hữu một món trùng với món hệ thống, **When** hệ thống tìm gợi ý bổ sung, **Then** món đó không xuất hiện trong gợi ý.
3. **Given** một gợi ý bổ sung hợp lệ, **When** người dùng lưu bộ đồ có món đó, **Then** hệ thống tạo bản sao thuộc người dùng, lưu bộ đồ tham chiếu bản sao, và không chạy lại phân tích ảnh.
4. **Given** việc tạo bản sao sẽ vượt hạn mức tủ đồ, **When** người dùng lưu bộ đồ, **Then** toàn bộ lần lưu bị từ chối, không có bản sao nào được tạo, và thông báo nêu rõ món gây vượt hạn mức.
5. **Given** món hệ thống đã bị quản trị viên gỡ bỏ giữa lúc gợi ý và lúc lưu, **When** người dùng lưu, **Then** hệ thống từ chối món đó với thông báo phù hợp và không tạo dữ liệu dở dang.
6. **Given** người dùng đang dùng gói miễn phí, **When** họ gọi gợi ý phối đồ, **Then** họ vẫn nhận được gợi ý bổ sung.

---

### User Story 3 - Xoay vòng để đồ lâu không dùng có cơ hội quay lại (Priority: P2)

Là người dùng, tôi muốn được gợi ý cả những món tôi đã lâu không mặc, thay vì ngày nào cũng thấy
những món vừa mặc hoặc vừa được gợi ý.

**Why this priority**: Cơ chế hiện tại đã được thiết kế nhưng không tạo ra thay đổi quan sát được, nên
người dùng vẫn thấy lặp lại. Đây là chất lượng cốt lõi của gợi ý, độc lập với phần gu.

**Independent Test**: Dựng một tủ đồ có cả món vừa lưu vào outfit gần đây và món lâu không dùng nhưng
cùng phù hợp ngữ cảnh; gọi gợi ý nhiều lần và kiểm tra món lâu không dùng xuất hiện trong bộ đồ hoặc
trong danh sách thay thế, đồng thời món vừa dùng không chiếm hết các vai trò.

**Acceptance Scenarios**:

1. **Given** một món vừa được lưu vào outfit trong vài ngày gần đây, **When** người dùng gọi gợi ý, **Then** món đó bị giảm ưu tiên so với món tương đương về độ phù hợp nhưng lâu không dùng.
2. **Given** một món chưa từng được dùng và đã có trong tủ từ lâu, **When** người dùng gọi gợi ý, **Then** món đó nhận ưu tiên của nhóm lâu không dùng, không bị coi là món mới.
3. **Given** một món vừa được hệ thống gợi ý nhưng người dùng không lưu, **When** họ gọi lại gợi ý ngay sau đó, **Then** món đó bị giảm ưu tiên trong vài ngày.
4. **Given** mọi món phù hợp cho một vai trò đều đang trong vùng làm mát, **When** người dùng gọi gợi ý, **Then** hệ thống vẫn trả về bộ đồ hoàn chỉnh thay vì bỏ trống vai trò.
5. **Given** người dùng chỉ đổi tên hoặc ảnh bìa của outfit, **When** họ lưu thay đổi, **Then** các món trong outfit không bị coi là vừa được sử dụng lại.
6. **Given** người dùng có nhiều bản sao của cùng một món, **When** hệ thống chấm điểm, **Then** tất cả bản sao dùng chung tín hiệu sử dụng gần nhất.

---

### User Story 4 - Hệ thống tự học gu từ những bộ đồ người dùng đã lưu (Priority: P2)

Là người dùng, tôi muốn hệ thống tự hiểu gu của tôi qua những bộ đồ tôi chọn lưu, để gợi ý ngày càng
sát ý tôi hơn mà tôi không phải nhập lại.

**Why this priority**: Biến hành vi sẵn có thành tín hiệu học là cách tăng chất lượng bền vững, nhưng
giá trị chỉ xuất hiện sau khi người dùng đã có vài outfit.

**Independent Test**: Lưu vài outfit có phong cách nhất quán, kiểm tra gu do hệ thống suy ra thay đổi
theo hướng đó và ảnh hưởng đến lần gợi ý kế tiếp, đồng thời gu người dùng tự khai vẫn nguyên vẹn.

**Acceptance Scenarios**:

1. **Given** người dùng lưu một outfit mới thành công, **When** hệ thống xử lý tín hiệu, **Then** gu do hệ thống suy ra được cập nhật trong vòng một phút và lần gợi ý sau phản ánh thay đổi.
2. **Given** người dùng đã khai gu thủ công, **When** hệ thống học từ outfit, **Then** gu đã khai không bị sửa, xoá hay ghi đè.
3. **Given** người dùng sửa outfit bằng cách thêm một món mới, **When** thay đổi được lưu, **Then** chỉ món mới tạo tín hiệu học.
4. **Given** cùng một sự kiện lưu outfit được xử lý hai lần, **When** hệ thống học, **Then** gu chỉ được cập nhật một lần.
5. **Given** người dùng đang dùng gói miễn phí, **When** họ lưu outfit, **Then** tín hiệu vẫn được ghi nhận nhưng không ảnh hưởng gợi ý cho tới khi họ có quyền lợi cá nhân hoá.

---

### User Story 5 - Người dùng mới được gợi ý gu từ chính tủ đồ của mình (Priority: P2)

Là người dùng vừa nâng cấp gói, tôi không biết nên khai báo phong cách nào, nên tôi muốn hệ thống gợi
ý những phong cách đang chiếm ưu thế trong tủ đồ để tôi chỉ cần xác nhận.

**Why this priority**: Giảm rào cản lớn nhất của tính năng gu là người dùng không biết nên nhập gì.
Chi phí thấp vì dữ liệu đã có sẵn trong tủ đồ.

**Independent Test**: Dựng tủ đồ có phân bố phong cách rõ ràng, gọi gợi ý gu và kiểm tra danh sách trả
về phản ánh đúng phân bố cùng số mẫu; xác nhận gợi ý không tự động áp dụng.

**Acceptance Scenarios**:

1. **Given** người dùng có quyền cá nhân hoá và tủ đồ có ít nhất một phong cách chiếm ưu thế, **When** họ yêu cầu gợi ý gu, **Then** hệ thống trả về danh sách phong cách kèm số lượng và tỷ lệ món tương ứng.
2. **Given** người dùng nhận gợi ý gu, **When** họ không xác nhận, **Then** hồ sơ gu không thay đổi.
3. **Given** tủ đồ chưa đủ dữ liệu để suy ra phong cách, **When** người dùng yêu cầu gợi ý gu, **Then** hệ thống trả về danh sách rỗng kèm thông báo chưa đủ dữ liệu.

---

### User Story 6 - Người dùng phản hồi để tinh chỉnh gu (Priority: P3)

Là người dùng, tôi muốn nói "thích" hoặc "không thích" với một bộ đồ được gợi ý, để hệ thống điều
chỉnh dần thay vì chỉ dựa vào việc tôi có lưu hay không.

**Why this priority**: Tăng tốc việc học, nhưng hệ thống đã có tín hiệu lưu outfit; phản hồi là lớp bổ
sung nên ưu tiên thấp hơn.

**Independent Test**: Gửi phản hồi thích và không thích cho hai bộ đồ khác nhau, kiểm tra gu suy ra
thay đổi theo hướng tương ứng và phản hồi trùng lặp không tạo tín hiệu thứ hai.

**Acceptance Scenarios**:

1. **Given** người dùng nhận một bộ đồ, **When** họ gửi phản hồi thích, **Then** hệ thống ghi nhận và lần gợi ý sau ưu tiên phong cách của bộ đồ đó hơn.
2. **Given** người dùng gửi phản hồi không thích, **When** hệ thống học, **Then** phong cách của bộ đồ đó bị giảm ưu tiên nhưng không bị loại vĩnh viễn.
3. **Given** người dùng gửi lại cùng một phản hồi cho cùng một bộ đồ, **When** hệ thống xử lý, **Then** không có tín hiệu trùng được tạo.
4. **Given** người dùng cố phản hồi cho bộ đồ không phải của mình, **When** hệ thống xử lý, **Then** yêu cầu bị từ chối.

---

### User Story 7 - Hoàn thiện một bộ đồ đã có (Priority: P3)

Là người dùng, tôi muốn chọn một vài món tôi đang có và hỏi hệ thống còn thiếu gì, để bổ sung mà
không phải chạy lại toàn bộ quy trình gợi ý.

**Why this priority**: Tái sử dụng đúng nguồn gợi ý bổ sung của US2, rẻ về chi phí và hữu ích cho
người dùng đã có ý tưởng phối đồ.

**Independent Test**: Gửi một tập món thuộc người dùng, kiểm tra hệ thống chỉ trả gợi ý cho vai trò
còn thiếu; gửi một món không thuộc người dùng và kiểm tra bị từ chối.

**Acceptance Scenarios**:

1. **Given** người dùng gửi một tập món đã sở hữu và còn thiếu một vai trò, **When** họ yêu cầu hoàn thiện, **Then** hệ thống trả về gợi ý cho đúng vai trò còn thiếu.
2. **Given** tập món đã đủ cấu trúc bộ đồ, **When** người dùng yêu cầu hoàn thiện, **Then** hệ thống trả về danh sách gợi ý rỗng thay vì báo lỗi.
3. **Given** tập món có món không thuộc người dùng, **When** người dùng yêu cầu hoàn thiện, **Then** yêu cầu bị từ chối.
4. **Given** người dùng gọi yêu cầu hoàn thiện, **When** hệ thống xử lý, **Then** không tiêu tốn lượt gợi ý AI trong ngày và không thay đổi outfit nào.

---

### Edge Cases

- **Người dùng để danh sách gu yêu thích rỗng**: được hiểu là "không có gu ưu tiên", không phải lỗi;
  danh sách gu muốn tránh vẫn được áp dụng.
- **Gu chứa giá trị lạ ngoài từ vựng phong cách**: hệ thống vẫn lưu và vẫn dùng được ở mức so khớp
  ngữ nghĩa; không từ chối dữ liệu do người dùng nhập.
- **Danh sách gu vượt số lượng tối đa**: hệ thống từ chối và nêu rõ giới hạn.
- **Bước tạo biểu diễn ngữ nghĩa cho gu thất bại**: dữ liệu gu vẫn được lưu; gợi ý chuyển sang mức so
  khớp từ khóa và không báo lỗi cho người dùng.
- **Gói trả phí hết hạn khi hồ sơ gu đang tồn tại**: dữ liệu được giữ, người dùng vẫn xem được nhưng
  không sửa được; gợi ý ngừng áp dụng gu cho tới khi gia hạn.
- **Người dùng nâng cấp gói sau khi đã có nhiều outfit**: gu học từ lịch sử được áp dụng ngay, không
  cần chờ outfit mới.
- **Món đồ chưa có biểu diễn ngữ nghĩa (đang phân tích)**: không tham gia học gu và không chặn quá
  trình cập nhật gu.
- **Outfit bị xoá sau khi đã học**: gu đã học không được hoàn tác trong phạm vi feature này.
- **Món hệ thống bị gỡ hoặc ngừng hoạt động giữa lúc gợi ý và lúc lưu**: lần lưu bị từ chối, không tạo
  dữ liệu dở dang.
- **Hai yêu cầu lưu outfit đồng thời cùng sao chép một món**: không tạo ra bộ đồ tham chiếu món không
  thuộc người dùng; bản sao trùng nếu có không làm hỏng bộ đồ.
- **Tủ đồ có nhiều bản sao của cùng một món**: tín hiệu sử dụng được hợp nhất theo món gốc trước khi
  chấm điểm.
- **Người dùng có ít hơn số món tối thiểu để phối đồ**: hành vi hiện có được giữ nguyên và thông báo
  vẫn rõ ràng.
- **Vai trò có món nhưng mọi ứng viên đều dưới ngưỡng chất lượng**: vai trò đó vẫn được coi là chưa
  phù hợp và nhận gợi ý bổ sung.
- **Mọi ứng viên cho một vai trò đều đang trong vùng làm mát**: vẫn trả bộ đồ hợp lệ, ưu tiên chất
  lượng hơn xoay vòng.
- **Người dùng gửi phản hồi cho bộ đồ đã bị xoá**: phản hồi bị từ chối nhưng không ảnh hưởng gu.
- **Sự kiện học gu bị xử lý lặp hoặc đến sai thứ tự**: kết quả cuối cùng nhất quán, không cộng dồn
  hai lần.
- **Người dùng gói miễn phí gọi các thao tác chỉ dành cho trả phí**: bị từ chối với thông báo nêu rõ
  quyền lợi, không lộ dữ liệu của người khác.
- **`.specify/feature.json` vẫn đang được theo dõi sau khi thêm vào danh sách bỏ qua**: coi là chưa
  hoàn thành yêu cầu bảo trì.

## Requirements *(mandatory)*

### Functional Requirements

#### A. Quyền lợi gói và giới hạn

- **FR-001**: Gói cước MUST có một cờ quyền lợi cá nhân hoá, mặc định tắt cho gói miễn phí.
- **FR-002**: Hệ thống MUST cho phép người dùng có quyền lợi tạo, xem, sửa và xoá hồ sơ gu của mình.
- **FR-003**: Hệ thống MUST từ chối thao tác ghi hồ sơ gu từ người dùng không có quyền lợi, với thông
  báo nêu rõ đây là quyền lợi của gói trả phí.
- **FR-004**: Người dùng không còn quyền lợi MUST vẫn xem được hồ sơ gu hiện có nhưng MUST NOT sửa
  được, và gu MUST NOT được áp dụng vào gợi ý. Do overview được cache tối đa 60 giây, hệ thống MUST
  ngừng áp dụng/ghi trong vòng tối đa 60 giây kể từ khi quyền lợi đổi; sai lệch trong cửa sổ này là
  dung sai đã biết và MUST được ghi trong tài liệu vận hành.
- **FR-005**: Số phong cách tối đa cho mỗi danh sách (yêu thích và muốn tránh) MUST là cấu hình của
  hệ thống (mặc định 4), độc lập với hạng gói.

#### B. Hồ sơ gu phong cách

- **FR-006**: Hồ sơ gu MUST gồm danh sách phong cách yêu thích, danh sách phong cách muốn tránh và
  trạng thái bật/tắt cá nhân hoá.
- **FR-007**: Hệ thống MUST chấp nhận phong cách là chuỗi tự do do giao diện gửi lên; MUST NOT bắt
  buộc giá trị thuộc một danh sách cố định.
- **FR-008**: Hệ thống MUST chuẩn hoá dữ liệu nhập bằng cách cắt khoảng trắng, loại trùng không phân
  biệt hoa thường và giới hạn độ dài từng giá trị; MUST NOT lưu giá trị rỗng.
- **FR-009**: Danh sách phong cách yêu thích rỗng MUST được hiểu là "không có gu ưu tiên", MUST NOT
  bị coi là lỗi.
- **FR-010**: Mỗi lần lưu hồ sơ thành công, hệ thống MUST tạo một biểu diễn ngữ nghĩa của tập phong
  cách để dùng cho việc so khớp với món đồ.
- **FR-011**: Nếu tạo biểu diễn ngữ nghĩa thất bại, hệ thống MUST vẫn lưu hồ sơ gu và MUST chuyển sang
  so khớp ở mức từ khóa cho tới khi biểu diễn được tạo lại thành công.
- **FR-012**: Người dùng MUST có thể tắt cá nhân hoá mà không xoá dữ liệu gu đã khai.
- **FR-013**: Gu do người dùng khai MUST NOT bị hệ thống tự động sửa, xoá hoặc ghi đè trong bất kỳ
  luồng học nào.

#### C. Áp dụng gu vào gợi ý phối đồ

- **FR-014**: Khi người dùng có quyền lợi và hồ sơ gu đang bật, gợi ý phối đồ MUST kết hợp gu cá nhân
  với ngữ cảnh của yêu cầu hiện tại.
- **FR-015**: Gu cá nhân MUST là ưu tiên mềm: được phép thay đổi thứ hạng và lựa chọn, nhưng MUST NOT
  loại bỏ món đồ cần thiết để tạo thành một bộ đồ hợp lệ.
- **FR-016**: Phong cách do người dùng nhập trong từng yêu cầu MUST được ưu tiên hơn gu lưu sẵn khi
  hai bên mâu thuẫn.
- **FR-017**: Phong cách muốn tránh MUST làm giảm thứ hạng của món đồ tương ứng và MUST NOT được trình
  bày như một điểm phù hợp.
- **FR-018**: Gu cá nhân MUST được truyền vào bước tạo phần giải thích để lý do gợi ý phản ánh đúng gu.
- **FR-019**: Khi quá trình tạo bộ đồ bằng AI thất bại và hệ thống dùng phương án dự phòng, phương án
  dự phòng MUST áp dụng cùng mức ưu tiên mềm cho gu cá nhân.

#### D. Gợi ý bổ sung ngoài tủ đồ

- **FR-020**: Sau khi tạo bộ đồ, hệ thống MUST xác định (a) các vai trò không có món nào trong tủ, và
  (b) các vai trò chỉ có ứng viên với chất lượng dưới ngưỡng cấu hình của hệ thống. Cả hai nhóm MUST
  được coi là "chưa phù hợp" và đủ điều kiện nhận gợi ý bổ sung.
- **FR-021**: Với các vai trò đó, hệ thống MUST tìm món trong tủ đồ hệ thống, loại trừ mọi món người
  dùng đã sở hữu. Kết quả MUST ưu tiên món khớp ngữ cảnh của yêu cầu (dịp, mùa, thời tiết, tông màu)
  và gu cá nhân đang bật (nếu có); gu MUST là ưu tiên mềm giống bộ đồ chính.
- **FR-022**: Kết quả gợi ý bổ sung MUST nằm trong một mảng riêng, tách khỏi danh sách món của bộ đồ
  và khỏi danh sách món thay thế.
- **FR-023**: Mỗi gợi ý bổ sung MUST có vai trò, thông tin món đồ và lý do gợi ý bằng tiếng Việt.
  Mỗi vai trò MUST có tối đa một gợi ý, và một phản hồi MUST có tối đa ba gợi ý bổ sung.
- **FR-024**: Gợi ý bổ sung MUST NOT được tính là một phần của bộ đồ, MUST NOT tiêu tốn lượt gợi ý AI
  và MUST NOT được trình bày như món người dùng đã sở hữu.
- **FR-025**: Nếu không có vai trò nào thiếu hoặc không có món phù hợp, hệ thống MUST trả về danh sách
  rỗng thay vì báo lỗi.
- **FR-026**: Gợi ý bổ sung MUST khả dụng cho mọi người dùng, không phụ thuộc gói cước.

#### E. Lưu outfit có món từ tủ đồ hệ thống

- **FR-027**: Thao tác lưu và sửa outfit MUST chấp nhận món đồ từ tủ đồ hệ thống, có thể được client
  chỉ định rõ nguồn gốc.
- **FR-028**: Khi client không chỉ định nguồn gốc, hệ thống MUST phân giải theo thứ tự: tủ đồ của
  người dùng, món của nhãn hàng đủ điều kiện, rồi tủ đồ hệ thống.
- **FR-029**: Khi lưu một món từ tủ đồ hệ thống, hệ thống MUST tạo một bản sao thuộc sở hữu người dùng
  và bộ đồ MUST tham chiếu bản sao đó.
- **FR-030**: Bản sao MUST dùng lại thông tin và hình ảnh sẵn có của món gốc, MUST NOT kích hoạt lại
  phân tích ảnh.
- **FR-031**: Việc tạo bản sao và tạo/cập nhật outfit MUST diễn ra nguyên tử; bất kỳ lỗi nào MUST
  hoàn tác toàn bộ, không để bản sao mồ côi hay bộ đồ thiếu món.
- **FR-032**: Nếu việc tạo bản sao làm vượt hạn mức tủ đồ, hệ thống MUST từ chối toàn bộ lần lưu và
  nêu rõ các món gây vượt hạn mức.
- **FR-033**: Một bộ đồ MUST NOT chứa hai mục trùng nhau cho cùng một món.
- **FR-034**: Thao tác sửa outfit MUST áp dụng đầy đủ các quy tắc của thao tác lưu, bao gồm kiểm tra
  sở hữu và giới hạn tủ đồ.
- **FR-035**: Đường dẫn khởi tạo tủ đồ từ bộ sưu tập hiện có MUST chỉ chấp nhận món thực sự thuộc tủ
  đồ hệ thống và chưa bị xoá; MUST NOT cho phép sao chép món riêng tư của người dùng khác.

#### F. Học gu từ outfit đã lưu

- **FR-036**: Mỗi lần lưu outfit thành công MUST tạo một tín hiệu học gu.
- **FR-037**: Tín hiệu học MUST được ghi nhận đúng một lần kể cả khi được xử lý lặp do cơ chế giao
  nhận ít nhất một lần.
- **FR-038**: Hệ thống MUST duy trì một gu do hệ thống suy ra, tách biệt hoàn toàn với gu người dùng
  khai.
- **FR-039**: Gu suy ra MUST được cập nhật theo cơ chế suy giảm để các bộ đồ gần đây có ảnh hưởng lớn
  hơn nhưng gu cũ không biến mất đột ngột.
- **FR-040**: Việc học MUST diễn ra bất đồng bộ và MUST NOT làm chậm hoặc làm hỏng thao tác lưu outfit.
- **FR-041**: Khi người dùng sửa outfit, hệ thống MUST chỉ tạo tín hiệu cho các món mới được thêm.
- **FR-042**: Đối với người dùng không có quyền cá nhân hoá, hệ thống MUST vẫn ghi nhận tín hiệu
  nhưng MUST NOT áp dụng gu suy ra vào gợi ý.
- **FR-043**: Gu suy ra MUST có trọng số thấp hơn gu người dùng khai và MUST NOT tạo ra bộ lọc cứng.

#### G. Khởi đầu không có dữ liệu gu

- **FR-044**: Người dùng có quyền cá nhân hoá MUST có thể yêu cầu gợi ý phong cách dựa trên tủ đồ
  hiện có.
- **FR-045**: Kết quả gợi ý phong cách MUST kèm số lượng món và tỷ lệ tương ứng để người dùng đánh
  giá độ tin cậy.
- **FR-046**: Gợi ý phong cách MUST NOT tự động áp dụng; người dùng MUST xác nhận trước khi hồ sơ
  thay đổi.
- **FR-047**: Khi tủ đồ không đủ dữ liệu, hệ thống MUST trả về danh sách rỗng kèm thông báo phù hợp.

#### H. Phản hồi của người dùng

- **FR-048**: Người dùng MUST có thể gửi phản hồi thích, không thích hoặc đã lưu cho một bộ đồ được
  gợi ý.
- **FR-049**: Mỗi bộ đồ được gợi ý MUST có định danh để phản hồi tham chiếu tới.
- **FR-050**: Hệ thống MUST chống trùng phản hồi theo người dùng, lần gợi ý (`recommendationId`) và
  loại phản hồi.
- **FR-051**: Phản hồi MUST ảnh hưởng đến gu suy ra với trọng số có giới hạn và MUST NOT vượt qua gu
  người dùng khai.
- **FR-052**: Hệ thống MUST từ chối phản hồi cho bộ đồ không thuộc người dùng gửi.
- **FR-053**: Phản hồi của người dùng không có quyền cá nhân hoá MUST vẫn được ghi nhận nhưng MUST
  NOT ảnh hưởng đến gợi ý cho tới khi người dùng có quyền lợi.

#### I. Xoay vòng và ưu tiên đồ lâu không dùng

- **FR-054**: Hệ thống MUST ghi nhận hai tín hiệu riêng biệt cho mỗi món: lần cuối được sử dụng và
  lần cuối được hệ thống gợi ý. Việc ghi tín hiệu "đã gợi ý" là best-effort: lỗi ghi MUST được log,
  MUST NOT chặn phản hồi, và lần gợi ý kế tiếp MUST thử cập nhật lại cho các món được trả về.
- **FR-055**: Gợi ý MUST giảm ưu tiên món vừa được sử dụng, với cửa sổ tối thiểu 7 ngày và có thể
  cấu hình.
- **FR-056**: Gợi ý MUST tăng ưu tiên món lâu không được sử dụng, bao gồm món chưa từng được sử dụng,
  bất kể yêu cầu có nêu phong cách hay dịp hay không.
- **FR-057**: Trạng thái "món mới" MUST được xác định theo thời điểm món được tạo, MUST NOT dựa trên
  việc món chưa từng được sử dụng.
- **FR-058**: Hệ thống MUST dành một phần tập ứng viên cho các món lâu không được sử dụng nhưng vẫn
  thoả điều kiện của yêu cầu.
- **FR-059**: "Vùng làm mát" MUST được hiểu là: món được sử dụng trong vòng bảy ngày gần nhất, hoặc
  món được hệ thống gợi ý trong vòng ba ngày gần nhất. Khi một vai trò có cả món đang trong vùng làm
  mát và món không trong vùng làm mát với chất lượng tương đương trong ngưỡng cho phép, hệ thống MUST
  ưu tiên món không trong vùng làm mát.
- **FR-060**: Nếu mọi ứng viên hợp lệ của một vai trò đều đang trong vùng làm mát, hệ thống MUST vẫn
  tạo bộ đồ hoàn chỉnh.
- **FR-061**: Các bản sao của cùng một món MUST chia sẻ tín hiệu sử dụng gần nhất khi chấm điểm.
- **FR-062**: Sửa outfit mà không thêm hoặc bớt món MUST NOT đánh dấu các món là vừa được sử dụng.

#### J. Hoàn thiện bộ đồ có sẵn

- **FR-063**: Người dùng MUST có thể yêu cầu gợi ý bổ sung cho một tập món đã sở hữu.
- **FR-064**: Hệ thống MUST kiểm tra quyền sở hữu của toàn bộ món trong tập đầu vào và MUST từ chối
  nếu có món không thuộc người dùng.
- **FR-065**: Kết quả MUST chỉ gồm gợi ý cho các vai trò còn thiếu, không sinh ra bộ đồ mới.
- **FR-066**: Thao tác hoàn thiện MUST NOT tiêu tốn lượt gợi ý AI và MUST NOT thay đổi bất kỳ outfit
  nào.

#### K. Bảo trì kho mã

- **FR-067**: Kho mã MUST bỏ qua file cấu hình cục bộ `.specify/feature.json` và MUST ngừng theo dõi
  file này.

#### L. Khả năng quan sát

- **FR-068**: Mỗi lần gợi ý phối đồ, hệ thống MUST ghi nhận tối thiểu: số món được dành chỗ cho nhóm
  lâu không dùng, số lần ưu tiên món ngoài vùng làm mát thay cho món trong vùng làm mát, tỷ lệ món đã
  được lưu trong 7 ngày gần nhất xuất hiện trong gợi ý mới, tỷ lệ món đã xuất hiện trong một gợi ý
  trước đó, tỷ lệ yêu cầu có ít nhất một món khớp gu, danh sách mã món nội bộ đã trả về và tập con
  thuộc nhóm lâu không dùng/vùng làm mát, cùng tập món lâu không dùng đủ điều kiện của người dùng
  (`stale_eligible_item_ids`) làm mẫu số. Dữ liệu này MUST đủ để đo SC-002, SC-003 và SC-004 bằng
  truy vấn vận hành, và MUST NOT chứa thông tin nhận dạng cá nhân ngoài mã người dùng nội bộ.

### Key Entities

- **Hồ sơ gu**: Gu phong cách gắn với một người dùng. Thuộc tính: danh sách phong cách yêu thích,
  danh sách phong cách muốn tránh, trạng thái bật/tắt, biểu diễn ngữ nghĩa của gu, gu do hệ thống
  suy ra, thời điểm cập nhật gần nhất.
- **Tín hiệu học gu**: Một sự kiện cho thấy gu của người dùng. Thuộc tính: nguồn (lưu outfit, phản
  hồi thích, phản hồi không thích), người dùng, bộ đồ hoặc bộ đồ được gợi ý liên quan, danh sách món
  tham gia, định danh chống trùng, thời điểm.
- **Món đồ hệ thống**: Món thuộc bộ sưu tập do quản trị viên quản lý, có thể được sao chép thành món
  thuộc sở hữu người dùng.
- **Bản sao món đồ**: Bản ghi sở hữu của người dùng trỏ tới cùng thông tin món gốc, dùng để bộ đồ
  tham chiếu và để tính hạn mức tủ đồ.
- **Gợi ý bổ sung**: Đề xuất một món ngoài tủ cho một vai trò còn thiếu. Thuộc tính: vai trò, món đồ,
  lý do.
- **Tín hiệu sử dụng**: Thời điểm gần nhất một món được đưa vào outfit.
- **Tín hiệu đã gợi ý**: Thời điểm gần nhất một món được hệ thống trả về cho người dùng.
- **Bản ghi quan sát gợi ý**: Thông tin tổng hợp của một lần gợi ý dùng để đo hiệu quả xoay vòng.
  Thuộc tính: mã người dùng nội bộ, số chỗ dành cho đồ lâu dùng, số lần ưu tiên ngoài vùng làm mát,
  tỷ lệ món lặp, thời điểm.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Ít nhất 80% người dùng có quyền cá nhân hoá hoàn tất việc thiết lập gu trong dưới 2 phút
  kể từ lần đầu mở màn hình gu.
- **SC-002**: Với người dùng có hồ sơ gu không rỗng, ít nhất 90% yêu cầu gợi ý trả về bộ đồ có ít
  nhất một món khớp với một phong cách yêu thích, trừ khi tủ đồ không có món nào phù hợp.
- **SC-003**: Sau 30 ngày kể từ khi bật rotation, tỷ lệ món đã được lưu vào outfit trong 7 ngày gần
  nhất xuất hiện trong các gợi ý mới không vượt quá 25%, đo bằng counter `recently_used_repeat_ratio`
  trong log gợi ý.
- **SC-004**: Ít nhất 70% món đã quá 30 ngày không được sử dụng (hoặc chưa từng dùng) xuất hiện trong
  ít nhất một gợi ý mỗi tháng; mẫu số là `stale_eligible_item_ids`, tử số là `stale_item_ids` trong log
  gợi ý (FR-068).
- **SC-005**: Ít nhất 90% gợi ý bổ sung trả về đúng vai trò còn thiếu và 0 gợi ý bị trình bày như món
  người dùng đã sở hữu.
- **SC-006**: 100% lần lưu outfit có món hệ thống mà vượt hạn mức bị từ chối nguyên tử; 0 bản sao mồ
  côi sau khi triển khai.
- **SC-007**: Thay đổi gu do hệ thống suy ra có hiệu lực trong vòng 60 giây sau khi người dùng lưu
  outfit.
- **SC-008**: 0 trường hợp gu người dùng khai bị hệ thống ghi đè hoặc xoá.
- **SC-009**: 0 trường hợp người dùng sao chép được món riêng tư của người dùng khác.
- **SC-010**: 95% yêu cầu gợi ý phối đồ trả kết quả trong dưới 5 giây.
- **SC-011**: 0 thay đổi phá vỡ client hiện tại: các trường mới là additive, client cũ bỏ qua được;
  các trường cũ giữ nguyên tên và kiểu dữ liệu. `recommendationId` và `personalizationEnabled` luôn có
  mặt nhưng client cũ không cần đọc.
- **SC-012**: Gợi ý bổ sung và thao tác hoàn thiện bộ đồ không làm giảm số lượt gợi ý AI còn lại của
  người dùng.
- **SC-013**: Dữ liệu quan sát của ít nhất 30 ngày gần nhất đủ để tính SC-002, SC-003 và SC-004 mà
  không cần triển khai thêm công cụ mới.
- **SC-014**: Ít nhất 90% gợi ý bổ sung khớp với ngữ cảnh của yêu cầu và, khi gu cá nhân đang bật,
  khớp ít nhất một phong cách yêu thích hoặc không thuộc phong cách muốn tránh.

## Assumptions

- **Giá trị phong cách là chuỗi tự do**: giao diện quyết định danh sách gợi ý; hệ thống không xác thực
  theo từ vựng. Với giá trị lạ, chất lượng giảm dần về so khớp ngữ nghĩa thay vì bị từ chối.
- **Quyền lợi mặc định tắt**: gói miễn phí mặc định tắt. Migration của feature bật một lần cho các gói
  đang active có giá > 0 (bao gồm premium hiện có); gói khuyến mãi giá 0 và gói tạo sau khi triển khai
  mặc định tắt và do admin bật thủ công. Cờ runtime là nguồn sự thật, không dùng `price > 0` để quyết
  định quyền lợi khi chạy.
- **Dung sai cache quyền lợi**: overview được cache tối đa 60 giây; việc ngừng áp dụng/ghi sau khi
  quyền lợi đổi có thể trễ tối đa 60 giây. Đây là dung sai đã biết, không phải vi phạm FR-004.
- **Giới hạn 4 phong cách** là giá trị mặc định của cấu hình hệ thống, có thể đổi mà không cần
  chuyển đổi dữ liệu người dùng.
- **Lưu outfit là đại diện cho việc sử dụng** trong phạm vi feature này; không có thao tác "đã mặc"
  riêng.
- **Học gu là bất đồng bộ**: chấp nhận nhất quán cuối cùng trong vòng 60 giây; thao tác lưu outfit
  không phụ thuộc vào kết quả học.
- **Tủ đồ hệ thống là nguồn duy nhất cho gợi ý bổ sung**; món của nhãn hàng vẫn hoạt động như hành vi
  gợi ý hiện tại, không trộn vào gợi ý bổ sung.
- **Món hệ thống hợp lệ** là món đang hoạt động, chưa bị xoá và đã có thông tin phân tích.
- **Ngưỡng chất lượng để coi một vai trò là "chưa phù hợp"** là cấu hình của hệ thống; giá trị cụ
  thể được chốt ở bước lập kế hoạch và có thể điều chỉnh mà không cần đổi dữ liệu người dùng.
- **Số lượng gợi ý bổ sung** là cấu hình của hệ thống (mặc định một món mỗi vai trò, tối đa ba món
  mỗi phản hồi) và có thể điều chỉnh độc lập với dữ liệu người dùng.
- **Cửa sổ vùng làm mát** mặc định là bảy ngày cho món vừa sử dụng và ba ngày cho món vừa được gợi ý;
  cả hai là cấu hình của hệ thống.
- **Hạ tầng log và bộ đếm hiện có được tái sử dụng** cho dữ liệu quan sát xoay vòng; không cần hệ
  thống phân tích mới.
- **Gợi ý bổ sung dùng chung ngữ cảnh và gu với bộ đồ chính**, không có bộ tiêu chí riêng.
- **Sao chép món tiêu tốn hạn mức tủ đồ** như mọi món khác.
- **Quy tắc cấu trúc bộ đồ hiện tại không đổi**: vẫn yêu cầu một trong hai cấu trúc hợp lệ và các
  vai trò lớp phủ theo quy định hiện hành.
- **Người dùng có ít nhất 5 món hoạt động** mới nhận gợi ý phối đồ, giữ nguyên hành vi hiện tại.
- **Cơ chế xử lý bất đồng bộ hiện có được tái sử dụng** cho việc học gu; không cần hệ thống lưu trữ
  mới.
- **Dữ liệu gu suy ra không hiển thị trực tiếp cho người dùng** trong phạm vi này; chỉ ảnh hưởng
  gợi ý.
- **Feature mang số 033** theo yêu cầu của người phát triển.
- **`.specify/feature.json` là file cục bộ** phục vụ công cụ phát triển, không phải một phần của sản
  phẩm.

## Out of Scope

- Thêm thao tác "đánh dấu đã mặc" riêng; feature này dùng lưu outfit làm đại diện.
- Hoàn tác gu đã học khi người dùng xoá outfit.
- Chuẩn hoá lại phong cách của toàn bộ món đồ về một bộ từ vựng cố định (bao gồm phân tích lại ảnh).
- Giao diện quản trị để quản lý danh mục phong cách.
- Đưa món của nhãn hàng vào gợi ý bổ sung hoặc ưu tiên thương mại trong gợi ý bổ sung.
- Cho người dùng xem và sửa trực tiếp gu do hệ thống suy ra.
- Gợi ý phong cách từ xu hướng bên ngoài hoặc dữ liệu cộng đồng.
- Hỗ trợ đa ngôn ngữ cho giá trị phong cách.
- Tái tạo lịch sử sử dụng cho các món đã có trước khi triển khai.
- Thay đổi quy tắc cấu trúc bộ đồ hoặc cách tính hạn mức tủ đồ hiện có.
