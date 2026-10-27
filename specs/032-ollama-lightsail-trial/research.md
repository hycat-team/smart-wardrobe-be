# Phase 0 Research: Ollama trên EC2 cho backend Lightsail

**Feature**: `032-ollama-lightsail-trial` | **Date**: 2026-10-26

Tài liệu này chốt toàn bộ quyết định kỹ thuật còn mở sau khi đọc code, cấu hình và runbook hiện
hữu. Mỗi mục dùng cấu trúc Decision / Rationale / Alternatives. Không còn câu hỏi kỹ thuật chưa
được giải quyết trước Phase 1.

## R1. EC2 public có kiểm soát, không dùng VPC peering

**Decision**: Một EC2 CPU-only tại `ap-southeast-1`, có Elastic IP và hostname riêng (bản ghi DNS
mới, không phải domain API). Security group chỉ cho phép cổng proxy 443 từ static IP của box
Lightsail; cổng 80 mở công cộng nhưng chỉ phục vụ ACME challenge và redirect, không proxy suy
luận; SSH đóng, quản trị qua Systems Manager Session Manager. Nginx trên EC2 terminate TLS, so khớp chính xác Bearer token lấy từ Secrets
Manager rồi proxy vào `127.0.0.1:11434`. Ollama chỉ bind loopback. Chứng chỉ lấy bằng Let's
Encrypt HTTP-01 cho hostname riêng; apply Terraform theo hai bước — cấp Elastic IP trước, tạo tay A
record của subdomain trỏ tới IP đó, rồi apply phần còn lại — nên không cần Route53 provider và
không đụng DNS của API.
Nếu không muốn mở cổng 80, phương án thay thế là self-signed CA và cấu hình `SSL_CERT_FILE` cho
container AI, nhưng phương án này cần thay đổi mount của Compose nên không được chọn.

**Rationale**: Box Lightsail nằm ngoài VPC AWS nên EC2 private không thể nhận kết nối trực tiếp.
VPC peering hoặc VPN overlay thêm thành phần mạng phải vận hành cho một thử nghiệm một tuần.
Public endpoint + SG allowlist + TLS + Bearer đạt mức cô lập cần thiết: không có Internet ingress
tới cổng suy luận, và request không có token đúng bị từ chối ngay tại proxy. Hostname riêng cũng
cho phép TLS hợp lệ mà không đụng DNS của API.

**Alternatives considered**:

- VPC peering Lightsail ↔ VPC: bảo mật hơn nhưng thêm VPC, route table, peering và cấu hình khó
  audit cho một tuần thử nghiệm.
- Tailscale/WireGuard: tránh mở cổng nhưng thêm một mạng overlay và một điểm vận hành mới.
- Expose trực tiếp `11434` hoặc coi API key rỗng là xác thực: không an toàn và sai hành vi Ollama.
- TLS self-signed không pin CA: dễ bị cấu hình `InsecureSkipVerify`; bị loại.

## R2. Terraform tối giản cho EC2

**Decision**: `deployments/terraform/ollama-ec2` quản lý EC2 `c6i.2xlarge` (x86, 8 vCPU/16 GiB),
Elastic IP, security group, EBS, IAM instance profile cho SSM, quyền đọc secret và Secrets Manager.
State là local, nằm ngoài repo, apply tay từ máy dev; không thêm S3/DynamoDB backend và không dùng
EKS, ALB, ACM, Kustomize, Helm, Route53 hay Vercel provider. Cloud-init cài Ollama pin version,
pull model pin digest, cấu hình Nginx proxy và pre-warm model.

**Rationale**: Repo chưa có IaC cho EC2; một module Terraform nhỏ giữ được khả năng tái tạo,
review plan và audit IAM/network, đồng thời không kéo theo công cụ mới. Local state hợp với thử
nghiệm một tuần do một người vận hành và không tạo thêm tài nguyên ngoài phạm vi spec. Cloud-init
giữ bootstrap gần tài nguyên và không cần cơ chế cấu hình ngoài.

**Alternatives considered**:

- Tạo EC2 thủ công bằng Console/CLI: nhanh nhưng khó tái tạo, dễ lệch security group.
- Ansible/Chef: thêm toolchain không cần cho một máy.
- EKS/ECS: vượt phạm vi và không cần cho một container Ollama.

## R3. Lightsail giữ nguyên, chỉ đổi env

**Decision**: Không sửa `deployments/docker-compose.aws.yml`, `deployments/nginx/templates/aws`,
`.github/workflows/deploy-aws.yml`, Dockerfile hay pipeline. Chỉ bổ sung biến AI trong
`/opt/closy-aws/.env.aws` trên box: `CHAT_TEXT_PRIMARY_PROVIDER=openai`,
`RECOMMENDATION_TEXT_PRIMARY_PROVIDER=openai`, `FREE_TEXT_PRIMARY_PROVIDER=openai`, endpoint đầy đủ
tới `/v1/chat/completions`, model Ollama, token proxy; vision/embedding giữ Gemini. `.env.aws.example`
chỉ thêm placeholder, không chứa secret.

**Rationale**: Constitution yêu cầu chuyển môi trường chỉ bằng config/DNS. Giữ nguyên artefact
Lightsail cũng bảo đảm rollback chỉ là đổi env và tạo lại container AI, đồng thời không tạo nhánh
code theo môi trường. Static IP của box Lightsail là đầu vào bắt buộc của allowlist security group;
runbook phải kiểm tra IP này trước mỗi lần bật thử nghiệm.

**Alternatives considered**:

- Thêm Ollama vào Compose Lightsail: tranh CPU/RAM với sáu container backend và cần sửa artefact
  deployment.
- Dùng EC2 private trong cùng VPC: cần peering như R1.
- Đổi domain API sang endpoint mới: trái yêu cầu giữ nguyên frontend và DNS API.

## R4. Adapter OpenAI được tái sử dụng nhưng phải khóa contract

**Decision**: Tiếp tục dùng `shared/infrastructure/ai/openai`, nhưng endpoint AWS phải là URL đầy
đủ `https://<ollama-host>/v1/chat/completions` vì adapter POST nguyên chuỗi cấu hình và không tự
nối path. Bổ sung typed upstream error giữ HTTP status, truyền `temperature` kể cả giá trị 0, luôn
phát provider/model winner ngay cả khi usage token bằng 0, validate structured response theo
`ResponseSchema`, và contract test cho request/header, non-stream, SSE, `[DONE]`, usage-only,
zero/missing usage, cancellation, malformed body và pinned Ollama version/model.

Model được pin là `qwen2.5:3b-instruct-q4_K_M` (non-thinking); `reasoning_effort` để trống và
không gửi field. Contract test phải chứng minh không có reasoning content/token, JSON vẫn hợp lệ.
`ReasoningEffort` vẫn được plumb từ `config.APIProviderConfig` → `app_ai.TextProviderConfig` →
payload để giữ tương thích provider khác, nhưng trial không dùng. Timeout thuộc về từng attempt:
Ollama 90 giây, Gemini fallback 30 giây, tổng nằm trong write timeout 120 giây hiện hữu; text
client không đặt `Timeout` toàn cục 30/45/30 giây nữa.

**Rationale**: Code hiện có đã gửi Bearer, `response_format` và `stream_options.include_usage`,
nhưng chưa có test adapter; mọi non-2xx bị ép thành lỗi 503 mất phân loại; temperature chưa được
đưa vào payload; streaming không luôn ghi nhận provider nếu usage vắng. Ollama chỉ cam kết một
subset của OpenAI API nên tương thích phải được chứng minh bằng code/test.

**Alternatives considered**:

- Chỉ đổi endpoint/model: không phát hiện khác biệt streaming/schema/usage và không sửa fallback
  sai.
- Viết Ollama adapter native `/api/chat`: nhân đôi integration trong khi contract OpenAI đủ cho
  scope.
- Dùng model thinking như `qwen3:4b`: Ollama 0.40.11 không tắt được thinking qua `/v1` (và
  `think:false` native cũng không), reasoning lẫn vào content, structured output và latency không
  đạt; bị loại sau local test.

## R5. Routing và fallback tối đa hai attempt

**Decision**: `chat_text`, `recommendation_text`, `free_text` dùng Ollama/OpenAI primary; vision
và embedding luôn giữ Gemini primary. Text service chạy đúng một primary attempt và tối đa một
Gemini fallback, không chạy `freeFailover` attempt thứ ba. Fallback chỉ khi timeout/deadline
upstream, transport/connection, HTTP 429, HTTP 5xx, response malformed/empty hoặc output/schema
validation thất bại trước khi trả content. Không fallback khi client `context.Canceled`, HTTP 4xx
khác 429, auth/config/unsupported request. Streaming chỉ fallback nếu chưa emit chunk; sau chunk
đầu thì trả lỗi trên stream, không ghép nội dung từ provider khác. Fallback phải qua limiter
Gemini. Mỗi attempt có deadline riêng (Ollama 90 giây, Gemini 30 giây) và một semaphore giới hạn
2 request text đồng thời; semaphore giữ chỗ trước limiter và nhả sau khi attempt kết thúc. Deadline
áp theo provider: `openai` dùng 90 giây; `google` primary giữ timeout hiện hữu của từng file config
(production 30/45/30; local 30/30/30). SSE chat dùng field hiện hữu `ai_chat_chunk_deadline_seconds`,
override 90 giây qua env trong trial để không cắt stream trước fallback; default 30 giây giữ nguyên
cho môi trường khác.

**Rationale**: `text_service.go` hiện fallback gần như mọi lỗi và không gọi limiter fallback;
`freeFailover` có thể tạo attempt thứ ba. HTTP 429 là ngoại lệ duy nhất trong nhóm 4xx vì đặc tả
xếp nó vào quá tải. Optional output validator phải chạy trong provider lifecycle; semantic check
sau khi provider đã trả về thì quá muộn để fallback đúng. Tổng 90 + 30 bằng write timeout 120 nên
fallback là best-effort ở biên; sự cố bị cắt phải ghi vào báo cáo thay vì tự nới timeout.

**Alternatives considered**:

- Fallback mọi lỗi: che lỗi request/auth và tạo chi phí Gemini không cần thiết.
- Retry Ollama rồi fallback: vượt giới hạn một attempt Ollama.
- Fallback sau khi stream đã phát chunk: tạo response trộn/duplicated cho client.

## R6. Tách admission/usage không tính tiền khỏi cost reservation

**Decision**: Refactor vòng đời thành attempt-aware callback chạy ngay trước mỗi provider call:

1. `provider=openai`: gọi đường `AdmitUnbilledAttempt` để kiểm tra daily attempt cap hiện hữu và
   tạo/cập nhật usage event với `logical_route = plan.Route` (giữ route nghiệp vụ), provider
   `openai`, `reserved_cost=actual_cost=0`; tuyệt đối không gọi pricing repository, phép tính tiền,
   `ReserveBillableAttempt` hoặc `ConfirmBillableAttempt`. Attempt `openai` vẫn tiêu thụ cap hiện
   hữu; không thêm quota mới. Rate limiter và semaphore vẫn áp dụng.
2. Nếu Ollama lỗi đủ điều kiện, đóng attempt bằng telemetry và trước khi gọi Gemini phải chạy
   `PrepareBillableAttempt`: khóa usage event theo `request_id` bằng `SELECT ... FOR UPDATE`, lấy
   pricing, cập nhật cùng event sang Gemini + reserved cost và cộng ledger. Không tạo event thứ hai
   vì `ai_usage_events.request_id` là UNIQUE.
3. Chỉ khi `PrepareBillableAttempt` thành công mới gọi Gemini; sau đó confirm/release trên cùng
   event theo provider/model thực tế.
4. Usage event database ghi provider thắng và kết quả cuối; từng attempt được audit bằng structured
   log/metric để không phải thêm bảng.
5. Callback attempt-aware được định nghĩa trong `shared/application/ai` (`AttemptHooks` hoặc tương
   đương) và được `text_service` gọi trước/sau mỗi provider call cho cả non-stream và stream;
   orchestrator (services/ai) cài đặt để gọi `AdmitUnbilledAttempt`/`PrepareBillableAttempt`.
   Callback lỗi thì không gọi provider; cancellation không fallback.
6. `CostUseCase.Prepare` không ghi event cho `provider=openai`; `AdmitUnbilledAttempt` là writer
   duy nhất và upsert theo `request_id` (idempotent: 2 lần gọi = 1 event, cap không tăng). Nếu hết
   daily cap, giữ hành vi degrade free route hiện hữu, không trả 429 mới.

Repository path non-monetary phải có tên/API riêng, không tái sử dụng hàm cost `Reserve` với số 0,
để dependency spy chứng minh OpenAI không chạm đường giữ chỗ tiền. `withUsageStamp`/`stampUsage`
phải đặt `Unbilled=true` khi `provider=openai` trước khi xét danh sách capability; Confirm có guard
theo provider để `openai` luôn cost 0 ngay cả khi caller truyền cờ sai. Cơ chế
`AI_UNBILLED_CAPABILITIES` hiện hữu không được dùng cho quy tắc này: nó miễn phí theo capability,
nên sẽ làm lượt Gemini fallback của text cũng thành miễn phí.

**Rationale**: `CostUseCase.Prepare` hiện tra giá trước call; đánh dấu unbilled theo capability sẽ
làm cả Gemini fallback thành miễn phí. Nếu chỉ xác nhận cost sau kết quả, hard cap Gemini có thể
bị vượt vì fallback chưa reserve. Callback trước từng attempt vừa giữ đúng reservation-first vừa
biết provider thật sự sắp được gọi.

**Alternatives considered**:

- Đánh dấu toàn bộ text capability unbilled: làm Gemini fallback có cost 0 sai yêu cầu.
- Tạo pricing row OpenAI giá 0: vẫn gọi pricing/reservation, bị đặc tả cấm.
- Skip toàn bộ admission: làm mất daily attempt/quota hiện hữu.
- Reserve Gemini worst-case ngay từ đầu: Ollama success vẫn chạm logic tiền tệ và reserve thừa.

## R7. Cấu hình, secret và giới hạn CPU

**Decision**: Secret runtime của EC2 (Bearer token) nằm trong AWS Secrets Manager, được cloud-init
đọc qua instance profile; không copy `.env.aws` hay secret qua workflow. Tạo `.env.ollama.example`
chỉ chứa placeholder. Trên Lightsail, token được đặt trong `.env.aws` hiện hữu và không commit.
Concurrency 2 được giữ ở hai đầu: một semaphore dùng chung cho mọi attempt text tới Ollama trong
Go AI Service (mặc định 2, cấu hình qua env) và `OLLAMA_NUM_PARALLEL=2` + `OLLAMA_MAX_QUEUE` trên
EC2. Timeout từng attempt là 90 giây cho Ollama và 30 giây cho Gemini; text client không còn
`Timeout` toàn cục 30/45/30 giây. RPM giữ nguyên giá trị hiện hữu — semaphore và queue Ollama là giới
hạn chính của trial, nên không thêm biến RPM. `ai_chat_chunk_deadline_seconds` là field hiện hữu,
được override 90 giây qua env trong trial và giữ default 30 ở nơi khác. Không thay đổi DB pool vì
số container Lightsail không tăng.

**Rationale**: Giữ secret ra khỏi Git/manifest/log. Concurrency 2 phù hợp với một EC2 CPU và tải
kiểm thử mặc định; semaphore phía Go bảo vệ bộ nhớ/connection của AI Service, còn giới hạn phía
Ollama tránh xếp hàng sâu trong runtime. Timeout 90/30 vừa khít `write_timeout_seconds = 120` nên
fallback luôn còn ngân sách. Ứng dụng vẫn nhận env, không import AWS SDK.

**Alternatives considered**:

- Gắn secret trực tiếp trong cloud-init hoặc biến GitHub: tăng rủi ro lộ và rotation kém.
- Giữ RPM cao: hàng đợi Ollama tăng, fallback xảy ra vì lý do cấu hình chứ không phải chất lượng.
- Giảm DB pool: không cần vì không nhân bản container.

## R8. Quan sát, chi phí và vòng đời ngày thứ 7

**Decision**: Log stdout có service/version/environment/correlation ID nhưng không có prompt/
secret; CloudWatch agent trên EC2 thu log proxy/Ollama và metric CPU/memory/disk. Metric tối thiểu
gồm provider/model, attempt/fallback reason, latency, schema failure, usage và cost; retention 14
ngày. Gắn tag `Project`, `Feature=032`, `Environment=aws-course`, `Owner`, `ExpiresOn`; AWS Budget
80 USD cảnh báo 50/80/100%. Hết ngày 7 chỉ sinh báo cáo go/no-go; không tự rollback, tắt hoặc
destroy.

**Rationale**: Trial phải cung cấp đủ evidence ra quyết định nhưng đặc tả cấm auto teardown.
Retention ngắn, tag và budget kiểm soát chi phí mà không cần một stack monitoring riêng.

**Alternatives considered**:

- Tự teardown ngày 7: trái yêu cầu và làm mất môi trường để điều tra.
- Dựng Prometheus/Grafana riêng: vận hành và chi phí vượt nhu cầu một tuần.
- Log prompt để debug: rủi ro dữ liệu nhạy cảm và không cần cho success criteria.

## R9. Ranh giới Lightsail/VPS và tác động frontend

**Decision**: Lightsail là môi trường duy nhất của backend trong thử nghiệm: artefact deployment
giữ nguyên, chỉ env AI thay đổi. VPS hoàn toàn ngoài phạm vi: không sửa file đặc thù, không deploy/
restart/probe, không đọc secret, không đổi DNS, không gọi workflow VPS và không lấy VPS làm
dependency. Frontend Vercel không đổi code/deploy; public API, event và DTO giữ nguyên. Readiness
`/readyz`, worker drain, EKS/Kubernetes, ALB/ACM và DNS cutover của API đều bị loại khỏi phạm vi;
không thêm endpoint public nào.

**Rationale**: Backend đã chạy ổn định trên Lightsail và yêu cầu mới cho phép giữ nguyên. Kiểm
chứng VPS bằng diff/allowlist phía repository là đủ; probe trực tiếp VPS sẽ tự vi phạm scope.

**Alternatives considered**:

- Nâng cấp đồng thời VPS bằng shared code: không cần thiết; VPS tiếp tục image cũ độc lập.
- Dùng VPS làm proxy/DNS/rollback automation: tạo phụ thuộc bị cấm.
- Thay base URL/frontend code: domain API giữ nguyên nên không cần.

## R10. Model qwen2.5:3b-instruct-q4_K_M, pin digest và kiểm chứng local trước

**Decision**: Dùng `qwen2.5:3b-instruct-q4_K_M` (non-thinking) cho cả test local và EC2, pin
version Ollama và digest model; `reasoning_effort` để trống. Trước khi lên EC2, chạy cùng
model/digest bằng Ollama native trên máy dev: GPU để chạy contract test và drill fallback, sau đó
`OLLAMA_NUM_GPU=0` để đo latency CPU tham chiếu. Chỉ khi contract test và fallback drill local đạt
mới apply Terraform; latency GPU local không được dùng thay SC-005.

**Rationale**: `qwen2.5:3b-instruct-q4_K_M` vừa 6 GiB VRAM của máy dev nên test chức năng nhanh,
đồng thời vẫn có khả năng chạy trên 8 vCPU của `c6i.2xlarge`; local test cho thấy nó trả lời không
reasoning và ổn định. `qwen3:4b` đã bị loại vì Ollama 0.40.11 không tắt được thinking qua `/v1`,
reasoning lẫn vào content và phá JSON/latency. Pin digest bảo đảm hành vi giữa local và EC2 giống
nhau. Test local trước giảm rủi ro phát hiện lỗi contract sau khi đã trả tiền EC2. Nếu benchmark
cho thấy recommendation không thể hoàn tất trong 90 giây, phải giảm prompt/output hoặc chấp nhận
fallback và ghi rõ SC-001 không đạt — không hạ ngưỡng sau khi đã bắt đầu đo.

**Alternatives considered**:

- `qwen3:4b`: mạnh hơn nhưng không tắt được thinking trên Ollama 0.40.11; reasoning lẫn content,
  structured output sai và latency cao — bị loại sau local test.
- `gemma3:4b`: đa ngôn ngữ tốt nhưng chậm hơn trên CPU; chưa thử.
- Chỉ test trên EC2: phát hiện lỗi contract/chứng chỉ/thinking muộn, tốn thời gian và tiền EC2.
- Dùng GPU local làm bằng chứng latency: không đại diện cho CPU-only, dễ chốt timeout sai.
