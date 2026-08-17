# Feature Specification: Brand RBAC Tập Trung & Bổ Sung Nghiệp Vụ Quản Trị Admin

**Feature Branch**: `012-brand-rbac-admin`

**Created**: 2026-08-17 | **Last Updated**: 2026-08-17

**Status**: Draft

**Input**: User description: "Tập trung Brand RBAC về middleware thay vì check trong từng usecase; xử lý lỗ hổng IDOR Brand Dashboard; bổ sung toàn bộ nghiệp vụ quản trị admin còn thiếu (payment resolution, subscription/plan, user admin, brand member admin, AI executor admin, audit log); phân quyền rõ ma trận brand owner/staff; đồng bộ mâu thuẫn docs-code lấy code làm chuẩn; các mục out-scope (RBAC toàn hệ thống, campaign, support ticket, virtual trial, notification đa kênh, swap/re-roll AI, AI OOTD/body-shape/garment decay) được đưa vào docs."

## User Scenarios & Testing

Các user story được xếp theo thứ tự ưu tiên. Mỗi story là một lát cắt nghiệp vụ độc lập, phát triển/kiểm thử/triển khai độc lập được.

---

### User Story 1 - Khóa lỗ hổng IDOR Brand Dashboard (Priority: P1)

Trước đây, bất kỳ người dùng đã đăng nhập nào cũng có thể đọc dữ liệu thống kê (khách hàng, điểm thưởng, doanh thu voucher...) của **mọi nhãn hàng** bằng cách truyền `brandId` bất kỳ vào endpoint brand dashboard, vì hệ thống không xác minh người dùng đó có phải thành viên của nhãn hàng hay không. Lỗ hổng này phải được đóng lại: chỉ thành viên đang hoạt động của nhãn hàng (chủ sở hữu/nhân viên) mới xem được dữ liệu dashboard của nhãn hàng đó.

**Why this priority**: Đây là lỗ hổng bảo mật nghiêm trọng (IDOR) làm lộ dữ liệu kinh doanh giữa các nhãn hàng. Ưu tiên cao nhất vì rủi ro pháp lý/uy tín; không cần bất kỳ tính năng nào khác triển khai trước.

**Independent Test**: Tạo tài khoản user thường (không phải member của brand A), gọi `GET /brand/dashboard/kpi-cards?brandId=<brandA>` → phải nhận 403. User là staff của brand A → nhận 200. User là owner của brand B → gọi brand A vẫn 403.

**Acceptance Scenarios**:

1. **Given** user đã đăng nhập nhưng không phải member active của brand A, **When** gọi bất kỳ endpoint `/brand/dashboard/*` hoặc `/brand/dashboard/drilldown/*` với `brandId=A`, **Then** nhận lỗi 403 (từ chối truy cập) và không trả dữ liệu nào của brand A
2. **Given** user là staff hoặc owner active của brand A, **When** gọi `/brand/dashboard/*` với `brandId=A`, **Then** nhận dữ liệu hợp lệ (200)
3. **Given** user là admin hệ thống (không phải member), **When** gọi `/brand/dashboard/*` với `brandId=A`, **Then** vẫn nhận dữ liệu (admin được phép bypass để hỗ trợ vận hành)
4. **Given** user gọi `/dashboard/analytics/query` với `targetDashboard=brand` và `brandId=A`, **When** user không phải member active của A, **Then** nhận 403
5. **Given** brand A đang ở trạng thái không hoạt động (suspended/archived), **When** member gọi dashboard brand A, **Then** nhận lỗi tương ứng và không trả dữ liệu

---

### User Story 2 - Brand RBAC tập trung vào middleware + cache membership (Priority: P1)

Việc kiểm tra quyền owner/staff hiện bị rải rác trong ~30 usecase, mỗi lần gọi đều truy vấn database. Cần đưa toàn bộ kiểm tra này lên middleware HTTP, chia brand portal thành 2 nhóm quyền rõ ràng (member = owner+staff; owner-only = cấu hình loyalty), và cache thông tin membership theo người dùng để giảm số lần truy vấn database.

**Why this priority**: Điều kiện tiên quyết để thêm route portal mới an toàn (không sợ quên check) và giảm tải DB. Gắn liền với User Story 1 (cùng cơ chế).

**Independent Test**: Mọi route `/brand-portal/brands/:brandId/*` không còn chứa lệnh kiểm tra quyền trong usecase; khi gọi route owner-only bằng tài khoản staff → 403; bằng owner → 200. Kiểm tra Redis có key cache membership sau lần gọi đầu tiên.

**Acceptance Scenarios**:

1. **Given** user là staff active của brand A, **When** gọi `PUT /brand-portal/brands/A/loyalty/program` hoặc `POST/PUT /brand-portal/brands/A/loyalty/tiers*`, **Then** nhận 403
2. **Given** user là owner active của brand A, **When** gọi các route owner-only nêu trên, **Then** nhận 200
3. **Given** user là owner/staff active của brand A, **When** gọi bất kỳ route portal còn lại của brand A, **Then** nhận 200
4. **Given** user không phải member active của brand A (hoặc brand A không active), **When** gọi route portal của brand A, **Then** nhận 403
5. **Given** một request portal, **When** kiểm tra giá trị context, **Then** có `brandId` và `brandRole` (role của user trong brand) được lưu theo chuẩn context của dự án (không hardcode key)
6. **Given** member thay đổi (thêm/đổi role/khóa) hoặc brand đổi trạng thái, **When** member gọi lại portal/dashboard, **Then** thông tin membership phản ánh thay đổi trong thời gian không quá TTL cache (invalidation kịp thời)
7. **Given** user thuộc nhiều brand (A, B), **When** gọi portal brand A và brand B, **Then** truy cập đúng theo membership từng brand, không lẫn lộn

---

### User Story 3 - Ma trận quyền owner/staff được tài liệu hóa (Priority: P2)

Quyền của chủ sở hữu (owner) và nhân viên (staff) trong brand portal hiện chỉ thể hiện trong code, tài liệu không nhất quán. Cần một tài liệu ma trận phân quyền chính thức, làm chuẩn cho cả middleware và docs API.

**Why this priority**: Loại bỏ mâu thuẫn tài liệu hiện tại, làm cơ sở duy nhất cho việc áp dụng quyền; không block vận hành nhưng cần trước khi thêm tính năng mới.

**Independent Test**: Đọc tài liệu ma trận có thể xác định được mỗi route portal thuộc nhóm nào; đối chiếu ma trận với middleware triển khai ở User Story 2 không có mâu thuẫn.

**Acceptance Scenarios**:

1. **Given** tài liệu ma trận quyền, **When** tra cứu route cấu hình loyalty (program/tiers), **Then** ghi rõ chỉ owner được thao tác
2. **Given** tài liệu ma trận quyền, **When** tra cứu các route vận hành (members, customers, claim-token, offline-purchase, points, benefits, conversations, items, dashboard), **Then** ghi rõ owner và staff đều được thao tác
3. **Given** tài liệu ma trận, **When** đối chiếu với middleware, **Then** không có route nào nằm sai nhóm quyền

---

### User Story 4 - Admin xử lý giao dịch thanh toán bất thường (Priority: P2)

Hệ thống thanh toán có các giao dịch cần kiểm tra thủ công (investigation required / reconciliation required). Nghiệp vụ xử lý đã có sẵn nhưng chưa có API cho quản trị viên dùng. Cần bổ sung API để admin xem danh sách/chi tiết giao dịch và thực hiện các thao tác: gửi lại kiểm tra, xác nhận đã thanh toán, xác nhận đã hủy, đánh dấu tạo link thất bại.

**Why this priority**: Vận hành thanh toán (đối soát thủ công) là nghiệp vụ quản trị quan trọng, hiện đang bị "treo" — đã có logic nhưng không dùng được.

**Independent Test**: Admin gọi danh sách giao dịch, chọn một giao dịch ở trạng thái cần kiểm tra, thao tác "xác nhận đã thanh toán" → giao dịch chuyển trạng thái và ví/đăng ký được cập nhật; user không phải admin gọi → 403.

**Acceptance Scenarios**:

1. **Given** admin đăng nhập, **When** gọi danh sách giao dịch (lọc theo trạng thái/thời gian/phân trang), **Then** nhận danh sách hợp lệ
2. **Given** giao dịch đang ở trạng thái cần kiểm tra, **When** admin gọi "retry investigation", **Then** giao dịch chuyển về trạng thái chờ đối soát lại
3. **Given** giao dịch hợp lệ kèm bằng chứng thanh toán, **When** admin gọi "resolve as paid", **Then** giao dịch được xác nhận đã thanh toán và ví/đăng ký được cập nhật
4. **Given** giao dịch, **When** admin gọi "resolve as cancelled" hoặc "resolve as creation failed", **Then** giao dịch chuyển trạng thái tương ứng (không cập nhật ví)
5. **Given** thao tác không hợp lệ (trạng thái hiện tại không khớp), **When** admin thực hiện, **Then** nhận lỗi rõ ràng và không thay đổi dữ liệu
6. **Given** user thường, **When** gọi các endpoint admin này, **Then** nhận 403

---

### User Story 5 - Admin quản lý subscription & gói cước (Priority: P2)

Quản trị viên cần quản lý danh sách gói cước (thêm/sửa/bật-tắt) và danh sách đăng ký của người dùng (xem, hủy gói, cấp gói thủ công) khi người dùng gặp sự cố thanh toán.

**Why this priority**: Hỗ trợ vận hành (hủy/cấp gói thủ công) giúp giữ khách hàng khi thanh toán lỗi; gói cước hiện không có giao diện quản trị.

**Independent Test**: Admin tạo một gói mới → xuất hiện trong danh sách gói (công khai); admin hủy gói của một user → user hạ cấp về free; admin cấp gói cho user → user nhận quyền tương ứng (bao gồm grant AI).

**Acceptance Scenarios**:

1. **Given** admin đăng nhập, **When** tạo/sửa/bật-tắt gói cước, **Then** danh sách gói (bao gồm public) phản ánh đúng thay đổi
2. **Given** gói đang bị tắt, **When** user thường gọi danh sách gói công khai, **Then** gói tắt không xuất hiện
3. **Given** user có gói premium, **When** admin hủy gói, **Then** user hạ cấp về free và quyền AI tương ứng bị thu hồi
4. **Given** user đang free, **When** admin cấp gói premium, **Then** user nhận gói premium và grant AI được đồng bộ
5. **Given** admin gọi danh sách subscription (lọc theo gói/trạng thái), **Then** nhận danh sách hợp lệ

---

### User Story 6 - Admin quản lý tài khoản người dùng (Priority: P2)

Quản trị viên hiện chỉ xem danh sách và khóa/mở khóa tài khoản. Cần bổ sung: xem chi tiết một người dùng, tạo người dùng thủ công, đổi vai trò (user/admin), vô hiệu hóa (soft) và quản lý phiên đăng nhập (xem/revoke).

**Why this priority**: Hỗ trợ vận hành nội bộ (tạo tài khoản đối tác, xử lý tài khoản vi phạm, thu hồi phiên bảo mật). Cần đảm bảo an toàn: admin không thể khóa/đổi chính mình hay thao tác lên admin khác bất cẩn.

**Independent Test**: Admin xem chi tiết một user; tạo user mới (role user) → đăng nhập được; cố đổi role một user thành admin → bị chặn; vô hiệu hóa → user không đăng nhập được; revoke phiên → token cũ hết hiệu lực.

**Acceptance Scenarios**:

1. **Given** admin đăng nhập, **When** xem chi tiết user (thông tin + subscription overview), **Then** nhận thông tin hợp lệ
2. **Given** admin tạo user mới (username/email/password/role user), **When** người dùng mới đăng nhập, **Then** đăng nhập thành công với đúng role
3. **Given** admin thường đổi role một user thành admin, **When** thực hiện, **Then** bị chặn (chỉ tài khoản admin seed mới được tạo/phong admin)
4. **Given** admin vô hiệu hóa một user, **When** user đó đăng nhập, **Then** bị từ chối và toàn bộ phiên cũ bị thu hồi
5. **Given** admin đang thao tác lên chính tài khoản của mình hoặc admin khác, **When** cố đổi role/khóa, **Then** bị chặn (không cho thực hiện)
6. **Given** admin xem danh sách phiên của một user và chọn revoke, **When** hoàn tất, **Then** các refresh token/phiên đó hết hiệu lực

---

### User Story 7 - Admin quản lý thành viên nhãn hàng (Priority: P3)

Quản trị viên hệ thống cần can thiệp vào thành viên của nhãn hàng (thêm nhân viên, đổi vai trò/trạng thái, chuyển quyền sở hữu, gỡ thành viên) khi cần hỗ trợ hoặc xử lý tranh chấp.

**Why this priority**: Hỗ trợ vận hành cấp hệ thống cho các nhãn hàng (hiện chỉ owner thao tác được). Không khẩn cấp bằng các mục P2.

**Independent Test**: Admin thêm một user làm staff của brand A → user đó truy cập được portal brand A; admin chuyển ownership từ user X sang Y → Y trở thành owner; admin gỡ member → không truy cập được nữa.

**Acceptance Scenarios**:

1. **Given** admin đăng nhập, **When** xem danh sách member của brand, **Then** nhận danh sách hợp lệ
2. **Given** admin thêm một user làm staff của brand, **When** user đó đăng nhập, **Then** truy cập được portal của brand
3. **Given** admin đổi vai trò/trạng thái của một member, **When** member dùng lại hệ thống, **Then** quyền phản ánh thay đổi
4. **Given** admin chuyển ownership giữa hai người dùng, **When** hoàn tất, **Then** người nhận trở thành owner; người chủ cũ được xử lý theo lựa chọn của admin khi chuyển (hạ xuống staff hoặc gỡ hoàn toàn)
5. **Given** admin gỡ một member, **When** member gọi portal brand, **Then** bị từ chối (403)

---

### User Story 8 - Admin quản trị AI executor (pricing, usage, grant) (Priority: P3)

Quản trị viên cần xem/sửa cấu hình giá mô hình AI, xem mức sử dụng AI theo người dùng, và xem/điều chỉnh grant chính sách AI — hiện toàn bộ chỉ chạy nội bộ (worker) hoặc qua dashboard.

**Why this priority**: Kiểm soát chi phí AI và hỗ trợ vận hành khi có sự cố về hạn mức; không khẩn cấp bằng bảo mật/thanh toán.

**Independent Test**: Admin xem danh sách giá model; sửa giá → áp dụng cho lần tính chi phí tiếp theo; xem usage theo user; tạo grant cho user → user được mở rộng hạn mức.

**Acceptance Scenarios**:

1. **Given** admin đăng nhập, **When** xem danh sách cấu hình giá model AI, **Then** nhận danh sách hợp lệ
2. **Given** admin cập nhật giá của một model, **When** phát sinh chi phí AI mới, **Then** chi phí tính theo giá mới
3. **Given** admin xem mức sử dụng AI theo user (lọc khoảng thời gian), **Then** nhận dữ liệu sử dụng/chi phí hợp lệ
4. **Given** admin xem danh sách grant của một user và tạo/điều chỉnh grant, **Then** hạn mức AI của user phản ánh thay đổi

---

### User Story 9 - Audit log cho hành động quản trị (Priority: P3)

Mọi hành động nhạy cảm của quản trị viên (đổi trạng thái user, duyệt/suspend brand, thay đổi gói, xử lý thanh toán, quản lý member, sửa giá AI...) phải được ghi lại để truy vết: ai làm gì, với đối tượng nào, lúc nào, từ IP nào.

**Why this priority**: Tuân thủ và truy vết khi có tranh chấp/điều tra; hiện hệ thống không có nhật ký audit tổng hợp.

**Independent Test**: Admin thực hiện một hành động (vd khóa user), kiểm tra bảng audit có bản ghi với đúng actor/action/target; user thường không có endpoint nào ghi audit được.

**Acceptance Scenarios**:

1. **Given** admin thực hiện một hành động quản trị được liệt kê, **When** hoàn tất, **Then** có bản ghi audit mới (actor, action, target, thời gian, IP)
2. **Given** có bản ghi audit, **When** truy vấn theo actor/target/thời gian, **Then** trả về đúng thông tin đã ghi
3. **Given** người dùng thường, **When** thực hiện hành động cá nhân (đổi mật khẩu...), **Then** không tạo bản ghi audit quản trị

---

### User Story 10 - Đồng bộ tài liệu docs-code và ghi nhận out-scope (Priority: P3)

Các mâu thuẫn giữa tài liệu và code cần được sửa theo hướng lấy code làm chuẩn; đồng thời các nghiệp vụ ngoài phạm vi (RBAC toàn hệ thống, campaign, support ticket, virtual trial, notification đa kênh, swap/re-roll AI, AI OOTD generator, body-shape validation, garment decay) cần được ghi nhận chính thức trong docs để tránh hiểu lầm sau này.

**Why this priority**: Cải thiện chất lượng tài liệu; không block vận hành nhưng cần hoàn tất để đóng feature.

**Independent Test**: Mở tài liệu đã sửa, đối chiếu với code (hằng số transaction_type, plan code, khái niệm role, ma trận quyền) không còn mâu thuẫn; tài liệu out-of-scope liệt kê đầy đủ các mục nêu trên.

**Acceptance Scenarios**:

1. **Given** tài liệu `dashboard-api-v2.md`, **When** tra cứu giá trị transaction_type, **Then** dùng chữ thường `direct_purchase` khớp code
2. **Given** tài liệu plan code, **When** tra cứu, **Then** phản ánh đúng tập gói thực tế (`free`, `premium-monthly`)
3. **Given** tài liệu kiến trúc, **When** đọc khái niệm người dùng theo gói, **Then** được mô tả là "hạng theo gói subscription", không gọi là role hệ thống
4. **Given** tài liệu ma trận quyền brand, **When** đối chiếu, **Then** không mâu thuẫn giữa các file docs
5. **Given** tài liệu out-of-scope, **When** đọc, **Then** liệt kê đủ 9 mục và lý do không làm

---

### Edge Cases

- **User thuộc nhiều brand**: quyền phải xác định theo cặp (user, brand), không theo user đơn lẻ; cache membership phải lưu theo user và hỗ trợ nhiều brand.
- **Stale cache sau thay đổi membership**: khi thêm/gỡ member, đổi role, đổi status brand → cần invalidate cache của user liên quan; thời gian lệch tối đa = TTL cache.
- **Admin thao tác lên chính mình / admin khác**: chặn tự khóa/đổi role bản thân, chặn admin thường thao tác lên tài khoản admin khác.
- **Phong/hạ admin**: chỉ tài khoản admin seed được tạo/phong admin mới; admin thường không được đổi role thành admin.
- **Retention audit log**: dữ liệu audit quá 12 tháng được dọn định kỳ; chỉ admin truy cập được bản ghi audit.
- **Owner cuối cùng**: không cho gỡ/chuyển ownership khi chỉ còn 1 owner (tránh brand mất người quản lý); khi chuyển ownership, admin chọn xử lý người chủ cũ (hạ staff hoặc gỡ hoàn toàn).
- **Giao dịch trạng thái không khớp khi resolve**: phải báo lỗi rõ ràng, không tự ý chuyển trạng thái.
- **Resolve as paid thiếu bằng chứng**: admin không cung cấp Reference/PaymentLinkID → báo lỗi, không cập nhật ví; admin phải dùng retry investigation hoặc resolve as cancelled thay thế.
- **Caching khi Redis không khả dụng**: nếu Redis lỗi, phải fallback về truy vấn database (không chặn người dùng hợp lệ).
- **Gói bị tắt khi user đang dùng**: hành vi rõ ràng (không gia hạn được, user hiện hữu vẫn dùng đến hết kỳ).
- **Endpoint public không thuộc phạm vi**: `/brands`, `/brand-items/:itemId` (public/optional-auth), `/subscriptions/plans`... không bị ảnh hưởng bởi middleware mới.

## Requirements

### Functional Requirements

- **FR-001**: Hệ thống MUST xác minh thành viên nhãn hàng (membership active + brand active) ở lớp middleware cho mọi route `/brand-portal/brands/:brandId/*` trước khi vào handler/usecase.
- **FR-002**: Hệ thống MUST phân brand portal thành 2 nhóm quyền: nhóm member (owner + staff) và nhóm owner-only (cấu hình loyalty program và loyalty tier). Route owner-only gồm: cập nhật loyalty program, tạo loyalty tier, cập nhật loyalty tier.
- **FR-003**: Hệ thống MUST xóa các lệnh kiểm tra quyền owner/staff khỏi usecase phục vụ HTTP (chuyển về middleware), đồng thời giữ helper kiểm tra cho các luồng nội bộ (worker) nếu còn dùng.
- **FR-004**: Hệ thống MUST kiểm tra quyền trả về 403 cho mọi endpoint `/brand/dashboard/*`, `/brand/dashboard/drilldown/*` khi người gọi không phải member active của brand, NGOẠI TRỪ admin hệ thống (được bypass).
- **FR-005**: Hệ thống MUST chặn truy vấn `targetDashboard=brand` tại `/dashboard/analytics/query` khi người gọi không phải member active của brand (không dựa vào role trả về mặc định).
- **FR-006**: Hệ thống MUST cache thông tin membership theo người dùng (hỗ trợ 1 user thuộc nhiều brand) để giảm truy vấn database; cache phải có TTL và được invalidate khi membership/brand thay đổi; khi cache không khả dụng phải fallback về database.
- **FR-007**: Hệ thống MUST lưu `brandId` và `brandRole` đã xác minh vào context của request theo chuẩn context hiện có của dự án (hằng số key + getter typed), không hardcode.
- **FR-008**: Hệ thống MUST sửa hành vi `GetUserBrandRole` để trả lỗi (thay vì mặc định coi là staff) khi user không phải member active của brand hoặc brand không active.
- **FR-009**: Hệ thống MUST cung cấp endpoint admin để danh sách/chi tiết giao dịch thanh toán, và thực hiện các thao tác: gửi lại kiểm tra, xác nhận đã thanh toán, xác nhận đã hủy, đánh dấu tạo link thất bại. Thao tác xác nhận đã thanh toán MUST bắt buộc có bằng chứng thanh toán (Reference + PaymentLinkID); thiếu bằng chứng → báo lỗi và không cập nhật ví/đăng ký.
- **FR-010**: Hệ thống MUST cung cấp endpoint admin quản lý gói cước (tạo/sửa/bật-tắt) và quản lý đăng ký người dùng (xem danh sách, hủy gói về free, cấp gói thủ công kèm đồng bộ grant AI).
- **FR-011**: Hệ thống MUST cung cấp endpoint admin quản lý người dùng: xem chi tiết, tạo người dùng, đổi vai trò (giới hạn trong vai trò user — không được phong admin), vô hiệu hóa (soft) và quản lý phiên đăng nhập (xem/revoke). Để hiển thị phiên hữu ích, hệ thống MUST ghi nhận thông tin thiết bị/IP/user-agent/lần hoạt động gần nhất tại thời điểm đăng nhập và refresh (bổ sung cột qua migration chuẩn).
- **FR-012**: Hệ thống MUST chỉ cho tài khoản admin seed được tạo/phong tài khoản admin mới; mọi admin khác MUST bị chặn khi cố đổi role thành admin, đổi role/khóa chính mình, hoặc thao tác lên tài khoản admin khác.
- **FR-013**: Hệ thống MUST cung cấp endpoint admin quản lý thành viên nhãn hàng: danh sách, thêm staff, đổi vai trò/trạng thái (kèm chuyển quyền sở hữu), gỡ thành viên.
- **FR-014**: Hệ thống MUST không cho gỡ/chuyển ownership khi brand còn duy nhất 1 owner active; khi chuyển ownership, admin MUST lựa chọn xử lý người chủ cũ: hạ xuống staff (vẫn truy cập portal) hoặc gỡ khỏi brand hoàn toàn.
- **FR-015**: Hệ thống MUST cung cấp endpoint admin cho AI executor: xem/sửa cấu hình giá model, xem mức sử dụng AI theo người dùng, xem/tạo/điều chỉnh grant chính sách AI.
- **FR-016**: Hệ thống MUST ghi lại nhật ký audit cho các hành động quản trị (actor, action, target type/id, nội dung, IP, thời gian) vào bảng dữ liệu chuyên dụng, tạo mới theo chuẩn migration hiện có; việc ghi MUST bất đồng bộ (không chặn nghiệp vụ chính) và đáng tin cậy (không mất bản ghi khi hệ thống message gặp sự cố — at-least-once); dữ liệu MUST được giữ 12 tháng với cơ chế dọn dẹp định kỳ (worker), và chỉ admin được xem/truy vấn qua endpoint admin.
- **FR-017**: Mọi endpoint admin mới MUST chỉ dành cho vai trò admin hệ thống (không phải user thường) và sử dụng cơ chế xác thực/bảo vệ route chuẩn.
- **FR-018**: Hệ thống MUST sửa các mâu thuẫn tài liệu theo chuẩn code: giá trị transaction_type chữ thường, tập gói cước thực tế, khái niệm "hạng gói subscription" thay vì "role", sửa lỗi liệt kê vai trò cũ, thống nhất ma trận quyền.
- **FR-019**: Hệ thống MUST ghi nhận chính thức trong docs các mục out-of-scope (RBAC toàn hệ thống giữ `users.role_slug`, campaign, support ticket, virtual trial, notification đa kênh + quản lý, swap/re-roll AI, AI OOTD generator, body-shape validation, garment decay) kèm lý do.
- **FR-020**: Mọi thay đổi database schema (bảng audit, nếu có bổ sung khác) MUST được triển khai qua file migration Goose theo chuẩn dự án (`-- +goose Up/Down`).

### Key Entities

- **BrandMember**: thành viên của nhãn hàng (user, brand, role owner/staff, status active/invited/disabled) — là đối tượng được cache và kiểm tra quyền.
- **Brand**: nhãn hàng (status pending_review/active/suspended/archived) — trạng thái active là điều kiện để truy cập portal/dashboard.
- **DepositTransaction / UserSubscription / SubscriptionPlan**: giao dịch thanh toán, đăng ký gói và gói cước — mục tiêu của nghiệp vụ admin 3A/3B.
- **User / RefreshToken**: người dùng và phiên đăng nhập — mục tiêu của nghiệp vụ admin 3C.
- **AIModelConfig / AIUsageEvent / AIPolicyGrant**: cấu hình giá AI, sự kiện sử dụng AI, grant chính sách — mục tiêu của nghiệp vụ admin 3E.
- **AdminAuditLog**: bản ghi nhật ký hành động quản trị (mới) — phục vụ truy vết.

## Success Criteria

### Measurable Outcomes

- **SC-001**: 100% endpoint brand dashboard/drilldown/analytics(brand) trả 403 cho người không phải member active (chỉ admin được bypass) — kiểm tra bằng kịch bản test IDOR.
- **SC-002**: 100% route brand portal không còn lệnh kiểm tra quyền trong usecase; quyền owner/staff được áp dụng thống nhất qua middleware.
- **SC-003**: Tỷ lệ request portal/dashboard không phát sinh truy vấn database để kiểm tra quyền đạt ≥ 90% khi cache hoạt động bình thường (đo bằng tỷ lệ cache hit so với tổng request).
- **SC-004**: Toàn bộ nghiệp vụ admin 3A–3F có endpoint hoạt động: payment resolution (4 thao tác + list/detail), plan CRUD + subscription list/cancel/grant, user admin (detail/create/role/deactivate/sessions), brand member admin (list/add/role/transfer/remove), AI executor (pricing/usage/grants), audit log ghi nhận mọi hành động admin.
- **SC-005**: Không có breaking change với các endpoint/cơ chế hiện có ngoài phạm vi; `make check` (fmt, tidy, generate-check, vet, test, build) pass.
- **SC-006**: 100% mâu thuẫn docs-code trong phạm vi (transaction_type, plan code, khái niệm role, ma trận quyền) được sửa theo chuẩn code; tài liệu out-of-scope có đủ 9 mục.
- **SC-007**: Không có hành vi mới nào vi phạm hiến pháp dự án (module boundary qua contract, DI qua Wire, migration Goose, apperror, client message tiếng Việt).

## Assumptions

- **Phạm vi RBAC**: Giữ nguyên cơ chế phân quyền hệ thống hiện tại (`users.role_slug` chỉ có admin/user) — không xây RBAC toàn hệ thống (out-of-scope). Feature này chỉ tập trung Brand RBAC (owner/staff) và bổ sung nghiệp vụ admin.
- **Admin và brand dashboard**: Admin hệ thống được phép xem dữ liệu mọi brand dashboard (bypass membership) để hỗ trợ vận hành — giữ hiện trạng, ghi rõ trong ma trận.
- **Cache membership**: Dùng Redis cache sẵn có của dự án, cache theo userID (danh sách membership nhiều brand). TTL ngắn (mặc định đề xuất 5 phút, có cấu hình) + invalidation khi có thay đổi member/brand; fallback về database khi Redis lỗi. Đánh đổi: thay đổi quyền được phản ánh chậm tối đa = TTL.
- **Kiểm tra quyền ở middleware là điểm chặn chính**: bỏ check trong usecase HTTP; giữ helper `RequireBrandRoleShared` (hiện có) cho các luồng nội bộ nếu còn dùng; worker nền không đi qua các method đã gỡ check.
- **Ma trận quyền owner/staff**: owner-only = cấu hình loyalty program + tier; còn lại (mọi route portal khác) = owner + staff. Đây là chuẩn từ code hiện tại.
- **Audit log**: tạo bảng mới `admin_audit_logs` qua migration Goose; ghi **bất đồng bộ qua outbox + RabbitMQ** (handler ghi outbox record → OutboxWorker publish → consumer module `admin` ghi bảng audit), áp dụng cho các handler admin (kể cả 2 route admin hiện có: đổi status user, đổi status/duyệt brand).
- **Nhận diện admin seed**: quản lý vận hành bằng SQL seed trực tiếp (team tự seed); KHÔNG cần cơ chế nhận diện admin seed lúc runtime trong code.
- **Ngôn ngữ**: message trả client/validation tiếng Việt, code/comment tiếng Anh, DTO JSON camelCase — theo chuẩn dự án.
- **Phụ thuộc**: module aiexecutor hiện chưa có tầng HTTP — cần dựng handler/router/provider mới và đăng ký vào Wire DI. Điều này cần thận trọng để không vi phạm ranh giới module (chỉ dùng contract/DTO).
- **Chuẩn migration**: mọi thay đổi DB qua `make migration-create` + `make migration-up`; không sửa trực tiếp file init.

## Clarifications

### Session 2026-08-17

- Q: Admin (không phải member) có xem được brand dashboard mọi brand không? → A: Có — admin được bypass membership check tại `/brand/dashboard/*` (giữ hiện trạng), ghi rõ trong ma trận. Portal brand vẫn yêu cầu membership kể cả admin.
- Q: Phạm vi endpoint admin Phase 3? → A: Đầy đủ 3A→3F (payment resolution, subscription/plan, user admin, brand member admin, AI executor admin, audit log).
- Q: Hướng audit log? → A: Tạo bảng mới `admin_audit_logs` qua migration Goose chuẩn dự án.
- Q: Cache kiểm tra membership? → A: Dùng Redis cache theo userID (hỗ trợ nhiều brand), TTL ngắn + invalidation khi member/brand đổi + fallback DB khi Redis lỗi.
- Q: Lưu context như thế nào? → A: Theo chuẩn `contextutils` hiện có (hằng số key + getter typed), không hardcode.
- Q: Vị trí tài liệu ma trận quyền? → A: Đặt tại `docs/system-design/brand-permission-matrix.md` (folder thiết kế hệ thống), reference từ `docs/api/constants/brand.md`.
- Q: Admin có được phong/hạ admin khác không? → A: Option C — Endpoint đổi role không được phép đổi thành admin; chỉ tài khoản admin seed được tạo/phong admin mới. Admin thường chỉ quản lý user (đổi trạng thái, tạo user role user, quản lý phiên); chặn thao tác lên admin khác và chính mình.
- Q: Audit log lưu bao lâu và ai xem? → A: Option B — Giữ 12 tháng với cleanup định kỳ (worker); chỉ admin được xem/truy vấn qua endpoint admin.
- Q: Xác nhận giao dịch đã thanh toán có cần bằng chứng không? → A: Option A — Bắt buộc bằng chứng (Reference + PaymentLinkID); không cho override không bằng chứng; thiếu bằng chứng dùng retry investigation hoặc resolve as cancelled.
- Q: Mức độ thông tin phiên đăng nhập? → A: Option A — Thêm cột theo dõi phiên (device/IP/UA/last used) qua migration chuẩn, ghi nhận lúc login và refresh; admin xem phiên có đầy đủ thông tin thiết bị.
- Q: Khi chuyển quyền sở hữu, xử lý người chủ cũ thế nào? → A: Option C — Admin lựa chọn khi chuyển: hạ xuống staff (vẫn truy cập portal) hoặc gỡ khỏi brand hoàn toàn.
- Q: Ghi audit đồng bộ hay bất đồng bộ? → A: **Bất đồng bộ qua outbox + RabbitMQ** (đúng hiến pháp §III): handler ghi outbox record → OutboxWorker publish → consumer module `admin` ghi `admin_audit_logs`; đảm bảo at-least-once.
- Q: Nhận diện admin seed lúc runtime? → A: Không cần — team tự quản admin seed bằng SQL seed trực tiếp; không xây cơ chế nhận diện trong code.
