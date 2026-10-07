# Feature Specification: Gỡ chặn đường sửa mức quyền tính phí AI (grant-repair-unblock)

**Feature Branch**: `027-grant-repair-unblock`

**Created**: 2026-10-07

**Status**: Draft

**Input**: User description: "Hai lỗi trong đường sửa grant khiến 18/28 tài khoản production bị từ chối vĩnh viễn khi chạy POST /admin/subscription-grants/restore ngày 2026-10-07. Lỗi 1: rejectStaleEvent từ chối vĩnh viễn khi grant hiện hành không đọc được occurred_at từ envelope — deadlock khởi tạo. Lỗi 2: EffectiveAt neo về subscription.started_at khiến effective_to == effective_from, vi phạm CHECK constraint SQLSTATE 23514, vốn bị phân loại nhầm là transient. Cần FR-001 lấy created_at làm đồng hồ thay thế; FR-002 buộc effective_from lớn hơn nghiêm mọi grant cũ, lệch tối đa 1 microsecond; FR-003 phân loại 23514 là refusal vĩnh viễn với mã lý do riêng."

> **Đây là đặc tả sửa lỗi cho các yêu cầu đã tồn tại của spec 024, KHÔNG phải tính năng mới.**
> Không yêu cầu nào ở đây thay đổi mô hình dữ liệu, không thêm endpoint, không thêm trường
> vào sự kiện, và không thay đổi nội dung hợp đồng của `subscription.plan.changed`.
> Điểm nó chạm vào là **cơ chế kiểm tra thứ tự sự kiện** của FR-033 và **ranh giới cửa sổ**
> của FR-031/FR-034.

## Bối cảnh vấn đề

Ngày 2026-10-07, lần chạy đầu tiên của công cụ khôi phục mức quyền trên production cho
28 tài khoản: **10 được sửa, 18 bị từ chối vĩnh viễn** và thông điệp nằm chết trong hàng
đỗi chết. Hai nguyên nhân độc lập, cùng xuất hiện trong cùng một lượt chạy.

**Nguyên nhân 1 — deadlock khởi tạo.** Bước kiểm tra thứ tự sự kiện cần so thời điểm xảy
ra của sự kiện đã tạo ra bản ghi mức quyền hiện hành, và thời điểm đó nằm trong ảnh
chụp chính sách. 22 bản ghi tồn tại từ trước khi ảnh chụp chính sách có phần `envelope`
không mang được thông tin này. Bước kiểm tra không lấy được đồng hồ thì từ chối vĩnh
viễn. Công cụ sửa duy nhất bị chặn bởi chính dữ liệu hỏng mà nó được gọi để sửa.

Hệ quả vượt ra ngoài bản thân việc sửa dữ liệu: **mọi lần đổi gói tương lai của những
tài khoản đó cũng bị từ chối y hệt.** Tài khoản nâng cấp thành công trên bản ghi đăng
ký, còn quyền AI vẫn ở mức cũ, và không có đường phục hồi.

**Nguyên nhân 2 — vi phạm ràng buộc dữ liệu, bị phân loại nhầm.** Công cụ ghim thời
điểm hiệu lực của bản ghi mới vào đúng ngày kỳ cước bắt đầu, rồi đóng bản ghi cũ bằng
chính giá trị đó làm mốc cắt. Khi bản ghi cũ bắt đầu đúng vào ngày đó, mốc cắt trùng
với điểm bắt đầu, và ràng buộc `effective_to` phải sau `effective_from` từ chối. Trên
production, **28/28 tài khoản** có đúng đặc điểm đó.

Lỗi này vốn được đóng nhãn là lỗi tạm thời, nên thông điệp bị thử lại 5 lần — mỗi lần
đều chắc chắn thất bại — rồi mới chết. Nhãn sai làm che mất việc đây là lỗi tất định.

## Quan hệ với các yêu cầu đang tồn tại — đọc trước khi kế hoạch

Đặc tả này **không** phủ định yêu cầu nào của spec 024, nhưng có một chỗ cần ghi thay đổi
tường minh thay vì sửa ngầm.

- **FR-033 (spec 024) có một tiền đề mà dữ liệu đang vi phạm.** Nguyên văn FR-033 yêu
  cầu từ chối mọi sự kiện có thời điểm xảy ra không mới hơn thời điểm **"đã ghi trong bản
  ghi mới nhất"**, và phần giải thích nói rõ điều kiện để làm được điều đó: *"mỗi bản ghi
  mức quyền tính phí phải lưu kèm thời điểm xảy ra của sự kiện đã tạo ra nó, trong ảnh
  chụp chính sách"*. 22 bản ghi đời trước **không thỏa tiền đề đó**. FR-033 không nói gì
  về trường hợp tiền đề không thỏa; code hiện tại chọn "từ chối vĩnh viễn" — đúng một cách
  chọn có chủ ý, nhưng nó biến dữ liệu cũ thành vĩnh viễn không sửa được. FR-001 ở đây
  **bổ sung** nhánh còn thiếu, không sửa nhánh đang có.
- **Một tài liệu đang mô tả sai.** `specs/024-admin-premium-grant/contracts/subscription-plan-changed-event.md`
  dòng 153 ghi mã lý do `stale_event` là so với *"bản ghi đang hiệu lực"*, trong khi dòng
  122 của chính file đó và FR-033 đã sửa phạm vi thành *"bản ghi mới nhất"*. Sai từ T164.
  Sẽ đính chính trong `tasks.md`.
- **FR-034 (spec 024)** cấm ràng buộc duy nhất ở tầng dữ liệu cho cột trạng thái, vì
  một tài khoản cần giữ nhiều bản ghi `active` theo thứ tự thời gian. FR-002 ở đây
  **không** thêm ràng buộc nào ở tầng dữ liệu; nó làm cho quy ước ứng dụng của FR-031 và
  FR-034 khớp với ràng buộc `CHECK` đã tồn tại.
- **Lệch phạm vi so sánh của FR-033 là hậu quả, không phải nguyên nhân.** Sau FR-001,
  mã lý do `stale_event` chỉ còn mang nghĩa "sự kiện thật sự đến muộn" — nhánh "envelope
  không đọc được" không còn từ chối nữa. Đây là hệ quả đúng đắn, nhưng phải nói ra.

## Clarifications

### Session 2026-10-07

- Q: Mục 4 của kiểm tra toàn vẹn đếm cả bản ghi `đã đóng`, mà công cụ khôi phục không
  bao giờ chạm vào chúng — vậy SC-005 có đạt được không? → A: Sửa mục 4 lọc theo **cửa
  sổ hiệu lực**, khớp đúng điều kiện mà bộ đọc dùng, **không** lọc theo trạng thái;
  thêm một dòng thông tin đếm riêng bản ghi lịch sử hỏng.
- Q: Sau khi sửa code, làm sao sửa được các tài khoản đã bị từ chối, khi khoá sự kiện
  của công cụ khôi phục đã được ghi nên chạy lại không có tác dụng? → A: Dùng đường ghi
  hành chính `POST /api/v1/admin/subscription-grants` với `Idempotency-Key` mới, và
  **tắt gửi email trong suốt thao tác** bằng `EMAIL_ENABLED=false`.

### Sai lệch đã phát hiện khi làm rõ — con số "18 tài khoản" là sai

Con số 18 trong phần bối cảnh là **số tài khoản mà lượt khôi phục hôm nay không sửa
được**, không phải số tài khoản cần sửa. Trong đó:

- **8 tài khoản** bị từ chối vì không đọc được `envelope` — grant cũ **vẫn còn hỏng**,
  đây mới là nhóm cần sửa.
- **10 tài khoản** bị từ chối vì vi phạm ràng buộc — nhưng giao dịch bị lùi nguyên tử,
  nên grant gốc của họ **vẫn tốt**. Thao tác "sửa" thất bại trên tài khoản không cần sửa.

Thao tác cần làm cho **8 tài khoản**, trong đó 6 là tài khoản thật thuộc gói `free` và 2
là tài khoản demo. Danh sách và cách thao tác nằm ở
[runbook.md](runbook.md) của mục này.

## User Scenarios & Testing *(bắt buộc)*

### User Story 1 - Khách hàng đổi gói vẫn được đổi quyền AI (Priority: P1)

Một khách hàng đang ở gói Premium nâng cấp hoặc hạ cấp. Trước đây, nếu bản ghi mức quyền
hiện hành của họ là bản ghi cũ chưa có ảnh chụp chính sách đầy đủ, thay đổi gói của họ
bị hệ thống bỏ qua âm thầm: gói trong bản ghi đăng ký đã đổi, còn quyền AI thì không.
Sau thay đổi này, thay đổi gói của họ được áp dụng như với mọi tài khoản khác.

**Vì sao ưu tiên P1**: Đây là hậu quả trực tiếp lên tiền của khách hàng. Tài khoản trả
tiền mà không nhận được quyền tương ứng, và không có đường phục hồi nào. Trên production,
một tài khoản `premium-monthly` đang ở đúng trạng thái này.

**Kiểm thử độc lập**: Chuẩn bị một tài khoản có bản ghi mức quyền hiện hành thiếu ảnh
chụp chính sách, phát một sự kiện đổi gói hợp lệ, xác nhận quyền AI của tài khoản thay
đổi theo gói mới — mà không cần bất kỳ thao tác sửa dữ liệu nào.

**Acceptance Scenarios**:

1. **Given** tài khoản có bản ghi mức quyền hiện hành không đọc được thời điểm xảy ra từ
   ảnh chụp chính sách, **When** một sự kiện thay đổi gói hợp lệ tới, **Then** quyền AI
   của tài khoản được cập nhật theo gói mới.
2. **Given** tài khoản có bản ghi mức quyền hiện hành không đọc được thời điểm xảy ra,
   **When** một sự kiện thay đổi gói tới **muộn hơn** thời điểm bản ghi đó được tạo,
   **Then** sự kiện bị từ chối, quyền đã bị thu hồi **không** được hồi sinh, và không
   bản ghi nào bị đóng.
3. **Given** tài khoản có bản ghi mức quyền hiện hành không đọc được thời điểm xảy ra,
   **When** hai sự kiện thay đổi gói tới gần như đồng thời, **Then** còn đúng một bản
   ghi mức quyền đang hiệu lực.

---

### User Story 2 - Quản trị viên sửa được tài khoản hỏng (Priority: P1)

Một quản trị viên phát hiện bằng kiểm tra toàn vẹn rằng tài khoản nào đang có bản ghi mức
quyền hỏng, rồi chạy công cụ khôi phục. Trước đây công cụ báo thành công rồi không sửa
được gì, và không có cách chạy lại vì khoá sự kiện đã được ghi. Sau thay đổi này, một lượt
chạy sửa được tài khoản, và kết quả thật được phản ánh trong phản hồi.

**Vì sao ưu tiên P1**: Nếu US1 chưa được sửa thì User Story này vô nghĩa — công cụ sửa
và cơ chế chặn là cùng một đường. Ưu tiên ngang với US1 vì nó là đường *vận hành* của
cùng một tai nạn.

**Kiểm thử độc lập**: Chuẩn bị một tài khoản có bản ghi mức quyền hiện hành bắt đầu đúng
vào ngày kỳ cước bắt đầu, chạy công cụ khôi phục, xác nhận bản ghi được dựng lại với ảnh
chụp chính sách đọc được.

**Acceptance Scenarios**:

1. **Given** tài khoản có bản ghi mức quyền hiện hành bắt đầu đúng vào ngày kỳ cước bắt
   đầu, **When** công cụ khôi phục phát lại sự kiện, **Then** bản ghi cũ được đóng và bản
   ghi mới được tạo, không có lỗi ràng buộc dữ liệu nào phát sinh.
2. **Given** một bản ghi mức quyền đang hiệu lực bình thường, **When** công cụ khôi phục
   chạy, **Then** thời điểm hiệu lực của bản ghi mới **không** bị dịch — cơ chế này
   không được thay đổi hành vi của trường hợp thường.
3. **Given** tài khoản có nhiều bản ghi mức quyền lịch sử, **When** công cụ khôi phục
   chạy, **Then** bản ghi mới bắt đầu **sau** mọi bản ghi trước đó, và không có hai bản
   ghi đang hiệu lực chồng lấn.

---

### User Story 3 - Lỗi tất định được báo đúng một lần (Priority: P2)

Một lỗi không thể tự khỏi — như vi phạm ràng buộc dữ liệu — được báo cáo ngay, đếm một
lần, và không bị thử lại. Trước đây nó bị thử 5 lần rồi chết, tiêu tốn lượt phân phối
và làm người vận hành tưởng đó là sự cố hạ tầng tạm thời.

**Vì sao ưu tiên P2**: Không gây mất quyền cho khách hàng. Nhưng nó là nguyên nhân khiến
một lỗi tất định bị trông như hạ tầng, và nó chính là thứ đã làm che mất cả hai nguyên
nhân trong lượt chạy hôm nay.

**Kiểm thử độc lập**: Gây ra một vi phạm ràng buộc dữ liệu có kiểm soát, xác nhận nó
được báo đúng một lần, không thử lại, và đi thẳng vào hàng đỗi chết.

**Acceptance Scenarios**:

1. **Given** một thao tác ghi vi phạm ràng buộc dữ liệu, **When** nó xảy ra, **Then** hệ
   thống bỏ qua thông điệp **một lần**, ghi một mã lý do riêng, và không thử lại.
2. **Given** một sự cố tạm thời thật sự — ví dụ kết nối cơ sở dữ liệu gián đoạn, **When**
   nó xảy ra, **Then** thông điệp vẫn được thử lại như trước. Việc phân loại lại không
   được biến lỗi tạm thời thành lỗi vĩnh viễn.

---

### Edge Cases

- **Tài khoản không có bản ghi mức quyền nào**: đường hiện hành đã xử lý đúng, sự kiện
  được chấp nhận vì không có gì để so sánh. Không thay đổi.
- **Bản ghi mới nhất đã đóng và không đọc được ảnh chụp chính sách**: phạm vi so sánh
  của FR-033 cố ý bao gồm bản ghi đã đóng, nên đây là trường hợp thường gặp, không phải
  hiếm. Nhánh thay thế đồng hồ phải áp dụng y hệt.
- **Thời điểm tạo bản ghi trùng khớp thời điểm xảy ra của sự kiện**: phải bị từ chối.
  Quy tắc "mới hơn" là nghiêm ngặt, không phải "không cũ hơn".
- **Lệch đồng hồ giữa hai máy làm thời điểm tạo bản ghi vượt tương lai so với thời điểm
  xảy ra của sự kiện**: một sự kiện hợp lệ có thể bị từ chối. Chấp nhận và ghi cảnh báo,
  vì hướng lựa chọn ngược lại sẽ cho phép hồi sinh quyền đã thu hồi — điều FR-033 cấm.
- **Bản ghi mới đã bắt đầu sau tất cả bản ghi cũ**: không có gì cần dịch. Đây là
  trường hợp phổ biến nhất và phải là trường hợp không đổi hành vi.
- **Nhiều sự kiện cho cùng một tài khoản chạy song song**: việc tính lại thời điểm bắt
  đầu phải xảy ra **bên trong** khoá theo tài khoản, nếu không hai giao dịch sẽ cùng
  tính một cách độc lập trên hai trạng thái khác nhau.
- **Sự kiện bị từ chối ở bước kiểm tra thứ tự có đóng bản ghi nào không**: không được.
  Bản ghi bị từ chối phải để nguyên toàn bộ.
- **Cửa sổ quyền bị dịch một microsecond có đổi thời điểm kết thúc không**: có, và đó là
  hành vi được chấp nhận — xem phần Giả định.

## Requirements *(bắt buộc)*

### Functional Requirements

- **FR-001**: Khi bản ghi mức quyền tính phí mới nhất của tài khoản không lấy được thời
  điểm xảy ra từ ảnh chụp chính sách, hệ thống AI MUST **không** từ chối sự kiện vĩnh
  viễn. Hệ thống AI MUST so sánh thứ tự bằng thời điểm tạo của chính bản ghi đó.

- **FR-002**: Việc so sánh ở FR-001 MUST giữ nguyên tính chặn hồi sinh mức quyền đã bị
  thu hồi. Một sự kiện có thời điểm xảy ra **không mới hơn** thời điểm tạo của bản ghi
  mới nhất MUST bị từ chối, đúng như một sự kiện không mới hơn thời điểm xảy ra đã ghi.

- **FR-003**: Hệ thống AI MUST ghi một dấu vết cảnh báo khi phải dùng thời điểm tạo bản
  ghi thay cho thời điểm xảy ra trong ảnh chụp chính sách, để một nhân sự vận hành đọc log
  thấy rằng bộ đếm từ chối đang chạy trên đồng hồ yếu hơn bình thường.

- **FR-004**: Thời điểm bắt đầu hiệu lực của bản ghi mức quyền mới MUST **lớn hơn nghiêm**
  thời điểm bắt đầu hiệu lực của mọi bản ghi trước đó của cùng tài khoản.

- **FR-005**: Việc tính lại ở FR-004 MUST thực hiện **bên trong** giao dịch, sau khi đã
  giữ khoá theo tài khoản, và MUST dùng trạng thái mà không giao dịch nào khác có thể thay
  đổi trong lúc đang tính. Nếu không, hai giao dịch sẽ cùng ra một quyết định từ hai
  trạng thái khác nhau.

- **FR-006**: Mức dịch tối đa của thời điểm bắt đầu hiệu lực do FR-004 gây ra MUST
  **không vượt quá 1 microsecond**. Ràng buộc này bảo đảm thay đổi không làm lệch đáng
  kể bất kỳ cửa sổ quyền nào.

- **FR-007**: FR-004 MUST là **không có tác dụng phụ** trong vận hành bình thường. Khi
  thời điểm hiệu lực của sự kiện đã lớn hơn mọi bản ghi trước đó, hệ thống MUST giữ nguyên
  thời điểm đó và không ghi bất kỳ thay đổi nào.

- **FR-008**: Việc dịch thời điểm bắt đầu hiệu lực MUST **không** làm hai bản ghi mức quyền
  của cùng một tài khoản chồng lấn. Bất kỳ hai bản ghi đang hiệu lực cùng lúc nào của
  một tài khoản MUST có khoảng thời gian không giao nhau, đúng như FR-034 của spec 024
  đòi hỏi ở tầng ứng dụng.

- **FR-009**: Một thao tác ghi vi phạm ràng buộc dữ liệu ở tầng cơ sở dữ liệu MUST được
  phân loại là từ chối vĩnh viễn, **không** phải sự cố tạm thời, và MUST dùng một mã
  lý do riêng để không nhập nhằng với các nguyên nhân từ chối đã có.

- **FR-010**: Việc phân loại lại ở FR-009 MUST **không** thay đổi cách xử lý của một sự
  cố tạm thời thật sự. Thông điệp do lỗi tạm thời MUST tiếp tục được thử lại như hiện tại.

- **FR-011**: Mã lý do mới ở FR-009 MUST xuất hiện trong bộ đếm từ chối công khai của dịch
  vụ AI, và mỗi lần xảy ra MUST được ghi đúng một dòng cảnh báo. Hệ thống MUST **không**
  tự đặt ngưỡng số cho mã lý do này.

- **FR-012**: Một sự kiện bị từ chối ở bất kỳ bước nào MUST **không** làm thay đổi bất kỳ
  bản ghi mức quyền nào của tài khoản đó. Việc bỏ qua và việc từ chối phải không có tác
  dụng phụ.

- **FR-013**: Bảng bản ghi mức quyền tính phí MUST NOT nhận thao tác ghi từ bất kỳ thành
  phần nào nào ngoài dịch vụ AI. Việc sửa dữ liệu đã hỏng MUST đi qua đường sự kiện, không
  qua câu lệnh SQL.

- **FR-014**: Kiểm tra toàn vẹn ở mục "ảnh chụp chính sách rỗng hoặc thiếu" MUST chỉ
  tính những bản ghi mà bộ đọc thực sự có thể trả về. Điều kiện lọc MUST **giống hệt**
  điều kiện mà bộ đọc dùng: **theo cửa sổ hiệu lực, và không lọc theo trạng thái.**

- **FR-015**: Lý do của FR-014 là bất biến trong `CHECK` và bộ đọc cùng nói về thứ này:
  một bản ghi có trạng thái `đã đóng` nhưng cửa sổ còn hiệu lực vẫn **được đọc**, nên
  nó phải nằm trong phạm vi kiểm tra. Ngược lại, một bản ghi `đã đóng` đã hết hạn thì không
  ảnh hưởng tài khoản nào và không thuộc phạm vi kiểm tra. Lọc theo trạng thái sẽ bỏ sót
  trường hợp đầu.

- **FR-016**: Kiểm tra toàn vẹn MUST vẫn **báo cáo** số bản ghi lịch sử có ảnh chụp chính
  sách không đọc được, dưới dạng thông tin **không** chặn. Số này MUST **không** được
  tính vào kết quả kiểm tra. Mục đích là để dữ liệu đời trước vẫn hiện diện khi ai đó
  đọc báo cáo, thay vì biến mất chỉ vì nó không còn ảnh hưởng ai.

### Ràng buộc kiến trúc

- **AC-01**: Dịch vụ AI MUST **không** đọc bảng của dịch vụ Account để suy ra gói cước
  hiện tại. Client subscription đã bị gỡ khỏi đồ thị phụ thuộc của AI (024 T178) và đặc
  tả này không được làm mất kết quả của việc gỡ đó.
- **AC-02**: MUST **không** thêm cờ đặc quyền kiểu *"bỏ qua kiểm tra thứ tự"* cho một
  loại sự kiện hoặc một nguồn phát cụ thể. Việc tạo một đường đặc quyền thứ hai để vá lỗi
  dữ liệu là đánh đổi xấu hơn việc sửa đúng cơ chế.
- **AC-03**: Không thay đổi hợp đồng của sự kiện `subscription.plan.changed`: các trường
  bắt buộc, tập loại sự kiện, và phản hồi khi sự kiện bị từ chối phải giữ nguyên.

### Key Entities

- **Bản ghi mức quyền tính phí**: Quyền AI hiện hành của một tài khoản theo một gói cước
  trong một khoảng thời gian. Thuộc tính quan tâm: tài khoản, gói, thời điểm bắt đầu và kết
  thúc hiệu lực, trạng thái, ảnh chụp chính sách, thời điểm tạo bản ghi.
- **Ảnh chụp chính sách**: Tài liệu hai phần ghi lại trong bản ghi mức quyền. Phần
  `envelope` mang thời điểm xảy ra của sự kiện đã tạo ra bản ghi. Phần `entitlements` mang
  giới hạn lúc cấp. Hai phần có thể vắng mặt ở dữ liệu tạo trước khi cấu trúc này tồn
  tại — đó là nguồn gốc của nguyên nhân 1.
- **Sự kiện thay đổi gói**: Tín hiệu Account phát ra khi gói cước của tài khoản đổi, mang
  thời điểm xảy ra, thời điểm hiệu lực, định danh bản ghi lịch sử nguồn và gói đích.
- **Khoá theo tài khoản**: Khoá giữ trong suốt giao dịch, đảm bảo mọi quyết định về bản ghi
  mức quyền của một tài khoản được đưa ra trên một trạng thái không giao dịch nào khác thay
  đổi được.

## Success Criteria *(bắt buộc)*

### Measurable Outcomes

- **SC-001**: 100% tài khoản có bản ghi mức quyền không đọc được ảnh chụp chính sách đầy
  đủ nhận được cập nhật quyền khi đổi gói. Không còn sự kiện thay đổi gói nào của những
  tài khoản đó bị bỏ qua.
- **SC-002**: 0 tài khoản nào có từ hai bản ghi mức quyền đang hiệu lực cùng lúc, trên mọi
  môi trường, sau bất kỳ lượt sửa nào.
- **SC-003**: Mức dịch thời điểm bắt đầu hiệu lực tối đa quan sát được là **1 microsecond**;
  trường hợp vận hành bình thường là **0**.
- **SC-004**: Một vi phạm ràng buộc dữ liệu được báo **đúng một lần** và đi thẳng vào hàng
  đỗi chết; số lần thử lại bằng 0.
- **SC-005**: Sau khi chạy công cụ khôi phục trên production, kiểm tra toàn vẹn trả về **0**
  ở mục "tài khoản có gói nhưng không có bản ghi mức quyền" và **0** ở mục "ảnh chụp chính
  sách rỗng hoặc thiếu" — với mục thứ hai đã thu hẹp phạm vi theo FR-014.
- **SC-008**: Con số bản ghi lịch sử có ảnh chụp chính sách không đọc được **vẫn được báo
  cáo** sau khi mục kiểm tra về 0. Nếu dòng thông tin này biến mất khỏi báo cáo, thì việc
  thu hẹp phạm vi đã xoá dữ liệu thay vì chỉ đo lại nó — và mức đó chưa được chấp nhận.
- **SC-006**: 0 thông điệp nào bị đưa vào hàng đỗi chết mà không để lại một dấu vết truy
  vết được gồm tài khoản, mã lý do và thông điệp nguyên nhân.
- **SC-007**: Một sự cố tạm thời thật sự vẫn được thử lại và cuối cùng thành công — việc
  phân loại lại không làm hỏng đường phục hồi đã có.

## Assumptions

- **Mức dịch 1 microsecond dịch luôn cả hai đầu cửa sổ.** Khi thời điểm bắt đầu bị dịch,
  thời điểm kết thúc tính theo thời lượn gói cũng dịch theo. Sai số bằng 1 microsecond trên
  một khoảng 30 ngày là vô nghĩa, và giữ cho cửa sổ liên tục là điều quan trọng hơn.
- **Không thêm bản ghi kiểu hậu kiệm.** Cách sửa được chọn là làm cho thời điểm hiệu lực
  đúng ngay từ đầu, không phải ghi một ngoại lệ và hy vọng đường đọc hiểu.
- **Mã lý do `stale_event` sau thay đổi chỉ còn nghĩa "sự kiện đến muộn thật sự".** Không
  cần mã lý do mới cho trường hợp ảnh chụp chính sách không đọc được, vì trường hợp đó
  không còn bị từ chối nữa.
- **Phạm vi so sánh của FR-033 giữ nguyên là "bản ghi mới nhất", bao gồm bản ghi đã đóng
  và bản ghi chờ tới.** Không thu hẹp về bản ghi đang hiệu lực.
- **Quy ước cảnh báo theo log và vô điều kiện của dịch vụ AI được giữ nguyên.** Hệ thống
  không có nền tảng số liệu, và một ngưỡng chưa ai hiệu chỉnh thì tệ hơn là không có.
- **Không cần thay đổi lược đồ cơ sở dữ liệu.** Ràng buộc `CHECK` đang tồn tại và đang
  đúng; vấn đề là mã không tuân theo nó.
- **Phải đính chính một dòng tài liệu hợp đồng sự kiện** mô tả sai phạm vi so sánh của
  `stale_event` — cùng file, dòng 122 viết đúng, dòng 153 viết sai.
- **Phải tạo `deferred.md`.** Đặc tả này đã biết ít nhất một phát hiện ngoài phạm vi
  (xem mục Ngoài phạm vi bên dưới), và hiến pháp yêu cầu ghi vào đó thay vì để thành task
  chưa đánh dấu.

## Ngoài phạm vi

- **Tự chữa grant bằng sự kiện tự phát** — đã ghi ở ADR 013 với điều kiện mở lại.
- **Nền tảng số liệu và ngưỡng cảnh báo** — ngoài phạm vi, là việc hạ tầng.
- **`ResolvePolicy` không lọc trạng thái bản ghi.** Hiện đang lọc theo cửa sổ hiệu lực
  nhưng không lọc trạng thái, nên một bản ghi `đã đóng` có cửa sổ còn hiệu lực sẽ được
  dùng. Hiện không xảy ra (kiểm tra toàn vẹn trả về 0 ở mục tương ứng) và đây là bất biến
  chỉ ở tầng ứng dụng, không có gì ở tầng dữ liệu bảo vệ. Ghi nhận, không sửa.
- **Bổ sung grant cho các tài khoản thiếu hẳn ở môi trường development.** Mười hai tài
  khoản development không có bản ghi mức quyền nào; đường suy giảm hai tầng xử lý đúng và
  đây không phải điều kiện production.
- **Thay đổi tập loại sự kiện, hợp đồng sự kiện, hoặc nội dung ảnh chụp chính sách.**

## Câu hỏi cần làm rõ

Không có. Các quyết định còn bỏ ngỏ đã được chốt trước khi đặc tả:

- Mức dịch tối đa **1 microsecond** — đã chốt, và FR-006 ghi lại để ràng buộc.
- Dùng **thời điểm tạo bản ghi** làm đồng hồ thay thế — đã chốt, FR-001.
- Việc sửa thực hiện ở **phía dịch vụ AI** trong giao dịch — đã chốt, FR-004 và FR-005.
- **Không** thêm cờ đặc quyền — đã chốt, AC-02.
- **Không** sửa dữ liệu bằng SQL — đã chốt, FR-013.