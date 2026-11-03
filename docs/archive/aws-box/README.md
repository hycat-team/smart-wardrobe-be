# Archive — Trục deploy AWS box (GĐ2)

> **Trạng thái**: archived ngày **2026-11-02** theo [ADR 017](../../decisions/017-retire-aws-box.md).
> Instance AWS Lightsail đã bị xoá; backend chỉ còn một trục deploy duy nhất là VPS
> (`deploy-vps-services.yml`). Không file nào trong thư mục này còn được CI/CD sử dụng.

## Vì sao lưu trữ

Trục thứ hai (`api-v2.closy.hycat.online` → box Lightsail 6 container + managed infra)
đã ngừng tồn tại cùng instance. Giữ nguyên workflow và tooling trong đường dẫn sống sẽ khiến
GitHub Actions vẫn hiện một workflow không thể chạy, và runbook vẫn dẫn người vận hành tới
hạ tầng không còn.

## Bản đồ file

| File trong archive | Đường dẫn gốc | Ghi chú |
|---|---|---|
| `deploy-aws.yml` | `.github/workflows/deploy-aws.yml` | Workflow CD trục box (`aws-production`, tag `aws-*`). GitHub chỉ chạy file nằm trong `.github/workflows/`. |
| `docker-compose.aws.yml` | `deployments/docker-compose.aws.yml` | Topology 6 container (http × 3 + worker × 3) + Nginx TLS. |
| `nginx/aws/*.template` | `deployments/nginx/templates/aws/` | Server block `api-v2` + `default.conf` của box. |
| `scripts/aws-env/*` | `deployments/scripts/aws-env/` | PowerShell sinh `.env.aws` từ `.env.production` và khối lệnh `gh secret set` / `scp`. |
| `scripts/rabbitmq-topology/03-migrate-aws.sh` | `deployments/scripts/rabbitmq-topology/03-migrate-aws.sh` | Migrate queue type trên CloudAMQP. Cần `common.sh` cùng thư mục gốc mới chạy được. |
| `env/.env.aws.example` | `.env.aws.example` | Template env của box (placeholder, không secret). |
| `env/.env.ollama.example` | `.env.ollama.example` | Template env EC2 Ollama (placeholder, không secret). |
| `terraform/ollama-ec2/**` | `deployments/terraform/ollama-ec2/` | Terraform EC2 Ollama: EIP, SG, EBS, IAM/SSM, Secrets Manager, Budget. |
| `ollama/*` | `deployments/ollama/` | Bootstrap EC2, test local, Nginx proxy template cho Ollama. |
| `runbooks/aws-runbook.md` | `docs/operations/aws-runbook.md` | Runbook dựng/teardown box. |
| `runbooks/ollama-trial-runbook.md` | `docs/operations/ollama-trial-runbook.md` | Runbook trial Ollama (feature 032). |
| `runbooks/ollama-trial-report.md` | `docs/operations/ollama-trial-report.md` | Báo cáo kết quả trial. |

## Không lưu trữ (giữ nguyên có chủ đích)

- `specs/032-ollama-lightsail-trial/` — hồ sơ feature (contract, plan, tasks; T039 đã WAIVED).
- ADR 005, 006, 011 — quyết định lịch sử, chỉ được ADR 017 tham chiếu.
- `docs/system-design/presentation/**` và `docs/SEP490_Report-Task2/**` — tài liệu báo cáo/demo.
- `deployments/scripts/aws-env/aws-local.env` và `deployments/scripts/aws-env/outcome/` — file
  **chứa secret thật**, vẫn nằm ở đường dẫn cũ để `.gitignore` (dòng 733–734) tiếp tục hiệu lực.
  Xoá sau khi đã rotate toàn bộ key managed.
- Nginx mode `services-redirect` và biến `API_V2_DOMAIN` trên VPS — vẫn dùng cho grace redirect,
  không phụ thuộc box.

## Cách khôi phục (nếu cần dựng lại trục box)

1. Move file về đúng đường dẫn gốc theo bảng trên (`git mv docs/archive/aws-box/deploy-aws.yml .github/workflows/`, ...).
2. Khôi phục environment `aws-production` + 5 secret `AWS_*` (`AWS_HOST`, `AWS_USER`,
   `AWS_SSH_PORT`, `AWS_SSH_KEY`, `AWS_KNOWN_HOSTS`) trên GitHub.
3. Dựng lại hạ tầng theo `runbooks/aws-runbook.md`, tạo DNS `api-v2`, cấp cert.
4. Trả `common.sh` / `01-check-drift.sh` / `04-verify.sh` về nhánh `aws` theo diff của ADR 017
   nếu muốn dùng lại tooling topology cho box.
