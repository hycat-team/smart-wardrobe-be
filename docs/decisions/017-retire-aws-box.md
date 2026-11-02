# ADR 017 - Retire the AWS box deployment axis

*   **Status**: Accepted
*   **Decision Date**: 2026-11-02
*   **Decision Maker**: repo owner
*   **Supersedes**: ADR 010 (phần trục box ngoài AWS — §2.3, §2.4 và các hệ quả liên quan)

## 1. Context

Instance AWS Lightsail chạy trục thứ hai (`api-v2.closy.hycat.online`, 6 container
`http`×3 + `worker`×3 + Nginx) đã bị xoá ngày 2026-11-02. Instance đó là compute duy nhất
của trục box ngoài; không còn DNS, TLS, host hay `docker compose ps` nào để vận hành.
Workflow `deploy-aws.yml` vì thế không còn đích đến: nó vẫn xuất hiện trong danh sách
GitHub Actions, vẫn nhận tag `aws-*`, và sẽ đỏ ở bước `scp`/`ssh` nếu ai đó chạy lại.

Sau khi box mất, VPS là môi trường duy nhất của backend: `deploy-vps-services.yml`
(3 service, 6 tiến trình trong 3 container) là đường deploy duy nhất còn tồn tại.
Trial Ollama (feature 032) cũng gắn với trục này: proxy EC2 chỉ mở 443 cho static IP
của box, nên trial không thể chạy tiếp dù EC2 còn hay không.

## 2. Decision

1. Archive `.github/workflows/deploy-aws.yml` (và toàn bộ tooling của trục box) vào
   `docs/archive/aws-box/`. GitHub chỉ thực thi file nằm trong `.github/workflows/`,
   nên archive là cách tắt workflow mà vẫn giữ nguyên nội dung để khôi phục. Bản đồ
   file gốc → archive nằm trong `docs/archive/aws-box/README.md`.
2. Archive các artefact deploy của box: `deployments/docker-compose.aws.yml`,
   `deployments/nginx/templates/aws/`, `deployments/scripts/aws-env/`,
   `deployments/scripts/rabbitmq-topology/03-migrate-aws.sh`, `.env.aws.example`.
3. Archive tooling trial Ollama trên EC2: `deployments/terraform/ollama-ec2/`,
   `deployments/ollama/`, `.env.ollama.example`, cùng hai tài liệu
   `docs/operations/ollama-trial-runbook.md` và `ollama-trial-report.md`.
4. Archive `docs/operations/aws-runbook.md`; mọi runbook/tài liệu còn sống không trỏ
   tới box nữa. `docs/operations/shared-reference.md` còn một trục VPS duy nhất.
5. Tooling topology (`rabbitmq-topology/`) thu về VPS-only: bỏ nhánh `aws` trong
   `01-check-drift.sh`/`04-verify.sh`, bỏ `AWS_*` trong `common.sh`.
6. Giữ nguyên có chủ đích: `specs/032-ollama-lightsail-trial/` (hồ sơ feature),
   ADR 005/006/011 (quyết định lịch sử), tài liệu báo cáo/demo
   (`docs/system-design/presentation/`, `docs/SEP490_Report-Task2/`), và nginx mode
   `services-redirect` + biến `API_V2_DOMAIN` trên VPS (dùng cho grace redirect,
   không phụ thuộc box).
7. Hai file chứa secret thật `deployments/scripts/aws-env/aws-local.env` và
   `deployments/scripts/aws-env/outcome/` **không** được move vào archive (sẽ thoát
   `.gitignore`). Chúng ở lại đường dẫn cũ cho tới khi owner rotate key và xoá tay.

## 3. Consequences

Tích cực: một đường deploy duy nhất để audit; Actions không còn workflow trỏ tới hạ tầng
đã xoá; runbook không dẫn người vận hành tới box không tồn tại; tooling topology chỉ còn
một nhánh nên ít lỗi cấu hình hơn.

Tiêu cực: mất khả năng dựng lại trục box trong một lệnh — phải move file về chỗ cũ, tạo
lại environment/secrets `AWS_*`, dựng hạ tầng theo runbook đã archive, và khôi phục nhánh
`aws` của tooling topology theo diff của ADR này. Trial Ollama không còn đường chạy cho tới
khi có môi trường thay thế; quyết định về provider AI text vẫn dựa trên báo cáo đã archive.
