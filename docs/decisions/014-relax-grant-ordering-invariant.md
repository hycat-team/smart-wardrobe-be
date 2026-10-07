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

Khi ảnh chụp chính sách không đọc được `occurred_at` từ vỏ bọc `envelope`, hệ thống lùi về
chính `created_at` của bản ghi — và `created_at` **bằng đúng** thời điểm sự kiện được ghi.
Hệ quả: sự kiện không mới hơn, nên bị từ chối vĩnh viễn. Đây là khoá chết, không phải chết
tạm thời — không có lần giao lại nào giúp ích, vì dữ liệu không đổi.

Có thể chỉ sửa đúng việc đọc `occurred_at`. Nhưng khi làm vậy, một sự kiện thực sự **cũ** sẽ
được chấp nhận với thời điểm bắt đầu bằng thời điểm của sự kiện cũ — tức nhỏ hơn thời điểm
bắt đầu của bản ghi đang có. Bản ghi mới khi đó nằm **trước** bản ghi cũ trên trục thời gian,
và việc cắt bản ghi cũ tại điểm bắt đầu mới sẽ tạo ra một trong hai lỗi tệ hơn:

- nếu bản ghi cũ bị cắt tại một điểm **sau** mốc bắt đầu mới, hai cửa sổ **chồng lấn**;
- nếu bản ghi mới bắt đầu **trước** mốc cắt, một tài khoản đã trả tiền bị từ chối bởi mọi
  lệnh gọi AI trong khe hở đó.

Nói cách khác: bất biến thứ tự và bất biến không-chồng-lấn không thể giữ cùng lúc khi mốc
thời gian đến muộn. Phải chọn cách nới.

## 2. Decision

Thay vì từ chối, hệ thống **dịch** mốc bắt đầu hiệu lực của bản ghi mới tới **ngay sau**
bản ghi mới nhất, rồi cắt bản ghi cũ tại chính mốc đó.

Ba điều kiện kèm theo, cả ba đều bắt buộc:

1. **Mức dịch không vượt quá 1 microsecond.** Sàn thứ tự là mốc lớn nhất giữa mốc sự kiện và
   mốc của bản ghi mới nhất cộng 1 µs. Một bản ghi có thể bị dịch nhiều lần nếu nhận nhiều
   sự kiện đến muộn, nhưng mỗi lần dịch là 1 µs, và tổng độ dịch bị chặn bởi `effective_to`.
2. **Mốc đã dịch là nguồn duy nhất cho cả bản ghi mới lẫn điểm cắt bản ghi cũ.** Tính đúng một
   lần, trong giao dịch, sau khi đã giữ khoá tài khoản; rồi dùng lại cho cả hai. Tính hai
   lần ở hai nơi là cách chắc chắn nhất để tạo ra khe hở hoặc chồng lấn.
3. **Trong vận hành bình thường thì không có tác dụng phụ.** Khi mốc hiệu lực của sự kiện đã
   lớn hơn mọi bản ghi trước, hệ thống giữ nguyên mốc đó và không ghi thêm gì. Dịch là
   biện pháp dự phòng, không phải đường đi thường lệ.

Về mặt bảo mật, sàn thứ tự được dựng trên cột thời gian nào là một phần của quyết định này
chứ không phải chi tiết triển khai:

- dùng `started_at` **khi nó không nằm trong tương lai** so với đồng hồ ứng dụng;
- nếu `started_at` không đọc được hoặc nằm trong tương lai, lùi sang `effective_from`.

`effective_from` là sàn thay thế vì nó chịu `CHECK` `effective_to > effective_from` và phải
nằm trong khoảng thời gian khách hàng thực sự mua. `started_at` là cột kỹ thuật, không ràng
buộc gì, nên đặt sàn quyền hạn vào nó là biến một giá trị kỹ thuật thành đồng hồ quyền hạn:
ai đẩy được `started_at` tới tương lai thì chặn được mọi cấp quyền sau đó của tài khoản đó
tới khi mốc đó trôi qua.

## 3. Consequences

**Tích cực**

- Khoá chết do `stale_event` biến mất. Sự kiện đến muộn được áp dụng thay vì bị lo vĩnh
  viễn.
- Không còn khe hở 1 µs và không còn chồng lấn, vì mốc cắt và mốc bắt đầu là một.
- Thao tác này là một phép tính thuần trên `GrantWindow`, kiểm thử được không cần cơ sở dữ
  liệu.
- Đường sửa dữ liệu hỏng thành đường hành chính `POST /api/v1/admin/subscription-grants`,
  không phải câu lệnh SQL.

**Tiêu cực**

- Đây là **sự nới lỏng thật**, không phải tối ưu cục bộ. Bản ghi mới nhất không còn bảo đảm
  rằng thời điểm bắt đầu của nó là lớn nhất tuyệt đối; nó chỉ lớn hơn *nghiêm*. Ai đọc bảng
  này giả định "thứ tự thời gian = thứ tự `effective_from`" phải biết điều này.
- Sai lệch tích luỹ dần theo số sự kiện đến muộn của cùng một tài khoản. Chặn bởi
  `effective_to` và bởi giới hạn 1 µp mỗi lần, nên vô hại về cửa sổ quyền — nhưng nó là
  sai lệch thật và không nên coi là vô hạn.
- Nhánh `rejectStaleEvent` không còn là nơi duy nhất giữ bất biến thứ tự, nên người đọc code
  phải theo `resolveGrantWindow` mới thấy toàn bộ. Đây là cái giá của việc dùng một nguồn
  sự thật thay cho hai lần tính.
- Quy tắc lùi sang `effective_from` khi `started_at` ở tương lai có thể bị kích hoạt bởi
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