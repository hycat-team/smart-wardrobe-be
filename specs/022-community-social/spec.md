# Feature Specification: Nhóm chức năng cộng đồng (Community Social)

**Feature Branch**: `022-community-social`

**Created**: 2026-09-25

**Status**: Draft

**Input**: User description: "Nhóm chức năng cộng đồng: đăng bài (outfit post gắn outfit, cascade khi xóa outfit; media post ảnh/video/không media), like, comment 2 cấp, follow user, cho phép xem danh sách followee và follower của 1 user, hỗ trợ search theo type là followee hoặc follower (optional), trang profile công khai theo username, search q (user + post, 200 ký tự đầu), feed type explore/following với hot ranking query-time, admin moderation posts+comments."

## Compliance & Governance (MANDATORY)

> Việc triển khai spec này **bắt buộc** tuân thủ [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md)
> và [AGENTS.md](../../AGENTS.md), cùng toàn bộ tài liệu trong `docs/development/` và
> `docs/system-design/` (đặc biệt `design-pattern.md`, `code-hygiene.md`, `agent-workflow.md`,
> `git-workflow.md`, `messaging-guide.md`). Khi xung đột, thứ tự ưu tiên:
> **Constitution > system-design > development > other docs**.
> Checklist bắt buộc đầy đủ: [compliance.md](./compliance.md).

## Quyết định hiện hành (authoritative)

> **Mục này là nguồn chân lý.** Phần `## Clarifications` bên dưới (nếu có) là lịch sử hội thoại;
> khi mâu thuẫn, mục này thắng. Agent implement chỉ cần đọc mục này + `plan.md` + `tasks.md`.

1. **Vị trí module**: nhóm chức năng cộng đồng là một module nghiệp vụ mới thuộc **Commerce Core** (dùng chung service với tủ đồ/outfit), không dựng service mới.
2. **Loại bài đăng**: có đúng **2 loại** — `outfit` (gắn một outfit có sẵn trong tủ đồ của người đăng) và `media` (chứa ảnh hoặc video, hoặc **không chứa media**). Giá trị enum dùng **chữ thường** theo quy ước hiện hành của dự án. Bài `outfit` bắt buộc có outfit; bài `media` không có outfit.
3. **Cascade outfit**: khi một outfit bị xóa, **mọi bài đăng loại outfit tham chiếu outfit đó bị xóa theo**. Không dùng ràng buộc khoá ngoại xuyên module; dùng sự kiện/cascade ở tầng ứng dụng.
4. **Like**: người dùng đăng nhập like/unlike một bài; trạng thái like gửi lên ở dạng rõ ràng (bật/tắt) để đảm bảo idempotent. Một người chỉ like một bài tối đa một lần.
5. **Comment**: tối đa **2 cấp** (cấp gốc và phản hồi cấp 1). Khi người dùng cố phản hồi vào một phản hồi (cấp 2), hệ thống **tự động gắn phản hồi mới lên comment gốc** thay vì báo lỗi.
6. **Follow**: quan hệ một chiều (theo dõi), không phải kết bạn; cho phép theo dõi và hủy theo dõi; không cho tự theo dõi chính mình.
7. **Danh sách follow**: xem được danh sách người mà một user đang theo dõi (gọi là `following`) và danh sách người đang theo dõi user đó (gọi là `followers`).
8. **Search trong danh sách follow (optional)**: hỗ trợ lọc/tìm kiếm theo `type` là `following` hoặc `followers`.
9. **Trang cá nhân công khai theo username**: truy cập được bằng username, hiển thị **tên, giới tính, ảnh đại diện**; kèm **số lượng follower/following/bài đăng** và **trạng thái đang theo dõi hay chưa**. Chưa cần tiểu sử (bio).
10. **Search tổng hợp**: một tham số `q` tìm đồng thời **tên người dùng** và **nội dung bài đăng**; trả về **2 tập kết quả độc lập** (users, posts), mỗi tập phân trang riêng. Với nội dung bài đăng, chỉ so khớp trong **200 ký tự đầu tiên** (ngưỡng cấu hình được).
11. **Feed**: hỗ trợ `type` = `explore` (tất cả bài công khai) hoặc `following` (chỉ bài của người mình theo dõi); mặc định là `explore`. Hỗ trợ sắp xếp `hot` (mặc định) hoặc `latest`.
12. **Hot ranking**: tính tại thời điểm truy vấn (query-time), **không** dùng bảng snapshot, cờ dirty hay worker làm mới điểm. Công thức dựa trên tương tác và mức suy giảm theo thời gian, có cửa sổ thời gian giới hạn và hệ số cấu hình được. Cả `explore` lẫn `following` dùng chung cơ chế.
13. **Admin moderation**: giữ tính năng quản trị cho bài đăng và comment (ẩn/xóa/khôi phục); **bỏ** phần quản trị mặt hàng bán lại (đã loại bỏ cùng tính năng resale).
14. **Tương thích API cũ**: giữ nguyên đường dẫn, envelope phản hồi và hình dạng DTO hiện có cho các thao tác bài đăng/like/comment/upload, chỉ bổ sung trường mới và endpoint mới nhằm giảm chi phí cho đội frontend. Các trường liên quan bán lại (trạng thái chuyển nhượng, người mua, giá bán...) bị loại bỏ.
15. **Không nằm trong đợt này**: thông báo (notification) cho like/comment/follow, tiểu sử (bio), chế độ riêng tư/bài chỉ người theo dõi, chặn/báo cáo, và đồng bộ bài đăng khi outfit bị chỉnh sửa (chỉ xử lý cascade khi xóa).
16. **Khách chưa đăng nhập (guest)**: chỉ đọc công khai (feed, bài, comment, trang cá nhân, search); mọi thao tác ghi cần đăng nhập. Guest gọi feed `type=following` nhận lỗi xác thực (401).
17. **Feed `following`** bao gồm cả bài của chính người xem. Bài của người dùng đang bị vô hiệu hóa/khóa bị ẩn khỏi công khai nhưng dữ liệu được giữ lại (không tự xóa).
18. **Vòng đời bài/comment**: dùng xóa mềm. Bài do tác giả tự xóa **không thể khôi phục** (kể cả admin); admin ẩn/xóa và khôi phục nội dung **do admin ẩn/xóa**. Tác giả vẫn thấy bài của mình khi bị admin ẩn, kèm trạng thái "đang bị ẩn". Đổi username không có chuyển hướng; URL profile cũ trả không tìm thấy.
19. **Giới hạn nội dung**: tối đa **10 media/bài** (ảnh ≤10MB, jpg/png/webp; video ≤100MB, mp4/webm, thời lượng ≤60s); `title` ≤150 ký tự; `content` ≤5000 ký tự; comment ≤1000 ký tự. Một outfit có thể gắn vào nhiều bài; khi cập nhật được đổi sang outfit khác thuộc sở hữu nhưng **không đổi loại bài**. Không có like cho comment.
20. **Thứ tự bình luận**: bình luận gốc **mới nhất trước**; phản hồi (cấp 1) trong mỗi gốc **cũ nhất trước**. Hiển thị số lượng phản hồi; không nhúng sẵn phản hồi trong danh sách feed. Xóa bình luận gốc còn phản hồi → hiển thị "bình luận đã bị xóa"; nếu không còn phản hồi thì ẩn hẳn.
21. **Danh sách follow**: sắp xếp theo lần theo dõi mới nhất trước; có nút theo dõi/bỏ theo dõi trong danh sách. Search trong danh sách follow: `type` **luôn tuỳ chọn**; khi bỏ trống trả về **cả hai** (following + follower) và mỗi mục mang trường `relation`.
22. **Search tổng hợp**: tìm theo cả `username` lẫn họ tên; `type` luôn tuỳ chọn (mặc định `all`); `q` tối thiểu 1 ký tự (đã trim), rỗng → kết quả rỗng; công khai. Thứ tự kết quả: users theo số follower giảm dần rồi mới nhất; posts theo hot.
23. **Hot ranking**: like×1, comment×2, cộng nền `+1`, `gravity=1.5`, cửa sổ 30 ngày (cấu hình được). Trang cá nhân vẫn sắp `latest`. Đợt này dùng tìm kiếm dự phòng truy vấn trực tiếp (chưa bật công cụ tìm kiếm chuyên dụng).
24. **Kỹ thuật**: phân trang theo chuẩn `PaginationQuery` (page/limit) hiện có; `publicId` bài là chuỗi ngẫu nhiên 32 ký tự; bộ đếm like/comment/follow lưu sẵn (denormalized) + cơ chế đối soát định kỳ; chống trùng bằng ràng buộc duy nhất ở tầng lưu trữ; comment lưu `parentCommentId` (luôn trỏ gốc) + `rootCommentId`; có log/metric cơ bản; media cần hỗ trợ tải video; moderation cần ghi nhật ký kiểm toán.
25. **Rate limit**: đăng bài 10/giờ, comment 60/giờ, follow 200/giờ, like 600/giờ (cấu hình được).
26. **Tài liệu frontend (deliverable bắt buộc)**: toàn bộ spec 022 phải kèm một tài liệu tích hợp frontend mô tả đầy đủ **phần giữ nguyên** so với code frontend hiện có, **phần mới**, và **phần chỉnh sửa/thay đổi**, kèm đặc tả request/response và hướng dẫn cập nhật, để đội frontend triển khai chỉ dựa trên tài liệu.
27. **Ẩn bài theo trạng thái user (cơ chế)**: Account publish event `identity.user.status.changed` khi admin đổi trạng thái user; community consume → set `status=hidden` + `hidden_reason=author_disabled` khi bị vô hiệu hóa/khóa, và đưa về `published` (xóa reason) khi hoạt động lại. Không join xuyên module.
28. **Comment/reply giữ dạng mảng** (tương thích FE cũ); phân trang chỉ áp cho bảng tin, danh sách like, danh sách follow và kết quả tìm kiếm.
29. **Nhãn follow**: query `type=following|followers`; trường `relation` = `following|follower`.
30. **Search user**: Account trả tối đa **N ứng viên** khớp (N cấu hình `Community.SearchUserCandidateCap`, mặc định 500); community enrich số follower từ `user_follow_stats`, sắp theo follower giảm dần rồi mới nhất, rồi phân trang.
31. **Rate limit** được áp ngay trong từng user story (không dồn về cuối).

## Clarifications

### Session 2026-09-25

- Q: Vị trí module? → A: Commerce Core (gần outfit/wardrobe).
- Q: Giá trị loại bài đăng? → A: `outfit` + `media`, chữ thường.
- Q: Cơ chế ranking? → A: Hot ranking query-time bản nhẹ (không snapshot/worker), có cửa sổ thời gian và hệ số cấu hình.
- Q: Hình dạng response search? → A: 2 tập kết quả độc lập, phân trang riêng.
- Q: Kiểu endpoint follow? → A: Bật/tắt theo trạng thái rõ ràng (idempotent), đồng nhất với like.
- Q: Reply vào comment cấp 2? → A: Tự động gắn lên comment gốc.
- Q: Notification đợt này? → A: Không làm.
- Q: Guest xem được gì? → A: Chỉ đọc công khai; thao tác ghi cần đăng nhập; guest gọi feed `following` nhận 401.
- Q: Feed `following` có gồm bài của chính mình / bài của user bị khóa? → A: Gồm bài của chính mình; bài của user bị vô hiệu hóa bị ẩn khỏi công khai.
- Q: Vòng đời xóa bài và quyền khôi phục? → A: Xóa mềm; bài tác giả tự xóa không thể khôi phục (kể cả admin); admin chỉ khôi phục nội dung do admin ẩn; tác giả vẫn thấy bài bị ẩn của mình kèm trạng thái.
- Q: Giới hạn media và nội dung? → A: 10 media/bài (ảnh ≤10MB jpg/png/webp; video ≤100MB mp4/webm ≤60s); title ≤150; content ≤5000; comment ≤1000 ký tự.
- Q: Một outfit gắn nhiều bài? Cập nhật có đổi loại bài? → A: Có thể gắn nhiều bài; đổi được outfit khác thuộc sở hữu nhưng không đổi loại bài.
- Q: Comment có like? → A: Không.
- Q: Thứ tự comment/reply và hiển thị phản hồi? → A: Gốc mới nhất trước, phản hồi cũ nhất trước; hiển thị số phản hồi; không nhúng sẵn phản hồi trong feed.
- Q: Xóa comment cha có phản hồi? → A: Hiển thị "bình luận đã bị xóa" nếu còn phản hồi; ẩn hẳn nếu không còn.
- Q: Thứ tự danh sách follow và nút thao tác? → A: Lần theo dõi mới nhất trước; có nút theo dõi/bỏ theo dõi trong danh sách.
- Q: `type` trong search danh sách follow? → A: Luôn tuỳ chọn; bỏ trống trả cả hai kèm trường `relation`.
- Q: `q` search user theo gì, tối thiểu bao nhiêu ký tự? → A: Theo cả username lẫn họ tên; tối thiểu 1 ký tự, rỗng → rỗng.
- Q: Thứ tự kết quả search? → A: Users theo follower giảm dần rồi mới nhất; posts theo hot.
- Q: Dùng công cụ tìm kiếm chuyên dụng hay dự phòng? → A: Dự phòng truy vấn trực tiếp đợt này; để hook công cụ chuyên dụng sau.
- Q: Cascade outfit đồng bộ hay bất đồng bộ? → A: Bất đồng bộ qua sự kiện, nhất quán trong ≤1 phút.
- Q: Bộ đếm lưu sẵn hay đếm khi truy vấn? → A: Lưu sẵn (denormalized) + đối soát định kỳ; chống trùng bằng ràng buộc duy nhất.
- Q: Phân trang dùng kiểu gì? → A: Theo chuẩn phân trang hiện có (page/limit).
- Q: `publicId` bài? → A: Chuỗi ngẫu nhiên 32 ký tự.
- Q: Audit log cho moderation? → A: Có; moderation từng mục một (không bulk); danh sách + hành động cơ bản.
- Q: Rate limit cụ thể? → A: post 10/giờ, comment 60/giờ, follow 200/giờ, like 600/giờ (cấu hình được).
- Q: Tài liệu cho frontend? → A: Bắt buộc có tài liệu tích hợp frontend cho toàn bộ spec 022 (giữ nguyên + mới + thay đổi).
- Q: Cơ chế ẩn bài của user bị vô hiệu hóa/khóa? → A: Account publish event `identity.user.status.changed` → worker community ẩn/hiện bài với `status=hidden` + `hidden_reason=author_disabled` (không join xuyên module).
- Q: Phân trang comment/reply? → A: Giữ mảng (tương thích FE); chỉ phân trang feed/like/follow/search.
- Q: Nhãn follow? → A: query `type=following|followers`; `relation=following|follower`.
- Q: Sắp user trong search theo follower khi dữ liệu khác owner? → A: Account trả tối đa N ứng viên (config), community enrich follower rồi sort + phân trang.
- Q: Tác giả thấy bài bị admin ẩn? → A: Có, viewer-aware.
- Q: Admin xóa/ẩn comment gốc cascade reply con? → A: Có, ẩn/hiện theo cây.
- Q: Rate limit đặt ở đâu? → A: Áp trong từng user story.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Đăng bài và xem bảng tin (Priority: P1)

Người dùng đã đăng nhập tạo một bài đăng mới. Với bài khoe outfit, người dùng chọn một outfit có sẵn trong tủ đồ của mình và viết nội dung. Với bài truyền thông, người dùng có thể đăng kèm ảnh, video, hoặc chỉ có nội dung chữ. Sau khi đăng, bài xuất hiện trên bảng tin cộng đồng và ở trang cá nhân của người đăng.

**Why this priority**: Đăng bài là năng lực cốt lõi — không có bài đăng thì mọi tính năng tương tác khác vô nghĩa.

**Independent Test**: Có thể kiểm thử độc lập bằng cách tạo lần lượt một bài outfit (gắn outfit hợp lệ) và một bài media (có ảnh, có video, và không media), rồi xác nhận cả ba xuất hiện đúng trên bảng tin và trang cá nhân.

**Acceptance Scenarios**:

1. **Given** người dùng có ít nhất một outfit trong tủ đồ, **When** người dùng tạo bài loại outfit gắn outfit đó kèm nội dung, **Then** bài được tạo thành công, hiển thị outfit và nội dung.
2. **Given** người dùng đã đăng nhập, **When** người dùng tạo bài media kèm ảnh/video hoặc không kèm media, **Then** bài được tạo thành công với đúng nội dung và media (nếu có).
3. **Given** người dùng tạo bài outfit nhưng không chọn outfit, **When** hệ thống xử lý, **Then** hệ thống từ chối với thông báo hợp lệ.
4. **Given** người dùng tạo bài media kèm nhiều media, **When** hệ thống lưu, **Then** thứ tự hiển thị media được giữ đúng như người dùng sắp xếp.
5. **Given** một bài đăng đã tồn tại, **When** chủ bài chỉnh sửa nội dung/media, **Then** bài cập nhật thành công và chỉ chủ bài được phép chỉnh sửa.

---

### User Story 2 - Like và bình luận (Priority: P1)

Người dùng xem một bài đăng, nhấn like hoặc bỏ like, và có thể bình luận. Người khác có thể phản hồi lại bình luận đó, nhưng chỉ sâu tối đa 2 cấp. Số lượng like và comment được hiển thị trên bài.

**Why this priority**: Tương tác là lý do tồn tại của bảng tin xã hội, và cần đi kèm với đăng bài để tạo giá trị tối thiểu.

**Independent Test**: Kiểm thử bằng cách like/unlike nhiều lần (kiểm tra tính idempotent), tạo bình luận gốc, phản hồi, và phản hồi vào phản hồi (kiểm tra gắn lên gốc), đồng thời xác nhận bộ đếm cập nhật đúng.

**Acceptance Scenarios**:

1. **Given** người dùng chưa like bài, **When** người dùng bật like, **Then** bộ đếm like tăng 1 và trạng thái like của người dùng là đã thích.
2. **Given** người dùng đã like bài, **When** người dùng bật like lần nữa (cùng trạng thái), **Then** hệ thống không tạo like trùng và bộ đếm không đổi.
3. **Given** một bài đăng công khai, **When** người dùng gửi bình luận gốc, **Then** bình luận được tạo ở cấp gốc và bộ đếm comment tăng 1.
4. **Given** một bình luận gốc tồn tại, **When** người dùng phản hồi, **Then** phản hồi thuộc cấp 1 của bình luận gốc đó.
5. **Given** một phản hồi cấp 1 tồn tại, **When** người dùng phản hồi vào phản hồi đó, **Then** phản hồi mới được tự động gắn lên bình luận gốc (không tạo cấp 3).
6. **Given** người dùng là chủ bình luận, **When** người dùng chỉnh sửa hoặc xóa bình luận, **Then** thao tác thành công; người khác không thể chỉnh sửa/xóa bình luận của người khác.

---

### User Story 3 - Theo dõi người dùng và xem trang cá nhân công khai (Priority: P2)

Người dùng mở trang cá nhân công khai của một người khác bằng username, xem tên, giới tính, ảnh đại diện, số follower/following và các bài đăng. Người dùng có thể theo dõi hoặc hủy theo dõi ngay trên trang đó, và xem danh sách người mà trang này đang theo dõi hoặc danh sách follower.

**Why this priority**: Follow và trang cá nhân tạo mạng lưới xã hội, là nền tảng cho feed `following`; nhưng có thể triển khai sau khi đăng bài/tương tác đã hoạt động.

**Independent Test**: Kiểm thử bằng cách theo dõi một user, mở trang cá nhân của user đó, xác nhận trạng thái theo dõi, số đếm, danh sách following/followers, và các bài đăng tương ứng.

**Acceptance Scenarios**:

1. **Given** hai người dùng khác nhau, **When** người A theo dõi người B, **Then** trạng thái theo dõi của A với B là "đang theo dõi", follower của B tăng và following của A tăng.
2. **Given** A đang theo dõi B, **When** A hủy theo dõi B, **Then** trạng thái trở về "chưa theo dõi", các bộ đếm giảm tương ứng.
3. **Given** người dùng đang xem trang cá nhân của chính mình, **When** trang tải, **Then** hệ thống không hiển thị nút theo dõi chính mình.
4. **Given** một username không tồn tại, **When** người dùng truy cập, **Then** hệ thống trả về "không tìm thấy".
5. **Given** một trang cá nhân có nhiều follower/following, **When** người dùng mở danh sách, **Then** danh sách phân trang và có thể lọc theo `type` là `following` hoặc `followers`.

---

### User Story 4 - Tìm kiếm người dùng và bài đăng (Priority: P2)

Người dùng nhập một từ khoá vào ô tìm kiếm và nhận về hai nhóm kết quả: những người dùng có tên khớp và những bài đăng có nội dung khớp (chỉ xét 200 ký tự đầu của bài).

**Why this priority**: Tìm kiếm giúp khám phá nội dung/người dùng nhưng không chặn luồng MVP.

**Independent Test**: Kiểm thử bằng cách tìm một từ khoá khớp cả tên người dùng lẫn nội dung bài, xác nhận hai nhóm kết quả trả về độc lập và đúng; và một từ khoá chỉ xuất hiện sau ký tự thứ 200 của bài thì không khớp.

**Acceptance Scenarios**:

1. **Given** tồn tại người dùng có tên khớp từ khoá, **When** người dùng tìm với `q`, **Then** nhóm users chứa người dùng đó.
2. **Given** tồn tại bài đăng có 200 ký tự đầu chứa từ khoá, **When** người dùng tìm với `q`, **Then** nhóm posts chứa bài đó.
3. **Given** từ khoá chỉ xuất hiện sau ký tự thứ 200 của nội dung bài, **When** người dùng tìm, **Then** bài đó không xuất hiện trong nhóm posts.
4. **Given** người dùng tìm với `type` chỉ định, **When** hệ thống xử lý, **Then** chỉ nhóm tương ứng được trả về (hoặc nhóm còn lại rỗng).
5. **Given** `q` rỗng hoặc chỉ khoảng trắng, **When** người dùng tìm, **Then** hệ thống trả về kết quả rỗng an toàn, không lỗi.

---

### User Story 5 - Bảng tin explore/following với hot ranking (Priority: P2)

Người dùng xem bảng tin mặc định (`explore`) được xếp theo độ "hot" dựa trên tương tác và độ mới; hoặc chuyển sang `following` để chỉ xem bài của những người mình theo dõi, cũng xếp theo hot. Người dùng cũng có thể chọn sắp xếp theo mới nhất.

**Why this priority**: Nâng cao trải nghiệm khám phá, nhưng bảng tin cơ bản đã có thể chạy với thứ tự mới nhất trước.

**Independent Test**: Kiểm thử bằng cách tạo các bài có tương tác/thời điểm khác nhau, xác nhận thứ tự hot khác thứ tự mới nhất; và kiểm tra `following` chỉ chứa bài của người đã theo dõi.

**Acceptance Scenarios**:

1. **Given** caller không truyền `type`, **When** lấy bảng tin, **Then** mặc định là `explore`.
2. **Given** có bài mới hơn nhưng ít tương tác và bài cũ hơn nhưng nhiều tương tác, **When** sắp xếp `hot`, **Then** thứ tự phản ánh điểm hot (không thuần theo thời gian) trong cửa sổ thời gian.
3. **Given** người dùng theo dõi một số người, **When** lấy bảng tin `following`, **Then** chỉ trả về bài của những người đang theo dõi (và bài của chính mình nếu có).
4. **Given** người dùng chưa theo dõi ai, **When** lấy `following`, **Then** trả về danh sách rỗng (hoặc chỉ bài của chính họ) không lỗi.
5. **Given** bài nằm ngoài cửa sổ thời gian hot, **When** sắp xếp `hot`, **Then** bài đó không lọt vào kết quả hot nhưng vẫn có thể xuất hiện khi sắp xếp `latest`.

---

### User Story 6 - Quản trị nội dung cộng đồng (Priority: P3)

Quản trị viên xem danh sách bài đăng và bình luận, có thể ẩn/xóa các nội dung vi phạm và khôi phục nội dung đã ẩn/xóa.

**Why this priority**: Cần cho vận hành và kiểm duyệt nhưng không chặn các luồng người dùng chính.

**Independent Test**: Kiểm thử bằng cách dùng tài khoản admin ẩn/xóa/khôi phục một bài và một bình luận, xác nhận trạng thái thay đổi đúng và nội dung bị ẩn/xóa không còn hiển thị với người dùng thường.

**Acceptance Scenarios**:

1. **Given** tài khoản admin, **When** admin ẩn một bài, **Then** bài không còn xuất hiện trong bảng tin/tìm kiếm/trang cá nhân với người dùng thường nhưng vẫn thấy trong danh sách quản trị.
2. **Given** tài khoản admin, **When** admin khôi phục một bài đã ẩn, **Then** bài hiển thị trở lại.
3. **Given** tài khoản admin, **When** admin xóa một bình luận vi phạm, **Then** bình luận và các phản hồi của nó không còn hiển thị với người dùng thường.
4. **Given** người dùng không phải admin, **When** gọi endpoint quản trị, **Then** hệ thống từ chối truy cập.

---

### Edge Cases

- **Xóa outfit có bài đăng loại outfit**: mọi bài tham chiếu bị xóa/hủy hiển thị theo; nếu outfit không có bài nào thì không ảnh hưởng.
- **Bài outfit trỏ tới outfit đã bị xóa trước đó**: không được hiển thị; thao tác tạo bài với outfit không tồn tại/không thuộc sở hữu bị từ chối.
- **Bài media không có media và không có nội dung**: từ chối (phải có ít nhất nội dung chữ hoặc media).
- **Media vượt giới hạn số lượng/loại không hợp lệ**: từ chối với thông báo hợp lệ.
- **Like/unlike lặp lại hoặc đồng thời**: idempotent, không tạo bản ghi trùng, bộ đếm không âm.
- **Tự theo dõi chính mình**: từ chối.
- **Theo dõi lặp lại / hủy khi chưa theo dõi**: idempotent, không lỗi.
- **Reply vào comment cấp 2**: tự gắn lên gốc.
- **Reply vào comment không tồn tại / không thuộc bài**: từ chối.
- **Bình luận cha bị xóa**: phản hồi con cũng không còn hiển thị (ẩn theo cha).
- **Người dùng bị vô hiệu hóa/xóa**: không thể đăng bài/tương tác/follow; không tự động xóa bài cũ trong phạm vi này, nhưng trang cá nhân xử lý hợp lý.
- **Truy cập trang cá nhân bằng username khác hoa thường**: xử lý không phân biệt hoa thường.
- **`q` trùng ký tự đặc biệt (%, _) hoặc rất dài**: xử lý an toàn, không lỗi truy vấn.
- **`q` chỉ khớp nội dung nằm sau 200 ký tự**: không xuất hiện.
- **Feed hot khi có ít dữ liệu**: vẫn trả danh sách hợp lệ, không chia cho 0.
- **Phân trang vượt số trang**: trả về danh sách rỗng, metadata hợp lệ.
- **Admin ẩn bài nhưng comment vẫn tồn tại**: comment không hiển thị qua bài đã ẩn với người dùng thường.
- **Guest gọi feed `following`**: nhận 401, không rò rỉ dữ liệu.
- **Guest thử like/comment/follow**: bị từ chối yêu cầu đăng nhập.
- **Tác giả tự xóa bài rồi muốn khôi phục**: không được, kể cả nhờ admin; khác với bài bị admin ẩn (khôi phục được).
- **Tác giả đổi username**: URL profile cũ trả không tìm thấy, không có chuyển hướng.
- **Vượt giới hạn media/số ký tự**: từ chối với thông báo hợp lệ (10 media, 150/5000/1000 ký tự, kích thước/định dạng/thời lượng media).
- **Cập nhật bài đổi loại bài**: bị từ chối; đổi outfit khác thuộc sở hữu: được phép.
- **Thứ tự hiển thị**: gốc mới nhất trước, phản hồi cũ nhất trước; feed không nhúng sẵn phản hồi.
- **Xóa bình luận gốc còn phản hồi**: hiển thị "bình luận đã bị xóa" thay vì mất cả luồng.
- **Search danh sách follow không truyền `type`**: trả cả following lẫn follower, mỗi mục có `relation`.
- **`q` chỉ gồm khoảng trắng hoặc rỗng**: trả kết quả rỗng, không lỗi.
- **Người dùng bị vô hiệu hóa sau khi đã có bài**: bài bị ẩn khỏi công khai nhưng dữ liệu vẫn còn.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST cho phép người dùng đã đăng nhập tạo bài đăng thuộc một trong hai loại: `outfit` hoặc `media`.
- **FR-002**: Bài loại `outfit` MUST gắn đúng một outfit thuộc sở hữu của người đăng; hệ thống MUST từ chối nếu outfit không tồn tại hoặc không thuộc sở hữu người đăng.
- **FR-003**: Bài loại `media` MUST hỗ trợ ảnh, video, hoặc không có media; mỗi media có loại, đường dẫn, định danh công khai và thứ tự hiển thị.
- **FR-004**: Hệ thống MUST yêu cầu bài đăng có nội dung chữ hoặc media; từ chối bài rỗng hoàn toàn.
- **FR-005**: Khi một outfit bị xóa, hệ thống MUST xóa (hoặc hủy hiển thị vĩnh viễn) mọi bài loại outfit tham chiếu outfit đó, xử lý bất đồng bộ nhưng bảo đảm cuối cùng nhất quán.
- **FR-006**: Hệ thống MUST cho phép chủ bài chỉnh sửa và xóa bài của mình; MUST NOT cho phép người khác làm điều đó.
- **FR-007**: Hệ thống MUST hỗ trợ like/unlike idempotent với trạng thái gửi lên rõ ràng; một người dùng chỉ like một bài tối đa một lần.
- **FR-008**: Hệ thống MUST duy trì bộ đếm like và comment trên mỗi bài, phản ánh đúng sau mọi thao tác.
- **FR-009**: Hệ thống MUST cho phép bình luận tối đa 2 cấp; khi phản hồi vào cấp 2, hệ thống MUST tự động gắn phản hồi lên comment gốc.
- **FR-010**: Hệ thống MUST cho phép chủ comment chỉnh sửa/xóa comment của mình; MUST NOT cho phép người khác.
- **FR-011**: Khi bình luận cha bị xóa, hệ thống MUST không hiển thị các phản hồi con với người dùng thường.
- **FR-012**: Hệ thống MUST cho phép người dùng theo dõi và hủy theo dõi người dùng khác; MUST NOT cho phép tự theo dõi chính mình; thao tác idempotent.
- **FR-013**: Hệ thống MUST cung cấp danh sách người mà user đang theo dõi (`following`) và danh sách người đang theo dõi user đó (`followers`), có phân trang.
- **FR-014**: Hệ thống MUST hỗ trợ tìm/lọc trong danh sách follow theo `type` là `following` hoặc `followers`; `type` là **tuỳ chọn**, khi bỏ trống hệ thống trả về cả hai nhóm và mỗi mục MUST mang trường `relation` cho biết quan hệ (`following` hoặc `follower`).
- **FR-015**: Hệ thống MUST cung cấp trang cá nhân công khai truy cập bằng username, hiển thị tên, giới tính, ảnh đại diện, số lượng follower/following/bài đăng và trạng thái theo dõi của người xem.
- **FR-016**: Hệ thống MUST xử lý username/email so khớp không phân biệt hoa thường.
- **FR-017**: Hệ thống MUST cung cấp tìm kiếm tổng hợp với tham số `q`, trả về hai tập kết quả độc lập (users, posts), mỗi tập phân trang riêng.
- **FR-018**: Với tìm kiếm bài đăng, hệ thống MUST chỉ so khớp trong phạm vi **200 ký tự đầu** của nội dung; ngưỡng này MUST cấu hình được.
- **FR-019**: Hệ thống MUST cho phép lọc bảng tin theo `type` = `explore` hoặc `following`, mặc định `explore` khi caller không truyền.
- **FR-020**: Bảng tin `following` MUST chỉ chứa bài của những người người dùng đang theo dõi (và bài của chính người dùng nếu có).
- **FR-021**: Hệ thống MUST hỗ trợ sắp xếp `hot` (mặc định) và `latest`; `hot` tính tại thời điểm truy vấn dựa trên tương tác và suy giảm theo thời gian, giới hạn trong một cửa sổ thời gian cấu hình được, không dùng bảng điểm lưu sẵn; cơ chế này MUST áp dụng chung cho cả `explore` và `following`.
- **FR-022**: Điểm hot MUST chỉ tính trên tập bài trong cửa sổ thời gian cấu hình (MUST NOT full-table scan).
- **FR-023**: Hệ thống MUST hỗ trợ phân trang cho bảng tin, danh sách like, danh sách follow và kết quả tìm kiếm. Danh sách bình luận gốc và phản hồi MUST giữ dạng mảng (không phân trang) để tương thích frontend hiện có.
- **FR-024**: Hệ thống MUST cung cấp endpoint lấy chữ ký tải media trực tiếp (ảnh và video) cho bài đăng.
- **FR-025**: Hệ thống MUST cung cấp cho quản trị viên khả năng xem danh sách, ẩn/xóa và khôi phục bài đăng và bình luận.
- **FR-026**: Nội dung bị ẩn/xóa MUST NOT xuất hiện trong bảng tin, tìm kiếm, danh sách comment hoặc trang cá nhân đối với người dùng thường.
- **FR-027**: Hệ thống MUST bảo vệ mọi thao tác ghi (đăng bài, like, comment, follow) sau xác thực; MUST từ chối người dùng không hợp lệ hoặc bị vô hiệu hóa.
- **FR-028**: Hệ thống MUST trả về thông báo lỗi tiếng Việt đúng chuẩn lỗi chung cho mọi trường hợp thất bại.
- **FR-029**: Hệ thống MUST giữ tương thích các đường dẫn/envelope/DTO hiện có cho bài đăng, like, comment, upload; loại bỏ các trường liên quan bán lại.
- **FR-030**: Hệ thống MUST áp dụng giới hạn tần suất phù hợp cho các thao tác nhạy cảm (đăng bài, comment, follow) để chống lạm dụng.
- **FR-031**: Hệ thống MUST bảo đảm không tạo bản ghi like/follow trùng khi có yêu cầu đồng thời.
- **FR-032**: Hệ thống MUST giữ đúng thứ tự media theo `sortOrder` khi lưu và khi trả về.
- **FR-033**: Hệ thống MUST cho phép khách chưa đăng nhập xem công khai feed, bài đăng, bình luận, trang cá nhân và kết quả tìm kiếm ở chế độ chỉ đọc; mọi thao tác ghi MUST yêu cầu đăng nhập.
- **FR-034**: Guest gọi bảng tin `type=following` MUST nhận lỗi xác thực (401).
- **FR-035**: Bài của người dùng đang bị vô hiệu hóa/khóa MUST NOT hiển thị công khai (giữ dữ liệu, không tự động xóa). Cơ chế: khi trạng thái user thay đổi, hệ thống MUST ẩn/hiện bài tương ứng qua sự kiện bất đồng bộ `identity.user.status.changed` (đặt `status=hidden` kèm `hidden_reason=author_disabled`, khôi phục khi user hoạt động lại), MUST NOT join xuyên module.
- **FR-036**: Việc xóa bài/bình luận MUST là xóa mềm. Bài do tác giả tự xóa MUST NOT khôi phục được kể cả bởi quản trị viên; quản trị viên ẩn/xóa và khôi phục nội dung do quản trị viên ẩn/xóa. Tác giả MUST vẫn thấy bài của mình khi bị ẩn, kèm trạng thái "đang bị ẩn".
- **FR-037**: Hệ thống MUST giới hạn tối đa 10 media mỗi bài (ảnh ≤10MB định dạng jpg/png/webp; video ≤100MB định dạng mp4/webm, thời lượng ≤60 giây), `title` ≤150 ký tự, `content` ≤5000 ký tự, bình luận ≤1000 ký tự; các ngưỡng này MUST cấu hình được.
- **FR-038**: Hệ thống MUST cho phép cập nhật bài đổi sang outfit khác thuộc sở hữu nhưng MUST NOT cho phép đổi loại bài; một outfit MAY được tham chiếu bởi nhiều bài.
- **FR-039**: Hệ thống MUST xếp bình luận gốc mới nhất trước và phản hồi trong mỗi bình luận gốc cũ nhất trước; MUST trả về số lượng phản hồi của bình luận gốc; MUST NOT nhúng sẵn danh sách phản hồi trong dữ liệu feed.
- **FR-040**: Khi bình luận gốc bị xóa và còn phản hồi, hệ thống MUST hiển thị dấu "bình luận đã bị xóa"; khi không còn phản hồi, hệ thống MUST ẩn hẳn khỏi danh sách.
- **FR-041**: Hệ thống MUST sắp danh sách follow theo lần theo dõi mới nhất trước và MUST hỗ trợ `q` (tuỳ chọn) để tìm trong danh sách.
- **FR-042**: Search tổng hợp MUST tìm người dùng theo cả username lẫn họ tên, không phân biệt hoa thường; `q` MUST tối thiểu 1 ký tự và nếu rỗng sau khi trim MUST trả kết quả rỗng an toàn.
- **FR-043**: Kết quả search MUST sắp người dùng theo số follower giảm dần rồi mới nhất, và sắp bài đăng theo hot. Do dữ liệu user thuộc sở hữu Account còn số follower thuộc community, hệ thống MUST lấy tối đa N ứng viên user khớp từ Account (N cấu hình được, mặc định 500), enrich số follower rồi sắp xếp và phân trang.
- **FR-044**: Hệ thống MUST áp dụng giới hạn tần suất cấu hình được cho các thao tác ghi (mặc định: đăng bài 10/giờ, bình luận 60/giờ, theo dõi 200/giờ, like 600/giờ).
- **FR-045**: Hệ thống MUST ghi nhật ký kiểm toán cho các hành động kiểm duyệt; moderation thực hiện từng mục một (không bulk).
- **FR-046**: Bản giao của tính năng MUST bao gồm một tài liệu tích hợp frontend cho toàn bộ spec 022, mô tả đầy đủ: (a) các phần API giữ nguyên so với code frontend hiện có, (b) các phần API mới, (c) các phần API chỉnh sửa/thay đổi (trường thêm/bỏ/đổi ngữ nghĩa), kèm đặc tả request/response và hướng dẫn cập nhật cho từng phần.
- **FR-047**: Việc đồng bộ bài đăng khi outfit bị xóa MUST được xử lý bất đồng bộ và đạt nhất quán trong vòng tối đa 1 phút.

### Key Entities *(include if feature involves data)*

- **Bài đăng (Post)**: nội dung do người dùng tạo; thuộc một người dùng; có loại (`outfit`/`media`), tiêu đề tuỳ chọn, nội dung, tham chiếu outfit (chỉ với loại outfit), bộ đếm like/comment, trạng thái hiển thị và thời điểm tạo/cập nhật. Có định danh công khai dùng trên đường dẫn.
- **Media bài đăng (Post Media)**: ảnh/video gắn với một bài; có loại, đường dẫn, định danh công khai, thứ tự. Thuộc sở hữu của bài và bị xóa theo bài.
- **Lượt thích (Post Like)**: quan hệ giữa người dùng và bài; mỗi cặp là duy nhất.
- **Bình luận (Post Comment)**: nội dung gắn với bài và người dùng; có tham chiếu tới comment gốc (nếu là phản hồi), đảm bảo tối đa 2 cấp; có trạng thái hiển thị.
- **Quan hệ theo dõi (Follow)**: quan hệ một chiều giữa người theo dõi và người được theo dõi; mỗi cặp là duy nhất; cấm tự theo dõi.
- **Hồ sơ công khai (Public Profile)**: góc nhìn công khai của một người dùng theo username, gồm tên, giới tính, ảnh đại diện và các số đếm quan hệ/bài đăng.
- **Outfit (tham chiếu)**: thực thể tủ đồ được bài outfit tham chiếu; khi bị xóa kéo theo bài đăng tương ứng.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Ở tải tham chiếu 50 request/giây, p95 của thao tác ghi (đăng bài/like/comment/follow) và tải danh sách (feed/danh sách) dưới 500ms; p95 tìm kiếm dưới 1 giây.
- **SC-002**: 100% bài loại outfit tham chiếu một outfit bị xóa đều không còn hiển thị cho người dùng thường trong vòng tối đa 1 phút sau khi outfit bị xóa.
- **SC-003**: 0 bản ghi like/follow trùng lặp được tạo ra trong kiểm thử đồng thời.
- **SC-004**: 100% bình luận được lưu chỉ ở tối đa 2 cấp; không tồn tại cấp 3.
- **SC-005**: Kết quả tìm kiếm không phân biệt hoa thường và không khớp nội dung nằm ngoài 200 ký tự đầu (0 sai lệch trong bộ kiểm thử).
- **SC-006**: Thứ tự bảng tin `hot` khác thứ tự thời gian thuần khi có tương tác khác biệt (thể hiện qua bài cũ tương tác cao xếp trên bài mới tương tác thấp).
- **SC-007**: Bảng tin `following` chỉ chứa bài của người được theo dõi (tỷ lệ chính xác 100% trong bộ kiểm thử).
- **SC-008**: 100% nội dung bị ẩn/xóa không còn hiển thị với người dùng thường nhưng vẫn quản trị được.
- **SC-009**: Người kiểm thử hoàn thành luồng "đăng bài → like → bình luận → theo dõi → xem trang cá nhân → tìm kiếm" trong dưới 5 phút mà không gặp lỗi.
- **SC-010**: Đội frontend có thể triển khai toàn bộ tính năng 022 chỉ dựa trên tài liệu tích hợp frontend (giữ nguyên + mới + thay đổi), không cần đọc mã nguồn backend; 100% endpoint/thay đổi hợp đồng được tài liệu bao phủ.
- **SC-011**: 100% giới hạn nội dung (số media, kích thước/định dạng/thời lượng, độ dài ký tự) được thực thi và trả thông báo lỗi rõ ràng khi vượt.

## Out of Scope

- Thông báo (notification) cho like/comment/follow.
- Tiểu sử (bio), liên kết ngoài, hay thông tin mở rộng khác trên trang cá nhân.
- Chế độ riêng tư, bài chỉ người theo dõi, chặn (block), báo cáo (report), ẩn danh.
- Bán lại/đổi chủ trang phục (resale/transfer) và mọi endpoint liên quan.
- Đồng bộ bài đăng khi outfit bị chỉnh sửa (chỉ cascade khi xóa).
- Đề xuất/ranking cá nhân hoá theo sở thích, tìm kiếm nâng cao (filter theo thời gian/loại/...).
- Kiểm duyệt tự động bằng AI cho nội dung cộng đồng.
- Đa ngôn ngữ cho nội dung người dùng tạo.

## Assumptions

- Người dùng cộng đồng là người dùng B2C đã đăng nhập; khách chưa đăng nhập chỉ xem được nội dung công khai (bảng tin, bài, comment, trang cá nhân) ở chế độ chỉ đọc.
- Mọi bài đăng mặc định công khai (không có chế độ riêng tư trong đợt này).
- Người dùng đã có tủ đồ và outfit hợp lệ trước khi đăng bài outfit; bài outfit tham chiếu outfit hiện hữu.
- Mỗi người dùng có username duy nhất (so khớp không phân biệt hoa thường), dùng làm định danh công khai cho trang cá nhân.
- Nền tảng đã có sẵn cơ chế tải media trực tiếp qua dịch vụ lưu trữ media hiện hành và cơ chế xử lý sự kiện bất đồng bộ (outbox + hàng đợi) để cascade/tìm kiếm.
- Nền tảng có sẵn cơ chế tìm kiếm với tuỳ chọn tắt và phương án dự phòng truy vấn trực tiếp; đợt này có thể dùng phương án dự phòng.
- Mặc định "bảng tin" trả về nội dung công khai và không bao gồm bài đã ẩn/xóa.
- Bộ đếm like/comment/follow được hiển thị ở mức "đủ chính xác"; sai lệch tạm thời trong tích tắc dưới tải cao là chấp nhận được miễn hội tụ nhanh.
- Giao diện hiện tại của frontend có thể phải cập nhật, nhưng các đường dẫn và cấu trúc phản hồi cốt lõi được giữ ổn định để giảm chi phí phía frontend.
- Phân trang dùng đúng chuẩn phân trang hiện có của dự án (page/limit, metadata `page/limit/totalItems/totalPages`); không dùng con trỏ (cursor) trong đợt này.
- Nội dung công khai mặc định; khách chỉ đọc; không có cơ chế riêng tư trong đợt này.
- Bộ đếm quan hệ và tương tác được lưu sẵn và đối soát định kỳ; sai lệch tạm thời ngắn dưới tải cao được chấp nhận miễn hội tụ nhanh.
- Tài liệu tích hợp frontend là một sản phẩm bàn giao chính thức của tính năng (nằm trong thư mục spec), không phải tùy chọn.
