# Specification Quality Checklist: Thử nghiệm Ollama trên EC2 và tính tiền Gemini-only

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-10-26
**Updated**: 2026-10-26 (bỏ phạm vi EKS, backend giữ nguyên Lightsail; chốt qwen2.5:3b-instruct-q4_K_M, c6i.2xlarge,
timeout 90/30, SC-005 tách ngưỡng, concurrency 2, TLS A record tay, Terraform local state)
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] Implementation details limited to user-mandated infrastructure choices; code-level design lives in plan/contracts
- [x] Focused on operational value, safety boundaries, and business rules
- [x] Written for technical and non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria describe observable outcomes
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Lightsail, EC2, AWS và VPS scope boundaries are explicit
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover Ollama serving, AI cost routing, Lightsail operation, and evaluation
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] OpenAI/Ollama cost bypass and Gemini-only pricing are independently testable
- [x] Model qwen2.5:3b-instruct-q4_K_M, cách pin version/digest và yêu cầu tắt thinking được đặc tả
- [x] Local-first validation trước EC2 có tiêu chí và không dùng latency GPU thay SC-005
- [x] Timeout 90/30, concurrency 2 và ngân sách 80 USD có trong yêu cầu/tiêu chí

## Scope Integrity

- [x] Không còn yêu cầu EKS/Kubernetes, ALB, ACM, ingress, DNS cutover của API
- [x] Không còn yêu cầu readiness `/readyz` hoặc worker drain
- [x] Không còn yêu cầu active/standby worker giữa hai môi trường
- [x] Backend Lightsail, Dockerfile, Compose, Nginx AWS và pipeline deploy được yêu cầu giữ nguyên
- [x] Ollama chỉ truy cập được từ static IP Lightsail, có TLS và Bearer; `11434` không public
- [x] VPS nằm ngoài phạm vi và được kiểm chứng bằng diff/repository evidence

## Notes

- EC2, Elastic IP, security group, Secrets Manager và Ollama được nêu vì đây là ràng buộc nền tảng
  do người dùng chỉ định; đặc tả không khóa cách tổ chức Terraform hay cấu trúc code triển khai.
- Quyết định đã chốt: `qwen2.5:3b-instruct-q4_K_M` pin version/digest; EC2 `c6i.2xlarge`; timeout Ollama 90s/Gemini
  30s; SC-005 có load profile 50 request/capability, concurrency 2, 30 phút; concurrency 2 ở Go +
  Ollama; giữ `AI_UNBILLED_CAPABILITIES=free_text,embedding`; TLS Let's Encrypt với A record tạo tay;
  Terraform local state; ngân sách cảnh báo 80 USD. `openai` vẫn tiêu thụ daily attempt cap hiện hữu
  và dùng chung usage event khi fallback Gemini (không tạo event thứ hai).
- `openai`/Ollama phải bỏ qua dependency pricing thật sự; không chấp nhận bảng giá 0 hoặc danh sách
  capability miễn phí. Gemini giữ nguyên luồng pricing/reservation hiện hữu.
- “Không đụng VPS” được kiểm chứng bằng diff và log phía AWS/repository; không chạy probe mới lên
  VPS.
- Không còn mục cần làm rõ trước khi chạy `speckit-clarify`, `speckit-plan` hoặc `speckit-tasks`.
