# Feature Specification: Đăng nhập Google (Google Login)

**Feature Branch**: `021-google-login`

**Created**: 2026-09-25

**Status**: Draft

**Input**: User description: "Thêm đăng nhập Google (redirect cho web + ID token cho mobile) vào module identity, auto-link theo email đã verify, migration bổ sung provider/provider_id/email_verified và password_hash nullable."

## Quyết định hiện hành (authoritative)

> **Mục này là nguồn chân lý.** Phần `## Clarifications` bên dưới (nếu có) là lịch sử hội thoại;
> khi mâu thuẫn, mục này thắng. Agent implement chỉ cần đọc mục này + `plan.md` + `tasks.md`.

1. **Phạm vi client**: hỗ trợ **cả web và mobile** — web dùng luồng **chuyển hướng (redirect)** (chính thức), mobile/app dùng **ID token** do Google Sign-In trả về. Web **có thể tuỳ chọn** dùng thêm ID token qua Google Identity Services (`POST /auth/google`) ngoài luồng redirect.
2. **Liên kết tài khoản (auto-link)**: khi Google trả về email đã xác thực (`email_verified = true`) trùng với một tài khoản hiện hữu đã từng xác thực email, hệ thống **tự động liên kết** danh tính Google vào tài khoản đó. Từ chối nếu thông tin Google chưa xác thực email, nếu tài khoản đang bị vô hiệu hóa, hoặc nếu email đã gắn với một danh tính Google khác.
3. **Tài khoản do admin tạo**: là tài khoản **chưa từng xác thực email** → **không tự động liên kết**; trả lỗi hướng dẫn người dùng đăng nhập bằng mật khẩu.
4. **Tài khoản tạo bằng Google**: **được phép thiết lập mật khẩu sau** thông qua luồng quên mật khẩu hoặc đổi mật khẩu lần đầu. Mật khẩu trở thành **không bắt buộc** đối với tài khoản.
5. **Username**: lấy **tiền tố trước dấu `@`** của email Google; xử lý trùng lặp khi tiền tố đã tồn tại.
6. **Email so sánh không phân biệt hoa thường**: luôn chuẩn hóa về chữ thường trước khi so sánh. `Duck@gmail.com` và `duCk@gmail.com` được coi là cùng một email.
7. **Avatar Google**: **bỏ qua** (không đồng bộ ảnh đại diện từ Google).
8. **Backfill**: toàn bộ người dùng cũ được đánh dấu **đã xác thực email**.
9. **Xóa tài khoản**: khi một tài khoản bị xóa, email phải được **ẩn danh (anonymize)** để giải phóng email cho người dùng mới. Hiện hệ thống chưa có luồng xóa tài khoản; đây là chính sách áp dụng khi luồng đó tồn tại.
10. **Luồng web (Authorization Code + redirect)** — giữ nguyên, chi tiết:
    1. FE gọi `GET /api/v1/auth/google?redirectUrl=<đích quay về>` (tên tham số camelCase `redirectUrl`, KHÔNG dùng `redirect_uri`).
    2. Backend **validate `redirectUrl`**: chỉ chấp nhận scheme `https` (trừ dev localhost) và **host khớp chính xác** với **một trong các origin frontend hợp lệ** (`server.front_end_origins`, danh sách dẫn xuất từ `FRONT_END_ORIGIN` đa giá trị); cho phép path/query/fragment phía sau nhưng KHÔNG cho host khác. KHÔNG dùng `HasPrefix`/`HasSuffix` trên chuỗi thô (dính userinfo trick `https://origin@evil.com` và suffix `.hehe`). **Đã chốt**: exact host (không cho subdomain) + scheme `https`; dev cho phép `http://localhost[:port]`; chuẩn hóa port và dấu chấm cuối rồi so hostname; cho phép path/query/fragment.
    3. Backend tạo `state` dùng một lần (lưu server-side, có TTL) và **nhúng đích `redirectUrl` đã validate vào state**; tạo authorization URL của Google rồi `302 → Google`.
    4. Người dùng đăng nhập/đồng ý tại Google; Google redirect về **callback cố định của backend** (`redirect_uri` đã đăng ký với Google).
    5. Backend nhận authorization code, exchange code phía server, xác minh danh tính Google, tạo phiên (Closy JWT + cookie).
    6. Backend `302 → redirectUrl` (đích lấy từ state). FE kiểm tra trạng thái đăng nhập qua `/me` kèm credentials (không nhét token vào URL).
    - **Luồng lỗi** (đã chốt): validate `redirectUrl` fail → JSON 400 (không chuyển hướng); các lỗi phát sinh sau khi đã có đích hợp lệ trong `state` (người dùng hủy đồng ý, `state` sai/hết hạn, code hết hạn, exchange thất bại, tài khoản không hợp lệ) → `302` về đích kèm `?error=<code>`.
    - **Quan hệ domain FE ↔ API** (đã chốt): FE và API cùng registrable domain (là subdomain của cùng một domain, ví dụ `closy.hycat.online` và `api.closy.hycat.online`) → giữ cookie phiên `SameSite=Strict`; FE chỉ cần gọi API kèm credentials/CORS.
11. **Không bổ sung endpoint liên kết/hủy liên kết thủ công** trong phạm vi này (auto-link là đủ; tránh phức tạp không cần thiết).
12. **Vai trò**: tài khoản tạo qua Google luôn có vai trò hệ thống `user` (không thêm vai trò mới).

## Clarifications

### Session 2026-09-25

- Q: Quy tắc so khớp host của `redirectUrl` trong luồng web? → A: Exact host, không cho subdomain; bắt buộc scheme `https` (dev cho phép `http://localhost[:port]`); chuẩn hóa port và dấu chấm cuối rồi so hostname; cho phép path/query/fragment phía sau.
- Q: Quan hệ domain FE ↔ API (ảnh hưởng `SameSite` cookie phiên)? → A: Cùng registrable domain (FE và API là subdomain cùng domain) → giữ cookie phiên `SameSite=Strict`, chỉ cần CORS credentials.
- Q: Hành vi luồng lỗi của redirect web? → A: Validate `redirectUrl` fail → JSON 400 (không redirect); mọi lỗi sau khi đã có đích hợp lệ trong `state` → redirect về FE kèm `?error=<code>`.
- Q: Google Client ID & kiểm tra `audience`? → A: Hỗ trợ danh sách nhiều Client ID (web/iOS/Android) cấu hình được; backend chấp nhận nếu `aud` thuộc danh sách.
- Q: Có cần hỗ trợ nhiều origin frontend không? → A: Có. `FRONT_END_ORIGIN` là danh sách nhiều origin (tùy môi trường); thêm biến dẫn xuất `server.front_end_origins` trong config (KHÔNG map từ env), tách từ chuỗi gốc ngay sau khi load; CORS và validate `redirectUrl` dùng biến này thay vì tự cắt chuỗi.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Đăng nhập/đăng ký nhanh bằng Google trên mobile (Priority: P1)

Người dùng mở ứng dụng di động, chọn "Đăng nhập bằng Google", chọn tài khoản Google của mình. Ứng dụng gửi thông tin danh tính Google về hệ thống. Nếu chưa có tài khoản, hệ thống tạo tài khoản mới; nếu đã có, hệ thống đăng nhập vào tài khoản đó. Người dùng nhận được phiên đăng nhập và dùng được ngay, không phải qua các bước đăng ký rườm rà.

**Why this priority**: Đây là giá trị cốt lõi của tính năng — giảm ma sát đăng ký/đăng nhập cho phần lớn người dùng và là điều kiện để các user story khác có ý nghĩa.

**Independent Test**: Có thể kiểm thử độc lập bằng cách gửi một danh tính Google hợp lệ của người dùng mới và của người dùng đã tồn tại, rồi xác nhận cả hai trường hợp đều nhận được phiên đăng nhập hợp lệ.

**Acceptance Scenarios**:

1. **Given** người dùng chưa có tài khoản và có tài khoản Google với email đã xác thực, **When** người dùng đăng nhập bằng Google, **Then** hệ thống tạo tài khoản mới với vai trò người dùng, trạng thái hoạt động, và trả về phiên đăng nhập hợp lệ.
2. **Given** người dùng đã có tài khoản được tạo bằng Google, **When** người dùng đăng nhập lại bằng Google, **Then** hệ thống đăng nhập vào đúng tài khoản cũ, không tạo thêm tài khoản.
3. **Given** danh tính Google có email chưa được xác thực, **When** người dùng đăng nhập bằng Google, **Then** hệ thống từ chối và không tạo tài khoản.

---

### User Story 2 - Tự động liên kết với tài khoản mật khẩu sẵn có (Priority: P1)

Người dùng đã có tài khoản đăng ký bằng email + mật khẩu (đã xác thực email qua OTP), nhưng chọn đăng nhập bằng Google lần đầu. Hệ thống nhận ra email đã xác thực trùng khớp và tự động liên kết danh tính Google vào tài khoản cũ. Sau đó người dùng có thể đăng nhập bằng cả Google lẫn mật khẩu trên cùng một tài khoản.

**Why this priority**: Ngăn chặn việc tạo tài khoản trùng và bảo đảm trải nghiệm liền mạch cho người dùng hiện hữu — nếu thiếu, tính năng sẽ gây trùng lặp dữ liệu và lỗi đăng nhập.

**Independent Test**: Kiểm thử bằng cách tạo tài khoản email/mật khẩu đã xác thực, sau đó đăng nhập Google với cùng email đã xác thực; xác nhận chỉ còn một tài khoản và cả hai phương thức đăng nhập đều hoạt động.

**Acceptance Scenarios**:

1. **Given** tài khoản email + mật khẩu đã xác thực email tồn tại và chưa gắn Google, **When** người dùng đăng nhập Google với cùng email đã xác thực, **Then** danh tính Google được liên kết vào tài khoản cũ và phiên đăng nhập được cấp.
2. **Given** tài khoản do admin tạo (chưa từng xác thực email), **When** người dùng đăng nhập Google với cùng email, **Then** hệ thống không liên kết và trả về thông báo hướng dẫn đăng nhập bằng mật khẩu.
3. **Given** email đã được liên kết với một danh tính Google khác, **When** một danh tính Google khác đăng nhập với cùng email, **Then** hệ thống từ chối.
4. **Given** email Google khác nhau về hoa thường với email tài khoản hiện có, **When** người dùng đăng nhập Google, **Then** hệ thống coi là cùng một email (không phân biệt hoa thường).

---

### User Story 3 - Đăng nhập Google trên web bằng chuyển hướng (Priority: P2)

Người dùng web bấm "Đăng nhập bằng Google", được chuyển sang trang xác thực của Google, sau khi đồng ý được chuyển ngược về hệ thống và nhận phiên đăng nhập dưới dạng cookie an toàn.

**Why this priority**: Cần thiết cho người dùng web nhưng có thể triển khai sau khi luồng ID token (mobile) đã hoạt động; độc lập về mặt kiểm thử.

**Independent Test**: Kiểm thử bằng cách bắt đầu luồng chuyển hướng, hoàn tất xác thực Google trong trình duyệt, và xác nhận người dùng quay về ứng dụng web ở trạng thái đã đăng nhập.

**Acceptance Scenarios**:

1. **Given** người dùng web chưa đăng nhập, **When** người dùng bắt đầu đăng nhập Google và hoàn tất xác thực, **Then** hệ thống trả về đúng ứng dụng web với phiên đăng nhập hợp lệ.
2. **Given** một yêu cầu chuyển hướng bị giả mạo hoặc hết hạn, **When** hệ thống nhận được phản hồi từ Google, **Then** hệ thống từ chối và không cấp phiên.
3. **Given** tham số chuyển hướng trỏ tới địa chỉ không hợp lệ, **When** hệ thống xử lý, **Then** hệ thống từ chối chuyển hướng tới địa chỉ đó.

---

### User Story 4 - Thiết lập mật khẩu cho tài khoản tạo bằng Google (Priority: P3)

Người dùng đã tạo tài khoản bằng Google muốn có thêm mật khẩu để đăng nhập bằng email/mật khẩu. Người dùng thiết lập mật khẩu lần đầu qua luồng quên mật khẩu hoặc đổi mật khẩu, sau đó có thể đăng nhập bằng cả hai cách.

**Why this priority**: Tăng tính linh hoạt cho người dùng nhưng không chặn luồng đăng nhập chính; có thể triển khai sau.

**Independent Test**: Kiểm thử bằng cách tạo tài khoản chỉ có Google, thực hiện luồng thiết lập mật khẩu, rồi đăng nhập bằng email + mật khẩu vừa tạo.

**Acceptance Scenarios**:

1. **Given** tài khoản chỉ có Google (chưa có mật khẩu), **When** người dùng thực hiện luồng quên mật khẩu và đặt mật khẩu mới, **Then** tài khoản có mật khẩu và có thể đăng nhập bằng email + mật khẩu.
2. **Given** tài khoản chỉ có Google, **When** người dùng cố đăng nhập bằng mật khẩu bất kỳ, **Then** hệ thống từ chối với thông báo hợp lý.

---

### Edge Cases

- **Email Google chưa xác thực**: từ chối tạo/liên kết tài khoản.
- **Email đã gắn danh tính Google khác**: từ chối để tránh chiếm tài khoản khi email đổi chủ.
- **Tài khoản đang bị vô hiệu hóa/tạm ngưng/khóa**: đăng nhập Google phải bị chặn giống đăng nhập thường.
- **Tài khoản do admin tạo (email chưa từng xác thực)**: không tự động liên kết.
- **Hai yêu cầu đăng nhập Google đồng thời cho cùng người dùng mới**: hệ thống không tạo ra hai tài khoản.
- **Đăng nhập Google trùng thời điểm với xác thực OTP đăng ký**: không tạo tài khoản trùng; trả lỗi rõ ràng.
- **Tiền tố username trùng**: sinh username thay thế hợp lệ thay vì thất bại.
- **Email khác hoa thường**: coi là cùng một email.
- **Danh tính Google không hợp lệ/giả mạo/hết hạn**: từ chối.
- **Tài khoản bị xóa (khi luồng xóa tồn tại)**: email được ẩn danh để giải phóng cho người dùng mới.
- **Tham số chuyển hướng web hết hạn hoặc dùng lại**: từ chối.
- **Đích chuyển hướng dùng host giả mạo**: `closy.hycat.online.hehe`, `https://closy.hycat.online@evil.com`, hoặc scheme `http` (ngoài localhost dev) → từ chối, không chuyển hướng.
- **Đích chuyển hướng là subdomain của origin** (`evil.closy.hycat.online`): từ chối (chỉ chấp nhận exact host).
- **Dev localhost**: `http://localhost:3000/...` được chấp nhận khi môi trường dev.
- **Đích chuyển hướng có path/query hợp lệ trên đúng host**: chấp nhận.
- **Người dùng hủy đồng ý tại Google**: chuyển hướng về đích kèm `?error=...`, không cấp phiên.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST cho phép người dùng xác thực bằng tài khoản Google từ cả ứng dụng web (qua chuyển hướng) và ứng dụng di động (qua ID token). Web MAY tuỳ chọn dùng thêm ID token qua Google Identity Services ngoài luồng redirect.
- **FR-002**: Hệ thống MUST xác minh tính hợp lệ của danh tính Google (nguồn phát hành, đối tượng nhận nằm trong danh sách Client ID hợp lệ, thời hạn, tính toàn vẹn) trước khi tin tưởng bất kỳ thông tin nào.
- **FR-003**: Hệ thống MUST từ chối đăng nhập/đăng ký bằng Google nếu email của danh tính Google chưa được xác thực.
- **FR-004**: Khi danh tính Google hợp lệ và chưa có tài khoản tương ứng, hệ thống MUST tạo tài khoản mới với vai trò hệ thống `user`, trạng thái hoạt động, và đánh dấu email đã xác thực.
- **FR-005**: Hệ thống MUST sinh `username` cho tài khoản mới bằng tiền tố trước dấu `@` của email Google, sau khi chuẩn hóa; nếu tiền tố đã tồn tại, hệ thống MUST sinh giá trị thay thế hợp lệ.
- **FR-006**: Khi danh tính Google có email đã xác thực trùng với tài khoản hiện hữu đã từng xác thực email, hệ thống MUST tự động liên kết danh tính Google vào tài khoản đó và cấp phiên đăng nhập.
- **FR-007**: Hệ thống MUST NOT tự động liên kết danh tính Google với tài khoản chưa từng xác thực email (ví dụ tài khoản do admin tạo); thay vào đó trả về thông báo hướng dẫn đăng nhập bằng mật khẩu.
- **FR-008**: Hệ thống MUST từ chối khi email đã được liên kết với một danh tính Google khác.
- **FR-009**: Hệ thống MUST so sánh email không phân biệt hoa thường trên mọi luồng đăng nhập/liên kết/tạo tài khoản.
- **FR-010**: Hệ thống MUST áp dụng cùng điều kiện hợp lệ của tài khoản (không bị xóa, đang hoạt động, không nằm trong danh sách khóa) cho đăng nhập Google như đăng nhập thường.
- **FR-011**: Phiên đăng nhập phát hành qua Google MUST dùng cùng cơ chế phiên hiện có (access token, refresh token, xoay vòng refresh, đăng xuất) để đảm bảo hành vi nhất quán; cookie phiên giữ `SameSite=Strict` vì FE và API cùng registrable domain.
- **FR-012**: Luồng chuyển hướng web MUST sử dụng `state` dùng một lần, có thời hạn, lưu phía máy chủ; `state` MUST mang theo đích quay về đã được validate để callback không đọc lại đích từ tham số do client cung cấp.
- **FR-013**: Hệ thống MUST validate đích quay về (`redirectUrl`) bằng cách phân tích URL và so khớp **scheme + host chính xác** với **một trong các origin frontend hợp lệ** (`server.front_end_origins`), cho phép path/query/fragment phía sau; MUST NOT dùng so khớp tiền tố/hậu tố trên chuỗi thô và MUST NOT chuyển hướng tới host khác. Chỉ chấp nhận **exact host** (không cho subdomain) và scheme `https`, ngoại lệ duy nhất là `http://localhost[:port]` khi môi trường dev; MUST chuẩn hóa port và dấu chấm cuối trước khi so hostname.
- **FR-014**: Người dùng chỉ có tài khoản Google MUST có thể thiết lập mật khẩu lần đầu thông qua luồng quên mật khẩu hoặc đổi mật khẩu.
- **FR-015**: Hệ thống MUST từ chối đăng nhập bằng mật khẩu đối với tài khoản chưa có mật khẩu, với thông báo hợp lý.
- **FR-016**: Hồ sơ người dùng MUST lưu được: nhà cung cấp danh tính, định danh nhà cung cấp, và trạng thái xác thực email; mật khẩu MUST trở thành không bắt buộc.
- **FR-017**: Toàn bộ người dùng hiện hữu MUST được đánh dấu đã xác thực email khi nâng cấp.
- **FR-018**: Hệ thống MUST xử lý được các yêu cầu đăng nhập Google đồng thời mà không tạo tài khoản trùng.
- **FR-019**: Hệ thống MUST NOT đồng bộ ảnh đại diện từ Google.
- **FR-020**: Hệ thống MUST trả về thông báo lỗi tiếng Việt, đúng chuẩn lỗi chung của hệ thống, cho mọi trường hợp thất bại.
- **FR-021** *(deferred — chỉ áp dụng khi luồng xoá tài khoản được thêm)*: Khi một tài khoản bị xóa, hệ thống MUST ẩn danh email của tài khoản đó để giải phóng email.
- **FR-022**: Hệ thống MUST áp dụng giới hạn tần suất cho các endpoint Google (`POST /auth/google`, `GET /auth/google`, `GET /auth/google/callback`) bằng cơ chế shield `/auth` hiện có, dùng cấu hình `claim_rate_limit` (key theo IP + định danh, `window_seconds`); MUST NOT để các endpoint này không được bảo vệ.
- **FR-023**: Luồng web MUST xử lý lỗi như sau: nếu `redirectUrl` không hợp lệ thì trả JSON 400 và không chuyển hướng; nếu lỗi phát sinh sau khi đã có đích hợp lệ trong `state` (hủy đồng ý, `state` sai/hết hạn, code hết hạn, exchange thất bại, tài khoản không hợp lệ) thì chuyển hướng về đích kèm tham số `error` và MUST NOT cấp phiên.

### Key Entities *(include if feature involves data)*

- **Tài khoản người dùng (Người dùng)**: đại diện cho người dùng hệ thống. Bổ sung thuộc tính: nhà cung cấp danh tính (ví dụ: cục bộ, Google), định danh duy nhất từ nhà cung cấp, trạng thái xác thực email. Thuộc tính mật khẩu trở thành không bắt buộc. Quan hệ: một người dùng có nhiều phiên đăng nhập.
- **Phiên đăng nhập (Phiên)**: đại diện cho một phiên đã xác thực của người dùng; dùng chung cơ chế với đăng nhập thường (không thay đổi cấu trúc).
- **Yêu cầu chuyển hướng tạm thời (Trạng thái chuyển hướng)**: đại diện cho một lần bắt đầu luồng đăng nhập web; dùng một lần, có thời hạn, gắn với điểm đến hợp lệ.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 90% người dùng hoàn tất đăng nhập hoặc đăng ký bằng Google trong vòng dưới 15 giây, không cần nhập biểu mẫu.
- **SC-002**: 100% trường hợp email Google chưa xác thực bị từ chối, không tạo và không liên kết tài khoản.
- **SC-003**: Không phát sinh tài khoản trùng lặp khi nhiều yêu cầu đăng nhập Google đồng thời cho cùng một người dùng mới (0 trùng lặp trong kiểm thử đồng thời).
- **SC-004**: 100% tài khoản đang bị vô hiệu hóa hoặc tạm ngưng không thể đăng nhập bằng Google.
- **SC-005**: Người dùng hiện hữu có thể tiếp tục đăng nhập bằng mật khẩu và/hoặc Google trên cùng một tài khoản mà không cần tạo tài khoản mới.
- **SC-006**: Tỷ lệ người dùng hoàn tất đăng ký mới tăng ít nhất 30% so với mức nền của luồng đăng ký email + mật khẩu hiện tại.

## Out of Scope

- Đồng bộ ảnh đại diện/tiểu sử từ Google.
- Endpoint liên kết/hủy liên kết Google thủ công.
- Các nhà cung cấp danh tính khác (Facebook, Apple, ...).
- Luồng xóa tài khoản tự phục vụ (chỉ định nghĩa chính sách ẩn danh email cho tương lai).
- Thay đổi cơ chế phiên/token hiện có (chỉ tái sử dụng).

## Assumptions

- Google là nhà cung cấp danh tính bên ngoài duy nhất được thêm trong phạm vi này; email/mật khẩu vẫn là phương thức cục bộ.
- Người dùng web dùng trình duyệt và có thể hoàn tất chuyển hướng tới Google; ứng dụng di động dùng Google Sign-In để lấy ID token.
- Tất cả người dùng hiện hữu đã xác thực email (đăng ký qua OTP) nên được đánh dấu đã xác thực khi nâng cấp.
- Danh sách origin frontend hợp lệ để chuyển hướng có sẵn trong cấu hình: `FRONT_END_ORIGIN` có thể chứa nhiều origin (tùy môi trường), được tách một lần thành biến dẫn xuất `server.front_end_origins`.
- Tài khoản tạo qua Google mặc định trạng thái hoạt động và không cần bước xác thực email bổ sung (Google đã xác thực).
- Cơ chế phiên, xoay vòng refresh token, đăng xuất và danh sách khóa tài khoản hiện có sẽ được tái sử dụng nguyên vẹn.
- Không thêm vai trò hệ thống mới; tài khoản Google luôn là vai trò `user`.
- Hệ thống có thể có nhiều Google OAuth Client ID (web/iOS/Android); danh sách audience hợp lệ được cấu hình, không hardcode.
