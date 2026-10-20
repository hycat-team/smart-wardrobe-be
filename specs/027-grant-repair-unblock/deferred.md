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

### D-06 — Ràng buộc `CHECK` của bảng mức quyền nằm trong migration của dịch vụ Account

- **Vấn đề**: Bảng `aiexecutor.ai_policy_grants` do
  `services/account/database/migrations/20260622131631_add_ai_cost_policy.sql` tạo, với
  ràng buộc `CHECK (effective_to IS NULL OR effective_to > effective_from)` viết **nội
  tuyến, không đặt tên** — nên PostgreSQL tự sinh tên `user_ai_policy_grants_check` theo
  tên bảng cũ. Sau đó bảng bị đổi tên thành `ai_policy_grants` và
  `20260801000000_separate_module_schemas.sql` chuyển nó sang schema `aiexecutor` bằng
  `ALTER TABLE ... SET SCHEMA`. **Cả hai thao tác đều giữ nguyên tên ràng buộc đã sinh tự
  động.** Hệ quả: bảng thuộc schema của dịch vụ AI, nhưng DDL của nó nằm trong lịch sử
  migration của dịch vụ Account, và `services/ai/database/migrations/` hiện **rỗng** (chỉ có
  `.gitkeep`).
- **Vì sao ngoài phạm vi**: FR-009 của 027 chỉ cần **đọc** mã lỗi Postgres, không cần thêm,
  sửa hay nâng cấp ràng buộc. 027 **không** tạo migration nào, nên nó không làm nợ này nặng
  thêm. Di chuyển DDL sang dịch vụ AI là một thay đổi lịch sử migration, ảnh hưởng
  `DATABASE_MIGRATION_TABLE` và thứ tự chạy khi khởi động — vượt xa một đặc tả sửa lỗi hai
  nhánh.
- **Gỡ bằng cách nào**: Một đặc tả riêng về quyền sở hữu DDL theo schema, chuyển
  `ai_policy_grants` (và các bảng `aiexecutor.*` còn lại) sang
  `services/ai/database/migrations/`, kèm bước kiểm tra tương thích cho môi trường đang
  chạy.
- **Ghi chú liên quan (DML)**: cùng chủ đề sở hữu, `database/scripts/06-cleanup_policy_grants.sql`
  và `database/scripts/11-seed_restore_test_data.sql` (có sẵn từ trước) **ghi** trực tiếp vào
  `aiexecutor.ai_policy_grants` bằng SQL. Đây là căng thẳng có sẵn với FR-013, không do 027
  tạo ra: `git diff` từ đầu đặc tả cho thấy 027 chỉ sửa `07-verify_policy_grants.sql`, và tệp
  đó **chỉ đọc**. Runbook của 027 cấm sửa dữ liệu bằng SQL. Việc chuyển quyền sở hữu bảng về
  dịch vụ AI (bước gỡ ở trên) tự nhiên bao trùm cả hai script này.
- **Ảnh hưởng tới task đã tick**: **Có — gián tiếp.** T016 phải **tự thêm** ràng buộc bằng
  `ALTER TABLE` trong cơ sở dữ liệu kiểm thử, vì `AutoMigrate` không tạo ra nó (thực thể
  `AIPolicyGrant` không có thẻ `check:`). Đây không phải sự cố của ràng buộc, nhưng nếu ai đó
   sửa lược đồ kiểm thử theo hướng "bỏ `ALTER TABLE` vì thừa" thì kiểm thử sẽ ngừng chứng
   minh điều gì và US3 sẽ trông như đã kiểm chứng.
- **Trạng thái (2026-10-16, spec 030 — RESOLVED)**: đã gỡ xong. Nội dung 11 file
  migration Account chạm `aiexecutor`/`dashboard` được viết lại (giữ tên +
  version); chain baseline độc lập ở đúng cây chủ
  (`services/ai/database/migrations/20261016213000_adopt_aiexecutor_ownership.sql`
  với tên tường minh `ai_policy_grants_effective_window_check` + khối rename
  một lần từ tên cũ,
  `services/commerce/database/migrations/20261016213001_adopt_dashboard_ownership.sql`);
  gate `migration-check` (3 luật G-ACCT/G-COMM/G-AI, 13 version miễn trừ đúng
  tên file) chặn vi phạm mới; rollback = restore backup. Dấu vết đầy đủ ở
  `specs/030-ddl-ownership-handover/` (hợp đồng, verification-log)
  và [ADR 016](../../docs/decisions/016-ddl-ownership-handover.md). Nội dung
  vấn đề gốc ở trên giữ nguyên làm lịch sử.

### D-07 — Lỗi gốc không được gắn vào lỗi từ chối vĩnh viễn

- **Vấn đề**: `GrantUseCase.reject(reason, userID, message)` (`GRANT/grant_uc.go`) trả
  `sharedevent.PermanentError` chỉ mang `RejectionError{Reason, Message}`, **không** mang lỗi
  gốc. Đường mới của 027 (vi phạm `CHECK` `23514`) vì vậy ghi được tài khoản và mã lý do vào
  dòng cảnh báo, và mang một thông điệp tiếng Việt nêu nguyên nhân, nhưng **không** gắn
  `*pgconn.PgError` (SQLSTATE và tên ràng buộc) vào dấu vết thư chết. Điều này đúng cho
  **mọi** đường `reject` hiện có, không riêng đường mới.
- **Vì sao ngoài phạm vi**: SC-006 đọc theo nghĩa "dấu vết có tài khoản, mã lý do và một
  thông điệp nêu nguyên nhân" thì đường mới **đã** đạt (`RejectionError.Message`). Đọc theo
  nghĩa chặt hơn — "phải có nguyên văn thông điệp driver" — thì đây là khoảng trống có **từ
  trước** 027, nằm ở thiết kế chung của `reject()`. Sửa nó là đổi hành vi của mọi đường từ
  chối, vượt phạm vi FR-009..FR-012. Ngoài ra FR-011 cấm ghi **quá một** dòng cảnh báo cho
  mỗi lần từ chối, nên không thể "chỉ thêm một dòng log nữa" mà không phải quyết định điều
  gì thay thế dòng hiện có.
- **Gỡ bằng cách nào**: Cho `RejectionError` mang lỗi gốc qua `Unwrap` (giữ `PermanentError`
  là marker), và ghi nó vào **cùng** dòng cảnh báo duy nhất mà `RecordRejection` phát ra —
  tức mở rộng `RecordRejection` nhận thêm `error`, hoặc ghi lỗi gốc ở nơi bắt lỗi trước khi
  bọc. Cần một đặc tả nhỏ, và cần đối chiếu với FR-011 để bảo đảm vẫn đúng một dòng.
- **Ảnh hưởng tới task đã tick**: **Có.** T018 khẳng định dấu vết thư chết có tài khoản, mã
  lý do và thông điệp; nếu SC-006 bị đọc chặt thành "phải có lỗi gốc", thì T018 chỉ đạt dấu
  vết ở mức thông điệp, không ở mức lỗi driver — `[X]` ở T018 nên được đọc kèm điều kiện này.

- **TRẠNG THÁI (2026-10-20 — RESOLVED)**: đã gỡ đúng hướng mô tả ở trên:
  - `RejectionError` có thêm `cause error` + `Unwrap() error`; `Error()` vẫn trả `Message` nên
    chuỗi mà log/khách hàng thấy không đổi.
  - `reject(reason, userID, message string, cause ...error)` — variadic để các đường không có
    lỗi gốc giữ nguyên; hai nhánh vi phạm `23514` (`grant_uc.go` create + retire) truyền `err`.
  - `RecordRejection(reason, userID string, cause ...error)` ghi `zap.NamedError("cause", …)` vào
    **cùng một dòng** cảnh báo (FR-011 vẫn đúng: test khẳng định `len(entries) == 1`).
  - Test khoá lại: `grant_error_class_integration_test.go` khẳng định `errors.As` tới
    `*pgconn.PgError`, `SQLSTATE == 23514`, và `ConstraintName` không rỗng;
    `TestGrantMetricsRejectionCarriesTheCauseOnTheSameLine` khẳng định field `cause` nằm trên
    dòng warn duy nhất. Không cần `pg_advisory`/hạ tầng gì thêm.
### D-08 — Cấp lại một gói CÓ thời hạn đang còn hiệu lực không sửa được kỳ hiện tại

- **Vấn đề**: Với một gói có thời hạn mà kỳ hiện tại chưa hết (tài khoản demo `user`,
  `premium-monthly`), đường sửa cấp lại đúng gói không chữa được bản ghi hỏng, và còn cộng
  thêm 30 ngày. Chuỗi: `ResolveExpiryFor` (`subscription_behavior.go:397`) cộng dồn
  `duration_days` vào hạn cũ khi cấp lại cùng gói còn hiệu lực → sự kiện `EXTENDED` mang
  `effectiveTo = hạn_cũ + 30d` → `EventGrantWindow` đặt `start = effectiveTo − durationDays =
  hạn_cũ`, và vì `hạn_cũ > now` nên bản ghi mới là `future` (`grant_window.go:64-70`) →
  `TruncateGrantsBefore(cutoff = hạn_cũ)` để bản ghi hỏng ở lại trong cửa sổ hiệu lực tới
  `hạn_cũ`. Bộ đọc lọc theo cửa sổ chứ không theo trạng thái
  (`ai_policy_grant_writer.go:90-96`), nên quyền AI của tài khoản vẫn suy giảm suốt phần còn
  lại của kỳ.
- **Vì sao ngoài phạm vi**: Đã chấp nhận có ý thức ngày 2026-10-07. Thiệt hại chỉ rơi vào
  **tài khoản demo** — đặc tả ghi rõ `premium-monthly` duy nhất là demo `user`, không có khách
  trả tiền nào trong danh sách. Sửa nó cần một trong hai thay đổi lớn hơn phạm vi 027: cho
  cấp gói hành chính **không** cộng dồn hạn (đổi hành vi phía Account, ảnh hưởng mọi đường
  admin grant), hoặc đổi ngữ nghĩa cửa sổ renewal ở phía AI để đóng bản ghi hỏng tại `now`
  và tạo một bản ghi đọc được cho phần còn lại của kỳ (phá quy tắc "không khe hở" và cần đặc
  tả riêng).
- **Gỡ bằng cách nào**: Một đặc tả riêng. Đường sạch nhất là tách "cấp hành chính" khỏi "mua"
  trong `ResolveExpiryFor`: cấp hành chính luôn tính kỳ mới từ `now`, không cộng dồn — vì
  không ai đã trả tiền cho kỳ được cộng thêm.
- **Ảnh hưởng tới task đã tick**: **Có.** T040 và SC-009 kỳ vọng cả 8 tài khoản được sửa; với
  tài khoản demo `premium-monthly` còn hiệu lực thì chỉ 7 tài khoản đạt. `[X]` ở T040 nên được
  đọc kèm điều kiện này.

- **TRẠNG THÁI (2026-10-20 — RESOLVED)**: tách "cấp hành chính" khỏi "kỳ đã trả tiền" đúng như
  hướng gỡ đề xuất:
  - `UserSubscription.ResolveAdminExpiryFor(plan, now)` mới: `DurationDays == nil` → `nil`, còn
    lại **luôn `now + duration`**, không cộng dồn hạn cũ.
  - `ApplySubscriptionPlanWithEvent` dùng method này (đây là workflow mà **cả hai** caller sản
    xuất — admin batch grant và campaign grant — đi qua; purchase settlement không đi qua nó, đã
    kiểm chứng: purchase dùng `ApplyTransition`/`Activate`).
  - `ResolveExpiryFor` giữ nguyên làm luật cho kỳ **đã trả tiền**; comment ghi rõ hiện không còn
    production caller. Test call-graph (`TestExpiryCallGraphMatchesTheComments`) khoá lại: 0 caller
    cho `ResolveExpiryFor`, đúng 1 caller cho `ResolveAdminExpiryFor`.
  - Hành vi **trước → sau**: admin cấp lại cùng gói đang còn hạn — trước `hạn_cũ + 30d` (và sinh
    bản ghi `future` khiến quyền AI suy giảm tới hết kỳ), sau **`now + 30d`**. Cấp lại **thay** phần
    còn lại, không cộng thêm; muốn kéo dài phải cấp gói có duration dài hơn hoặc gói khác.
  - Campaign không đổi hành vi: condition 5 (`SkipAlreadyOnPlan`, `admission_uc.go:115`) vẫn chặn
    ca "đã đúng gói và kỳ chưa hết" trước khi tới workflow; test condition 5 vẫn xanh.
  - Test khoá: 3 ca entity (`TestResolveAdminExpiryForNeverStacks`), workflow
    (`TestRegrantingALiveSamePlanKeepsAutoRenewAndStartsAFreshPeriod`), admin idempotency
    (`TestAdminGrantBatchDifferentKeyGrantsAgain` sửa kỳ vọng: khoá khác cấp lại nhưng không cộng
    ngày).
- **TRẠNG THÁI (2026-10-20 — RESOLVED, docs-only)**: bất biến đã được viết thành tài liệu chuẩn tại
  [policy-grant-window.md](../../docs/system-design/policy-grant-window.md): predicate
  `effective_from <= t AND (effective_to IS NULL OR effective_to > t)`, **không lọc `status`**; bảng
  status × cửa sổ; hai consumer (reader `GetCurrentGrant` và checker
  `database/scripts/07-verify_policy_grants.sql` §4/§4b); quy tắc sửa đồng bộ hai bên. Link từ
  `ai-cost-control.md` và `grant-restore-runbook.md`. Không đổi code.