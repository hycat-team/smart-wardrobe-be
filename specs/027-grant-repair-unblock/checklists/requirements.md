# Specification Quality Checklist: grant-repair-unblock

**Purpose**: Kiểm tra tính đầy đủ và chất lượng đặc tả trước khi lập kế hoạch
**Created**: 2026-10-07
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] CHK001 Không có chi tiết triển khai (ngôn ngữ, framework, API)
- [x] CHK002 Tập trung vào giá trị và nhu cầu, không vào cách làm
- [x] CHK003 Viết cho người đọc hiểu nghiệp vụ, không chỉ cho lập trình viên
- [x] CHK004 Đủ các mục bắt buộc

**Ghi chú CHK001 — cần nói rõ phạm vi kiểm tra**: mô tả đặc tả có nhắc tên hàm, tên bảng
và mã lỗi ở phần "Quan hệ với các yêu cầu đang tồn tại" và phần Bối cảnh. Đây là
**cố ý** và không vi phạm: đặc tả sửa lỗi cho một cơ chế đã tồn tại không thể mô tả mà
không nói nó sửa cái gì, và việc ghi tên file giúp người đọc kiểm chứng lại luận điểm.
Các mục yêu cầu bắt buộc (FR-001..FR-013, AC-01..AC-03, SC-001..SC-007) đều viết ở
mức hành vi và ràng buộc, không chỉ định thuật toán hay cấu trúc dữ liệu.

## Requirement Completeness

- [x] CHK005 Không còn nhãn [NEEDS CLARIFICATION]
- [x] CHK006 Yêu cầu kiểm thử được và không mơ hồ
- [x] CHK007 Tiêu chí thành công đo lường được
- [x] CHK008 Tiêu chí thành công không phụ thuộc công nghệ
- [x] CHK009 Đủ kịch bản chấp nhận
- [x] CHK010 Đã nhận diện trường hợp biên
- [x] CHK011 Phạm vi được xác định rõ
- [x] CHK012 Đã nhận diện phụ thuộc và giả định

**Ghi chú CHK008 — điểm cần biết**: SC-002, SC-004, SC-006 và SC-007 là tiêu chí vận
hành, mô tả trạng thái hệ thống chứ không mô tả trải nghiệm người dùng cuối. Đặc tả
này không có giao diện người dùng, và đối tượng bị ảnh hưởng là quyền lợi của khách
hàng chứ không phải một màn hình. Các tiêu chí được viết theo kết quả quan sát được
("0 tài khoản", "đúng một lần", "0 lần thử lại") chứ không theo tên bảng hay tên hàm,
nên chúng vẫn kiểm chứng được mà không cần biết cách hiện thực.

## Feature Readiness

- [x] CHK013 Mọi yêu cầu chức năng có tiêu chí chấp nhận rõ ràng
- [x] CHK014 Kịch bản người dùng bao phủ các luồng chính
- [x] CHK015 Đặc tả đáp ứng các kết quả đo lường ở Tiêu chí thành công
- [x] CHK016 Không rò rỉ chi tiết triển khai vào yêu cầu

## Đối chiếu với hiến pháp và spec 024

- [x] CHK017 Không vi phạm nguyên tắc quyền sở hữu dữ liệu (§I, FR-027) — FR-013
- [x] CHK018 Không đảo ngược quyết định gỡ client subscription (024 T178) — AC-01
- [x] CHK019 Không thêm cờ đặc quyền — AC-02
- [x] CHK020 Không thay đổi hợp đồng sự kiện — AC-03
- [x] CHK021 Nêu rõ quan hệ với FR-033, kể cả tiền đề mà dữ liệu hiện tại vi phạm
- [x] CHK022 Không thêm ràng buộc duy nhất ở tầng dữ liệu (FR-034 của spec 024)
- [x] CHK023 Ghi nhận phát hiện ngoài phạm vi vào `deferred.md` chứ không để thành task
  chưa đánh dấu (hiến pháp §Governance)

## Tài liệu cần tạo ở giai đoạn kế hoạch

- [x] CHK024 `plan.md` — đã tạo. Mục 1 chốt `resolveGrantWindow` và `EventGrantWindow`; Mục 2
  chốt `retirePreviousGrants` đổi chữ ký. Phát hiện cấu trúc quan trọng nhất: FR-005 **không
  thể** thực hiện bằng cách vá một chỗ, vì `retirePreviousGrants` và `buildGrant` mỗi cái
  tự tính lại cửa sổ của riêng nó.
- [x] CHK025 ADR 014 — đã tạo `docs/decisions/014-relax-grant-ordering-invariant.md`. Ghi
  rõ đây là **sự nới lỏng thật**: bản ghi mới nhất không còn bảo đảm có thời điểm bắt đầu
  lớn nhất tuyệt đối, chỉ lớn hơn nghiêm. Nêu cả hệ quả về lệch đồng hồ.
- [x] CHK026 **Không** cần `frontend-guide.md`. R-004 kết luận nhãn của
  `grant_rejected_by_reason` là `string` tự do, không phải kiểu liệt kê đóng, nên khóa mới
  không làm vỡ client đọc theo khoá. Kết luận này **có điều kiện**: Mục 4 của `plan.md` và
  K4 của `quickstart.md` yêu cầu xác nhận bằng chứng trước khi đóng. Nếu nhãn hoá ra là
  kiểu đóng, mục này mở lại và `frontend-guide.md` trở thành bắt buộc.
- [x] CHK027 `deferred.md` — đã tạo, 5 mục. D-03 ghi rõ nó **làm suy yếu** FR-014/FR-015:
  nếu ai đó đổi bộ đọc để lọc theo trạng thái thì mục 4 của kiểm tra toàn vẹn và SC-005 hỏng
  theo.
- [x] CHK028 Dòng mô tả sai phạm vi `stale_event` trong
  `contracts/subscription-plan-changed-event.md` — đính chính trong Mục 6 của `plan.md`,
  giao cho một mục của `tasks.md`.
- [x] CHK029 `runbook.md` — đã tạo trong thư mục gốc spec. Chốt đường sửa là đường ghi
  hành chính `POST /admin/subscription-grants` với khoá sự kiện mới, và bắt buộc tắt
  `EMAIL_ENABLED` trong suốt thao tác.

## Ghi chú về làm rõ

Hai câu hỏi đã hỏi trong phiên `2026-10-07`, cả hai đều lộ ra mâu thuẫn trong bản đặc tả
viết trước đó và đều được sửa:

- Câu hỏi về phạm vi mục 4 của kiểm tra toàn vẹn lộ ra rằng **SC-005 không thể đạt
  được** như đang viết, vì công cụ khôi phục không bao giờ chạm vào bản ghi lịch sử đã
  đóng mà mục đó đếm cả. Đã thêm FR-014..FR-016 và sửa SC-005.
- Câu hỏi về cách sửa 8 tài khoản đã bị chặn lộ ra rằng **con số "18 tài khoản" trong
  phần bối cảnh là sai** — đó là số tài khoản mà lượt khôi phục không sửa được, không phải
  số tài khoản cần sửa. Mười trong số đó vốn đã có grant tốt. Đã sửa và ghi vào mục
  Clarifications.

Điểm dễ đọc sai nhất của đặc tả này, đã ghi ở Notes: mã lý do `stale_event` hiện mang
**hai** nghĩa khác nhau, và FR-001 chỉ loại bỏ một nghĩa trong số đó.

## Notes

- Đặc tả này là **sửa lỗi**, không phải tính năng mới: không có endpoint mới, không có
  trường sự kiện mới, không đổi lược đồ.
- Nguyên nhân bốn lỗi và phép đo production đều đã được xác minh bằng log dịch vụ AI và
  truy vấn chỉ đọc trước khi viết đặc tả, nên không còn giả định nào đáng ngờ về dữ
  liệu thật.
- Điểm dễ đọc sai nhất của đặc tả này: mã lý do `stale_event` hiện mang **hai** nghĩa
  khác nhau, và FR-001 chỉ loại bỏ một nghĩa trong số đó. Người đọc `quickstart.md` của
  spec 024 và người đọc runbook vận hành cần biết điều này.