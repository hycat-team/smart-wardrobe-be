# ADR 013 - Ghi nhận nợ kỹ thuật: tự chữa grant AI khi tài khoản không có grant

*   **Status**: Accepted
*   **Decision Date**: 2026-10-07
*   **Decision Maker**: Duck

## 1. Context

Trong lượt sửa panic của dịch vụ AI, phát hiện một tài khoản không có bản ghi trong
`aiexecutor.ai_policy_grants`. Tài khoản đó đã đăng ký và đã gọi AI, và đường đọc đã
xử lý đúng bằng cơ chế suy giảm hai tầng của FR-006 — nhưng tài khoản không có
`grant` riêng, nên mọi lần gọi đều ghi một dòng log `fallback tier 1`.

Câu hỏi được nêu là: *mỗi người dùng lẽ ra phải có một grant đang hiệu lực, vậy
tại sao không cho đường đọc tự cấp ngay khi phát hiện thiếu, giống cách `GET /me`
cấp gói mặc định khi tài khoản chưa có gói?*

Hai ý tưởng tưởng như một, nhưng tách ra là thấy ngay vấn đề nằm ở đâu.

**Ý tưởng A — giữ nguyên kiến trúc async, dùng công cụ sửa có sẵn.** Đây là trạng
thái hiện tại: `PlanChangedWorker` tiêu thụ `subscription.plan.changed`, và
`POST /api/v1/admin/subscription-grants/restore` phát lại sự kiện cho các tài
khoản thiếu grant. Đường đọc chỉ đọc, không ghi (FR-042).

**Ý tưởng B — tự chữa: đường đọc phát một message yêu cầu cấp grant.** Đường đọc vẫn
không ghi bảng grant, nên không vi phạm FR-042. Nó chỉ bắn một message và để worker
xử lý. Về mặt hình thức đây là cách đúng.

Nhưng B không khả thi trong kiến trúc hiện tại, và lý do nằm ở dữ liệu chứ không
phải ở quy tắc:

- `buildGrant` (`services/ai/.../grant/grant_uc.go:566-590`) trả về `nil` khi
  `event.ToPlanID == uuid.Nil`. Không có mã dựng được bản ghi grant.
- Payload cần **9 trường bắt buộc**. Dịch vụ AI tự lấy được đúng một trong số đó
  (`aiCostPolicyId`, từ bảng `ai_cost_policies` của chính nó). Bốn trường plan thì
  không — chúng nằm trong bảng của Account.
- FR-024 và T178 đã quyết định rằng AI **không được** lấy dữ liệu gói từ bảng
  subscription: client subscription bị gỡ khỏi đồ thị phụ thuộc của AI, và lý do
  được ghi lại là để một contract nằm sẵn trong DI chính là mở cửa cho đúng kiểu đọc
  chéo miền mà feature 024 sinh ra để đóng.

Nên với B, thứ AI có thể bắn đi chỉ là `userId`. Ai đó phải biến nó thành payload
đầy đủ, và người duy nhất có dữ liệu đó là Account. Đó là một topic mới do AI phát
cho Account, Account tiêu thụ và phát lại `subscription.plan.changed` với
`sourceEventId` và `occurredAt` mới.

**Hai phương án đã loại, ghi lại để không ai phải phân tích lại:**

- *Sửa trong AI bằng gRPC đồng bộ.* Đảo ngược T178, và làm cho đường đọc của AI phụ
  thuộc vào việc Account còn sống hay không — một phụ thuộc chạy ngược với thiết kế
  sẵn sàng cao hiện tại.
- *Cho phép ghi bản ghi grant ngay trên đường đọc.* Bị FR-042 cấm rõ ràng, và
  `research.md:201` đã liệt kê phương án "bản ghi khẩn cấp" dưới cột loại. Nguyên
  nhân gốc là bộ dò cũ từng đóng-xoá bản ghi ngay trong lúc tra cứu, biến một sai
  lệch dữ liệu nhỏ thành mất dữ liệu vĩnh viễn.

**Về mặt kỹ thuật, vì sao B vẫn nguy hiểm kể cả khi nó khả thi:** cơ chế chống stale
giả định `effective_from` tăng đơn điệu, và `rejectStaleEvent` chặn mọi sự kiện có
`occurred_at` không mới hơn bản ghi hiện tại. Một grant cứu hộ được đóng dấu `now`
sẽ **mới hơn** sự kiện `ACTIVATED` cũ đang nằm trong hàng đỗi, khiến sự kiện hợp lệ
đó bị từ chối `stale_event` và đẩy vào hàng đỗi chết. Đó chính xác là kịch bản đã xảy
ra ở spec 025, nơi một campaign grant bị đóng dấu sớm hơn sự kiện free-plan đã có và
tài khoản kết thúc ở gói Premium nhưng vẫn giới hạn AI mức miễn phí.

## 2. Decision

**Lượt này không xây tự chữa.** Dữ liệu hiện có được sửa bằng công cụ đã có sẵn:
gọi `POST /api/v1/admin/subscription-grants/restore` một lần, có `dryRun` trước và
xác minh bằng `database/scripts/07-verify_policy_grants.sql` mục 1 sau.

Lý do đủ để hoãn, theo thứ tự sức nặng:

- **Khoảng trống này là dữ liệu đời trước, không phải rò rỉ đang chạy.** Cả ba cửa
  tạo tài khoản đều cấp gói mặc định **trong cùng transaction** tạo user và ghi
  outbox (`register_uc.go:190-211`, `google_auth_uc.go:248-262`,
  `admin_uc.go:239-249`). Relay outbox giữ bản ghi cho tới khi publish được, nên một
  tài khoản mới không thể mất grant chỉ vì hàng đợi tạm thời sập.
- **Suy giảm hai tầng đã phủ đúng khoảng hở này**, và sau khi vá panic nó chạy đúng
  như thiết kế: request vẫn có trần chi phí, vẫn bị đếm lượt, token ceiling lấy từ
  chính sách mặc định. Không có khách nào nào bị thiệt hại.
- **B là một feature thật**, không phải một task sửa lỗi: topic mới trong hàng đỗi,
  consumer mới ở Account, đăng ký topology, script triển khai. Chi phí đó không
  biện minh cho một khoảng trống đang đóng.

Nợ kỹ thuật được ghi lại ở đây để khi có tín hiệu thật (bộ đếm
`cost_fallback_by_reason.tier1_default_policy` tăng liên tục trên dữ liệu **mới**)
thì có một đường đã nghiên cứu sẵn thay vì phải phân tích lại từ đầu.

**Một lượt chạy thật đã làm rõ điều §2 chưa nói đủ.** Ngày 2026-10-07, chạy
`RestorePolicyGrants` trên production cho 28 tài khoản: **10 được sửa, 18 bị từ chối
vĩnh viễn.** Nguyên nhân không phải dữ liệu đời trước như giả định ở đây, mà là hai
lỗi trong chính đường sửa:

1. `rejectStaleEvent` từ chối sự kiện khi grant hiện hành không có `envelope` —
   đúng dữ liệu cần sửa là thứ nó không đọc được. Deadlock khởi tạo.
2. `RestorePolicyGrants` ghim `EffectiveAt = subscription.started_at`, rồi đóng
   grant cũ với cutoff bằng giá trị đó. Khi grant cũ bắt đầu đúng tại `started_at`
   thì `effective_to == effective_from` và constraint
   `user_ai_policy_grants_check` từ chối. Trên production, **28/28 tài khoản** rơi
   vào trường hợp này.

Hai lỗi này được theo dõi ở spec 027, không phải ở đây. Điều ADR này sửa là kết luận
"chạy một lượt là xong": không đúng, và lượt đó còn làm mất vĩnh viễn khả năng chạy
lại qua cùng endpoint, vì event key đã bị ghi.

**Điều kiện để mở lại:** bộ đếm `cost_fallback_by_reason.tier1_default_policy` tăng
trên dữ liệu **tạo sau** ngày quyết định này. Đó là thứ phân biệt "dữ liệu đời trước"
với "đường ghi đang hỏng". Với lượt chạy đã nêu, con số này bằng 0 trên production —
không phải vì mọi thứ ổn, mà vì lỗi nằm ở chỗ khác và im lặng hơn.

## 3. Consequences

**Tích cực**

- Không thêm topic, consumer hay phụ thuộc nào. AI vẫn là consumer thuần của
  `subscription.plan.changed` và không có đường nào đọc bảng của Account.
- Sửa được dữ liệu hiện có bằng công cụ đã có, có sổ ghi và có thể chạy lại an toàn
  (event key `RESTORE_GRANT:<userID>:<version>` giữ nguyên nên lần hai bị bỏ qua).
- Khoảng hở giữa lúc sự kiện đến và lúc grant có hiệu lực vẫn được xử lý bằng suy giảm
  hai tầng, đúng như SC-008 yêu cầu: không thao tác AI nào bị từ chối trong khoảng đó.

**Tiêu cực**

- Một tài khoản hỏng grant trong tương lai vẫn sẽ không tự được sửa; nó sẽ phục vụ ở
  mức suy giảm cho tới khi có người chạy restore.
- Không có cơ chế phát hiện tự động. Việc phát hiện phụ thuộc vào việc ai đó đọc bộ
  đếm, và lượt này chỉ ghi tài liệu cách đọc chứ không dựng cảnh báo — hệ thống
  không có metrics backend, và dựng một cái là việc hạ tầng chứ không phải kiến trúc.
- Bán kính của lần chạy restore đầu tiên rộng hơn nhiều so với "sửa một tài khoản":
  nó đóng và tạo lại grant cho **mọi** tài khoản còn hạn, kể cả tài khoản đang có
  grant tốt, và snapshot được dựng lại từ cấu hình chính sách tại thời điểm chạy.
- **Lượt chạy thật đã chứng minh công cụ này không sửa được phần lớn tài khoản, và
  không chạy lại được.** 18/28 bị từ chối vĩnh viễn; lần chạy thứ hai báo
  `already_restored` cho tất cả vì event key đã được ghi. Một lượt sửa thất bại để
  lại trạng thái mà không có đường quay lại — đó là khiến runbook phải thêm hai mục
  chẩn đoán chặn trước khi chạy.

**Rủi ro được chấp nhận**

Suy giảm tier-1 dùng hạn mức trực tiếp từ chính sách live thay vì từ snapshot đóng
băng. Với tài khoản free, đó là cùng một con số mà snapshot sẽ mang, nên không mất
gì. Với một tài khoản đã trả tiền mà grant bị mất, hạn mức của nó sẽ bám theo cấu
hình live thay vì giá trị đã đóng băng — nghĩa là một thay đổi cấu hình về sau sẽ có
hiệu lực với tài khoản đó. Đây chính là điều FR-029b cấm, và nó được chấp nhận **vì
đây là khoảng hở sửa chữa, không phải trạng thái bình thường**: không có snapshot thì
không có gì để lấy đi, và suy giảm còn hơn là từ chối dịch vụ. Tài khoản free là
trường hợp duy nhất đã quan sát thực tế.