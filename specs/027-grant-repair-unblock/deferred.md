# Deferred — 027 Sửa tắc hai nguyên nhân chặn vĩnh viễn

Việc phát hiện trong khi làm 027 nhưng **không** thuộc việc của 027. Không mục nào ở đây
chặn việc đóng 027; không mục nào là một nhiệm vụ của 027.

Tệp này tồn tại vì một ô vuông chưa tick trong `tasks.md` đưa ra một tuyên bố khác.
`- [ ]` nghĩa là *tính năng này chưa xong*. Việc thuộc về tính năng khác, hoặc thuộc về
hạ tầng kiểm thử, không phải việc dở dang của tính năng này; ghi nó vào `tasks.md` hoặc
giữ một tính năng đã đóng trông như còn mở, hoặc gây áp lực buộc điều kiện thoát phải bị
bịa ra sai.

## Mục

### D-01 — Tự chữa grant bằng sự kiện tự phát

- **Vấn đề**: Dữ liệu grant trên production hỏng và chỉ được phát hiện khi bộ đếm
  `grant_rejected_by_reason` tăng. Không có cơ chế nào tự phát ra sự kiện khi bản ghi không
  đọc được, nên phải có người chạy [runbook.md](runbook.md) mới biết.
- **Vì sao ngoài phạm vi**: Đã ghi thành [ADR 013](../../docs/decisions/013-deferred-lazy-grant-self-heal.md)
  với điều kiện mở lại. 027 sửa **cơ chế** làm hỏng dữ liệu, không sửa việc phát hiện. Sửa
  cơ chế mà không có cơ chế phát hiện thì sự cố lặp lại vẫn được phát hiện bằng mắt.
- **Gỡ bằng cách nào**: Điều kiện mở lại trong ADR 013. Nếu 027 chứng minh được sự cố do cơ
  chế phát ra chứ không phải do dữ liệu nhập sai, thì điều kiện đó còn chưa đạt.
- **Ảnh hưởng tới task đã tick**: không.

### D-02 — Nền tảng số liệu và ngưỡng cảnh báo

- **Vấn đề**: FR-011 yêu cầu mã lý do mới xuất hiện trong bộ đếm công khai nhưng **không**
  tự đặt ngưỡng cảnh báo. Hiện chưa có nền tảng số liệu nào cảnh báo tự động; một nhân sự
  vận hành phải tự mở trang sức khoẻ để nhìn thấy bộ đếm này.
- **Vì sao ngoài phạm vi**: Việc của hệ thống hạ tầng quan sát, không phải việc của một
  đặc tả sửa hai nhánh hỏng của đường sửa grant. Đặc tả cấm tự đặt ngưỡng là có chủ ý — một
  ngưỡng đặt sai trong 027 sẽ là một cam kết vận hành mà ai cũng đặt, không ai tin.
- **Gỡ bằng cách nào**: Một đặc tả riêng cho nền tảng số liệu, nơi quy tắc đặt ngưỡng được
  chốt một lần và áp cho mọi mã lý do. Đừng đặt ngưỡng cho từng mã ở đây.
- **Ảnh hưởng tới task đã tick**: không.

### D-03 — `ResolvePolicy` không lọc theo trạng thái bản ghi

- **Vấn đề**: Bộ đọc lọc theo cửa sổ hiệu lực chứ **không** lọc theo trạng thái, nên một bản
  ghi `đã đóng` cửa sổ còn hiệu lực vẫn được dùng. Hiện chưa xảy ra — kiểm tra toàn vẹn mục
  tương ứng trả về 0 — nhưng 027 vừa **dựa vào** chính bất biến này ở FR-014/FR-015.
- **Vì sao ngoài phạm vi**: Đây là một **bất biến tương đương đúng đắn**, không phải lỗi dữ
  liệu. Bản ghi `đã đóng` với cửa sổ còn hiệu lực vẫn mô tả đúng quyền mà khách hàng đã
  trả tiền, nên bộ đọc trả nó là hành vi đúng. Sửa bộ đọc thành "chỉ lấy `active`" là sẽ
  làm một bản ghi đóng có cửa sổ còn hiệu lực biến mất khỏi quyền của khách — tức là tạo
  đúng loại sự cố mà 027 đang sửa. Đặc tả 027 ghi nhận, không sửa.
- **Gỡ bằng cách nào**: Định nghĩa lại bất biến ở `docs/system-design/` cho cả hai bên — bộ
  đọc và kiểm tra toàn vẹn phải dùng **cùng một** định nghĩa cửa sổ hiệu lực. 027 đã làm đúng
  việc đó ở mục 4; việc còn lại là viết bất biến xuống tài liệu để lần sau không phải suy ra.
- **Ảnh hưởng tới task đã tick**: **Có** — nó là lý do FR-014 phải lọc theo cửa sổ hiệu lực.
  Nếu ai đó đổi bộ đọc để lọc theo trạng thái thì mục 4 của kiểm tra toàn vẹn sẽ sai theo và
  SC-005 sẽ hỏng theo.

### D-04 — Bổ sung grant cho các tài khoản thiếu ở môi trường development

- **Vấn đề**: Mười hai tài khoản development không có bản ghi mức quyền nào. Đường dựng hai
  tầng xử lý dựng này **không phải** điều kiện production — production đã có đủ dữ liệu.
- **Vì sao ngoài phạm vi**: Chúng chỉ ảnh hưởng việc chạy thử ở development, không ảnh
  hưởng khách hàng. 027 sửa một đường hỏng đã gây thiệt hại thật; dựng dữ liệu tiện nghiệm
  là việc riêng và nên đi kèm đường dựng dữ liệu thay vì một lệnh SQL thủ công — mà lệnh
  SQL thủ công thì vi phạm FR-013.
- **Gỡ bằng cách nào**: Một đặc tả cho đường dựng dữ liệu phát triển, sinh qua đường hành
  chính `POST /api/v1/admin/subscription-grants` với khoá chống trùng mới, thay vì SQL.
- **Ảnh hưởng tới task đã tick**: không.

### D-05 — Thay đổi tập loại sự kiện, hợp đồng sự kiện, hoặc nội dung ảnh chụp chính sách

- **Vấn đề**: 027 sửa hành vi nhận sự kiện và cách tính cửa sổ. Nó **không** thay đổi loại
  sự kiện, không thay đổi hợp đồng sự kiện, và không thay đổi nội dung ảnh chụp chính sách.
  Cả ba đều có thể bị hiểu là thuộc phạm vi khi đọc nhanh.
- **Vì sao ngoài phạm vi**: Đây là ranh giới, không phải một khoản nợ. Nêu ở đây để
  `tasks.md` không mở rộng phạm vi theo hiểu lầm — mở rộng vào loại sự kiện sẽ đụng tới
  hiến pháp §III và mở rộng vào ảnh chụp chính sách sẽ đụng tới §VI.
- **Gỡ bằng cách nào**: Đặc tả riêng, có đánh giá hiến pháp riêng.
- **Ảnh hưởng tới task đã tick**: không. Nói rõ để `[X]` ở AC-02 (không thêm đặc quyền)
  và FR-012 (từ chối không có tác dụng phụ) được đọc là vô điều kiện.