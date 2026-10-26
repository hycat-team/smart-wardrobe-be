# Feature 032 - input variables for the Ollama trial EC2.
#
# Contract sources:
#   specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md section 2 (EC2 bootstrap env)
#   specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md section 3 (outputs)
#   specs/032-ollama-lightsail-trial/plan.md section 4 (infrastructure)
#
# Never put a real account id, secret or token in this file or in any committed tfvars.

# ------------------------------------------------------------------------------
# 1. Placement and tags
# ------------------------------------------------------------------------------
variable "aws_region" {
  description = "AWS region for the trial host. Pinned by the feature to ap-southeast-1."
  type        = string
  default     = "ap-southeast-1"

  validation {
    condition     = var.aws_region == "ap-southeast-1"
    error_message = "The trial host must run in ap-southeast-1 (cost and latency were measured there)."
  }
}

variable "project" {
  description = "Value of the Project tag."
  type        = string
  default     = "smart-wardrobe"
}

variable "feature" {
  description = "Value of the Feature tag. Pinned to 032 for this trial."
  type        = string
  default     = "032"
}

variable "environment" {
  description = "Value of the Environment tag."
  type        = string
  default     = "aws-course"
}

variable "owner" {
  description = "Value of the Owner tag. TODO(operator): replace with the real owner identity before apply."
  type        = string
  default     = "TODO-owner"
}

variable "expires_on" {
  description = "Value of the ExpiresOn tag (trial day 7). Reminder only: nothing is destroyed automatically."
  type        = string
  default     = "2026-11-02"
}

variable "tags" {
  description = "Optional extra tags merged on top of the required feature tags."
  type        = map(string)
  default     = {}
}

# ------------------------------------------------------------------------------
# 2. Instance, AMI and storage
# ------------------------------------------------------------------------------
variable "instance_type" {
  description = "EC2 instance type. c6i.2xlarge = 8 vCPU / 16 GiB x86, CPU-only inference."
  type        = string
  default     = "c6i.2xlarge"

  validation {
    condition     = var.instance_type == "c6i.2xlarge"
    error_message = "The trial is pinned to c6i.2xlarge so SC-005 numbers stay comparable."
  }
}

variable "instance_ami_id" {
  description = "Optional explicit AMI id. Empty means resolve the latest Ubuntu 24.04 amd64 id from SSM."
  type        = string
  default     = ""
}

variable "instance_ami_ssm_parameter" {
  description = "SSM public parameter used to resolve the AMI when instance_ami_id is empty."
  type        = string
  default     = "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id"
}

variable "root_volume_size_gb" {
  description = "Root EBS gp3 size in GiB (OS, packages, logs)."
  type        = number
  default     = 16

  validation {
    condition     = var.root_volume_size_gb >= 16
    error_message = "Root volume must be at least 16 GiB for nginx, certbot and logs."
  }
}

variable "data_volume_size_gb" {
  description = "Dedicated EBS gp3 size in GiB mounted for Ollama models. Trial pins 50 GiB."
  type        = number
  default     = 50

  validation {
    condition     = var.data_volume_size_gb >= 50
    error_message = "The trial pins a 50 GiB gp3 data volume for the Ollama model store."
  }
}

variable "data_volume_device_name" {
  description = "Device name of the data volume, mounted at /var/lib/ollama by bootstrap."
  type        = string
  default     = "/dev/sdf"
}

# ------------------------------------------------------------------------------
# 3. Network: Elastic IP, security group allowlist
# ------------------------------------------------------------------------------
variable "proxy_allowed_cidr" {
  description = "Static IP of the Lightsail box as /32. TODO(operator): fill from baseline. Never 0.0.0.0/0."
  type        = string
  default     = ""

  validation {
    condition     = can(cidrhost(var.proxy_allowed_cidr, 0)) && endswith(var.proxy_allowed_cidr, "/32")
    error_message = "proxy_allowed_cidr must be a single IPv4 address in /32 form, e.g. 203.0.113.10/32."
  }

  validation {
    condition     = var.proxy_allowed_cidr != "0.0.0.0/0"
    error_message = "Port 443 must never be open to the world; only the Lightsail static IP may connect."
  }
}

variable "proxy_https_port" {
  description = "Public TLS port for the proxy (inference traffic)."
  type        = number
  default     = 443
}

variable "proxy_acme_port" {
  description = "Public port 80, ACME challenge and redirect only. It must never proxy inference."
  type        = number
  default     = 80
}

# ------------------------------------------------------------------------------
# 4. Proxy, TLS and Bearer auth
# ------------------------------------------------------------------------------
variable "proxy_hostname" {
  description = "Dedicated A record for the Ollama proxy (NOT the API domain). TODO(operator): fill before apply."
  type        = string
  default     = ""
}

variable "proxy_acme_email" {
  description = "Contact email for Let's Encrypt HTTP-01 registration."
  type        = string
  default     = ""
}

variable "proxy_bearer_secret_name" {
  description = "Secrets Manager secret name holding the proxy Bearer token. The instance profile may read only this secret."
  type        = string
  default     = "ollama-trial/proxy-bearer"
}

variable "proxy_bearer_token" {
  description = <<-EOT
    Optional Bearer token value to store in Secrets Manager at apply time.
    Prefer passing it via TF_VAR_proxy_bearer_token from the operator secret store.
    If T019 generates it instead, add the hashicorp/random provider there and use a
    random_password resource. Never commit the token, never write it to a tracked
    tfvars file, never print it in output.
  EOT
  type        = string
  default     = null
  sensitive   = true
  nullable    = true
}

# TLS certificate paths are derived by bootstrap.sh from proxy_hostname
# (/etc/letsencrypt/live/<hostname>/{fullchain.pem,privkey.pem}); they are not inputs.

# ------------------------------------------------------------------------------
# 5. Ollama runtime (pinned version and model digest)
# ------------------------------------------------------------------------------
variable "ollama_version" {
  description = "Pinned Ollama runtime version. TODO(operator): fill from `ollama --version` in Phase A."
  type        = string
  default     = ""
}

variable "ollama_model" {
  description = "Pinned model tag for the trial."
  type        = string
  default     = "qwen3:4b"
}

variable "model_digest" {
  description = "Pinned model digest (sha256:...). TODO(operator): fill from `ollama list` in Phase A."
  type        = string
  default     = ""

  validation {
    condition     = var.model_digest == "" || startswith(var.model_digest, "sha256:")
    error_message = "model_digest must be empty or start with sha256:."
  }
}

variable "ollama_host" {
  description = "Bind address. Loopback only; the proxy is the sole caller."
  type        = string
  default     = "127.0.0.1:11434"

  validation {
    condition     = var.ollama_host == "127.0.0.1:11434"
    error_message = "Ollama must bind loopback 127.0.0.1:11434; never expose the inference port."
  }
}

variable "ollama_num_parallel" {
  description = "Ollama parallel requests. Must match AI_TEXT_MAX_CONCURRENCY (2)."
  type        = number
  default     = 2

  validation {
    condition     = var.ollama_num_parallel >= 1
    error_message = "ollama_num_parallel must be at least 1."
  }
}

variable "ollama_max_queue" {
  description = "Finite Ollama queue depth; an unbounded queue hides overload."
  type        = number
  default     = 8

  validation {
    condition     = var.ollama_max_queue >= 1
    error_message = "ollama_max_queue must be finite and at least 1."
  }
}

variable "ollama_keep_alive" {
  description = "How long the model stays loaded between requests during the trial week."
  type        = string
  default     = "30m"
}

variable "ollama_num_gpu" {
  description = "GPU count forced for the CPU-only trial. 0 disables GPU layers even if a driver appears."
  type        = number
  default     = 0
}

# ------------------------------------------------------------------------------
# 6. Cost guard and observability
# ------------------------------------------------------------------------------
variable "budget_limit_usd" {
  description = "Monthly AWS Budget limit for the trial in USD (FR-028)."
  type        = number
  default     = 80
}

variable "budget_alert_thresholds" {
  description = "Budget alert thresholds in percent of budget_limit_usd."
  type        = list(number)
  default     = [50, 80, 100]

  validation {
    condition     = contains(var.budget_alert_thresholds, 50) && contains(var.budget_alert_thresholds, 80) && contains(var.budget_alert_thresholds, 100)
    error_message = "Budget alerts must include 50, 80 and 100 percent of the 80 USD trial budget."
  }
}

variable "cloudwatch_log_retention_days" {
  description = "Log retention for the trial. Contract: 14 days. TODO(T043): apply in cloud-init/agent config."
  type        = number
  default     = 14
}

# ------------------------------------------------------------------------------
# 7. Resources owned by main.tf (T019) - listed here so the Phase 1 skeleton is
#    traceable; they are not configurable variables.
# ------------------------------------------------------------------------------
# - aws_instance.ollama            : c6i.2xlarge, no key pair, IMDSv2 required, EBS gp3
#                                    root + 50 GiB data volume, instance profile.
# - aws_eip.ollama                 : Elastic IP, associated with the instance; output
#                                    elastic_ip feeds the manual A record step.
# - aws_security_group.ollama      : 443 from proxy_allowed_cidr/32, 80 only for ACME,
#                                    11434 loopback only, no SSH.
# - aws_iam_role / profile         : AmazonSSMManagedInstanceCore + read-only access to
#                                    the proxy Bearer secret; no other permission.
# - aws_secretsmanager_secret      : proxy Bearer token (proxy_bearer_secret_name).
# - aws_budgets_budget             : budget_limit_usd with 50/80/100% alerts.
# - aws_cloudwatch_log_group       : 14-day retention (T043 completes agent/metrics).
# Nothing here may create EKS/Kubernetes, ALB, ACM, Route53 or an auto-destroy schedule.
