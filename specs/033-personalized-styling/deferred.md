# Deferred — Cá nhân hoá phối đồ và gợi ý bổ sung (033)

Work discovered while building Cá nhân hoá phối đồ và gợi ý bổ sung (033) that is **not this feature's job**.
Nothing here blocks closing 033; nothing here is a task of 033.

This file exists because an unchecked box in `tasks.md` makes a different claim.
`- [ ]` means *this feature is unfinished*. Work belonging to another feature, or
to the test infrastructure itself, is not this feature's unfinished business, and
recording it as such either keeps a closed feature looking open or pressures the
exit condition into being falsified.

## Rules for this file

- Every entry states **what it is**, **why it is out of scope**, and **what would
  unblock it**. An entry that only names a symptom is not an entry.
- An entry that undermines an already-ticked task's evidence **must say so and name
  that task ID**. Otherwise `[X]` reads as unconditional when it is not.
- If an entry turns out to belong to this feature after all, it moves back into
  `tasks.md` as a real task. **This file is not a parking lot.**
- Report every entry in the skill's final report. Deferring silently defeats the
  purpose of recording it.

## Entries

### D-01 — `make lint` cần shell `cmd.exe` trên Windows

- **Vấn đề**: `make lint` (`fmt-check` + `vet`) chỉ fail khi recipe được shell ngoài là PowerShell: biến
  `$files` trong `FMT_CHECK_CMD` bị PowerShell của outer shell expand trước khi vào `powershell -Command`,
  nên lệnh con nhận `if ()`. Đây không phải lỗi mã nguồn 033. Trên Windows, dùng
  `make SHELL=C:/Windows/System32/cmd.exe lint` (đúng convention đã dùng ở feature 031) thì cả `fmt-check`
  và `vet` đều xanh.
- **Vì sao ngoài phạm vi**: Cách gọi `make` phụ thuộc shell của máy dev/CI, không phải hành vi feature.
  Không cần sửa `Makefile` để feature 033 đóng.
- **Gỡ bằng cách nào**: Ghi lệnh `SHELL=cmd.exe` vào tài liệu/CI Windows. Nếu muốn `make lint` trần chạy
  được dưới PowerShell thì mới cần sửa `FMT_CHECK_CMD` (script `.ps1`/`.cmd` riêng).
- **Ảnh hưởng tới task đã tick**: T087 dùng `SHELL=cmd.exe`; các task đã tick dùng `gofmt -l` + `make vet`
  hoặc `make SHELL=cmd.exe lint`; không có kết luận nào bị mất hiệu lực.

### D-02 — `make` / `make vet` tạo file rác tên `NUL` trên Windows

- **Vấn đề**: Makefile dòng 20–23 dùng redirect `2>NUL` cho Windows trong biến `GIT_SHA` (`$(shell git rev-parse --short HEAD 2>/dev/null 2>NUL || echo dev)`). Khi recipe chạy qua shell POSIX (Git Bash/sh) trên Windows, `NUL` bị hiểu là tên file thường thay vì thiết bị null, nên mỗi lần chạy `make`/`make vet` sinh một file 0 byte tên `NUL` ở thư mục repo. File này lẫn vào `git status` nếu không dọn.
- **Vì sao ngoài phạm vi**: Cùng nguyên nhân harness/OS với D-01; không liên quan tới feature 033. Đây là lỗi portability của Makefile dùng chung.
- **Gỡ bằng cách nào**: Tách redirect theo shell thay vì theo `OS`: dùng `2>/dev/null` khi shell là sh và `2>nul` (chữ thường, hoặc `>nul 2>&1`) chỉ trong `cmd /c`; hoặc bỏ `GIT_SHA` khỏi `$(shell ...)` và tính trong recipe bằng PowerShell. Thêm `.gitignore` entry `NUL` như biện pháp tạm, nhưng fix gốc vẫn ở `Makefile`.
- **Ảnh hưởng tới task đã tick**: Không ảnh hưởng nội dung task nào; chỉ ảnh hưởng tính sạch của workspace khi chạy gate.

### D-03 — `make test-db-integration` trước đây bỏ qua `services/commerce`

- **Vấn đề**: Trước T093, target `test-db-integration` chỉ chạy `services/account` và `services/ai`, nên toàn bộ test `TestIntegration*` của `services/commerce` (migration, outbox, repository mới của 033) không hề chạy trong gate DB-integration. Vì vậy mọi tuyên bố "DB integration đã xanh" trước T093 không bao hàm Commerce.
- **Vì sao ngoài phạm vi**: Đây là thiếu sót của test harness dùng chung (026), không phải logic 033. T093 đã sửa đúng phần thuộc 033 (thêm dòng Commerce); phần còn lại — thiết kế harness, CI job `db-integration` có chạy đúng ba service hay không — thuộc owner của harness.
- **Gỡ bằng cách nào**: Đã gỡ một phần bằng T093 (Makefile + `docs/development/local-setup.md`). Việc còn lại: xác nhận CI job `db-integration` dùng đúng target này và có `TEST_DATABASE_URL` trỏ DB đã migrate đủ ba service; nếu CI tự liệt kê service riêng thì đồng bộ với Makefile.
- **Ảnh hưởng tới task đã tick**: T022/T087 và mọi task từng được coi là "đã kiểm chứng bằng DB integration" trước T093 — bằng chứng của chúng **không** bao gồm Commerce. Sau T093, cần chạy lại `make test-db-integration` với DB thật để xác nhận (xem blocker trong báo cáo).

### D-04 — (Đã xử lý bởi T086) Swagger và nginx route map đang cũ, cần `make generate` khi close-out

> **Trạng thái**: T086 đã chạy `make generate` và commit artifact sinh tự động; mục này giữ lại để
> traceability, không còn là việc mở.

- **Vấn đề**: Feature 033 thêm route `POST /ai/outfit-recommendations/feedback`, `POST /ai/outfit-completions`, `GET|PUT|DELETE /me/style-profile`, `GET /me/style-profile/suggestions`; thêm field `recommendationId`/`suggestions` vào response gợi ý và `personalizationEnabled` vào DTO subscription. Các artifact sinh tự động (`api/swagger/`, nginx route map) chưa được regenerate trong phạm vi task tài liệu này, nên vẫn mô tả bề mặt API cũ.
- **Vì sao ngoài phạm vi**: Đây là công việc của T086 (chạy `make generate` và kiểm tra diff), không phải của T080–T083/T088/T092/T093. Ghi ở đây để close-out không bỏ sót; **không** phải deferral vĩnh viễn.
- **Gỡ bằng cách nào**: Chạy `make generate` (wire + swagger + nginx route map) ở bước T086, kiểm tra diff chỉ chứa thay đổi của 033, rồi commit cùng feature.
- **Ảnh hưởng tới task đã tick**: Không có task nào đã tick dựa trên Swagger/nginx sinh tự động. T086 vẫn đang mở và là nơi xử lý.

### D-05 — Prior vector suy ra từ tủ đồ cho cold-start (inferred-vector prior)

- **Vấn đề**: `GET /me/style-profile/suggestions` hiện chỉ trả phân bố nhãn style kèm `itemCount`/`ratio`/`sampleSize`; người dùng phải xác nhận rồi `PUT`. Không có vector trung bình của tủ được dùng làm prior cho recommendation trước khi người dùng lưu hồ sơ, nên người dùng mới chưa có `style_embedding` chỉ nhận cá nhân hoá ở mức từ khoá (hoặc không có).
- **Vì sao ngoài phạm vi**: Plan D10 chủ động defer: dùng inferred embedding làm prior sẽ **tự áp dụng** gu mà người dùng chưa đồng ý, vi phạm FR-046. Đây là enhancement sau v1, không phải thiếu sót của 033.
- **Gỡ bằng cách nào**: Thiết kế lại luồng cold-start cho phép prior **chỉ khi** người dùng bật một lựa chọn rõ ràng (opt-in) hoặc chỉ dùng prior trong lần gợi ý đầu rồi bỏ; cần spec/ADR mới và test đo tác động lên SC-003/SC-004. Không sửa trong 033.
- **Ảnh hưởng tới task đã tick**: Không. T027/T028 (cold-start suggestions) vẫn đúng phạm vi đã chốt; chỉ nêu rõ giới hạn để không ai kỳ vọng vector prior.

### D-06 — Hai tín hiệu "avoid-style" có thể cộng dồn khi request và hồ sơ trùng nhãn

- **Vấn đề**: `ranking/scoring.go` có hai nhánh trừ điểm độc lập: nhánh cũ theo `intent.ExcludedStyles` (−0.35, từ parser request) và nhánh mới theo `intent.AvoidStyle` (−0.30, từ `profile.excluded_styles`). Nếu người dùng vừa nhập "tránh streetwear" trong request vừa lưu `excludedStyles = ["streetwear"]`, một món streetwear bị trừ **0.65** thay vì mức tối đa dự kiến. Gu vẫn là ưu tiên mềm (không filter), nhưng mức phạt có thể lấn át các tín hiệu khác.
- **Vì sao ngoài phạm vi**: Đây là tương tác giữa parser request (tín hiệu cũ) và profile (tín hiệu 033). Hợp nhất hai nguồn cần quyết định sản phẩm về việc "request có được phép trùng gu" và có thể đổi hành vi của các client cũ; không nằm trong phạm vi phase tài liệu và không chặn close-out.
- **Gỡ bằng cách nào**: Hợp nhất trước khi scoring: dedupe `ExcludedStyles ∪ AvoidStyle` theo nhãn chuẩn hoá, hoặc chỉ áp một mức phạt lớn nhất. Thêm test cho trường hợp trùng nhãn. Nơi sửa: `recommendation/ranking/scoring.go` + `personalization_helper.go`.
- **Ảnh hưởng tới task đã tick**: T031/T032 (scoring cá nhân hoá) — bằng chứng unit test hiện có kiểm từng nhánh riêng, **không** kiểm trường hợp trùng nhãn giữa request và profile.

### D-07 — Định nghĩa `sampleSize` của cold-start hẹp hơn câu chữ hợp đồng

- **Vấn đề**: Hợp đồng ghi `sampleSize` = "số active item có style không rỗng". Code (`profile_helper.go:buildStyleSuggestions` + `distinctFashionItemIDs`) thực tế (a) dedupe theo `fashion_item_id`, nên hai row `wardrobe_items` cùng fashion item chỉ tính một; và (b) chỉ lấy tối đa `overview.MaxWardrobeItems` item qua `GetActiveWardrobeItemsByUserID`. Với người dùng đang vượt hạn mức sau hạ gói (còn item bị khoá) hoặc có bản sao trùng fashion item, `sampleSize` nhỏ hơn "số active item" theo nghĩa đen — `ratio` vẫn nằm trong [0,1] nên không sai số học, nhưng con số hiển thị lệch kỳ vọng.
- **Vì sao ngoài phạm vi**: Đây là ranh giới giữa contract DTO và cách Wardrobe contract giới hạn item theo gói; sửa cần thống nhất lại định nghĩa với client và có thể đổi số liệu hiển thị (không chỉ là bug nội bộ). Không chặn close-out vì `sampleSize` vẫn nhất quán với chính các `itemCount`/`ratio` trả về.
- **Gỡ bằng cách nào**: Chốt lại định nghĩa trong `contracts/personalization-api.md` (ví dụ "số fashion item phân biệt trong phạm vi hạn mức gói") và/hoặc bỏ cap `MaxWardrobeItems` cho mục đích thống kê cold-start; thêm test cho user vượt hạn mức và user có hai row cùng fashion item.
- **Ảnh hưởng tới task đã tick**: T027/T028 — unit test hiện có xác nhận "chỉ item có style được đếm" nhưng **không** phủ trường hợp vượt hạn mức/bản sao trùng, nên bằng chứng của hai task này hẹp hơn câu chữ hợp đồng.

### D-08 — `personalization.gap_score_threshold` không có default trong code

- **Vấn đề**: Khác với `suggestion_max_total`/`suggestion_max_per_role` (có fallback hằng số trong `suggestion_pick_helpers.go`), `detectOutfitGapRoles` nhận thẳng `uc.cfg.Personalization.GapScoreThreshold`. Nếu config thiếu key này (hoặc file config cũ), giá trị là `0`, khi đó chỉ vai trò **không có món nào** mới bị coi là gap; vai trò có ứng viên điểm thấp sẽ không được gợi ý bổ sung. Config ship hiện tại có `0.35` nên hành vi đúng, nhưng độ bền config thấp hơn các knob còn lại.
- **Vì sao ngoài phạm vi**: Đây là khác biệt về độ bền cấu hình (defensive default) trong pipeline gợi ý, không phải sai hành vi với config đã ship; sửa kèm validation config thuộc phạm vi hardening chung.
- **Gỡ bằng cách nào**: Thêm hằng `defaultGapScoreThreshold = 0.35` trong `recommendation` và dùng khi giá trị `<= 0`; hoặc validate `personalization.gap_score_threshold > 0` trong `shared/config/validation.go`. Thêm test cho config zero.
- **Ảnh hưởng tới task đã tick**: T039/T040 (gap-fill suggestions) — test hiện dùng config có 0.35 nên không phủ nhánh zero.

### D-09 — Đường dẫn tài liệu `design-pattern.md` dễ bị trỏ nhầm sang `docs/development/`

- **Vấn đề**: Trong quá trình đối chiếu tài liệu cho 033, `docs/development/` **không** chứa `design-pattern.md`; file chuẩn nằm ở `docs/system-design/design-pattern.md` (được hiến pháp §VII và `docs/development/code-hygiene.md` tham chiếu). Không tìm thấy link gãy trong repo hiện tại, nhưng cách nhóm tài liệu trong `AGENTS.md` (§3 liệt kê cả `docs/development/` và `docs/system-design/`) khiến người đọc dễ tìm sai thư mục.
- **Vì sao ngoài phạm vi**: Đây là vấn đề discoverability của cây tài liệu chung, không phải nội dung feature 033; sửa `AGENTS.md`/mục lục docs ảnh hưởng mọi feature.
- **Gỡ bằng cách nào**: Thêm một dòng trỏ rõ trong `AGENTS.md` §3 (hoặc `docs/development/` README) rằng convention implementation nằm ở `docs/system-design/design-pattern.md`; không di chuyển file.
- **Ảnh hưởng tới task đã tick**: Không.

### D-10 — Hai đường tiêu thụ quota tủ đồ còn thiếu lock

- **Vấn đề**: Sau fix race của 033, `SaveOutfit`/`UpdateOutfit` và `InitClosetFromCatalog` đã gọi
  `LockUserWardrobe` (`pg_advisory_xact_lock(hashtext(userID))`) trước khi đếm. `CloneWardrobeItem`
  (`wardrobe/application/usecase/wardrobe/item/item_uc_write.go`) và `BatchUploadWardrobeItems` vẫn đếm
  ngoài transaction/không lấy lock, nên có thể đua với hai đường đã khoá và vượt `max_wardrobe_items`.
- **Vì sao ngoài phạm vi**: FR-032 của 033 chỉ nói đường lưu outfit có món catalog; hai đường này là
  pre-existing behaviour của Wardrobe, không thuộc thay đổi 033.
- **Gỡ bằng cách nào**: Dời `CountByUserID` vào `IUnitOfWork`, gọi `LockUserWardrobe(txCtx, userID)` trước
  count ở cả hai đường; thêm test đồng thời 2 goroutine trên DB thật.
- **Ảnh hưởng tới task đã tick**: T043/T044 (clone khi lưu outfit) và T046 (catalog-init) đã đóng đường
  của 033; hai đường còn lại vẫn hở và cần task riêng.
