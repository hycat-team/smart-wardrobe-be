# Feature Specification: Cơ chế chiến dịch tặng gói cho tài khoản đăng ký mới

**Feature Branch**: `025-signup-campaign-grant`

**Created**: 2026-10-01

**Status**: Draft

**Input**: User description: "Tạo một cơ chế chiến dịch (campaign) tặng gói cước cho các tài khoản đăng ký mới. Chiến dịch đầu tiên phục vụ buổi ra mắt ngày 00:00 ngày 02/10/2026 giờ Việt Nam: 50 tài khoản đầu tiên tạo mới qua luồng đăng ký email hoặc đăng nhập Google sẽ được cấp gói `premium-monthly` trong 1 tháng, không bật tự động gia hạn. Sau khi đủ 50 suất, chiến dịch tự ngủ và không cấp thêm. Chiến dịch chạy qua hàng đỗi tin nhắn với một worker xử lý, chấp nhận độ trễ, và phải chống được tình huống nhiều phiên bản hệ thống cấp đồng thời làm vượt hạn mức. Tài khoản do quản trị viên tạo không tính vào chiến dịch. Người dùng được nhận gói sẽ nhận được email thông báo. QUAN TRỌNG — mã nguồn viết ra KHÔNG cần phải xoá sau khi dùng xong: sau này sẽ còn các chiến dịch tương tự, và mỗi lần chỉ cần khai báo mục tiêu (goal) mới là chạy được, không phải sửa hay triển khai lại mã nguồn."

## Thuật ngữ: tên trong văn xuôi và mã thật trong cơ sở dữ liệu

Tài liệu này dùng **tiếng Việt trong văn xuôi** nhưng **mã thật trong cơ sở dữ liệu** khi nói về giá trị cụ thể.

| Cách viết trong văn xuôi | Mã thật trong cơ sở dữ liệu | Ở đâu | Ghi chú |
|---|---|---|---|
| gói Premium 1 tháng | `premium-monthly` | `subscription.subscription_plans.slug` | **không** phải `premium` và **không** phải `premium-default` |
| gói miễn phí | `free` | `subscription.subscription_plans.slug` | hạn mức 100 món tủ, 100 bộ đồ, 5 lượt gợi ý/ngày, 3 lượt trò chuyện/ngày |
| tự động gia hạn | `is_auto_renew_enabled` | `subscription.user_subscriptions` | mặc định là **tắt**; chỉ tài khoản chủ động bật mới bật |
| lượt đã cấp quyền lợi | `subscription.campaign_claims` | bảng lượt cấp, khoá `UNIQUE (user_id)` | **mỗi tài khoản chỉ có tối đa 1 lượt cho TOÀN BỘ cơ chế**, không phải 1 lượt cho mỗi chiến dịch. Cột `campaign_code` chỉ ghi lại chiến dịch nào đã cấp. |
| phần của lượt cấp | `grant_bucket` | `subscription.campaign_claims` | `'main'` hoặc `'reserve'`. Dùng để đếm tách phần chính và phần dự phòng; **không** suy ra từ cột mốc ưu tiên (xem `data-model.md` mục 3.1) |
| tổng hạn mức | `quota` của chiến dịch | thuộc cấu hình chiến dịch | phần chính; chiến dịch ra mắt khai báo 50 |
| hạn mức dự phòng | `reserve` của chiến dịch | thuộc cấu hình chiến dịch | phần bù cho tài khoản bị trượt; mặc định 0; chiến dịch ra mắt khai báo 5 |
| trần cứng | `quota` + `reserve` | tính từ cấu hình chiến dịch | chiến dịch ra mắt là 55; **không bao giờ vượt** |
| mốc ưu tiên | thời điểm tạo của tài khoản lấp đầy `quota` | lưu trên bản ghi lượt cấp đó | chốt một lần rồi không đổi |
| chiến dịch | theo mã, ví dụ `launch-2026-10` | thuộc cấu hình chiến dịch | mỗi chiến dịch một mã riêng, không trùng nhau |
| sự kiện tài khoản được tạo | `identity.user.created` | hàng đỗi sự kiện | phát **vô điều kiện** từ cả 3 đường tạo tài khoản; có cờ `createdByAdmin` |
| loại thay đổi gói | `ACTIVATED` | lịch sử thay đổi gói | tập loại là **đóng** và dùng chung cho cả hai dịch vụ; bên nhận từ chối vĩnh viễn loại ngoài tập. Lần cấp gói của chiến dịch dùng **cùng** loại với lần cấp gói của quản trị viên |

> **Cảnh báo nhầm lẫn thường gặp nhất.** `premium-monthly` và `premium-default` là **hai mã khác nhau thuộc hai bảng khác nhau thuộc hai dịch vụ khác nhau**: `premium-monthly` là mã gói cước, `premium-default` là mã chính sách tính phí AI. Không được lấn sang bảng khác.

## Bối cảnh

### Ba quyết định đã chốt trước khi viết đặc tả

| # | Câu hỏi | Quyết định | Hệ quả trong tài liệu này |
|---|---|---|---|
| 1 | "50 tài khoản đầu tiên" tính theo trục nào, khi tin nhắn tới không đúng thứ tự | **Ưu tiên tài khoản được tạo sớm hơn**, chấp nhận cho phép vượt 50 một chút để bù người bị trượt vì tin nhắn tới muộn | FR-021 tới FR-033, SC-001 tới SC-001d. "50" thành mốc mục tiêu, trần cứng là 55. Bảng ở mục "Giới hạn đã biết" nêu rõ hai thứ tự cho kết quả **khác nhau**. |
| 2 | Nhiều chiến dịch cùng hiệu lực thì một tài khoản nhận thế nào | **Mỗi tài khoản chỉ tham gia một chiến dịch**; chiến dịch được chọn là chiến dịch **bắt đầu muộn nhất** mà tài khoản đủ điều kiện | FR-011, FR-012, FR-033. Lựa chọn chiến dịch là hàm thuần túy của cấu hình và thời điểm tạo tài khoản, không phụ thuộc thứ tự xử lý. |
| 3 | Có cần giao diện tra cứu trạng thái cho người vận hành không | **Có**, giao diện quản trị chỉ đọc: danh sách chiến dịch + danh sách tài khoản đã cấp, phân biệt được ba trạng thái | User Story 7, FR-065 tới FR-072, SC-011. Khối lượng lớn nhất còn lại của công việc. |

Ba quyết định này được chốt ở trên đặc tả. Quyết định 1 là quyết định **có đánh đổi thật**: nó phá vỡ tính tuyệt đối của con số 50, và cái đổi lại là sự công bằng cho tài khoản đăng ký sớm.

### Cơ chế này phải phục vụ nhiều lần, không phải một lần

Yêu cầu ban đầu chỉ nói về một con số cụ thể: 50 tài khoản đầu tiên trong ngày 02/10/2026. Nhưng yêu cầu bổ sung làm thay đổi bản chất công việc:

> *"mã nguồn viết ra KHÔNG cần phải xoá sau khi dùng xong: sau này sẽ còn các chiến dịch tương tự, và mỗi lần chỉ cần khai báo mục tiêu mới là chạy được, không phải sửa hay triển khai lại mã nguồn."*

Hệ quả là công việc **không** viết ra một chức năng một lần dùng cho buổi ra mắt. Nó viết ra một **cơ chế chiến dịch** dùng lại được, và con số "50 ngày 02/10" chỉ là **chỉ thị cấu hình đầu tiên** của cơ chế đó. Bối cảnh này được đặt ở đầu tài liệu vì nó quyết định phần lớn các yêu cầu bên dưới.

Ba hệ quả cụ thể:

| Hệ quả | Nghĩa là gì trong yêu cầu |
|---|---|
| Chiến dịch là **dữ liệu**, không phải mã | Thêm chiến dịch mới bằng khai báo, không bằng cách viết thêm hàm |
| Chiến dịch phải **ngủ** được | Khi không có chiến dịch nào chạy, hệ thống vẫn hoạt động bình thường, chỉ là không cấp gói. Không có lỗi, không có cảnh báo giả |
| Chiến dịch phải **thức dậy** được | Khởi động lại chiến dịch không được làm mất dữ liệu đã cấp trước đó. Một chiến dịch không bao giờ cấp lại cho tài khoản đã được cấp |

### Ba bài toán kỹ thuật đã kiểm chứng trên mã nguồn hiện có

#### Bài toán 1 — Đếm "50 tài khoản đầu tiên" phải chính xác tuyệt đối

Hệ thống chạy nhiều phiên bản (mỗi phiên bản một bộ worker riêng), và tin nhắn trong hàng đỗi được giao **ít nhất một lần**, nghĩa là cùng một thông báo có thể tới worker nhiều lần. Ba hệ quả phải xử lý:

- Hai worker cùng đọc "hiện đã cấp 49" rồi cùng cấp ⇒ vượt hạn mức.
- Cùng một thông báo tới hai lần ⇒ cùng một tài khoản bị cấp gói hai lần.
- Worker chết giữa chừng ⇒ dở một lượt cấp, phải cấp lại được mà không cấp đúp.

Đây là lý do hạn mức và hành động cấp gói phải nằm trong **cùng một giao dịch duy nhất**, không tách ra. Nếu tách, sẽ tồn tại khe hở giữa lúc "đếm" và lúc "ghi", và khe hở đó chính là 51.

#### Bài toán 2 — Không được để việc cấp gói làm treo đăng ký

Đăng ký là hành động người dùng đang chờ. Nếu đăng ký phải chờ tin nhắn từ hàng đỗi rồi mới xong, mọi tài khoản đều treo khi hàng đỗi trục trặc. Yêu cầu đã chọn phương án **chấp nhận độ trễ**: đăng ký hoàn tất ngay, việc cấp gói diễn ra sau, và nếu có độ trễ thì người dùng nhận gói muộn hơn chứ không bị treo.

#### Bài toán 3 — Lượt dùng AI hằng ngày đang bị đếm thiếu khi thao tác chồng nhau

Đây không phải vấn đề do chiến dịch gây ra, mà là vấn đề có sẵn phát hiện khi khảo sát. Hệ thống cộng dồn số lượt dùng AI trong ngày theo cách "đọc số hiện tại, cộng thêm một, ghi lại" qua ba câu lệnh **không nằm trong giao dịch nào**. Khi hai thao tác dùng AI của cùng một người chạy cùng lúc, cả hai đọc cùng một số, và lần ghi của thao tác sau đè lên lần ghi của thao tác trước.

Hệ quả đo được: người dùng **dùng 3 lượt AI thật nhưng hệ thống chỉ ghi nhận 1 lượt** — tức là vượt hạn mức mà không bị chặn. Hạn mức ngày là thứ quyết định có thu được tiền hay không, nên đây là rò rỉ doanh thu chứ không chỉ là sai số hiển thị.

Cơ chế khoá đúng đã tồn tại trong dữ liệu nhưng chưa ai dùng: bản ghi hạn mức ngày có sẵn cách đọc khoá để trì hoãn thao tác ghi. Vì vậy việc sửa là đóng lại một lỗ hổng đã có đường dẫn, không phải xây mới.

### Đã kiểm chứng: mọi đường đổi gói đều phát sự kiện

Chiến dịch không tạo tài khoản, nó chỉ **đổi gói** cho tài khoản đã có. Vì vậy điểm mấu chốt là: sự kiện thay đổi gói có được phát trên mọi đường đổi gói hay không — chứ không chỉ lúc tạo tài khoản. Đã rà toàn bộ mã nguồn module gói cước:

| Kiểm tra | Kết quả |
|---|---|
| Số đường ghi vào bảng gói của tài khoản | 12 |
| Trong đó phát sự kiện thay đổi gói | **12 / 12** |
| Câu lệnh SQL ghi thô nào ghi vòng qua không | 0 |

Bao gồm cả trường hợp gói **không đổi** (bật/tắt tự động gia hạn) — vẫn phát, vì bên nhận mới là bên quyết định có xử lý hay không.

**Hai ràng buộc phát hiện kèm, đã thành yêu cầu ở FR-092 tới FR-095:**

1. Tập loại thay đổi gói là **đóng** và dùng chung cho cả hai dịch vụ. Bên nhận **từ chối vĩnh viễn** mọi loại ngoài tập, đưa thông báo vào hàng đỗi chết — không phải thử lại. Nghĩa là thêm một loại mới cho lần cấp gói chiến dịch sẽ tạo ra một điểm hỏng mới, không phải một cải tiến.
2. Lần cấp gói của chiến dịch sẽ mang **cùng loại sự kiện** với lần cấp gói của quản trị viên, vì cả hai đều đi qua cùng một quy ước. Hệ quả: **lịch sử thay đổi gói không phân biệt được hai đường này**; nguồn gốc chỉ nằm ở nhật ký kiểm toán.

### Cần sửa trước, rồi mới xây chiến dịch

Vì sao sửa lỗi đếm hạn mức nằm chung trong phạm vi công việc này, thay vì tách riêng: chiến dịch trao quyền lợi ngay lập tức cho tài khoản mới, nên số lượt dùng AI sẽ tăng vọt ngay sau khi được cấp gói. Nếu bộ đếm vẫn đếm thiếu, tài khoản vừa nhận quyền lợi có nguy cơ vượt hạn mức ngay trong ngày đầu tiên — đúng lúc cần nhất là nó phải được đo đúng.

### Một vấn đề đã phát hiện nhưng quyết định không sửa trong lần này

Trong lúc khảo sát cơ chế tính phí AI, phát hiện thêm một vấn đề có sẵn, **không do công việc này gây ra**, và **không ảnh hưởng tới chiến dịch**:

- Số lượt dùng AI có trả phí hằng ngày đang được đếm theo **tài khoản và ngày lịch**, thay vì theo **gói cước đang có hiệu lực**.
- Vì vậy, khi một tài khoản dùng nhiều AI hơn hạn mức gói Premium, rồi gói đó hết hạn và tài khoản trở về gói miễn phí, tài khoản đó vẫn bị coi như đã dùng quá hạn mức cho tới hết ngày, và bị đẩy sang phương án xử lý cục bộ.
- Chiều ngược lại — từ gói miễn phí lên gói Premium — **không** xảy ra, vì gói Premium không dùng bộ đếm đó để chặn.

Đánh giá: đây là ranh giới mờ giữa lỗi và chủ ý, không làm mất tiền, và sửa nó đòi hỏi thay đổi cách bản ghi quyền AI được gắn với gói cước — thuộc lõi kiểm soát chi phí, nằm ngoài phạm vi công việc này. Quyết định là **ghi nhận là vấn đề đã biết, không sửa**, và ghi vào tài liệu công việc để không ai nhầm rằng chiến dịch là thủ phạm. Xem FR-075.

## Clarifications

### Session 2026-10-01

- Q: Cơ chế dự phòng + mốc ưu tiên có bảo đảm "50 tài khoản sớm nhất" bất kể thứ tự tin nhắn tới không, và nếu không thì nên hy sinh điều gì? → A: **Chấp nhận phụ thuộc thứ tự, hạ yêu cầu công bằng.** Giữ cơ chế cấp gói ngay với hạn mức dự phòng; sửa tiêu chí thành điều kiện "không tài khoản nào tạo sau mốc ưu tiên được cấp gói", và ghi nhận tường minh rằng khi tin nhắn tới ngược thứ tự thì tập tài khoản được chọn **không** phải 50 người sớm nhất. Lý do: giữ được yêu cầu cấp gói ngay sau đăng ký, tránh bắt tài khoản đăng ký sớm phải chờ đóng cửa sổ mới có gói.
  - **Hiệu chỉnh 2026-10-01 (giữ nguyên quyết định, sửa cách diễn đạt)**: câu trả lời trên viết tiêu chí là "không tài khoản nào tạo sau mốc ưu tiên được cấp gói" — văn xuôi đó **hiểu sai phạm vi**. Điều kiện `created_at < watermark` chỉ áp cho **nhánh phần dự phòng**; 50 lượt cấp phần chính được quyết định *trước khi* mốc ưu tiên được chốt, nên nhiều tài khoản tạo sau mốc vẫn được cấp từ phần chính và đó là hành vi đúng. Điều kiện đúng phải viết là: *"không tài khoản nào được cấp **từ phần dự phòng** mà tạo sau mốc ưu tiên"*, và phải kiểm tra theo `grant_bucket`. Mắc lỗi này sẽ làm kịch bản kiểm chứng K2.1 báo đỏ oan. Chi tiết ở mục "Giới hạn đã biết" và `quickstart.md` K2.1.
- Q: Hệ thống cam kết cấp gói trong bao lâu kể từ lúc đăng ký, và có cơ chế đóng sót không? → A: **Hai tầng — cam kết 5 phút cho đường đi bình thường, cộng cơ chế quét bù mỗi 15 phút.** Lý do: cam kết thời gian mà không có lưới an toàn thì bị vỡ đúng lúc hệ thống hỏng, và những tài khoản đó mất quyền lợi một cách im lặng; còn chỉ có lưới an toàn mà không cam kết thời gian thì người dùng và bộ phận hỗ trợ không biết phải chờ bao lâu.
- Q: Hệ thống có nên chuyển sang phát hướng publisher — luồng tạo tài khoản chỉ bắn một sự kiện "tài khoản được tạo" và không biết chiến dịch tồn tại hay không — thay vì bắn một sự kiện mang tính "có thể dính chiến dịch"? → A: **Có, đổi sang publisher-driven.** Sự kiện trở thành *sự thật về chính module tạo tài khoản* chứ không phải tín hiệu cho chiến dịch, phát vô điều kiện, và mọi tính năng sau này chỉ cần viết thêm bộ nhận thông báo mà không phải sửa lại luồng đăng ký. Lý do: nó cắt hết ràng buộc giữa module tạo tài khoản với chiến dịch, và biến chi phí phát thông báo thành chi phí nền tảng cố định thay vì phát sinh theo chiến dịch.
- Q: Hệ thống có nên thêm một loại sự kiện đổi gói riêng cho lần cấp gói của chiến dịch không, để lịch sử gói tự phân biệt được với lần cấp gói của quản trị viên? → A: **Không thêm, dùng chung loại sự kiện hiện có, phân biệt nguồn gốc bằng nhật ký kiểm toán.** Lý do: tập loại sự kiện là **đóng** và dùng chung cho cả hai dịch vụ, bên nhận **từ chối vĩnh viễn** mọi loại ngoài tập và đưa thông báo vào hàng đỗi chết — thêm loại mới là thay đổi hợp đồng sự kiện vượt biên, đúng thứ cần tránh. Nhật ký kiểm toán đã ghi nguồn gốc theo FR-038 nên thông tin phân biệt vẫn đủ.
- Q: Có thể chỉ phát `identity.user.created` mà không phát `subscription.plan.changed` lúc tạo tài khoản không, để mỗi lần tạo tài khoản chỉ bắn một chủ đề? → A: **Không, giữ cả hai.** Lý do: sự kiện đổi gói phát lúc cấp phát gói mặc định là tín hiệu **duy nhất** để dịch vụ AI biết tài khoản mới được cấp quyền chi phí, và nó được phát ở đúng chỗ đã có sẵn mọi thông tin cần. Bỏ nó buộc phải chọn giữa: cho dịch vụ AI nghe thêm chủ đề mới (và nếu hai bên lệch nhịp triển khai thì **mọi** tài khoản mới mất quyền), hoặc gộp làm một chủ đề (và buộc module tạo tài khoản phải biết tới gói cước, đúng cái ràng buộc mà thiết kế hướng publisher đã cố bỏ). Một dòng hàng đỗi thừa là phần rẻ nhất trong toàn bộ thiết kế này.
- Q: Sự kiện tạo tài khoản phát từ 2 đường (tự đăng ký) hay 3 đường (gồm cả quản trị viên)? → A: **Cả 3 đường, kèm cờ cho biết tài khoản có do quản trị viên tạo hay không**, và bên chiến dịch tự bỏ qua tài khoản có cờ đó. Lý do: sự kiện trung thực với thực tế, và mọi tính năng sau này đều có dữ liệu đầy đủ để tự quyết định. Đồng thời tài khoản do quản trị viên tạo **vẫn phải được cấp phát gói miễn phí mặc định** như mọi tài khoản khác, và đường cấp gói thủ công cho tài khoản đó vẫn phải hoạt động bình thường.

## User Scenarios & Testing *(bắt buộc)*

### User Story 1 - Người dùng mới đủ điều kiện nhận gói quà tặng (Priority: P1)

Tôi đăng ký tài khoản mới bằng email hoặc bằng Google trong thời gian chiến dịch đang chạy. Tôi thuộc số tài khoản đủ điều kiện theo hạn mức của chiến dịch. Tôi mong đăng ký hoàn tất ngay, và **trong vòng 5 phút** tôi tìm thấy tài khoản của mình đang ở gói Premium với thời hạn 1 tháng và **không** bị tự động gia hạn. Tôi không cần làm gì thêm và tôi không bị mất tiền nào.

**Why this priority**: Đây là giá trị mà chiến dịch tạo ra. Không có câu chuyện này thì các câu chuyện còn lại không có ý nghĩa.

**Independent Test**: Đăng ký một tài khoản mới, đợi hết độ trễ đã công bố, kiểm tra gói của tài khoản đó. Không cần chạy email, không cần chạy chiến dịch thứ hai — câu chuyện này tự đứng vững.

**Acceptance Scenarios**:

1. **Given** chiến dịch đang chạy và còn suất, **When** một người đăng ký tài khoản mới bằng email và xác nhận mã xác thực, **Then** tài khoản đó cuối cùng ở gói Premium với thời hạn 1 tháng kể từ lúc đăng ký, và cờ tự động gia hạn ở trạng thái tắt.
2. **Given** chiến dịch đang chạy và còn suất, **When** một người đăng nhập lần đầu bằng Google bằng một địa chỉ chưa có tài khoản, **Then** tài khoản mới đó nhận gói quà tặng giống trường hợp đăng ký bằng email.
3. **Given** chiến dịch đang chạy, **When** một người đã có tài khoản đăng nhập Google lại lần thứ hai, **Then** không có gói quà tặng nào được cấp thêm cho họ.
4. **Given** chiến dịch đang chạy, **When** một người đăng nhập Google mà địa chỉ đó đã thuộc một tài khoản có sẵn, **Then** tài khoản đó **không** bị tính là tài khoản mới của chiến dịch.
5. **Given** bất kỳ trường hợp nào ở trên, **When** người dùng hoàn tất đăng ký, **Then** yêu cầu đăng ký trả về thành công trong thời gian như một đăng ký bình thường, không phụ thuộc vào việc chiến dịch có xử lý xong hay không.
6. **Given** hệ thống bị gián đoạn khoảng 10 phút, **When** trong lúc gián đoạn đó có 20 tài khoản đủ điều kiện được tạo, **Then** sau khi hệ thống phục hồi, những tài khoản đó vẫn được cấp gói nhờ cơ chế quét bù, chứ không tự động mất quyền lợi.
7. **Given** một chiến dịch có thời điểm kết thúc, **When** một tài khoản được tạo sau thời điểm kết thúc đó, **Then** tài khoản đó không được cấp gói, kể cả khi thông báo của nó tới muộn.

---

### User Story 2 - Người dùng được báo rõ quyền lợi đã nhận (Priority: P2)

Tôi vừa được cấp gói quà tặng. Tôi mong nhận được một email nói rõ tôi đang ở gói nào, hết hạn vào ngày nào, và gói đó **không tự động gia hạn** nên tôi sẽ không bị trừ tiền. Email không được khiến tôi nghĩ rằng mình đã mua gói hay đã trả tiền.

**Why this priority**: Không có thông báo, quyền lợi tặng xuất hiện trở nên khó chấp nhận vì không biết mình có. Đây cũng là câu chuyện tách biệt được vì nó nằm sau câu chuyện cấp gói, không nằm trên đường chính.

**Independent Test**: Cấp gói cho một tài khoản, kiểm tra hộp thư của địa chỉ đó. Có thể bỏ qua hoàn toàn phần kiểm tra hạn mức.

**Acceptance Scenarios**:

1. **Given** tài khoản vừa được cấp gói quà tặng, **When** thao tác cấp gói hoàn tất, **Then** địa chỉ email của tài khoản đó nhận được một email nêu tên gói và ngày hết hạn.
2. **Given** cùng tình huống, **Then** email nói rõ gói không tự động gia hạn, và **không** nói hoặc không gợi ý rằng người dùng đã thanh toán.
3. **Given** cùng tình huống, **Then** email **không** nhắc đến số lượng suất của chiến dịch, vì thông tin đó không liên quan tới người nhận và sẽ gây thắc mắc cho những người không nhận được.
4. **Given** tài khoản không có địa chỉ email, **When** thao tác cấp gói hoàn tất, **Then** gói vẫn được cấp thành công và chỉ việc gửi email bị bỏ qua.
5. **Given** email không gửi được, **Then** gói đã cấp vẫn giữ nguyên, và sự cố gửi email phải để lại dấu vết để tìm kiểm tra được.

---

### User Story 3 - Hạn mức chiến dịch không bao giờ bị vượt (Priority: P1)

Tôi là người vận hành. Tôi khai báo tổng hạn mức 50 và hạn mức dự phòng 5 cho chiến dịch ra mắt, nên trần cứng là 55. Tôi cần một điều bảo đảm tuyệt đối: **không bao giờ có tài khoản thứ 56**. Tôi chấp nhận việc con số thực tế có thể là 50 hoặc 55, vì quan trọng hơn là nó **không bao giờ vượt trần tôi đã khai báo**. Điều này phải đúng kể cả khi nhiều phiên bản hệ thống chạy đồng thời, kể cả khi cùng một thông báo bị giao lại nhiều lần, và kể cả khi một phiên bản bị tắt giữa chừng rồi khởi động lại.

**Why this priority**: Hạn mức là cam kết với khách hàng và là điều kiện để chiến dịch đáng tin. Vượt trần là lỗi tệ nhất của một chiến dịch: nó tốn chi phí cấp gói thật và không thể thu hồi.

**Independent Test**: Chạy chiến dịch với trần nhỏ, cho nhiều tài khoản đăng ký cùng lúc từ nhiều phiên bản hệ thống, rồi đếm số tài khoản thực sự được cấp gói.

**Acceptance Scenarios**:

1. **Given** còn đúng 1 suất trong tổng hạn mức và có 20 thông báo tới cùng lúc trên 5 phiên bản hệ thống, **When** các thông báo được xử lý xong, **Then** đúng 1 tài khoản nhận gói, và 19 tài khoản còn lại vẫn ở gói miễn phí.
2. **Given** một tài khoản đã được cấp gói, **When** cùng thông báo đó tới lại lần thứ hai, **Then** tài khoản đó **không** bị cộng thêm thời hạn, và thời hạn vẫn là 1 tháng chứ không phải 2 tháng.
3. **Given** đã cấp đủ tổng hạn mức và còn hạn mức dự phòng, **When** có thông báo mới tới, **Then** tài khoản đó **vẫn được cấp gói** nếu thời điểm tạo của nó sớm hơn mốc ưu tiên, và không được cấp nếu tạo sau mốc ưu tiên.
4. **Given** đã tiêu cả tổng hạn mức lẫn hạn mức dự phòng, **When** có thông báo mới tới, **Then** không có tài khoản nào được cấp gói nữa, và hệ thống vẫn hoạt động bình thường.
5. **Given** đã tiêu hết trần, **Then** số suất đã dùng được bảo toàn sau khi khởi động lại hệ thống, và không có tài khoản nào được cấp lại lần hai.
6. **Given** một phiên bản bị tắt giữa lúc đang xử lý một thông báo, **When** phiên bản đó khởi động lại, **Then** tài khoản đó nhận gói **đúng một lần**, không phải hai lần và không phải zero lần.

---

### User Story 4 - Người dùng không bị mất lượt dùng AI vì thao tác chồng nhau (Priority: P2)

Tôi là người dùng thường xuyên. Tôi thỉnh thoảng bấm gợi ý phối đồ, hoặc mở ứng dụng trên hai thiết bị cùng lúc. Tôi mong mỗi lần bấm đều được tính vào hạn mức ngày của tôi, và tôi phải bị chặn đúng khi đã hết hạn mức. Tôi không mong nhận thêm lượt nào, và tôi cũng không mong phải chờ lâu hơn bình thường vì thao tác trước của tôi chưa xong.

**Why this priority**: Hạn mức ngày là thứ quyết định dịch vụ AI trả phí có được phép chạy hay không. Đếm thiếu là rò rỉ chi phí. Nó ảnh hưởng mọi tài khoản trên mọi gói, không riêng chiến dịch, nên càng nên sửa sớm.

**Independent Test**: Chạy nhiều thao tác dùng AI đồng thời cho cùng một tài khoản ở sát hạn mức, rồi đối chiếu số lượt hệ thống ghi nhận với số thao tác thực sự đã dùng. Không cần chạy chiến dịch nào.

**Acceptance Scenarios**:

1. **Given** tài khoản đã dùng 29 trên tổng 30 lượt trong ngày, **When** ba thao tác dùng AI cùng chạy một lúc, **Then** sau khi cả xong, bộ đếm ghi nhận đúng 30 lượt, không phải 29 và không phải 32.
2. **Given** tài khoản đã dùng hết hạn mức ngày, **When** thao tác dùng AI tiếp tục, **Then** thao tác đó bị từ chối với thông báo lỗi hợp lệ, không phải thao tác lọt qua rồi mới bị ghi nhận sai.
3. **Given** tài khoản bắt đầu dùng AI ở ngày mới, **When** lượt dùng được ghi nhận, **Then** số lượt bắt đầu lại từ 0 một cách tự động, không cần thao tác thủ công nào.
4. **Given** nhiều người dùng đồng thời, **When** họ dùng AI cùng lúc, **Then** việc đếm của mỗi người không ảnh hưởng tới việc đếm của người khác.
5. **Given** thao tác dùng AI thông thường, **When** nó hoàn tất, **Then** thời gian chờ của người dùng không tăng so với trước khi sửa.

---

### User Story 5 - Người vận hành mở được chiến dịch mới mà không phải sửa mã nguồn (Priority: P3)

Tôi là người vận hành. Sau khi buổi ra mắt kết thúc, tôi sẽ còn muốn chạy các chiến dịch khác: tặng gói cho 100 tài khoản đầu tiên trong một tuần, hoặc tặng một gói khác. Tôi mong chỉ cần khai báo mã chiến dịch, gói cước, hạn mức và thời gian chạy. Tôi **không** muốn sửa mã nguồn, **không** muốn viết thêm hàm, và **không** muốn triển khai lại hệ thống nếu có thể tránh.

**Why this priority**: Đây là yêu cầu bổ sung làm thay đổi bản chất công việc. Nếu không có câu chuyện này thì những gì xây ra chỉ là một chức năng một lần dùng, đúng thứ mà người dùng đã nói rõ là không muốn.

**Independent Test**: Cấu hình một chiến dịch thứ hai với hạn mức khác, gói khác và thời gian khác, rồi cho tài khoản mới đăng ký. Không sửa bất kỳ tệp mã nguồn nào.

**Acceptance Scenarios**:

1. **Given** chiến dịch ra mắt đã đủ hạn mức, **When** tôi khai báo một chiến dịch mới cùng dạng, **Then** chiến dịch mới chạy mà không cần sửa mã nguồn.
2. **Given** tôi không khai báo chiến dịch nào, **When** hệ thống chạy, **Then** hệ thống hoạt động bình thường, không cấp gói cho ai, không ghi cảnh báo lỗi, và mọi chức năng khác vẫn hoạt động.
3. **Given** một chiến dịch đã kết thúc, **When** tôi chạy lại chiến dịch đó, **Then** số suất đã cấp trước đó vẫn được ghi nhận, và không tài khoản nào bị cấp gói lần hai.
4. **Given** tôi khai báo sai — ví dụ trỏ tới một gói cước không tồn tại, hoặc hạn mức bằng 0 hoặc âm, **When** hệ thống khởi động, **Then** hệ thống **không khởi động được** và báo rõ đúng chỗ sai, thay vì khởi động rồi âm thầm không cấp gói cho ai.
5. **Given** một chiến dịch đã đặt thời gian kết thúc, **When** thời gian đó trôi qua, **Then** chiến dịch tự ngủ mà không cần thao tác tắt thủ công.

---

### User Story 6 - Tài khoản do quản trị viên tạo không bị tính vào chiến dịch (Priority: P3)

Tôi là quản trị viên. Tôi tạo tài khoản cho khách hàng thân thiết hoặc để kiểm thử. Những tài khoản đó không phải tài khoản tăng trưởng tự nhiên. Tôi mong chúng **không** tiêu tốn suất của chiến dịch, tôi vẫn nhận được gói miễn phí mặc định cho chúng như mọi tài khoản khác, và tôi vẫn cấp gói cho chúng bằng đường cấp gói thủ công sẵn có như trước.

**Why this priority**: Nếu không tách được, mỗi tài khoản kiểm thử sẽ âm thầm ăn mất một suất quà tặng của khách hàng thật. Đây là câu chuyện nhỏ nhưng rẻ nhất để viết và tốn kém nhất để sửa sau.

**Independent Test**: Tạo một tài khoản bằng đường quản trị viên, rồi kiểm tra ba thứ: số suất chiến dịch không bị giảm, tài khoản có gói miễn phí, và cấp gói thủ công vẫn chạy.

**Acceptance Scenarios**:

1. **Given** chiến dịch đang chạy và còn suất, **When** quản trị viên tạo một tài khoản mới, **Then** số suất chiến dịch không bị giảm, và tài khoản đó không nhận gói quà tặng.
2. **Given** quản trị viên vừa tạo tài khoản, **When** kiểm tra tài khoản đó, **Then** nó vẫn ở gói miễn phí như mọi tài khoản mới khác.
3. **Given** tài khoản do quản trị viên tạo, **When** quản trị viên cấp gói cho tài khoản đó bằng đường cấp thủ công, **Then** việc cấp gói thủ công vẫn hoạt động bình thường, không bị ảnh hưởng bởi chiến dịch.
4. **Given** quản trị viên vừa tạo tài khoản, **When** một tính năng khác đang nghe sự kiện "tài khoản được tạo" chạy vào, **Then** nó nhận được thông báo cho tài khoản đó, kèm dấu hiệu cho biết tài khoản do quản trị viên tạo, để tự quyết định có xử lý hay không.

---

### User Story 7 - Người vận hành trả lời được khách hàng đang hỏi (Priority: P2)

Tôi là quản trị viên, và có khách hàng nhắn hỏi "tôi có nằm trong số tài khoản được tặng không". Tôi cần mở một màn hình và thấy ngay: chiến dịch đang chạy, đã tặng được bao nhiêu trên tổng, còn bao nhiêu suất, và **riêng tài khoản của khách hàng đó** thì rơi vào trường hợp nào — đã được tặng, đủ điều kiện nhưng hết suất, hay không đủ điều kiện. Ba trường hợp đó cần trả lời khác nhau, vì cách tôi nói với khách hàng cũng khác nhau.

**Why this priority**: Không có màn hình này thì mỗi câu hỏi của khách hàng đều phải tra cứu thủ công, và khách hàng không có cách nào tự xác minh. Đây là hậu quả trực tiếp của việc chấp nhận cho phép vượt hạn mức một chút: phải trả lời được câu hỏi "vậy tôi có được không".

**Independent Test**: Mở chức năng tra cứu với một chiến dịch đã chạy, không cần tạo thêm tài khoản nào. Chỉ cần một chiến dịch có số liệu thật là kiểm chứng được ngay.

**Acceptance Scenarios**:

1. **Given** một chiến dịch đang chạy, **When** tôi mở danh sách chiến dịch, **Then** tôi thấy mã chiến dịch, gói được tặng, thời gian chạy, tổng hạn mức, hạn mức dự phòng, số lượt đã cấp, số suất còn lại, và trạng thái.
2. **Given** tôi mở danh sách tài khoản đã cấp của một chiến dịch, **When** danh sách có nhiều hơn một trang, **Then** tôi xem được từng trang, và có thể lọc theo khoảng thời điểm cấp.
3. **Given** tôi tra một tài khoản chưa từng đăng ký, **Then** hệ thống trả lời **không đủ điều kiện**, không phải danh sách rỗng không giải thích.
4. **Given** tôi tra một tài khoản đủ điều kiện nhưng chiến dịch đã hết suất, **Then** hệ thống trả lời **đủ điều kiện nhưng hết suất** — khác hẳn câu trả lời cho một tài khoản đã được tặng.
5. **Given** tôi tra một mã chiến dịch không tồn tại, **Then** hệ thống báo lỗi thân thiện nói mã đó không có, thay vì trả về lỗi kỹ thuật.
6. **Given** tôi dùng tài khoản người dùng thường, **When** tôi thử mở chức năng này bằng đường dẫn trực tiếp, **Then** bị từ chối, và không lộ ra danh sách tài khoản nào.
7. **Given** tôi chỉ xem, **When** tôi tra cứu nhiều lần, **Then** số lượt đã cấp và số suất còn lại **không thay đổi**, và không có email nào được gửi đi.

---

### Edge Cases

**Về sự kiện tạo tài khoản**
- Sự kiện "tài khoản được tạo" tới mà hệ thống chưa bật chiến dịch, rồi chiến dịch được bật ⇒ tài khoản đó không được hưởng, vì nó tạo trước thời điểm bắt đầu.
- Chiến dịch đang chạy nhưng tài khoản tạo trước lúc bật ⇒ không được hưởng, và **không** được tính vào suất nào.
- Sự kiện tới **hai lần** cho cùng một tài khoản (do cơ chế quét bù chạy song song) ⇒ chỉ cấp một lần.
- Tài khoản do quản trị viên tạo ⇒ vẫn có gói miễn phí, không tiêu suất chiến dịch, và lý do bỏ qua là "do quản trị viên tạo" chứ không phải "không đủ điều kiện".
- Sự kiện không đọc được nội dung, hoặc thiếu mã tài khoản ⇒ loại bỏ, không thử lại vô hạn; cơ chế quét bù vẫn đóng sót.
- Chiến dịch không khai báo gói cước hợp lệ ⇒ hệ thống **không khởi động được**, chứ không phải bỏ qua âm thầm.

**Về độ trễ và đóng sót**
- Hệ thống bị gián đoạn, tài khoản đủ điều kiện được tạo trong lúc gián đoạn ⇒ phải được cấp gói khi hệ thống phục hồi, nhờ cơ chế quét bù, không được mất quyền lợi im lặng.
- Cơ chế quét bù chạy song song với đường xử lý tin nhắn trên cùng một tài khoản ⇒ phải cho ra **một** lần cấp gói, không phải hai.
- Cơ chế quét bù chạy trên nhiều phiên bản hệ thống cùng lúc ⇒ không được vượt trần.
- Tài khoản được tạo sau thời điểm kết thúc chiến dịch, nhưng thông báo tới lúc chiến dịch còn đủ suất ⇒ **không** được cấp gói, vì thời điểm tạo đã nằm ngoài cửa sổ chiến dịch.
- Chiến dịch không khai báo thời điểm kết thúc, nhưng đã đạt trần ⇒ vẫn tự ngủ, không chờ thông báo nào nữa.
- Trễ quá 5 phút xảy ra do một nguyên nhân riêng biệt, không phải do mất tin nhắn ⇒ phải ghi nhận khác với trường hợp tin nhắn bị mất.

**Về tính đúng đắn của hạn mức**
- Hạn mức còn đúng 1 suất và có nhiều thông báo tới cùng lúc trên nhiều phiên bản hệ thống ⇒ phải có đúng 1 tài khoản được cấp.
- Cùng một thông báo tới lại lần thứ hai, lần thứ ba ⇒ tài khoản đó chỉ được cấp một lần, thời hạn vẫn là 1 tháng.
- Phiên bản hệ thống bị tắt giữa lúc đang cấp gói ⇒ thông báo phải được giao lại và cấp đúng một lần.
- Chiến dịch đã đầy hạn mức nhưng thông báo cũ vẫn nằm trong hàng đỗi ⇒ xử lý bình thường, không cấp, không báo lỗi.

**Về thời điểm**
- Hệ thống được triển khai **trước** thời điểm bắt đầu chiến dịch, và có tài khoản đăng ký trong khoảng giữa lúc chờ và lúc bắt đầu ⇒ tài khoản đó không được tính suất, và suất của nó không bị tiêu mất.
- Thông báo xử lý **chậm**, tới sau khi chiến dịch đã kết thúc theo thời gian khai báo ⇒ cần một quyết định rõ ràng về việc có cấp hay không.
- Thời gian khai báo không có múi giờ ⇒ cần một quy tắc thống nhất để mọi môi trường hiểu giờ theo cùng một cách.

**Về điều kiện đủ điều kiện**
- Tài khoản đã có gói Premium còn hạn nhận thông báo tới ⇒ không được cộng thêm thời hạn thành 2 tháng.
- Tài khoản vừa được hạ về gói miễn phí, thời hạn gói cũ chưa dọn xong ⇒ không được cấp chồng.
- Người dùng dùng ngay hết hạn mức AI gói miễn phí trong lúc chờ được cấp gói ⇒ sau khi được cấp gói, họ phải có đủ hạn mức gói mới, không bị chặn bởi lượt đã dùng của gói cũ.
- Ghi cấp gói bị giữa chừng do dữ liệu lỗi ⇒ phải để lại dấu vết đủ để tìm ra, không được im lặng.

**Về nhiều chiến dịch**
- Hai chiến dịch cùng có hiệu lực ⇒ **một tài khoản chỉ tham gia một chiến dịch**, và chiến dịch được chọn là chiến dịch **bắt đầu sau cùng** trong số những chiến dịch mà tài khoản đó đủ điều kiện. Việc chọn này phải cho **cùng một kết quả** bất kể thông báo tới trước hay sau.
- Chiến dịch thứ hai dùng gói cước khác, ví dụ một gói theo năm thay vì theo tháng ⇒ coi là kết quả mong muốn, không cần hỏi: thời hạn cấp phải lấy từ định nghĩa của gói cước, không viết cứng 30 ngày.
- Tài khoản đã tham gia chiến dịch A, sau đó có chiến dịch B bắt đầu muộn hơn và tài khoản đó vẫn đủ điều kiện cho B ⇒ tài khoản **không** được tham gia B, vì mỗi tài khoản chỉ có một lần tham gia trong đời qua lớp chiến dịch này.

**Về vận hành**
- Cần phân biệt chiến dịch tạm ngủ với chiến dịch đang chạy nhưng hư hỏng không cấp được gói ⇒ phải phân biệt được bằng dấu hiệu quan sát được, không phải đoán.
- Người vận hành cần xem được: chiến dịch nào đang chạy, đã cấp bao nhiêu trên tổng hạn mức, còn bao nhiêu suất, và những tài khoản nào đã được cấp.
- Có người hỏi "tôi có nằm trong số tài khoản đầu không" vào một thời điểm mà số liệu đang chạy ⇒ giao diện tra cứu phải trả về con số tại thời điểm hỏi, và phải phân biệt được "chưa đủ điều kiện" với "đủ điều kiện nhưng hết suất".
- Khai báo trùng mã chiến dịch, hoặc khai báo mã rỗng ⇒ phải báo lỗi khởi động, không được âm thầm bỏ qua.
- Người dùng khác không có quyền quản trị mở giao diện tra cứu ⇒ phải bị từ chối, không được rò rỉ danh sách tài khoản đã được cấp gói.

**Về "50 tài khoản đầu tiên" nghĩa là gì khi tin nhắn tới không đúng thứ tự**
- Phần mềm chấp nhận độ trễ, nên hai tài khoản đăng ký cách nhau 2 giây có thể được xử lý theo thứ tự ngược lại.
- Tài khoản đăng ký thứ 50 có thể tới xử lý sau tài khoản đăng ký thứ 55.
- Quyết định đã chốt: **quyền ưu tiên thuộc về tài khoản được tạo sớm hơn**, và hệ thống **được phép vượt hạn mức một chút** để bù cho tài khoản bị trượt vì tin nhắn tới muộn.
- Hệ quả bắt buộc phải nói rõ: "50" không còn là con số tuyệt đối nữa, nên cần một **hạn mức dự phòng** khai báo trước để trần vẫn là một con số tuyệt đối, và cần một **mốc thời gian** chốt lại tại thời điểm hết ngân sách chính để các lần bù về sau so sánh được với nhau, bất kể thứ tự xử lý.

## Requirements *(bắt buộc)*

### Phạm vi và nguyên tắc chung

- **FR-001**: Công việc này phải tạo ra một **cơ chế chiến dịch dùng lại được**, không phải một chức năng viết riêng cho con số 50 của ngày 02/10/2026.
- **FR-002**: Thêm một chiến dịch mới phải chỉ cần **khai báo mục tiêu**, không được yêu cầu sửa mã nguồn, viết thêm hàm, hay tạo bản ghi cấu hình riêng trong mã nguồn. Ràng buộc này áp dụng **từ phía bên nhận thông báo** chứ không chỉ phía cấu hình: mọi tính năng sau này dựa trên sự kiện tạo tài khoản — gửi email chào mừng, thương giới thiệu, đo tăng trưởng tín hiện tại — đều phải chạy được chỉ bằng cách khai báo chiến dịch.
- **FR-003**: Mã nguồn viết ra trong phạm vi công việc này **phải được giữ lại sau khi chiến dịch đầu tiên kết thúc**, và phải hoạt động đúng đối với các chiến dịch sau đó.
- **FR-004**: Hệ thống phải chạy đúng khi **không có chiến dịch nào được khai báo**: mọi chức năng hiện có vẫn hoạt động, không có lỗi, không có cảnh báo giả báo động.
- **FR-005**: Toàn bộ hành vi phụ thuộc vào môi trường phải được thể hiện bằng **cấu hình**, không được viết cứng trong mã nguồn. Việc chuyển đổi giữa môi trường chỉ được cần đổi cấu hình, không được cần đổi mã nguồn.

### Khai báo chiến dịch

- **FR-006**: Mỗi chiến dịch phải có một **mã định danh** do người vận hành đặt, mã này không được trùng với mã của chiến dịch khác.
- **FR-007**: Mỗi chiến dịch phải khai báo: mã định danh, mã gói cước được tặng, tổng hạn mức, hạn mức dự phòng, thời điểm bắt đầu, và thời điểm kết thúc.
- **FR-008**: Thời điểm bắt đầu và kết thúc phải được khai báo theo một **quy ước thời gian thống nhất** mà mọi môi trường đều hiểu giống nhau. Với chiến dịch ra mắt, mốc bắt đầu là **00:00 ngày 02/10/2026 theo giờ Việt Nam**.
- **FR-009**: Thời điểm kết thúc phải **không bắt buộc**. Chiến dịch khai báo không có thời điểm kết thúc thì tự ngủ khi đủ hạn mức.
- **FR-010**: Hệ thống phải **từ chối khởi động** khi có khai báo sai: mã rỗng, mã trùng, tổng hạn mức bằng 0 hoặc âm, hạn mức dự phòng âm, thời điểm kết thúc không sau thời điểm bắt đầu, hoặc mã gói cước không tồn tại. Thông báo lỗi phải chỉ đúng chi tiết sai.
- **FR-011**: Hệ thống phải chấp nhận **nhiều chiến dịch cùng tồn tại trong cấu hình**, và phải xác định chiến dịch áp dụng cho một tài khoản bằng một **quy tắc thuần túy, không phụ thuộc thứ tự xử lý**: trong số các chiến dịch mà tài khoản đó đủ điều kiện, lấy chiến dịch **bắt đầu muộn nhất**. Hai tài khoản có cùng thời điểm tạo phải được xếp vào cùng một chiến dịch, bất kể thông báo tới trước hay sau.
- **FR-012**: Mỗi tài khoản được tham gia **nhiều nhất một lần** trong toàn bộ cơ chế chiến dịch. Một tài khoản đã có lượt cấp thì mọi chiến dịch sau đó đều bỏ qua tài khoản đó, kể cả chiến dịch bắt đầu muộn hơn, kể cả khi khởi động lại hệ thống, kể cả khi chạy lại đúng chiến dịch đó.
- **FR-013**: Chiến dịch phải **tự ngủ** khi đã cấp đủ hạn mức, và phải **tự ngủ** khi đã qua thời điểm kết thúc nếu có khai báo. Không cần thao tác tắt thủ công.

### Điều kiện đủ điều kiện

- **FR-014**: Tài khoản đủ điều kiện là tài khoản **được tạo mới** qua một trong hai đường: đăng ký bằng email, hoặc đăng nhập Google bằng một địa chỉ chưa có tài khoản.
- **FR-015**: Tài khoản **không** đủ điều kiện nếu được tạo bởi quản trị viên, dù đi qua cùng cơ chế tạo tài khoản.
- **FR-016**: Tài khoản **không** đủ điều kiện nếu nó chỉ liên kết một định danh Google vào một tài khoản đã tồn tại, vì tài khoản đó không phải mới.
- **FR-017**: Tài khoản **không** đủ điều kiện nếu thời điểm nó được tạo nằm **trước** thời điểm bắt đầu của chiến dịch, hoặc **sau** thời điểm kết thúc nếu chiến dịch có khai báo thời điểm kết thúc.
- **FR-018**: Tài khoản đang ở gói cước đó và còn thời hạn **không** được cộng thêm thời hạn. Thời hạn của nó phải giữ nguyên.
- **FR-019**: Thời hạn được cấp phải lấy từ **định nghĩa của gói cước** được khai báo, không được viết cứng. Với gói `premium-monthly` hiện tại, thời hạn là 1 tháng.
- **FR-020**: Sau khi chiến dịch ngừng hoạt động, tài khoản đã được cấp gói quà tặng phải tự trở về gói miễn phí khi hết thời hạn, và **không** bị trừ tiền.

### Tính đúng đắn của hạn mức

Quyết định đã chốt là **ưu tiên tài khoản được tạo sớm hơn, đánh đổi bằng việc cho phép vượt hạn mức một chút**. Để trần vẫn là một con số tuyệt đối, hạn mức của chiến dịch gồm hai phần, và việc dùng phần dự phòng được điều kiện hóa bằng một mốc thời gian chốt lại.

- **FR-021**: Mỗi chiến dịch phải khai báo **tổng hạn mức** (phần chính) và **hạn mức dự phòng** (phần bù). Hạn mức dự phòng **không bắt buộc phải viết ra**; khi bỏ trống thì mặc định là 0, tức là trần cứng bằng đúng tổng hạn mức. Chiến dịch ra mắt khai báo tổng hạn mức 50 và hạn mức dự phòng 5, nên trần cứng là 55.
- **FR-022**: Hệ thống phải bảo đảm **tuyệt đối** rằng số tài khoản được cấp gói của một chiến dịch **không bao giờ vượt quá tổng hạn mức cộng hạn mức dự phòng**, trong mọi tình huống. Không có ngoại lệ và không có đường vòng nào vượt được con số này. Ràng buộc này phải đúng **cả khi nhiều phiên bản hệ thống chạy đồng thời** và cùng xử lý thông báo của một chiến dịch.
- **FR-023**: *(Đã gộp vào FR-022. Giữ số hiệu để các tham chiếu chéo cũ vẫn đúng; nội dung không còn tách riêng.)*
- **FR-024**: Việc kiểm tra hạn mức và hành động cấp gói phải xảy ra trong **cùng một giao dịch không thể tách**. Không được có khoảng thời gian nào mà hạn mức đã được kiểm tra nhưng lượt cấp chưa được ghi.
- **FR-025**: Thông báo bị giao lại phải **không** tạo ra lần cấp thứ hai cho cùng một tài khoản, và **không** cộng thêm thời hạn.
- **FR-026**: Khi phiên bản hệ thống bị tắt giữa lúc đang xử lý, thông báo phải được giao lại và xử lý lại được mà kết quả vẫn là **đúng một lần cấp gói**.
- **FR-027**: Khi hạn mức đã đầy, hệ thống phải bỏ qua thông báo một cách **bình thường**, không được coi là lỗi, không được làm gián đoạn.
- **FR-028**: Khi dữ liệu lượt cấp đã ghi phải được **giữ nguyên** sau khi khởi động lại, để chiến dịch không bao giờ cấp vượt trần trong các lần chạy sau.

**Quy tắc ưu tiên theo thời điểm tạo tài khoản**

- **FR-029**: Trong khi số lượt đã cấp **nhỏ hơn tổng hạn mức**, mọi tài khoản đủ điều kiện đều được cấp, không xét thêm điều kiện thứ tự nào.
- **FR-030**: Tại thời điểm lượt thứ `tổng hạn mức` được ghi, hệ thống phải **chốt lại mốc ưu tiên** là thời điểm tạo của chính tài khoản đó, và lưu lại mốc đó. Mốc ưu tiên phải mang tính bền vững và **không đổi** theo thứ tự xử lý sau đó.
- **FR-031**: Sau khi hết phần chính, một tài khoản **chỉ** được cấp từ phần dự phòng khi thời điểm tạo của nó **sớm hơn mốc ưu tiên** đã chốt. Tài khoản tạo sau mốc ưu tiên thì không được cấp, kể cả khi phần dự phòng còn suất.
- **FR-032**: Phần dự phòng là **trần**, không phải mục tiêu. Nếu hết chiến dịch mà phần dự phòng vẫn chưa dùng hết thì số lượt thực tế có thể **ít hơn** tổng hạn mức cộng hạn mức dự phòng. Đây là hành vi đúng, không phải lỗi.
- **FR-033**: Một tài khoản bị từ chối vì đã tạo sau mốc ưu tiên phải được ghi nhận với lý do **"đủ điều kiện nhưng hết suất"**, tách bạch với lý do **"không đủ điều kiện"**. Hai trường hợp này phải trả lời khác nhau khi người vận hành tra cứu.

**Giới hạn đã biết và đã chấp nhận của cơ chế này**

Cơ chế cấp gói dựa trên **thứ tự tin nhắn tới**, nên nó **không** bảo đảm được "đúng 50 tài khoản được tạo sớm nhất" khi tin nhắn tới không đúng thứ tự. Cụ thể với `tổng hạn mức = 50`, `hạn mức dự phòng = 5`, 55 tài khoản đăng ký liên tiếp:

| Thứ tự tin nhắn tới | 50 lượt cấp đầu | Mốc ưu tiên | Lượt bù từ dự phòng | Tổng | Tập tài khoản được cấp |
|---|---|---|---|---|---|
| Đúng thứ tự đăng ký | tài khoản 1 tới 50 | tài khoản 50 | không có | **50** | 1 tới 50 (đúng 50 người sớm nhất) |
| Ngược thứ tự | tài khoản 55 tới 6 | tài khoản 6 | tài khoản 5, 4, 3, 2, 1 | **55** | 1 tới 55 — **toàn bộ 55 tài khoản**, không phải 50 người sớm nhất |

Hai dòng này **khác nhau**, và đó là hệ quả trực tiếp của việc cơ chế dựa trên thứ tự tin nhắn tới.

**FR-031 chỉ ràng buộc phần dự phòng, không ràng buộc phần chính.** Đây là điểm dễ hiểu sai nhất
của toàn bộ cơ chế:

- Trong 50 lượt cấp đầu, hệ thống **không** so sánh thời điểm tạo với mốc ưu tiên — mốc ưu tiên tới
  đúng lúc lượt thứ 50 được ghi, tức là **sau khi** cả 50 lượt đó đã quyết định. Ở thứ tự ngược, các
  tài khoản 7 tới 55 đều tạo **sau** mốc (tài khoản 6) mà vẫn được cấp gói từ phần chính. Đó là hành
  vi đúng, không phải lỗi.
- Chỉ từ lượt thứ 51 trở đi, phần dự phòng mới áp điều kiện `created_at < watermark`. Ở thứ tự
  ngược, 5 tài khoản đó là 5, 4, 3, 2, 1 — đều tạo **trước** mốc — nên cả 5 đều được cấp.
- Ở thứ tự thuận, 5 tài khoản còn lại là 51 tới 55 — đều tạo **sau** mốc (tài khoản 50) — nên
  không tài khoản nào được cấp từ dự phòng.

Vì thế câu hỏi "có tài khoản nào tạo sau mốc ưu tiên mà vẫn được cấp không" **không có câu trả lời
duy nhất** — nó phụ thuộc vào việc lượt cấp đó lấy từ phần chính hay phần dự phòng, và câu trả lời
đúng phải kiểm tra theo `grant_bucket` chứ không kiểm tra trên toàn bộ lượt cấp. Cơ chế **không**
bảo đảm *danh tính* của 50 người được chọn khi thứ tự tới bị đảo. Muốn bảo đảm điều đó thì phải
chốt danh sách ứng viên trước rồi mới chọn, tức phải chờ đóng cửa sổ — đánh đổi bằng việc tài
khoản đăng ký sớm không có gói cho tới lúc đó. Quyết định là **không chờ**, xem mục Clarifications
phiên 2026-10-01.

### Trao quyền lợi

- **FR-034**: Tài khoản đủ điều kiện phải ở gói cước được tặng, với thời hạn **đủ dài** — ít nhất là toàn bộ thời hạn của gói tính từ thời điểm cấp. *(Hiệu chỉnh 2026-10-01, sau khi kiểm thử end-to-end: bản gốc ghi "tính từ thời điểm tài khoản được tạo", và điều đó không thực thi được.)* Lý do: dịch vụ AI giữ một ràng buộc đơn điệu trên `effective_from` để chặn sự kiện giao lại làm tua quyền của tài khoản. Nếu đóng dấu sự kiện cấp gói bằng thời điểm tạo tài khoản, nó sẽ **cũ hơn** sự kiện cấp gói miễn phí phát ra micro-giây sau đó trong cùng giao dịch đăng ký, bị AI từ chối với lý do `stale_event`, và bị đẩy vào hàng đỗi chết — tài khoản có gói Premium nhưng vẫn giới hạn AI mức miễn phí, không phục hồi được. Sự kiện vì thế mang **thời điểm xử lý**; thời hạn tính từ thời điểm cấp cộng `duration_days` của gói (FR-019). Với tài khoản nhận cấp ngay, hai cách tính trùng nhau đến từng giây; với tài khoản phải cứu bằng quét bù, cách này cho thêm (không bắt), đúng hướng cho một chương trình khuyến mãi.
- **FR-035**: Cờ tự động gia hạn của tài khoản được cấp gói phải ở trạng thái **tắt**. Hệ thống không được tự bật tự động gia hạn cho bất kỳ tài khoản nào chưa chủ động yêu cầu.
- **FR-036**: Cấp gói phải ghi lại lịch sử thay đổi gói theo đúng cùng một quy ước mà mọi đường thay đổi gói khác đang dùng, để dữ liệu hạn mức của dịch vụ AI cập nhật theo. Chi tiết ràng buộc ở FR-092 tới FR-095.
- **FR-037**: Dữ liệu quyền tính phí AI của tài khoản được cấp gói phải được cập nhật sang chính sách của gói mới, với **thời hạn kết thúc khớp** với thời hạn gói cước.
- **FR-038**: Mọi thay đổi trạng thái gói phải được ghi vào nhật ký kiểm toán, ghi rõ nguồn gốc là chiến dịch và mã chiến dịch, và không ghi một quản trị viên cụ thể làm người thực hiện.
- **FR-039**: Nếu có bất kỳ bước nào trong quá trình cấp gói thất bại thì **toàn bộ** thay đổi phải được hoàn tác, để không tồn tại trạng thái nửa vời như: đã tiêu suất nhưng chưa có gói, hoặc đã có gói nhưng chưa ghi lịch sử.
- **FR-040**: Nếu thành công, thông tin gói mà tài khoản nhận trên màn hình và qua giao diện phải phản ánh đúng gói vừa được cấp, không bị dữ liệu cũ che che.

### Quy ước ghi lịch sử đổi gói và phân biệt nguồn gốc

Đã kiểm chứng trên mã nguồn hiện có. Mục này ghi lại các ràng buộc đã kiểm tra để không phải khám phá lại khi lập kế hoạch.

- **FR-092**: Công việc này **không được** thêm loại thay đổi gói mới vào tập loại dùng chung giữa hai dịch vụ. Tập loại là **đóng**, và bên nhận **từ chối vĩnh viễn** mọi loại không nằm trong tập, đưa thông báo vào hàng đỗi chết. Thêm loại mới là thay đổi hợp đồng sự kiện vượt biên và tạo ra một điểm hỏng mới: nếu bên nhận chưa kịp triển khai, mọi lần cấp gói của chiến dịch sẽ mất sạch.
- **FR-093**: Lần cấp gói của chiến dịch phải ghi lịch sử với **cùng loại sự kiện** mà một lần cấp gói quản trị viên ghi. Không được dùng loại sự kiện để phân biệt hai đường này.
- **FR-094**: Nguồn gốc "được cấp bởi chiến dịch" phải chỉ được ghi ở **nhật ký kiểm toán**, theo FR-038. Người vận hành tra cứu nguồn gốc của một lần cấp gói phải dùng nhật ký kiểm toán, **không** dựa vào lịch sử thay đổi gói.
- **FR-095**: Việc kiểm thử phải chứng minh rằng sự kiện thay đổi gói được phát trên **ít nhất** sáu đường: cấp phát gói mặc định lúc tạo tài khoản, mua gói, cấp gói quản trị viên, gia hạn, hạ về gói miễn phí khi hết hạn, và cấp gói của chiến dịch. Bỏ sót đường nào nghĩa là bên nhận bị bỏ hở trên đường đó, và việc thiếu đó **không** báo lỗi ở bất kỳ đâu.

### Không làm gián đoạn việc đăng ký

- **FR-041**: Yêu cầu tạo tài khoản phải hoàn tất thành công **không phụ thuộc** vào việc chiến dịch có xử lý xong hay không.
- **FR-042**: Nếu việc ghi sự kiện `identity.user.created` vào hàng đỗi không thành công, yêu cầu đăng ký phải **thất bại** thay vì tạo ra một tài khoản mà không ai biết là chưa từng được thông báo. Đây là đánh đổi có chủ ý: ưu tiên không mất thông báo hơn là đăng ký thành công. Hành vi này **không phải hành vi mới**: luồng đăng ký hiện đã phụ thuộc vào việc ghi hàng đỗi thành công, qua đường cấp phát gói mặc định.
- **FR-043**: Thời gian hoàn tất của yêu cầu tạo tài khoản khi có chiến dịch chạy phải **không chậm hơn đáng kể** so với khi không có chiến dịch.

### Độ trễ chấp nhận được và đóng sót

Quyết định đã chốt là **hai tầng**: một cam kết thời gian cho đường đi bình thường, và một cơ chế quét định kỳ để đóng trường hợp sót do sự cố. Xem mục Clarifications phiên 2026-10-01.

- **FR-078**: Hệ thống phải cam kết cấp gói cho tài khoản đủ điều kiện trong vòng **5 phút** kể từ thời điểm tạo tài khoản, tính trên đường xử lý bình thường. Đây là con số dùng để trả lời người dùng và bộ phận hỗ trợ.
- **FR-079**: Hệ thống phải có **cơ chế quét bù chạy định kỳ** (mặc định mỗi 15 phút) để phát hiện tài khoản đủ điều kiện mà lượt cấp chưa được ghi, thay vì chỉ tin vào một lần xử lý duy nhất của tin nhắn.
- **FR-080**: Cơ chế quét bù phải cho **cùng một kết quả** với đường xử lý tin nhắn: cùng một tài khoản, dù tới qua tin nhắn hay qua lần quét bù, chỉ được cấp đúng một lần và phải tuân theo cùng bộ luật hạn mức, mốc ưu tiên và điều kiện đủ điều kiện.
- **FR-081**: Cơ chế quét bù phải dùng **chung khoá và chung bộ luật hạn mức** với đường xử lý tin nhắn, để hai đường chạy song song trên nhiều phiên bản hệ thống không thể vượt trần.
- **FR-082**: Trường hợp trễ quá 5 phút phải được ghi nhận như **sự kiện quan sát được**, kèm số tài khoản bị trễ và độ trễ lớn nhất, để phân biệt được "hệ thống chậm" với "hệ thống hỏng" theo FR-059.
- **FR-083**: Thời gian chạy của cơ chế quét bù phải **khai báo được**, không viết cứng, để môi trường khác nhau có thể đặt khác nhau.

### Sự kiện tạo tài khoản

Quyết định đã chốt là **hướng publisher**: luồng tạo tài khoản phát một sự kiện về *chính nó*, không phải một tín hiệu cho chiến dịch. Sự kiện có tên `identity.user.created`.

| | Nếu phát theo hướng chiến dịch | Nếu phát theo hướng publisher (**đã chọn**) |
|---|---|---|
| Sự kiện mang nghĩa gì | "tài khoản này có thể dính chiến dịch" | "một tài khoản đã được tạo" |
| Luồng tạo tài khoản có biết chiến dịch tồn tại không | Có, phải đọc cấu hình để quyết định có phát không | **Không** |
| Mỗi tính năng sau cần gì | Một topic riêng **và** sửa lại luồng tạo tài khoản | Chỉ viết thêm bộ nhận thông báo |

- **FR-084**: Luồng tạo tài khoản phải phát **đúng một** sự kiện `identity.user.created`, trong **cùng giao dịch** với việc ghi bản ghi tài khoản, và phải phát **vô điều kiện** — không phụ thuộc có chiến dịch nào đang chạy hay không.
- **FR-085**: Luồng tạo tài khoản **không được** phụ thuộc vào cấu hình chiến dịch, và **không được** biết sự tồn tại của chiến dịch. Đây là điều kiện cấm, không phải khuyến nghị.
- **FR-086**: Sự kiện phải được phát từ **cả ba** đường tạo tài khoản: đăng ký bằng email, đăng nhập Google tạo mới, và tạo bởi quản trị viên. Sự kiện phải mang **cờ cho biết tài khoản có do quản trị viên tạo hay không**.
- **FR-087**: Sự kiện `identity.user.created` **không được** chứa dữ liệu nhận diện cá nhân — không địa chỉ email, không tên, không số điện thoại. Chỉ được chứa: mã tài khoản, nguồn đăng ký, thời điểm tạo tài khoản, và cờ quản trị viên. Mọi dữ liệu cần thêm phải được bên nhận thông báo đọc từ cơ sở dữ liệu.
- **FR-088**: Sự kiện phải đi qua **cùng đường ghi hàng đỗi sẵn có** với mọi sự kiện khác, và phải mang mã liên lạc theo FR-064, để lỗi hàng đỗi và độ trễ đều quan sát được theo cùng một cách.
- **FR-089**: Sự kiện là **đường tắt** để cấp gói nhanh, **không** phải cơ chế bảo đảm tính đúng đắn. Tính đúng đắn do cơ chế quét bù ở FR-079 đảm bảo, và trần độ trễ do FR-078 quy định. Sự kiện bị thất lạc **không được** dẫn tới mất quyền lợi cho bất kỳ tài khoản đủ điều kiện nào.
- **FR-090**: *(Đã gộp vào FR-002. Giữ số hiệu để các tham chiếu chéo cũ vẫn đúng; nội dung không còn tách riêng.)*
- **FR-091**: *(Đã gộp vào FR-050. Giữ số hiệu để các tham chiếu chéo cũ vẫn đúng; nội dung không còn tách riêng.)*
- **FR-096**: Lúc tạo tài khoản phải phát **cả hai** chủ đề: `identity.user.created` và `subscription.plan.changed`. Không được bỏ chủ đề nào.
- **FR-097**: Công việc này **không được** thay đổi cách dịch vụ AI nhận biết tài khoản mới. Sự kiện `subscription.plan.changed` phát lúc cấp phát gói mặc định là tín hiệu **duy nhất**, và nó phải là con đường duy nhất tạo bản ghi quyền chi phí cho tài khoản mới. Không cho dịch vụ AI nghe thêm chủ đề mới, và không để dịch vụ AI tra cứu gói cước.
- **FR-098**: Module tạo tài khoản **không được** tra cứu gói cước hay chính sách chi phí để làm phong bị thông báo `identity.user.created`. Nội dung thông báo giữ nguyên như FR-087. Lý do: mỗi chủ đề mang thông tin không chồng lấn từ một chủ sở hữu dữ liệu khác nhau — chủ đề tài khoản mang mã tài khoản, thời điểm tạo và cờ quản trị viên; chủ đề đổi gói mang mã gói và mã chính sách chi phí. Gộp chúng nghĩa là một trong hai module phải biết về việc của module kia.

**Ba phương án đã cân nhắc và loại bỏ** — ghi lại để không tranh luận lại ở giai đoạn kế hoạch:

| Phương án | Vì sao loại |
|---|---|
| Cho dịch vụ AI nghe **cả hai** chủ đề, đồng thời bỏ lần phát `subscription.plan.changed` lúc tạo | Lúc tạo tài khoản vẫn phát **hai** chủ đề, nên không tiết kiệm gì; và nếu giữ cả hai lần phát thì consumer nhận hai thông báo cho cùng một tài khoản, buộc phải tự chống trùng |
| Gộp còn **một** chủ đề, cho dịch vụ AI dựng quyền gói miễn phí từ thông báo tài khoản | Buộc module tạo tài khoản phải tra cứu gói mặc định để lấy mã gói và mã chính sách — vi phạm tinh thần FR-085. Thêm nữa, nếu dịch vụ AI chưa kịp có bộ nhận thông báo thì **mọi** tài khoản mới đều không có bản ghi quyền và rơi vào đường dự phòng. Rủi ro này trúng đúng vào thời điểm ra mắt. |
| Suy ra cờ quản trị viên từ dữ liệu thay vì lấy từ đường gọi | **Không khả thi.** Bảng tài khoản không có trường nào ghi ai đã tạo; hàng do quản trị viên tạo và hàng tự đăng ký **giống hệt nhau về cấu trúc**. Cờ này chỉ lấy được từ đường gọi, tức chỉ người phát ở module tài khoản mới biết. |

### Thông báo cho người dùng

- **FR-044**: Tài khoản được cấp gói phải nhận được **một** email thông báo, nêu tên gói và ngày hết hạn.
- **FR-045**: Email phải nói rõ gói đó **không tự động gia hạn**, và **không** được gợi ý rằng người dùng đã thanh toán.
- **FR-046**: Email **không** được nhắc đến tổng hạn mức của chiến dịch, vì thông tin đó không liên quan tới người nhận.
- **FR-047**: Tài khoản không có địa chỉ email thì vẫn được cấp gói; chỉ bước gửi email bị bỏ qua.
- **FR-048**: Email **không bao giờ** được gửi cho tài khoản không được cấp gói.
- **FR-049**: Việc gửi email thất bại **không được** làm mất gói đã cấp, nhưng phải để lại dấu vết đủ để tìm kiểm tra.

### Loại trừ tài khoản của quản trị viên

- **FR-050**: Tạo tài khoản qua đường quản trị viên không được tiêu tốn suất của bất kỳ chiến dịch nào, và tài khoản đó không được nhận gói quà tặng. Phía bên chiến dịch phải bỏ qua tài khoản này với **lý do ghi nhận riêng**, tách bạch với lý do "không đủ điều kiện" và với lý do "đủ điều kiện nhưng hết suất" (nội dung gộp từ FR-091).
- **FR-051**: Tài khoản do quản trị viên tạo phải **vẫn** được cấp phát gói miễn phí mặc định như mọi tài khoản khác, và việc cấp gói thủ công cho tài khoản đó bằng đường quản trị phải **không bị ảnh hưởng** bởi sự tồn tại của cơ chế chiến dịch.

### Sửa bộ đếm hạn mức dùng AI hằng ngày

- **FR-052**: Việc ghi nhận lượt dùng AI phải được thực hiện theo cách **chống được việc hai thao tác cùng lúc ghi đè lên nhau**, kể cả khi nhiều phiên bản hệ thống đang phục vụ.
- **FR-053**: Khi số lượt đã dùng bằng 29 trên tổng 30, và ba thao tác dùng AI cùng hoàn tất, bộ đếm cuối cùng phải là **30**, không phải 29 và không phải 32.
- **FR-054**: Bộ đếm phải tự về 0 khi sang ngày mới mà **không cần** thao tác thủ công nào.
- **FR-055**: Việc ghi bộ đếm không được **khóa** một tài khoản lâu hơn mức cần thiết. Thao tác chỉ đọc bộ đếm phải tiếp tục hoạt động khi thao tác khác đang ghi.
- **FR-056**: Việc sửa bộ đếm **không được** làm thay đổi số hạn mức mà bất kỳ gói cước nào đang quy định.
- **FR-057**: Tài khoản đã dùng hết hạn mức ngày phải bị từ chối **trước khi** hệ thống trả tiền cho nhà cung cấp AI.
- **FR-058**: Công việc sửa bộ đếm phải **không làm thay đổi** hợp đồng giao tiếp với hệ thống khác. Các hệ thống gọi tới chức năng đọc và ghi bộ đếm phải tiếp tục dùng được như cũ.

### Quan sát được

- **FR-059**: Người vận hành phải phân biệt được ba trạng thái: chiến dịch đang chạy, chiến dịch tạm ngủ vì đủ hạn mức, và chiến dịch **đang chạy nhưng không cấp được gói vì hỏng**. Trạng thái thứ ba không được hiện ra giống hai trạng thái kia.
- **FR-060**: Mỗi lần cấp gói thành công phải để lại dấu vết cho biết: mã chiến dịch, tài khoản, gói được cấp, thời hạn, thời điểm cấp.
- **FR-061**: Mỗi lần bỏ qua phải để lại dấu vết cho biết **lý do** bỏ qua: tài khoản do quản trị viên tạo, chiến dịch tạm ngủ, hết phần chính và tạo sau mốc ưu tiên, hết cả hạn mức dự phòng, thông báo trùng, ngoài thời gian, tài khoản không đủ điều kiện, tài khoản đã tham gia chiến dịch trước, hay lỗi tạm thời.
- **FR-062**: Hệ thống phải phân biệt được lỗi **tạm thời** với lỗi **không bao giờ tự khỏi**. Lỗi tạm thời phải được thử lại, lỗi không tự khỏi phải bị loại bỏ để không làm nghẽn hàng đợi.
- **FR-063**: Thông báo không đọc được nội dung phải bị loại bỏ khỏi hàng đỗi thay vì được thử lại vô hạn.
- **FR-064**: Mọi thông báo phải mang theo một mã liên lạc xuyên suốt để điều tra được từ lúc phát sinh tới lúc xử lý xong.

### Giao diện tra cứu cho người vận hành

- **FR-065**: Hệ thống phải cung cấp chức năng tra cứu **chỉ đọc** cho quản trị viên về chiến dịch, gồm: mã chiến dịch, mã gói cước được tặng, thời điểm bắt đầu, thời điểm kết thúc, tổng hạn mức, hạn mức dự phòng, số lượt đã cấp, số suất còn lại, mốc ưu tiên đã chốt hay chưa, và trạng thái hiện tại.
- **FR-066**: Hệ thống phải cung cấp chức năng tra cứu **chỉ đọc** danh sách tài khoản đã được cấp gói theo một mã chiến dịch, gồm định danh tài khoản, thời điểm cấp, thời hạn được cấp, và gói được cấp.
- **FR-067**: Danh sách tài khoản đã cấp phải được **phân trang**, và phải chấp nhận lọc theo khoảng thời điểm cấp.
- **FR-068**: Chức năng tra cứu phải trả về con số tại **thời điểm được hỏi**, và phải cho phép phân biệt ba trạng thái của một tài khoản: đã được cấp, **đủ điều kiện nhưng hết suất**, và không đủ điều kiện. Nếu không phân biệt được ba trạng thái thì không đáp ứng được câu hỏi của khách hàng.
- **FR-069**: Chức năng tra cứu **chỉ đọc** và **không được làm thay đổi** bất kỳ dữ liệu nào. Nó không được tiêu tốn suất, không được cấp gói, và không được gửi email.
- **FR-070**: Chức năng tra cứu phải **chỉ dành cho quản trị viên**. Tài khoản người dùng thường gọi vào phải bị từ chối, kể cả khi biết đường dẫn.
- **FR-071**: Chức năng tra cứu phải chịu cùng một giới hạn tần suất với các chức năng quản trị khác đang có, để một vòng dò không làm tê liệt hệ thống.
- **FR-072**: Chức năng tra cứu phải trả về lỗi thân thiện khi mã chiến dịch không tồn tại, thay vì trả về lỗi kỹ thuật hoặc danh sách rỗng không giải thích.

### Ngoài phạm vi

- **FR-073**: Công việc này **không** thay đổi giá, mức quyền lợi, hay định nghĩa của bất kỳ gói cước nào.
- **FR-074**: Công việc này **không** cho phép thay đổi hạn mức, hạn mức dự phòng, hay thời gian chiến dịch từ giao diện tra cứu. Việc đó chỉ thực hiện được bằng cách khai báo cấu hình, theo FR-021.
- **FR-075**: Công việc này **không** thay đổi cách dịch vụ AI tính phí và kiểm soát chi phí, ngoài phạm vi FR-052 tới FR-058. Trong đó, vấn đề số lượt dùng AI có trả phí bị đếm theo tài khoản thay vì theo gói đang hiệu lực được **ghi nhận là vấn đề đã biết và không sửa trong lần này**.
- **FR-076**: Công việc này **không** cung cấp giao diện tra cứu cho chính khách hàng; việc trả lời khách hàng "tôi có được tặng không" thuộc trách nhiệm bộ phận hỗ trợ, dựa trên chức năng tra cứu dành cho quản trị viên.
- **FR-077**: Công việc này **không** hỗ trợ điều kiện lọc theo nguồn đăng ký trong lần đầu. Cơ chế được thiết kế để bổ sung điều kiện lọc về sau bằng cách khai báo, không cần sửa mã nguồn.

### Key Entities *(nếu đặc tả có liên quan tới dữ liệu)*

- **Chiến dịch (Campaign)**: Một chương trình tặng quyền lợi cho nhóm tài khoản đáp ứng điều kiện trong một khoảng thời gian. Thuộc tính: mã định danh, mã gói cước được tặng, tổng hạn mức, thời điểm bắt đầu, thời điểm kết thúc. Không lưu thành một bảng riêng trong phạm vi lần đầu — thuộc tính nằm trong cấu hình.
- **Lượt cấp (Campaign claim)**: Một bản ghi ghi nhận "tài khoản này đã tiêu một suất của chiến dịch này". Thuộc tính: mã chiến dịch, tài khoản, gói cước được cấp, mã gói, thời điểm cấp, thời điểm tạo tài khoản, cờ cho biết lượt cấp đó có lấp đầy phần hạn mức chính hay không. Ràng buộc quan trọng: **mỗi tài khoản chỉ có tối đa một lượt cấp trong toàn bộ cơ chế**. Chính ràng buộc này vừa chống cấp trùng vừa là cách đếm suất đã dùng.
- **Mốc ưu tiên (Priority watermark)**: Thời điểm tạo của tài khoản đã lấp đầy phần hạn mức chính. Được lưu **bên trong dữ liệu lượt cấp**, trên chính bản ghi lượt cấp đó, và phải là **duy nhất** trong phạm vi chiến dịch. Sau khi chốt thì không đổi, để mọi lần so sánh về sau cho cùng một kết quả bất kể thứ tự xử lý.
- **Tài khoản (Account)**: Người dùng của hệ thống. Thuộc tính liên quan: thời điểm tạo, nguồn đăng ký, ai là người tạo.
- **Gói cước (Subscription plan)**: Định nghĩa gói đang tồn tại, gồm tên hiển thị, thời hạn, giá, các hạn mức quyền lợi. Chiến dịch **tham chiếu** gói cước, không sao chép nó.
- **Đăng ký gói của tài khoản (User subscription)**: Gói hiện tại của tài khoản, gồm gói đang dùng, thời hạn, cờ tự động gia hạn, ảnh chụp quyền lợi tại thời điểm thay đổi.
- **Bản ghi lịch sử thay đổi gói (Subscription event)**: Ghi lại mỗi lần gói của tài khoản thay đổi. Có khóa chống trùng theo tài khoản và số phiên bản. Là nguồn phát tín hiệu cho dịch vụ AI.
- **Bộ đếm hạn mức ngày (Daily usage counter)**: Số lượt gợi ý phối đồ và số lượt trò chuyện đã dùng trong ngày của tài khoản, cùng ngày đã về 0 lần gần nhất.
- **Quyền tính phí AI (AI cost policy grant)**: Bản ghi quyền chạy tính phí AI của tài khoản ở dịch vụ AI, gồm chính sách áp dụng, mức quyền, khoảng thời gian có hiệu lực. Được cập nhật từ lịch sử thay đổi gói.

## Success Criteria *(bắt buộc)*

### Measurable Outcomes

- **SC-001**: Với chiến dịch ra mắt khai báo tổng hạn mức 50 và hạn mức dự phòng 5, sau khi xử lý xong toàn bộ thông báo, **số tài khoản thực sự ở gói Premium do chiến dịch cấp nằm trong khoảng từ 50 đến 55**, và **không bao giờ vượt 55** trên mọi cấu hình phân bố tải đã thử.
- **SC-001b**: Với chiến dịch ra mắt, **không lượt cấp nào lấy từ phần dự phòng mà có thời điểm tạo sau mốc ưu tiên**. Đây là điều kiện công bằng **bảo đảm được**. Điều kiện này **giới hạn cho phần dự phòng** — trong 50 lượt cấp phần chính có những tài khoản tạo sau mốc ưu tiên, vì mốc chỉ được chốt tại lượt thứ 50. Vì vậy phải kiểm tra theo `grant_bucket`, không áp lên toàn bộ lượt cấp. Nó **không** phải điều kiện "đúng 50 tài khoản được tạo sớm nhất", vì điều đó **không thể bảo đảm** khi tin nhắn tới không đúng thứ tự. Xem mục Clarifications phiên 2026-10-01 và mục "Giới hạn đã biết".
- **SC-001d**: Khi tin nhắn tới **đúng thứ tự thời điểm tạo tài khoản**, kết quả phải là đúng 50 tài khoản được tạo sớm nhất, và hạn mức dự phòng không được dùng. Khi tin nhắn tới **ngược thứ tự**, kết quả có thể là 55, và tập tài khoản được chọn sẽ **không** phải 50 người sớm nhất. **Cả hai đều là hành vi đúng**, không phải lỗi.
- **SC-001c**: Khi phần dự phòng không dùng hết, số lượt thực tế **ít hơn** 55, và hệ thống báo trạng thái "tạm ngủ" chứ không báo lỗi.
- **SC-002**: Khi 100 thông báo của cùng một tài khoản được giao lại nhiều lần, tài khoản đó vẫn chỉ được cấp gói **đúng một lần** và thời hạn vẫn là 1 tháng.
- **SC-003**: 100% thông báo đã gửi cho chiến dịch đều có kết quả xử lý rõ ràng: hoặc được cấp gói, hoặc bị bỏ qua kèm lý do. Không thông báo nào bị mất không dấu vết.
- **SC-004**: Yêu cầu tạo tài khoản khi có chiến dịch chạy hoàn tất trong thời gian **không chậm hơn 10%** so với khi không có chiến dịch.
- **SC-005**: 100% tài khoản được cấp gói nhận được email trong vòng **60 giây** kể từ lúc được cấp gói, và 0 email nào được gửi cho tài khoản không được cấp gói.
- **SC-006**: 100% tài khoản được cấp gói có cờ tự động gia hạn ở trạng thái tắt. Trong thử nghiệm giả lập ngày hết hạn, **0 đồng** bị trừ.
- **SC-007**: Khi hết thời hạn gói quà tặng, tài khoản tự trở về gói miễn phí **mà không cần thao tác thủ công**.
- **SC-008**: Khi tài khoản được cấp gói, thông tin hạn mức dùng AI của nó khớp với định nghĩa của gói mới, và mức quyền đó tự hết hiệu lực đúng khi gói hết hạn.
- **SC-009**: Với 1000 thao tác dùng AI đồng thời trên các tài khoản đang sát hạn mức, tổng số lượt được ghi nhận **không nhỏ hơn** số thao tác thực sự đã dùng.
- **SC-010**: Thêm một chiến dịch mới chỉ cần **thêm cấu hình, không sửa tệp mã nguồn nào** và không cần tạo bảng mới.
- **SC-011**: Một người vận hành trả lời được "chiến dịch này đã cấp bao nhiêu trên tổng hạn mức, còn bao nhiêu suất" và "tài khoản X đã được tặng, đủ điều kiện mà hết suất, hay không đủ điều kiện" trong thời gian **dưới 1 phút**, bằng chức năng tra cứu dành cho quản trị viên.
- **SC-012**: Sau khi đạt hạn mức, trạng thái chiến dịch quan sát được là **tạm ngủ do đủ hạn mức**, phân biệt được với trạng thái **lỗi cấu hình** và với trạng thái **hỏng không cấp được**.
- **SC-013**: Hệ thống khởi động, chạy, và phục vụ đầy đủ chức năng khi **không khai báo chiến dịch nào**, với 0 cảnh báo lỗi liên quan tới chiến dịch.
- **SC-014**: Cấu hình chiến dịch sai bất kỳ trường bắt buộc nào đều bị **từ chối khi khởi động** kèm thông báo chỉ đúng trường đó, thay vì khởi động thành công rồi âm thầm không cấp gói.
- **SC-015**: **95%** tài khoản đủ điều kiện nhận được gói trong vòng **5 phút** kể từ lúc đăng ký, khi hệ thống vận hành bình thường.
- **SC-016**: **100%** tài khoản đủ điều kiện nhận được gói trong vòng **1 giờ** kể từ lúc đăng ký, kể cả khi hệ thống bị gián đoạn, nhờ cơ chế quét bù. Không tài khoản đủ điều kiện nào mất quyền lợi vì một lần gián đoạn của hệ thống.

## Assumptions

- Hạ tầng sẵn có đủ dùng: không cần thêm hạ tầng mới. Công việc bổ sung **hai** chủ đề mới vào hệ thống sự kiện đang chạy: `identity.user.created` và một chủ đề dành riêng cho chiến dịch, cùng một worker xử lý tin nhắn và một cơ chế quét bù chạy theo lịch — tất cả vào bộ khung đang có. Dự án đã có sẵn mẫu worker chạy theo lịch để làm theo. Giao diện tra cứu dùng lại cơ chế xác thực và phân quyền quản trị đang có, không mở cơ chế mới.
- Phát `identity.user.created` vô điều kiện làm chi phí cố định: mỗi lần tạo tài khoản, kể cả khi không có chiến dịch nào, đều thêm một bản ghi hàng đỗi. Đổi lại, mọi tính năng sau này dựa trên việc tạo tài khoản không phải sửa lại luồng đăng ký. Quy đổi này là **có chủ đích** theo FR-084 và FR-090.
- Con số 5 phút trong FR-078 là **cam kết vận hành dự kiến**, chưa đo được trên dữ liệu thật, vì hệ thống chưa từng chạy chiến dịch nào. Cơ chế quét bù làm cho con số này không quyết định việc tài khoản có bị mất quyền lợi hay không, nên khi đo được thực tế có thể chỉnh lại mà không phải đổi thiết kế.
- Cơ chế quét bù tìm tài khoản đủ điều kiện chưa có lượt cấp bằng cách quét trong cửa sổ thời gian của các chiến dịch đang mở, giới hạn ở quy mô nhỏ. Nếu sau này một chiến dịch chạy quy mô hàng trăm nghìn tài khoản, cách quét này có thể không còn phù hợp và cần thay bằng cách khác.
- Chiến dịch đầu tiên khai báo **tổng hạn mức 50 và hạn mức dự phòng 5**. Giá trị dự phòng 5 là con số quyết định theo phán đoán, chưa có cơ sở dữ liệu thực tế nào đo được độ trễ xử lý tin nhắn; nó có thể được chỉnh bằng khai báo mà không cần sửa mã nguồn.
- Quyết định ưu tiên theo thời điểm tạo tài khoản là **hy sinh tính tuyệt đối của con số 50** lấy sự công bằng cho tài khoản đăng ký sớm. Hệ quả được chấp nhận: chi phí cấp gói có thể vượt 50 tài khoản, và con số 50 trở thành mốc mục tiêu chứ không phải trần. Trần cứng là 55.
- Mỗi tài khoản tham gia **nhiều nhất một lần** trong toàn bộ cơ chế. Điều này đơn giản hoá dữ liệu và tránh việc một tài khoản bị cộng thời hạn nhiều lần, đồng thời giữ được đảm bảo trong FR-012. Hệ quả: nếu muốn cho tài khoản cũ tham gia chiến dịch mới thì phải dùng đường cấp gói thủ công, không qua chiến dịch.
- Chiến dịch được chọn cho một tài khoản là chiến dịch **bắt đầu muộn nhất** mà tài khoản đó đủ điều kiện. Quy tắc này thuần túy theo cấu hình và thời điểm tạo tài khoản, nên **không phụ thuộc** thứ tự tin nhắn tới.
- Không hỗ trợ điều kiện lọc theo nguồn đăng ký trong lần đầu. Chiến dịch ra mắt tặng cho **cả hai** nguồn. Cơ chế được thiết kế để bổ sung điều kiện lọc về sau bằng khai báo.
- Không cung cấp giao diện tra cứu cho chính khách hàng. Bộ phận hỗ trợ trả lời câu hỏi "tôi có được tặng không" dựa trên chức năng tra cứu dành cho quản trị viên.
- Hạ tầng thay đổi gói sẵn có đã lo được phần thông báo cho dịch vụ AI. Công việc này chỉ gọi lại đường đó, **không** sửa cơ chế đó.
- Cơ chế đếm hạn mức ngày của mỗi tài khoản đã có sẵn cách đọc khoá để trì hoãn thao tác ghi; việc sửa là đóng lại lỗ hổng đã có đường dẫn, không phải xây cơ chế mới.
- Thời điểm bắt đầu và kết thúc được hiểu theo **giờ Việt Nam**, và giá trị khai báo phải nêu rõ múi giờ để không phụ thuộc múi giờ của máy chủ.
