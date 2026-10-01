# Ops scripts

Script vận hành, gom theo từng nhóm công việc. Mỗi nhóm một thư mục riêng kèm `README.md`
riêng — đọc README của nhóm đó trước khi chạy, không đọc chung.

| Thư mục | Môi trường | Ngôn ngữ | Dùng khi |
|---|---|---|---|
| [`aws-env/`](aws-env/README.md) | Local (Windows) | PowerShell | Dựng/gửi `.env.aws` lên AWS box từ VPS `.env.production` |
| [`rabbitmq-topology/`](rabbitmq-topology/README.md) | VPS + AWS box | Bash | Kiểm tra & đồng bộ queue type (classic ↔ quorum) khi `topology.go` đổi |

## Quy ước

- **Mỗi nhóm là một thư mục**, có README riêng. Thêm nhóm mới thì tạo thư mục + README, không
  nhồi script vào gốc `scripts/`.
- **File sinh ra chứa secret phải được gitignore.** Xem mục liên quan trong `.gitignore` —
  repo liệt kê từng path cụ thể thay vì wildcard, để dễ review khi thêm/sửa.
- **Ưu tiên `Makefile target` hơn lệnh thủ công** khi có target tương đương
  (`AGENTS.md` §4). Script trong đây là fallback cho thao tác trên box từ xa, không phải
  đường chính trong dev loop.