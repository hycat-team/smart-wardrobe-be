# Quy trình làm việc với Git Branch (Branch Strategy)

Dự án áp dụng quy trình Git Flow rút gọn:

- `main`: Nhánh chứa mã nguồn ổn định nhất đang chạy trên production.
- `dev`: Nhánh tích hợp chính cho môi trường staging và phát triển chung.
- Nhánh tính năng: Tạo từ `dev` có tiền tố `feat/` (ví dụ: `feat/digital-sample-lab`).
- Nhánh sửa lỗi: Tạo từ `dev` hoặc `main` có tiền tố `bugfix/` hoặc `hotfix/`.

Mọi Pull Request (PR) cần được duyệt bởi ít nhất 1 thành viên khác và phải vượt qua tất cả các kiểm tra CI (Lints, Tests, Build) trước khi merge.
