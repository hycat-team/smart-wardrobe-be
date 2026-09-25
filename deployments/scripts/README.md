# AWS env helper scripts

Bộ 2 script PowerShell giúp dựng `AWS EXTERNAL BOX` env từ VPS `.env.production` và sinh
khối lệnh copy-paste để nạp secret/env lên box. Đã được "hợp thức hóa" từ `docs/temp/`.

## Files

| File | Tracked | Mục đích |
|---|---|---|
| `aws-env-bootstrap.ps1` | ✅ | Đọc VPS `.env.production` + template `.env.aws.example`, sinh `outcome/.env.aws` với các giá trị cần điền (TODO) |
| `aws-env-to-copyblock.ps1` | ✅ | Validate `outcome/.env.aws`, sinh `outcome/aws-secrets-to-set.txt` (khối lệnh gh secret / scp / recreate) |
| `aws-local.env.example` | ✅ | Mẫu file settings nhạy cảm cục bộ |
| `aws-local.env` | ❌ gitignored | Settings nhạy cảm cục bộ (KeyPath, Box IP, repo...); script tự đọc |
| `outcome/` | ❌ gitignored | Mọi file sinh ra (`.env.aws`, `aws-secrets-to-set.txt`, `aws_known_hosts`) — chứa secret |

> `aws-local.env` và `outcome/` **không bao giờ được commit**.

## Chuẩn bị

```powershell
# từ repo root
Copy-Item deployments/scripts/aws-local.env.example deployments/scripts/aws-local.env
# điền VPS_ENV_PROD / AWS_BOX_IP / AWS_SSH_KEY_PATH / GITHUB_REPO
```

## Sử dụng

```powershell
# 1. Sinh .env.aws (điền các dòng '# TODO' trong outcome/.env.aws)
powershell -ExecutionPolicy Bypass -File deployments/scripts/aws-env-bootstrap.ps1

# 2. Sinh khối lệnh secrets/upload (sau khi hết '# TODO')
powershell -ExecutionPolicy Bypass -File deployments/scripts/aws-env-to-copyblock.ps1
#    (tuỳ chọn) thêm -WithHeredoc để có fallback tạo .env.aws trực tiếp trên box
```

## Ghi chú theo môi trường

- `GOOGLE_REDIRECT_URI` là **đặc thù AWS box** (`https://api-v2.closy.hycat.online/api/v1/auth/google/callback`)
  nên **không copy từ VPS**; script tự set constant. URI này phải được đăng ký trong Google Cloud Console.
- `GOOGLE_CLIENT_IDS` / `GOOGLE_CLIENT_SECRET`: copy từ VPS **nếu** `.env.production` đã có; nếu chưa
  (feature 021 mới thêm), script phát sinh dòng `# TODO` để bạn điền.
- Các khoá VPS-only (`API_DOMAIN`, `COMPOSE_PROJECT_NAME`, `NGINX_MODE`, `INGRESS_*_URL`, ...) không thuộc
  template AWS nên được bỏ qua một cách chủ đích.

## Option precedence

`CLI param` > `aws-local.env` > default. Các khoá trong `aws-local.env`:

`VPS_ENV_PROD`, `AWS_BOX_IP`, `AWS_SSH_KEY_PATH`, `GITHUB_REPO`,
`AWS_ENV_TEMPLATE`, `AWS_ENV_FILE`, `AWS_COPYBLOCK_OUT`, `AWS_KNOWN_HOSTS`.

## Bảo mật

- Không commit `aws-local.env`, `outcome/`, hay bất kỳ file chứa secret.
- `outcome/aws-secrets-to-set.txt` chứa secret (kể cả heredoc khi dùng `-WithHeredoc`) — **xoá sau khi dùng**.
- File `.env.aws` chỉ nên nằm trên box (`/opt/closy-aws/.env.aws`, `chmod 600`).
