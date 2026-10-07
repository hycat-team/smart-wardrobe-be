# ADR 014 - Nới lỏng bất biến thứ tự của bản ghi mức quyền

- **Status**: Accepted
- **Decision Date**: 2026-10-07
- **Decision Maker**: Lê Hoàng (tech lead)
- **Supersedes**: ADR 013 §2 — *bất biến thứ tự nghiêm*

## 1. Context

Bản ghi mức quyền của một tài khoản tạo thành một dãy theo thứ tự thời gian. Từ spec 024, quy
tắc vận hành là: **từ chối vĩnh viễn** bất kỳ sự kiện nào không mới hơn thời điểm tạo của
bản ghi mới nhất, với mã lý do `stale_event`. Lý do là bảo vệ FR-033 của spec 024: không
được phép hồi sinh một mức quyền đã bị thu hồi.

Quy tắc này giả định sự kiện luôn **mới hơn** bản ghi đang có. Giả định đó sai trong một
tình huống cụ thể, và sự cố đã xảy ra trên production ngày 2026-10-07.

Khi ảnh chụp chính sách không đọc được `occurred_at` từ vỏ bọc `envelope`, hệ thống **không
đọc cột thời gian nào của bản ghi cả** — nó từ chối vĩnh viễn ngay tại đó. Hàm kiểm tra thứ tự
(`rejectStaleEvent`, `grant_uc.go:365`) gọi `ExtractOccurredAt` để lấy `occurred_at` từ
`policy_snapshot`; đọc lỗi thì trả về `stale_event` và không có đường lùi nào cả. Đó là khoá
chết, không phải chết tạm thời — không có lần giao lại nào giúp ích, vì dữ liệu không đổi.

Có thể chỉ sửa đúng việc đọc `occurred_at`: lùi về cột thời gian của bản ghi. Nhưng khi làm
vậy, một sự kiện thực sự **cũ** sẽ được chấp nhận với thời điểm bắt đầu bằng thời điểm của
sự kiện cũ — tức nhỏ hơn thời điểm bắt đầu của bản ghi đang có. Bản ghi mới khi đó nằm
**trước** bản ghi cũ trên trục thời gian, và việc cắt bản ghi cũ tại điểm bắt đầu mới sẽ tạo
ra một trong hai lỗi tệ hơn:

- nếu bản ghi cũ bị cắt tại một điểm **sau** mốc bắt đầu mới, hai cửa sổ **chồng lấn**;
- nếu bản ghi mới bắt đầu **trước** mốc cắt, một tài khoản đã trả tiền bị từ chối bởi mọi
  lệnh gọi AI trong khe hở đó.

Nói cách khác: bất biến thứ tự và bất biến không-chồng-lấn không thể giữ cùng lúc khi mốc
thời gian đến muộn. Phải chọn cách nới.

## 2. Decision

Thay vì từ chối, hệ thống **dịch** mốc bắt đầu hiệu lực của bản ghi mới tới **ngay sau**
bản ghi mới nhất, rồi cắt bản ghi cũ tại chính mốc đó.

**Có hai sàn, không phải một.** Đây là điểm dễ làm sai nhất của quyết định này, nên nêu ở
đây chứ không để trong tài liệu triển khai:

| Sàn | Công thức | Trả lời câu hỏi nào |
| --- | --- | --- |
| **sàn cũ** | `created_at` của bản ghi mới nhất, nếu đọc được và không nằm trong tương lai; nếu không thì `effective_from` của chính bản ghi đó | "sự kiện này có **mới hơn** lần ghi bản ghi gần nhất không" — bảo vệ FR-033 của spec 024 |
| **sàn mới** | `max(sàn cũ, effective_from của bản ghi mới nhất)` | "mốc bắt đầu mới có nằm **sau mọi** cửa sổ đã có không" — bảo vệ FR-004/FR-008 |

Chỉ dùng `sàn cũ` là **sai**, và sai theo cách im lặng. Với một bản ghi `tương lai`,
`created_at` nằm ở quá khứ còn `effective_from` nằm ở tương lai, nên `sàn cũ + 1 µs` vẫn
nhỏ hơn `effective_from` của bản ghi đó. Bản ghi mới bắt đầu **trước** nó, và
`TruncateGrantsBefore` — chỉ chạm dòng có `effective_from < cutoff` — **không** cắt được
dòng `tương lai`. Kết quả: hai cửa sổ đang hiệu lực chồng lấn, tức đúng thứ mà hàm
`GetNewestGrantForOrdering` được viết ra để ngăn.

Năm điều kiện kèm theo, tất cả đều bắt buộc:

1. **Mức dịch là mức nhỏ nhất cần thiết và không vượt quá 1 microsecond ngoài sàn mới**, tức
   `EffectiveFrom − sàn_mới ≤ 1 µs**. Cần nói rõ vì cách hiểu sai rất dễ xảy ra: con số này
   đo **độ dư quanh sàn**, **không** phải tổng độ dịch so với thời điểm của sự kiện. Một sự
   kiện đến muộn hơn sàn 5 giây thì tổng dịch là 5 giây 1 µs — và điều đó là bắt buộc, vì nếu
   giới hạn *tổng* dịch ở 1 µs thì FR-004 không thể thoả và FR-008 vỡ. Sự kiện có thể bị dịch
   nhiều lần nếu nhận nhiều
   sự kiện đến muộn, nhưng mỗi lần dịch là 1 µs, và tổng độ dịch bị chặn bởi `effective_to`.
2. **Mốc đã dịch là nguồn duy nhất cho cả bản ghi mới lẫn điểm cắt bản ghi cũ.** Tính đúng một
   lần, trong giao dịch, sau khi đã giữ khoá tài khoản; rồi dùng lại cho cả hai. Tính hai
   lần ở hai nơi là cách chắc chắn nhất để tạo ra khe hở hoặc chồng lấn.
3. **Trong vận hành bình thường thì không có tác dụng phụ.** Khi mốc hiệu lực của sự kiện đã
   lớn hơn **sàn mới**, hệ thống giữ nguyên mốc đó và không ghi thêm gì. Dịch là biện pháp dự
   phòng, không phải đường đi thường lệ.
4. **Phần dựng `sàn cũ` là một hàm thuần dùng chung.** Nhánh thu hồi bản ghi vì hạ cấp tới
   một gói không định giá được cũng kiểm tra thứ tự, và nó **không** tạo bản ghi mới nên không
   cần dịch cửa sổ — nhưng nó dùng cùng hàm kiểm tra, và nếu chỉ sửa một nhánh thì nhánh còn
   lại giữ nguyên đúng khoá chết này. Một lần sửa nửa vời tệ hơn là không sửa gì, vì nó tạo
   cảm giác đã xong.
5. **Sau khi dịch, cửa sổ phải còn khả dụng.** Nếu `sàn mới + 1 µs` không nhỏ hơn
   `effective_to`, cửa sổ rỗng và hệ thống từ chối vĩnh viễn thay vì ghi một bản ghi không có
   khoảng thời gian. Đây là quyết định có tác dụng phụ với khách hàng, nên nó phải là một yêu
   cầu riêng (FR-017) chứ không chỉ là hệ quả ngầm trong mã.

Về mặt bảo mật, sàn cũ được dựng trên cột thời gian nào là một phần của quyết định này chứ
không phải chi tiết triển khai:

- dùng `created_at` **khi nó đọc được và không nằm trong tương lai** so với đồng hồ ứng dụng;
- nếu `created_at` không đọc được hoặc nằm trong tương lai, lùi sang `effective_from`.

`effective_from` là sàn thay thế vì nó chịu `CHECK` `effective_to > effective_from` và phải
nằm trong khoảng thời gian khách hàng thực sự mua. `created_at` là cột kỹ thuật, không ràng
buộc gì, nên đặt sàn quyền hạn vào nó là biến một giá trị kỹ thuật thành đồng hồ quyền hạn:
ai đẩy được `created_at` tới tương lai thì chặn được mọi cấp quyền sau đó của tài khoản đó
tới khi mốc đó trôi qua. Cả hai nhánh phải để lại dòng cảnh báo, vì nhánh lùi là nhánh mà một
sự cố giả mạo dữ liệu sẽ tập trung vào.

`updated_at` bị loại trong sự cứu vãn: nó đổi mỗi lần một bản ghi bị cắt, nên nó phản ánh
thời điểm sửa bản ghi chứ không phản ánh thời điểm sự kiện được ghi. Dùng nó làm sàn sẽ
dịch sai.

Sự cố không nằm ở việc đọc cột nào, mà ở chỗ `rejectStaleEvent` **không đọc cột nào cả**:
nó gọi `ExtractOccurredAt` để lấy `occurred_at` từ `envelope` trong ảnh chụp chính sách, và
khi đọc không được thì **từ chối vĩnh viễn** — không có đường lùi. Đó là lý do FR-001 phải
lùi về `created_at`, và cũng là lý do quyết định này cần một cột dự phòng.

**Một điều dễ hiểu sai về phạm vi quyết định này.** `resolveGrantWindow` — hàm thực hiện
dịch — **không phải hàm thuần**: nó đọc bản ghi mới nhất qua repository, và phải đọc **bên
trong** giao dịch sau khi đã giữ khoá tài khoản. Nếu nó là hàm thuần thì FR-005 mất hết ý
nghĩa. Chỉ phần dựng `sàn cũ` từ một bản ghi đã đọc mới là hàm thuần, và vì vậy mới dùng
chung được cho cả hai nhánh ở điều kiện 4.

## 3. Consequences

**Tích cực**

- Khoá chết do `stale_event` biến mất. Sự kiện đến muộn được áp dụng thay vì bị lo vĩnh
  viễn.
- Không còn khe hở 1 µs và không còn chồng lấn, vì mốc cắt và mốc bắt đầu là một.
- Phần dựng `sàn cũ` là một phép tính thuần và kiểm thử được không cần cơ sở dữ liệu. Bản thân
  `resolveGrantWindow` thì **không** thuần: nó đọc bản ghi mới nhất, và phải đọc bên trong
  giao dịch sau khi đã giữ khoá — nếu không thì FR-005 mất hết ý nghĩa.
- Đường sửa dữ liệu hỏng thành đường hành chính `POST /api/v1/admin/subscription-grants`,
  không phải câu lệnh SQL.

**Tiêu cực**

- Đây là **sự nới lỏng thật**, không phải tối ưu cục bộ. Bản ghi mới nhất không còn bảo đảm
  rằng thời điểm bắt đầu của nó là lớn nhất tuyệt đối; nó chỉ lớn hơn *nghiêm*. Ai đọc bảng
  này giả định "thứ tự thời gian = thứ tự `effective_from`" phải biết điều này.
- Sai lệch tích luỹ dần theo số sự kiện đến muộn của cùng một tài khoản. Chặn bởi
  `effective_to` và bởi giới hạn 1 µs quanh sàn mới ở mỗi lần dịch, nên vô hại về cửa sổ quyền — nhưng nó là
  sai lệch thật và không nên coi là vô hạn.
- Nhánh `rejectStaleEvent` không còn là nơi duy nhất giữ bất biến thứ tự, nên người đọc code
  phải theo `resolveGrantWindow` mới thấy toàn bộ. Đây là cái giá của việc dùng một nguồn
  sự thật thay cho hai lần tính.
- Quy tắc lùi sang `effective_from` khi `created_at` ở tương lai có thể bị kích hoạt bởi
  **lệch đồng hồ** giữa dịch vụ AI và PostgreSQL chứ không chỉ bởi can thiệp. Đây là hành
  vi an toàn: sàn lùi về giá trị ràng buộc chặt hơn, nên phải lùi cả khi nguyên nhân là
  đồng hồ lệch.

**Điều không đổi**

- FR-033 của spec 024 vẫn giữ: một sự kiện có thời điểm xảy ra **không mới hơn** thời điểm
  tạo của bản ghi mới nhất vẫn bị từ chối, y hệt một sự kiện không mới hơn thời điểm xảy ra
  đã ghi. Việc dịch mốc chỉ áp dụng cho sự kiện **mới hơn**, ở phần còn lại của khoảng cách.
- FR-034 của spec 024 vẫn giữ: các cửa sổ của một tài khoản không chồng lấn.

## 4. Bổ sung (2026-10-07)

Khi ứng dụng thực hiện, một lỗi phân loại được phát hiện và xử lý ở đặc tả chứ không phải ở
đây: vi phạm `CHECK` đã bị xếp vào nhóm "tạm thời" và bị thử lại vô hạn. Lý do và cách sửa
thuộc FR-009..FR-011 của [spec 027](../../specs/027-grant-repair-unblock/spec.md); nó không
làm thay đổi quyết định ở §2, chỉ là cách thực thi nó.