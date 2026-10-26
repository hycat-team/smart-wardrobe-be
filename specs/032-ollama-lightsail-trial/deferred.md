# Deferred — Thử nghiệm Ollama trên EC2 và tính tiền Gemini-only

Work discovered while building feature 032 that is **not this feature's job**.
Nothing here blocks closing feature 032; nothing here is a task of feature 032.

## Entries

### D-01 — Terraform module assumes an existing default VPC/subnet

- **Vấn đề**: `deployments/terraform/ollama-ec2/main.tf` deliberately omits `subnet_id` and relies on
  the account's default VPC/default subnet. An AWS account without a default VPC fails at the second
  apply step.
- **Vì sao ngoài phạm vi**: the trial was scoped to a minimal EC2 module with no VPC/route-table
  design; choosing or creating a VPC is environment topology work, not the Ollama trial.
- **Gỡ bằng cách nào**: add `vpc_id`/`subnet_id` variables (or a small network module) to
  `deployments/terraform/ollama-ec2/` once the AWS account topology is decided.
- **Ảnh hưởng tới task đã tick**: không (T019/T020 are code-complete; live apply is still operator-run).

### D-02 — AI attempt metrics live on Lightsail, outside the EC2 CloudWatch module

- **Vấn đề**: the EC2 Terraform module creates CloudWatch log/metric resources for the Ollama host
  and proxy. The AI service runs on Lightsail, so the contract metrics `ai_ttft_ms`,
  `ai_latency_ms`, `ai_fallback_total`, `ai_schema_failure_total`, `ai_thinking_token_total` are
  emitted as structured logs on Lightsail rather than as CloudWatch metrics in this module.
- **Vì sao ngoài phạm vi**: moving Lightsail logs/metrics into CloudWatch is an observability
  platform change; feature 032 must not alter the Lightsail deployment artifacts or add a monitoring
  stack.
- **Gỡ bằng cách nào**: a separate observability feature can ship a log shipper/metric filter for the
  Lightsail AI containers, or the trial report can continue to aggregate the structured logs.
- **Ảnh hưởng tới task đã tick**: T043 creates EC2/Nginx metrics only; T044 emits the AI fields in
  structured logs. The report must state that AI metrics are log-derived, not CloudWatch metrics.

### D-03 — Attempt log cost is read from usage events, not from the attempt result

- **Vấn đề**: `AttemptResult` carries no monetary cost, so `cost_micro_vnd` on a per-attempt log line
  is `0`. Actual cost is confirmed in `ai_usage_events` after the winning provider is known.
- **Vì sao ngoài phạm vi**: adding cost to every attempt result would change the AI service's
  application contract and duplicate the reservation/confirm ledger; the trial's source of truth for
  cost is the usage-event ledger.
- **Gỡ bằng cách nào**: if per-attempt cost is required later, extend `AttemptResult` with a
  post-confirm cost field and log it from the orchestrator after `Confirm`.
- **Ảnh hưởng tới task đã tick**: T044 logs `cost_micro_vnd=0`; the report and runbook already say to
  read actual cost from `ai_usage_events`.

### D-04 — Build/app version is not injected into the AI service

- **Vấn đề**: there is no existing build-version symbol in the repository. The attempt log reads
  `APP_VERSION` and allows it to be empty, so `version` can be blank until the deployment pipeline
  injects a value.
- **Vì sao ngoài phạm vi**: build metadata injection belongs to the image/deployment pipeline, not to
  the Ollama trial feature; modifying `deploy-aws.yml` is explicitly forbidden by FR-002.
- **Gỡ bằng cách nào**: have the existing pipeline set `APP_VERSION` (image tag or commit SHA) on the
  AI containers; the telemetry field will pick it up without code changes.
- **Ảnh hưởng tới task đã tick**: T044 sets the field from `APP_VERSION`; an empty value is a
  deployment gap, not a missing code path.

### D-05 — Vision/embedding attempts are not emitted as per-attempt log lines

- **Vấn đề**: `AttemptHooks` is attached only when `plan.Provider == "openai"` (trial text). Vision
  and embedding run on Gemini without an `ai provider attempt` log line; their usage is still in
  `ai_usage_events`.
- **Vì sao ngoài phạm vi**: the trial only changes the text provider path; adding attempt telemetry
  to vision/embedding touches their call sites and is an observability expansion rather than a trial
  requirement. The runbook points to `ai_usage_events` for those two capabilities.
- **Gỡ bằng cách nào**: extend `AnalyzeImage`/`GenerateEmbeddings` with the same telemetry helper in a
  separate observability change, or relax contract §1 to say per-attempt logs are text-only.
- **Ảnh hưởng tới task đã tick**: T044's per-attempt log covers trial text only; the report must not
  claim vision/embedding attempt logs exist.
