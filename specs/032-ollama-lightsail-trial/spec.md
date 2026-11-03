# Đặc tả tính năng: Thử nghiệm Ollama trên EC2 và tính tiền Gemini-only

**Feature Branch**: `032-ollama-lightsail-trial`

**Created**: 2026-10-26

**Status**: Closed - demo-only validation; decision = adjust

**Input**: Thử nghiệm trong 1 tuần việc chạy Ollama bằng CPU trên một máy EC2 riêng và dùng nó làm
provider text chính cho Go AI Service, trong khi backend ba dịch vụ tiếp tục chạy trên AWS
Lightsail đúng như hiện tại. Không triển khai EKS, không đổi ALB/ACM/DNS của API, không thay đổi
Dockerfile, Compose hay Nginx của Lightsail ngoài cấu hình môi trường. Mọi lượt dùng provider `openai`
(bao gồm Ollama OpenAI-compatible) phải bỏ qua hoàn toàn bước tính tiền; chỉ Gemini được đi qua
luồng giá và giữ chỗ chi phí. Không thao tác, không phụ thuộc và không thay đổi deployment VPS độc
lập. Môi trường AWS chỉ phục vụ môn học, không có request hoặc người dùng thật nên không đặt mục
tiêu uptime.

## Clarifications

### Session 2026-10-26

- Q: Backend chạy ở đâu trong tuần thử nghiệm? → A: Giữ nguyên trên AWS Lightsail như hiện tại;
  không dùng EKS/Kubernetes, không đổi ALB/ACM/DNS của API, chỉ thay đổi cấu hình môi trường.
- Q: Ollama chạy ở đâu và được quản lý thế nào? → A: Một EC2 CPU-only riêng, có Elastic IP, quản
  lý bằng Terraform tối giản; quản trị qua Systems Manager, không mở SSH.
- Q: Box Lightsail kết nối tới EC2 bằng cách nào? → A: EC2 dùng public endpoint nhưng security
  group chỉ cho phép static IP của box Lightsail; proxy bắt buộc TLS và Bearer token, Ollama chỉ
  bind loopback.
- Q: Trong tuần thử nghiệm, Ollama làm primary cho capability nào? → A: Tất cả tác vụ text;
  Gemini giữ vision và embedding.
- Q: Khi Ollama thất bại, chính sách fallback sang Gemini nên là gì? → A: Một lần Ollama, rồi
  fallback Gemini cho lỗi kỹ thuật/output.
- Q: Khi kết thúc 1 tuần thử nghiệm, trạng thái mặc định nên là gì? → A: Giữ nguyên cấu hình
  Ollama đến khi có quyết định thủ công; không tự rollback, tắt hay teardown.
- Q: Readiness `/readyz` và worker drain có còn trong phạm vi? → A: Không; hai hạng mục này bị
  loại khỏi phạm vi cùng với EKS.

### Session 2026-10-26 (chốt trước khi lập kế hoạch)

- Q: Model Ollama nào được dùng và pin thế nào? → A: `qwen2.5:3b-instruct-q4_K_M`; pin version Ollama và digest
  model. Trước khi lên EC2, chạy cùng model/digest trên máy dev bằng Ollama native (GPU để test
  chức năng, sau đó ép CPU để đo latency).
- Q: Loại EC2 nào? → A: `c6i.2xlarge` x86 (8 vCPU/16 GiB) tại `ap-southeast-1`, on-demand; ngân
  sách cảnh báo 80 USD cho tuần thử nghiệm.
- Q: Timeout cho một attempt text? → A: Ollama tối đa 90 giây; Gemini fallback tối đa 30 giây;
  tổng vẫn nằm trong `server.write_timeout_seconds = 120` hiện hữu.
- Q: SC-005 xử lý thế nào khi recommendation prompt lớn trên CPU? → A: Tách ngưỡng theo
  capability — `chat_text`/`free_text` TTFT ≤15 giây; `recommendation_text` TTFT ≤60 giây và
  hoàn tất ≤120 giây; ≥95% yêu cầu Ollama hoàn tất trong 90 giây.
- Q: Giới hạn đồng thời được đặt ở đâu? → A: Semaphore trong Go AI Service (mặc định 2) kết hợp
  `OLLAMA_NUM_PARALLEL=2` và `OLLAMA_MAX_QUEUE` trên EC2.
- Q: Danh sách `AI_UNBILLED_CAPABILITIES` trong tuần thử nghiệm? → A: Giữ
  `free_text,embedding`; quy tắc miễn phí cho provider `openai` triển khai theo provider, không
  dựa vào danh sách capability.
- Q: TLS và DNS cho proxy Ollama? → A: Subdomain riêng; cấp Elastic IP trước, tạo tay A record rồi
  apply phần còn lại; Let's Encrypt HTTP-01; cổng 80 chỉ phục vụ ACME challenge/redirect.
- Q: Terraform state và cách apply? → A: Local state ngoài repo, apply tay từ máy dev; không thêm
  backend S3/DynamoDB và không đụng pipeline.
- Q: Model có thinking mode, xử lý thế nào? → A: Model trial là `qwen2.5:3b-instruct-q4_K_M`
  (non-thinking). `AI_OPENAI_REASONING_EFFORT` để trống; contract test vẫn phải chứng minh response
  không có reasoning content/token và structured output hợp lệ. Không dùng model thinking như
  qwen3 vì Ollama 0.40.11 không tắt được thinking qua `/v1`.
- Q: Attempt `openai` có tiêu thụ daily attempt cap hiện hữu không? → A: Có — giữ nguyên quota
  hiện hữu, không thêm quota mới; `openai` chỉ được miễn phần tính tiền, không được miễn giới hạn
  vận hành. Fallback Gemini dùng lại đúng usage event của request, không tạo event thứ hai.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Ollama phục vụ text AI qua EC2 (Priority: P1)

Người vận hành có thể tạo một EC2 CPU-only chạy Ollama, đặt sau proxy có xác thực và chỉ cho box
Lightsail truy cập, sau đó trỏ cấu hình provider text của AI Service sang endpoint đó mà không
thay đổi Dockerfile hay kiến trúc deployment của backend.

**Why this priority**: Đây là điều kiện để thử nghiệm Ollama. Nếu endpoint không được cô lập,
không xác thực hoặc không thay thế được provider text bằng cấu hình thì thử nghiệm không an toàn
hoặc không đo được kết quả.

**Independent Test**: Áp dụng Terraform cho EC2, chờ proxy và model sẵn sàng, cập nhật biến môi
trường provider text trên Lightsail rồi chạy request kiểm thử. Xác minh request do box Lightsail
gửi được chấp nhận, request từ nguồn khác hoặc thiếu/sai token bị từ chối, và cổng suy luận không
truy cập được từ Internet công cộng.

**Acceptance Scenarios**:

1. **Given** EC2 và proxy Ollama đã sẵn sàng với model được pin, **When** cấu hình provider text
   trên Lightsail trỏ tới proxy, **Then** `chat_text`, `recommendation_text` và `free_text` được
   phục vụ bởi Ollama và provider/model thắng được ghi nhận trong log/metric.
2. **Given** một request hợp lệ từ static IP của box Lightsail, **When** proxy nhận request,
   **Then** request được chuyển tới Ollama qua loopback; request từ IP khác, thiếu token hoặc sai
   token bị từ chối và không tạo truy cập ẩn danh.
3. **Given** một yêu cầu vision hoặc embedding, **When** Go AI Service chọn provider, **Then**
   Gemini vẫn là primary và yêu cầu không được chuyển sang Ollama.
4. **Given** EC2 khởi động lại, **When** proxy và model chưa sẵn sàng, **Then** trạng thái này quan
   sát được và không có request nào bị coi là thành công giả; sau khi model sẵn sàng, traffic được
   phục vụ lại mà không cần đổi cấu hình.

---

### User Story 2 - Dùng Ollama nhưng chỉ Gemini được tính tiền (Priority: P1)

Người dùng sử dụng các tính năng AI qua Go AI Service như trước. Khi Ollama phục vụ yêu cầu qua
giao thức OpenAI-compatible, hệ thống không tra giá, không giữ chỗ chi phí và không tính tiền;
khi Gemini thực sự phục vụ yêu cầu có tính phí, cơ chế tính tiền Gemini hiện hữu vẫn được áp
dụng.

**Why this priority**: Tích hợp OpenAI đã tồn tại nhưng luồng chi phí hiện tại có thể chạy trước
lúc gọi provider, và cơ chế miễn phí hiện tại đang phân loại theo capability. Chỉ đổi endpoint là
chưa đủ để bảo đảm Ollama không bị chặn bởi dữ liệu giá hoặc ghi nhận chi phí sai.

**Independent Test**: Chạy các luồng AI áp dụng được với một kho giá giả có bộ đếm lời gọi. Mọi
yêu cầu do provider `openai`/Ollama phục vụ phải thành công khi không có cấu hình giá và bộ đếm
truy vấn giá/giữ chỗ vẫn bằng 0; yêu cầu do Gemini phục vụ phải đi qua đúng luồng giá và xác nhận
chi phí hiện hữu.

**Acceptance Scenarios**:

1. **Given** provider được chọn là `openai` và endpoint trỏ tới proxy Ollama, **When** một yêu cầu
   AI được thực hiện, **Then** yêu cầu không tra bảng giá, không giữ chỗ chi phí và ghi chi phí
   tiền tệ bằng 0.
2. **Given** không tồn tại cấu hình giá cho model Ollama, **When** Go AI Service gọi model đó,
   **Then** yêu cầu không bị từ chối chỉ vì thiếu cấu hình giá.
3. **Given** provider được chọn là Gemini, **When** một yêu cầu có tính phí thành công, **Then**
   hệ thống vẫn tra giá, giữ chỗ và xác nhận chi phí theo quy tắc hiện hữu.
4. **Given** lượt gọi Ollama đầu tiên bị timeout, quá tải, lỗi kết nối/5xx hoặc trả output không
   hợp lệ, **When** fallback được kích hoạt, **Then** hệ thống gọi Gemini đúng một lần; nếu Gemini
   phục vụ kết quả thì chỉ lượt Gemini được đưa qua luồng tính tiền và lượt Ollama thất bại không
   tạo chi phí tiền tệ.
5. **Given** Ollama trả về số token hoặc không trả đủ số token, **When** hệ thống ghi nhận usage,
   **Then** dữ liệu vận hành vẫn thể hiện provider, model, trạng thái và số token biết được nhưng
   không kích hoạt phép tính tiền cho provider `openai`.
6. **Given** một yêu cầu vision hoặc embedding phát sinh, **When** Go AI Service chọn provider,
   **Then** Gemini vẫn là primary và yêu cầu không được chuyển sang Ollama trong phạm vi thử
   nghiệm.

---

### User Story 3 - Vận hành thử nghiệm trên đường deploy Lightsail hiện hữu (Priority: P2)

Người vận hành có thể bật/tắt thử nghiệm bằng thay đổi biến môi trường trên box Lightsail, phát
hành qua quy trình hiện hữu và quay lại Gemini-only nếu cần, trong khi Dockerfile, Compose, Nginx,
pipeline và deployment VPS giữ nguyên.

**Why this priority**: Một thử nghiệm kéo dài một tuần cần đường bật/tắt lặp lại được và không
làm sai lệch môi trường hiện hữu; thay đổi hạ tầng lớn không còn nằm trong phạm vi.

**Independent Test**: Chỉ sửa biến môi trường AI trên Lightsail, chạy lại quy trình deploy hiện
hữu, xác minh sáu container vẫn khỏe và request text đi qua Ollama; sau đó đổi cấu hình về Gemini
và xác minh dịch vụ phục hồi mà không cần thay đổi Dockerfile hay Nginx.

**Acceptance Scenarios**:

1. **Given** chỉ cấu hình provider/endpoint/model/token thay đổi, **When** quy trình deploy
   Lightsail hiện hữu chạy, **Then** đúng các container bị ảnh hưởng được tạo lại, sáu container
   vẫn sẵn sàng và không có thay đổi Dockerfile, Compose hay Nginx.
2. **Given** thử nghiệm đang bật, **When** kiểm tra phạm vi, **Then** không có tài nguyên hay
   workflow EKS/Kubernetes nào được tạo, không có thay đổi ALB/ACM/DNS của API và không có lệnh
   nào nhắm vào VPS.
3. **Given** Ollama hoặc proxy không khả dụng, **When** request AI text được gửi, **Then** hệ
   thống xử lý theo chính sách fallback, không treo backend và không làm hỏng các API khác.

---

### User Story 4 - Đánh giá kết quả thử nghiệm sau 1 tuần (Priority: P2)

Nhóm dự án có đủ số liệu để quyết định tiếp tục, điều chỉnh hay dừng Ollama dựa trên độ ổn định,
độ trễ AI, tỷ lệ fallback, tài nguyên EC2, chi phí AWS và khả năng vận hành.

**Why this priority**: Mục tiêu là một thử nghiệm có quyết định đầu ra, không phải chỉ chứng minh
rằng container Ollama có thể khởi chạy.

**Independent Test**: Sau khoảng thử nghiệm, tạo báo cáo từ dữ liệu quan sát đã thu thập và kiểm
tra báo cáo có đủ chỉ số, sự cố và kết luận đạt/không đạt cho từng tiêu chí thành công.

**Acceptance Scenarios**:

1. **Given** thử nghiệm đang chạy, **When** một request AI được xử lý, **Then** vận hành có thể
   xác định dịch vụ, phiên bản, môi trường, provider, model, độ trễ, fallback và kết quả mà không
   lộ nội dung nhạy cảm.
2. **Given** hết thời gian thử nghiệm, **When** nhóm xem báo cáo, **Then** báo cáo so sánh kết quả
   thực tế với toàn bộ tiêu chí thành công và đưa ra khuyến nghị go/no-go có căn cứ.
3. **Given** đã hết ngày thứ 7 nhưng chưa có quyết định thủ công, **When** hệ thống bước sang thời
   gian chờ quyết định, **Then** cấu hình Ollama tiếp tục được giữ nguyên, việc quan sát không bị
   dừng và không có rollback, tắt hay teardown tự động.

### Edge Cases

- Ollama đạt giới hạn CPU, hàng đợi nội bộ hoặc timeout: yêu cầu phải bị giới hạn có kiểm soát, có
  tín hiệu quan sát được và chỉ fallback theo chính sách đã xác định.
- Request bị client hủy hoặc bị từ chối do lỗi validation/4xx: không được fallback sang Gemini và
  không được tiếp tục xử lý ngầm sau khi request đã kết thúc.
- Ollama trả dữ liệu text streaming, structured output hoặc usage khác với OpenAI: luồng tích hợp
  phải phát hiện không tương thích qua kiểm thử code cụ thể, không được mặc định rằng cùng
  endpoint nghĩa là cùng hành vi.
- Ollama không trả usage: lượt gọi vẫn có thể hoàn tất, usage thiếu phải được đánh dấu; không được
  tự suy ra chi phí hoặc tra giá để bù.
- Endpoint Ollama bị gọi thiếu hoặc sai thông tin xác thực: yêu cầu phải bị từ chối và không được
  chuyển thành truy cập công khai.
- Provider `openai` vô tình được cấu hình tới một endpoint trả phí: theo quy tắc phạm vi này, hệ
  thống vẫn không tính tiền nội bộ vì chỉ Gemini được áp dụng bước tính tiền; cấu hình sai phải
  tạo cảnh báo vận hành rõ ràng.
- Ollama thất bại rồi Gemini fallback thành công: usage và chi phí phải gắn đúng provider đã thực
  sự phục vụ kết quả cuối, không tính trùng hai lượt.
- Static IP của box Lightsail thay đổi hoặc security group bị thu hẹp: request từ Lightsail bị
  chặn; vận hành phải phát hiện và cập nhật allowlist trước khi coi thử nghiệm là hợp lệ.
- EC2 hoặc proxy khởi động lại giữa lúc có request: không được trả kết quả giả; lỗi phải phân loại
  đúng để fallback hoạt động và trạng thái model/proxy phải quan sát được.
- Model hoặc version Ollama bị thay đổi ngoài ý muốn: phải phát hiện qua kiểm tra version/digest
  và không âm thầm đổi hành vi của thử nghiệm.
- Model bật thinking ngoài ý muốn (hoặc bị đổi sang model thinking): response chứa reasoning
  token/field, structured output lệch hoặc độ trễ tăng; phải phát hiện bằng contract test và dừng
  trước khi bắt đầu đo.
- Thay đổi code dùng chung được phát hành lên Lightsail nhưng VPS vẫn giữ image cũ: hai môi
  trường phải tiếp tục hoạt động độc lập, không đòi hỏi nâng cấp đồng thời.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Backend MUST tiếp tục chạy trên AWS Lightsail theo đúng topology sáu container hiện
  hữu (HTTP và worker cho Account, Commerce, AI); thử nghiệm MUST NOT tạo hoặc yêu cầu
  EKS/Kubernetes, ALB, ACM, ingress hay thay đổi DNS của API.
- **FR-002**: Thay đổi phía Lightsail MUST chỉ gồm cấu hình môi trường (provider, endpoint, model,
  token, giới hạn) và image AI mới chứa thay đổi code của tính năng, được build/phát hành bởi
  pipeline hiện hữu; Dockerfile, `docker-compose.aws.yml`, template Nginx và pipeline deploy MUST
  NOT bị sửa để phục vụ thử nghiệm.
- **FR-003**: Supabase PostgreSQL, Redis Cloud, CloudAMQP và Cloudinary MUST tiếp tục là các dịch
  vụ dữ liệu dùng chung hiện hữu; thử nghiệm MUST NOT di chuyển hoặc sao chép có thẩm quyền các
  dịch vụ này vào AWS.
- **FR-004**: Deployment VPS độc lập cùng toàn bộ file cấu hình, workflow, secret, DNS, host, dữ
  liệu và runtime đặc thù VPS MUST NOT bị sửa, truy cập để triển khai hoặc trở thành phụ thuộc của
  thử nghiệm.
- **FR-005**: Frontend MUST tiếp tục được phục vụ bởi Vercel và dùng domain API AWS hiện hữu; hợp
  đồng API công khai MUST giữ tương thích ngược và không cần thay đổi phía client.
- **FR-006**: Mọi thay đổi code dùng chung cần thiết cho thử nghiệm MUST giữ nguyên hành vi Gemini
  hiện có và MUST NOT yêu cầu một nhánh code riêng cho Lightsail hay VPS.
- **FR-007**: Ollama MUST chạy trên một EC2 CPU-only riêng, dùng EBS gp3 cho model, pin version
  Ollama và digest model; quản trị MUST qua Systems Manager, không mở SSH.
- **FR-008**: EC2 MUST dùng Elastic IP và security group chỉ cho phép cổng proxy từ static IP của
  box Lightsail; nếu dùng Let's Encrypt HTTP-01, cổng 80 MUST chỉ phục vụ ACME challenge/redirect
  và MUST NOT proxy suy luận. Ollama MUST chỉ bind `127.0.0.1:11434` và MUST NOT nhận kết nối trực
  tiếp từ Internet công cộng.
- **FR-009**: Proxy MUST dùng TLS cho lưu lượng từ Lightsail, xác thực Bearer token lấy từ AWS
  Secrets Manager hoặc biến môi trường được quản lý, từ chối thông tin xác thực thiếu/sai và không
  ghi secret hoặc prompt nhạy cảm vào log.
- **FR-010**: Hạ tầng EC2 MUST được quản lý bằng Terraform tối giản (EC2, Elastic IP, security
  group, EBS, IAM/SSM, Secrets Manager); business code MUST NOT import AWS SDK chỉ để gọi Ollama.
- **FR-011**: Trước khi coi tích hợp hoàn tất, nhóm MUST kiểm tra code thực tế của adapter OpenAI và
  tất cả call site AI áp dụng được; không được kết luận tương thích chỉ vì repository đã có
  provider `openai`.
- **FR-012**: Việc kiểm tra code OpenAI-compatible MUST bao phủ các đường text thường, streaming,
  structured output, usage/token, timeout, lỗi và fallback; kiểm thử hồi quy MUST chứng minh
  vision/embedding vẫn đi Gemini và không vô tình gọi Ollama.
- **FR-013**: Mọi lượt gọi có provider được phân loại là `openai`, bao gồm Ollama
  OpenAI-compatible, MUST có chi phí tiền tệ bằng 0 bất kể route, model hoặc usage được trả về.
- **FR-014**: Trước khi gọi provider `openai`, hệ thống MUST bỏ qua hoàn toàn truy vấn cấu hình
  giá, phép tính chi phí và thao tác giữ chỗ/xác nhận chi phí; MUST NOT dùng một bản ghi giá bằng
  0 hoặc danh sách capability miễn phí để giả lập việc bỏ qua.
- **FR-015**: Lượt gọi `openai` MUST vẫn có thể ghi nhận usage vận hành gồm provider, model, tác
  vụ, trạng thái, độ trễ và token biết được, nhưng bản ghi này MUST NOT kích hoạt logic tiền tệ.
- **FR-016**: Gemini MUST là provider duy nhất đủ điều kiện đi qua logic tiền tệ; lượt Gemini có
  tính phí MUST giữ nguyên quy tắc tra giá, giữ chỗ trước khi gọi và xác nhận/hoàn tác, còn
  capability Gemini không tính phí MUST giữ nguyên hành vi hiện hữu.
- **FR-017**: Khi có fallback giữa Ollama và Gemini, logic usage và chi phí MUST dựa trên từng
  attempt và provider thực sự phục vụ kết quả; MUST NOT tính chi phí Gemini cho kết quả Ollama
  hoặc tính trùng chi phí.
- **FR-018**: Bộ kiểm thử MUST chứng minh bằng quan sát lời gọi dependency rằng các đường đi
  `openai` không gọi kho giá/giữ chỗ, còn các đường đi Gemini có tính phí vẫn gọi đúng dependency
  tương ứng.
- **FR-019**: Hệ thống MUST có giới hạn đồng thời và timeout phù hợp với năng lực CPU để quá tải
  Ollama không làm cạn kiệt tài nguyên của Go AI Service; giới hạn đồng thời mặc định là 2 ở cả
  Go AI Service và Ollama, attempt Ollama tối đa 90 giây, attempt Gemini fallback tối đa 30 giây.
- **FR-020**: Hệ thống MUST cung cấp log và metric đủ để kiểm tra provider/model thắng, attempt,
  lý do fallback, độ trễ, lỗi, usage và CPU/bộ nhớ/disk của EC2; MUST NOT log prompt, token xác
  thực hoặc secret.
- **FR-021**: Sau thử nghiệm, nhóm MUST có báo cáo go/no-go nêu số liệu thực tế, sự cố, chi phí
  AWS, giới hạn quan sát được và các hạng mục còn thiếu trước khi dùng production.
- **FR-022**: Trong tuần thử nghiệm, Ollama MUST là primary cho `chat_text`, `recommendation_text`
  và `free_text`. `summary` và `rewriter` MUST cũng đi Ollama theo route nghiệp vụ hiện hữu:
  route free dùng nhóm cấu hình `free_text`, route paid dùng nhóm `chat_text`; trial đặt **cả hai**
  nhóm primary = `openai`. Gemini MUST tiếp tục là primary cho vision và embedding.
- **FR-023**: Mỗi request text MUST gọi Ollama tối đa một lần rồi được fallback sang Gemini tối đa
  một lần chỉ khi Ollama timeout, quá tải, lỗi kết nối/5xx hoặc trả output không qua validation;
  lỗi validation/4xx từ request và client cancellation MUST NOT kích hoạt fallback.
- **FR-024**: Hết ngày thứ 7 MUST NOT tự động rollback, tắt hoặc teardown EC2/Ollama; cấu hình
  Ollama MUST tiếp tục active và việc quan sát MUST tiếp tục cho đến khi người vận hành ghi nhận
  quyết định thủ công.
- **FR-025**: Thử nghiệm MUST kéo dài một tuần trong môi trường AWS môn học không có request hoặc
  người dùng thật; uptime MUST NOT là tiêu chí quyết định.
- **FR-026**: Model Ollama MUST được pin cả version runtime và digest; tuần thử nghiệm dùng
  `qwen2.5:3b-instruct-q4_K_M` (non-thinking). Contract test MUST chứng minh response trả client
  không chứa reasoning content/token và structured output vẫn hợp lệ.
- **FR-027**: Trước khi lên EC2, nhóm MUST chạy cùng model/digest trên máy dev qua Ollama native,
  hoàn tất contract test của adapter và các kịch bản fallback; latency đo bằng GPU local chỉ là
  tham chiếu chức năng và MUST NOT được dùng thay cho SC-005.
- **FR-028**: Ngân sách tuần thử nghiệm MUST có cảnh báo ở 50%, 80% và 100% của mức 80 USD; mọi
  tài nguyên MUST được tag `Project`, `Feature=032`, `Environment=aws-course`, `Owner`,
  `ExpiresOn` và không tài nguyên nào bị teardown tự động sau ngày thứ 7.

### Key Entities *(include if feature involves data)*

- **Ollama Endpoint**: cấu hình thử nghiệm gồm Elastic IP, hostname proxy, cổng, model, version,
  digest, tham chiếu token và danh sách static IP Lightsail được phép.
- **AI Provider Attempt**: một lần thử gọi provider cụ thể, gồm provider, model, tác vụ, kết quả,
  độ trễ, token biết được, lý do lỗi/fallback và phân loại có áp dụng tiền tệ hay không.
- **Operational Usage Record**: dữ liệu quan sát của lượt dùng AI; tách việc đo provider/model/
  token khỏi việc tính tiền để lượt Ollama vẫn quan sát được với chi phí bằng 0.
- **Trial Observation**: một mẫu metric hoặc sự cố trong tuần thử nghiệm, liên kết với provider,
  model và thời điểm để phục vụ quyết định go/no-go.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% sáu container trên Lightsail vượt smoke test; 100% request kiểm thử text
  **không rơi vào fallback đủ điều kiện** được Ollama phục vụ (mọi fallback đủ điều kiện được đếm
  riêng và báo cáo, không tính vào mẫu "Ollama phục vụ"); diff của tính năng cho thấy 0 thay đổi
  đối với Dockerfile, `docker-compose.aws.yml`, template Nginx AWS, pipeline deploy và 0 tài
  nguyên EKS/Kubernetes.
- **SC-002**: 100% kịch bản `openai`/Ollama trong bộ kiểm thử chạy thành công khi thiếu cấu hình
  giá, ghi chi phí bằng 0 và có đúng 0 lời gọi tới kho giá hoặc thao tác giữ chỗ; 100% kịch bản
  Gemini có tính phí vẫn đi qua luồng giá hiện hữu.
- **SC-003**: 100% request từ IP ngoài allowlist bị chặn, 100% request thiếu/sai xác thực bị từ
  chối, 100% request hợp lệ từ box Lightsail được chấp nhận và 100% phép thử truy cập trực tiếp
  `11434` từ Internet công cộng bị chặn.
- **SC-004**: Nhật ký và diff của tính năng cho thấy 0 lệnh, probe hoặc thao tác tới VPS và 0 thay
  đổi đối với artefact đặc thù VPS; việc kiểm chứng chỉ dựa trên bằng chứng phía AWS/repository,
  không chạy kiểm tra mới trực tiếp lên VPS.
- **SC-005**: Load profile: 50 request cho mỗi capability text (`chat_text`, `free_text`,
  `recommendation_text`), concurrency 2, trong cửa sổ 30 phút; p95 tính trên mẫu này. Trong profile
  đó, ít nhất 95% `chat_text` và `free_text` bắt đầu trả dữ liệu trong 15 giây; ít nhất 95%
  `recommendation_text` bắt đầu trả dữ liệu trong 60 giây và hoàn tất trong 120 giây; ít nhất 95%
  attempt Ollama hoàn tất trong 90 giây. Attempt vượt 90 giây bị hủy và fallback Gemini được thử
  trong ngân sách còn lại; request bị `write_timeout_seconds = 120` cắt được ghi nhận là failure,
  không phải success, và không làm API khác mất khả năng đáp ứng.
- **SC-006**: Vận hành có thể chuyển cấu hình text về Gemini-only trong tối đa 15 phút bằng thay
  đổi biến môi trường và tạo lại container, không mất dữ liệu đã xác nhận và không thao tác lên
  VPS.
- **SC-007**: Sau một tuần, báo cáo go/no-go có đủ 100% chỉ số bắt buộc: kết quả smoke test, kiểm
  soát truy cập/xác thực, lỗi, độ trễ AI, tỷ lệ fallback, hàng đợi, CPU/bộ nhớ/disk EC2 và chi phí
  AWS; uptime không phải tiêu chí quyết định.
- **SC-008**: 100% kịch bản hồi quy hợp đồng API vượt kiểm thử; mọi kịch bản text chọn Ollama làm
  primary, mọi kịch bản vision/embedding vẫn chọn Gemini, và frontend Vercel cùng các dịch vụ dữ
  liệu bên ngoài không phải di chuyển hay đổi hợp đồng.
- **SC-009**: 100% kịch bản timeout, quá tải, lỗi kết nối/5xx và output Ollama không hợp lệ **trước
  khi emit chunk đầu** gọi Gemini đúng một lần; 100% lỗi giữa stream không gọi Gemini, trả lỗi trên
  stream và không ghi cost; 100% kịch bản client cancellation hoặc request 4xx gọi Gemini đúng 0
  lần.
- **SC-010**: Khi chưa có quyết định sau ngày thứ 7, hệ thống có 0 hành động tự động rollback/tắt/
  teardown; quyết định cuối cùng có danh tính người quyết định, thời điểm cùng lựa chọn được audit.
- **SC-011**: Trước khi lên EC2, cùng model/digest chạy local bằng Ollama native và 100% contract
  test adapter cùng kịch bản fallback vượt qua; kết quả latency GPU local không được dùng để thay
  hay hạ SC-005.

## Assumptions

- Backend tiếp tục chạy trên box Lightsail hiện hữu với sáu container và pipeline deploy AWS hiện
  hữu; thử nghiệm không yêu cầu thay đổi kiến trúc deployment.
- Box Lightsail có static IP ổn định trong tuần thử nghiệm và có thể kết nối ra endpoint HTTPS của
  EC2; nếu static IP thay đổi, allowlist của security group phải được cập nhật trước khi đo.
- EC2 và một hostname proxy riêng (bản ghi DNS mới, không phải domain API hiện hữu) có thể được
  tạo trong AWS; việc thêm bản ghi này không được coi là thay đổi DNS của API.
- Gemini tiếp tục tồn tại như provider hiện hữu hoặc fallback trong tuần thử nghiệm, nên logic
  tính tiền Gemini phải được bảo toàn.
- Giá trị provider `openai` là phân loại không tính tiền trong phạm vi sản phẩm này, kể cả khi
  endpoint bị cấu hình khác Ollama; chỉ provider Gemini được phép đi qua bước tính tiền.
- Các dịch vụ Supabase PostgreSQL, Redis Cloud, CloudAMQP và Cloudinary cho phép kết nối an toàn
  từ Lightsail và EC2 trong hạn mức hiện tại.
- Tài khoản AWS, quyền tạo EC2/Elastic IP/security group/Secrets Manager và quyền quản trị DNS
  cần thiết có sẵn trước khi triển khai.
- Môi trường AWS hiện tại chỉ phục vụ môn học, không có request hoặc người dùng thật; không cần
  bảo đảm uptime hoặc bảo vệ phiên người dùng đang hoạt động.
- Mức tải đồng thời 2 yêu cầu AI là tải kiểm thử mặc định để đánh giá CPU; không được hạ tiêu chí
  sau khi bắt đầu đo mà không ghi rõ trong báo cáo.
- Máy dev dùng để test trước là laptop i5-13500HX/20 thread, 15.7 GiB RAM, RTX 4050 Laptop 6 GiB;
  GPU chỉ dùng cho test chức năng, mọi kết luận về latency và timeout lấy từ EC2 `c6i.2xlarge`.
- Một tuần thử nghiệm có ngân sách cảnh báo 80 USD; nếu chi phí vượt mức này, nhóm phải ghi nhận
  trong báo cáo go/no-go chứ không tự động hạ cấu hình hoặc teardown.
- Không có yêu cầu thay đổi giao diện hoặc hợp đồng frontend; nếu quá trình kiểm tra phát hiện bắt
  buộc phải đổi contract, thay đổi đó phải được đặc tả riêng.

### Ranh giới ngoài phạm vi

- Mọi thay đổi, nâng cấp, restart, deploy, kiểm tra secret hoặc chuyển lưu lượng trên VPS.
- EKS, Kubernetes, ALB, ACM, ingress, ExternalDNS hoặc thay đổi DNS của domain API.
- Readiness `/readyz`, worker drain và các thay đổi lifecycle worker chỉ phục vụ Kubernetes.
- Di chuyển database, cache, message broker hoặc image storage sang dịch vụ AWS.
- Chuyển frontend khỏi Vercel.
- Dùng GPU, huấn luyện/fine-tune model hoặc cam kết thay thế Gemini vĩnh viễn.
- Tính tiền cho OpenAI/Ollama hoặc tạo bản ghi giá 0 để lách bước pricing.
- Thay đổi gói cước, quota, API công khai hoặc trải nghiệm người dùng ngoài ảnh hưởng hiệu năng
  được đo.
