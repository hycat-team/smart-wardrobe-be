# Feature 032 - Ollama CPU trial host (EC2 + Elastic IP + proxy secret).
#
# Contract sources:
#   specs/032-ollama-lightsail-trial/plan.md section 4
#   specs/032-ollama-lightsail-trial/research.md R1, R2, R7, R8
#   specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md sections 2 and 3
#
# Scope guardrails:
#   - Local state only: no backend block here (versions.tf), no S3, no DynamoDB.
#   - No EKS/Kubernetes, ALB, ACM, Route53 or Vercel provider.
#   - No SSH ingress and no public 11434: the only inference path is 443 from the
#     Lightsail static IP, through the TLS/Bearer proxy on the instance.
#   - No auto teardown. ExpiresOn is a reminder tag; nothing is destroyed on day 7.
#
# Two-step apply (operator):
#   1) terraform apply -target=aws_eip.ollama     -> create the Elastic IP only
#   2) create the A record for var.proxy_hostname -> point it at output elastic_ip
#   3) terraform apply                            -> everything else
# Terraform never touches the API domain or any Route53 zone.

# ------------------------------------------------------------------------------
# 0. Derived inputs
# ------------------------------------------------------------------------------
data "aws_ssm_parameter" "ubuntu_ami" {
  count = var.instance_ami_id == "" ? 1 : 0
  name  = var.instance_ami_ssm_parameter
}

locals {
  ami_id = var.instance_ami_id != "" ? var.instance_ami_id : data.aws_ssm_parameter.ubuntu_ami[0].value

  # Rendered into cloud-init: cloud-init has no access to this repository, so the
  # bootstrap script and the proxy template travel with the user data. The file()
  # values are plain strings, so the shell `$VAR` and nginx `$http_*` references
  # inside them are NOT re-interpolated by Terraform.
  bootstrap_script = file("${path.module}/../../ollama/bootstrap.sh")
  nginx_template   = file("${path.module}/../../ollama/nginx-ollama.conf.template")

  cloudwatch_log_group_name = aws_cloudwatch_log_group.ollama.name

  # One budget notification per (email, threshold) pair. Terraform cannot nest a
  # dynamic block inside another dynamic block, so the product is built here.
  budget_notifications = flatten([
    for email in var.budget_alert_emails : [
      for threshold in var.budget_alert_thresholds : {
        email     = email
        threshold = threshold
      }
    ]
  ])
}

# ------------------------------------------------------------------------------
# 1. Network: security group + Elastic IP
# ------------------------------------------------------------------------------
resource "aws_security_group" "ollama" {
  name        = "ollama-trial-proxy"
  description = "Feature 032 Ollama trial: 443 from the Lightsail static IP only, 80 for ACME only"

  # The only inference ingress. A /32 keeps the proxy unreachable from anywhere
  # except the Lightsail box.
  ingress {
    description = "TLS proxy from the Lightsail static IP only"
    from_port   = var.proxy_https_port
    to_port     = var.proxy_https_port
    protocol    = "tcp"
    cidr_blocks = [var.proxy_allowed_cidr]
  }

  # Port 80 exists for the Let's Encrypt HTTP-01 challenge and the redirect to
  # 443. The nginx template never proxies inference on this port.
  ingress {
    description      = "ACME HTTP-01 challenge and redirect only"
    from_port        = var.proxy_acme_port
    to_port          = var.proxy_acme_port
    protocol         = "tcp"
    cidr_blocks      = ["0.0.0.0/0"]
    ipv6_cidr_blocks = ["::/0"]
  }

  # No ingress rule for 22 (administration is SSM only) and none for 11434
  # (Ollama binds 127.0.0.1). Do not add either. The explicit deny is implicit:
  # an AWS security group only allows what is listed above.

  egress {
    description = "Outbound for package install, Ollama model pull, CloudWatch and SMTP-free operation"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "ollama-trial-proxy"
  }

  lifecycle {
    precondition {
      condition     = var.proxy_allowed_cidr != ""
      error_message = "proxy_allowed_cidr must be the Lightsail static IP in /32 form before apply (baseline section 1 of the runbook)."
    }
  }
}

# Step 1 of the two-step apply. The EIP has no dependency on the instance, so
# `terraform apply -target=aws_eip.ollama` works before the host exists.
resource "aws_eip" "ollama" {
  domain = "vpc"

  tags = {
    Name = "ollama-trial-eip"
  }
}

resource "aws_eip_association" "ollama" {
  instance_id   = aws_instance.ollama.id
  allocation_id = aws_eip.ollama.id
}

# ------------------------------------------------------------------------------
# 2. IAM: SSM administration, one readable secret, CloudWatch agent
# ------------------------------------------------------------------------------
data "aws_iam_policy_document" "ec2_assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ollama" {
  name               = "ollama-trial-instance"
  description        = "Feature 032 trial host: SSM, the proxy Bearer secret and CloudWatch agent only"
  assume_role_policy = data.aws_iam_policy_document.ec2_assume_role.json
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.ollama.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy_attachment" "cloudwatch_agent" {
  role       = aws_iam_role.ollama.name
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
}

# Least privilege: the instance may read exactly this one secret.
data "aws_iam_policy_document" "proxy_secret_read" {
  statement {
    sid       = "ReadProxyBearerOnly"
    effect    = "Allow"
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [aws_secretsmanager_secret.proxy_bearer.arn]
  }
}

resource "aws_iam_role_policy" "proxy_secret_read" {
  name   = "ollama-trial-read-proxy-bearer"
  role   = aws_iam_role.ollama.id
  policy = data.aws_iam_policy_document.proxy_secret_read.json
}

resource "aws_iam_instance_profile" "ollama" {
  name = "ollama-trial-instance"
  role = aws_iam_role.ollama.name
}

# ------------------------------------------------------------------------------
# 3. Proxy Bearer token (Secrets Manager)
# ------------------------------------------------------------------------------
# The token value is an input, never generated here: a generated value would have
# to be printed or stored in state output for the operator to copy into the
# Lightsail .env.aws, and outputs must not carry secrets (env contract section 4).
resource "aws_secretsmanager_secret" "proxy_bearer" {
  name                    = var.proxy_bearer_secret_name
  description             = "Feature 032 Ollama trial proxy Bearer token (compare with <CAP>_TEXT_PRIMARY_API_KEY on Lightsail)"
  recovery_window_in_days = 0

  tags = {
    Name = "ollama-trial-proxy-bearer"
  }

  lifecycle {
    precondition {
      condition     = var.proxy_bearer_token != null && var.proxy_bearer_token != ""
      error_message = "Pass the proxy Bearer token as TF_VAR_proxy_bearer_token (long random value). It must match <CAP>_TEXT_PRIMARY_API_KEY on Lightsail and must never be committed."
    }
  }
}

resource "aws_secretsmanager_secret_version" "proxy_bearer" {
  secret_id     = aws_secretsmanager_secret.proxy_bearer.id
  secret_string = var.proxy_bearer_token
}

# ------------------------------------------------------------------------------
# 4. Instance and model volume
# ------------------------------------------------------------------------------
resource "aws_instance" "ollama" {
  ami                    = local.ami_id
  instance_type          = var.instance_type
  iam_instance_profile   = aws_iam_instance_profile.ollama.name
  vpc_security_group_ids = [aws_security_group.ollama.id]

  # The instance gets an ephemeral public IP first so cloud-init can install
  # packages and pull the model before the Elastic IP association lands; the EIP
  # then replaces it and is the address the A record points at.
  associate_public_ip_address = true

  # No subnet_id on purpose: the trial host lives in the account's default VPC and
  # its default subnet, so the module stays a single small file. An account with
  # no default VPC fails at apply; that is an environment prerequisite, not a
  # module option (runbook section 8.2).
  #
  # No key_name: SSH is closed and administration goes through SSM Session
  # Manager, which is why the instance profile above exists.
  #
  # IMDSv2 is required so a prompt-injection or SSRF bug in a future dependency
  # cannot read the instance credentials over the metadata endpoint.
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
    instance_metadata_tags      = "enabled"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = var.root_volume_size_gb
    encrypted   = true
  }

  user_data = templatefile("${path.module}/templates/cloud-init.yaml.tftpl", {
    aws_region                = var.aws_region
    ollama_host               = var.ollama_host
    ollama_model              = var.ollama_model
    model_digest              = var.model_digest
    ollama_version            = var.ollama_version
    ollama_num_parallel       = var.ollama_num_parallel
    ollama_max_queue          = var.ollama_max_queue
    ollama_keep_alive         = var.ollama_keep_alive
    ollama_num_gpu            = var.ollama_num_gpu
    proxy_https_port          = var.proxy_https_port
    proxy_acme_port           = var.proxy_acme_port
    proxy_allowed_cidr        = var.proxy_allowed_cidr
    proxy_hostname            = var.proxy_hostname
    proxy_acme_email          = var.proxy_acme_email
    proxy_bearer_secret_arn   = aws_secretsmanager_secret.proxy_bearer.arn
    data_volume_device_name   = var.data_volume_device_name
    cloudwatch_log_retention_days = var.cloudwatch_log_retention_days
    cloudwatch_log_group_name = local.cloudwatch_log_group_name
    bootstrap_script          = local.bootstrap_script
    nginx_template            = local.nginx_template
  })

  tags = {
    Name = "ollama-trial-c6i-2xlarge"
  }

  lifecycle {
    # Changing the user data of a running trial host would replace the instance
    # and invalidate the certificate/DNS step. Re-run bootstrap.sh over SSM
    # instead, then destroy/recreate deliberately if a rebuild is really wanted.
    ignore_changes = [user_data]

    precondition {
      condition     = var.ollama_version != ""
      error_message = "ollama_version must be pinned to the version measured in Phase A (runbook section 1)."
    }

    precondition {
      condition     = var.model_digest != ""
      error_message = "model_digest must be pinned to the sha256 digest recorded in Phase A so local and EC2 behaviour match."
    }

    precondition {
      condition     = var.proxy_hostname != ""
      error_message = "proxy_hostname must be the dedicated Ollama A record (never the API domain)."
    }

    precondition {
      condition     = var.proxy_acme_email != ""
      error_message = "proxy_acme_email is required by Let's Encrypt registration."
    }
  }
}

resource "aws_ebs_volume" "ollama_data" {
  availability_zone = aws_instance.ollama.availability_zone
  size              = var.data_volume_size_gb
  type              = "gp3"
  encrypted         = true

  tags = {
    Name = "ollama-trial-model-store"
  }
}

resource "aws_volume_attachment" "ollama_data" {
  device_name = var.data_volume_device_name
  volume_id   = aws_ebs_volume.ollama_data.id
  instance_id = aws_instance.ollama.id
}

# ------------------------------------------------------------------------------
# 5. Observability and cost guard
# ------------------------------------------------------------------------------
# T043 adds metric filters/dashboard on top of this group; the group exists now so
# the agent configuration rendered by cloud-init has somewhere to write.
resource "aws_cloudwatch_log_group" "ollama" {
  name              = "/ollama-trial/ec2"
  retention_in_days = var.cloudwatch_log_retention_days

  tags = {
    Name = "ollama-trial-logs"
  }
}

resource "aws_budgets_budget" "trial" {
  name         = "ollama-trial-80usd"
  budget_type  = "COST"
  limit_amount = tostring(var.budget_limit_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  # AWS Budgets is account-scoped; the provider's regional endpoint resolves the
  # global service. The limit is the monthly trial ceiling, alerts at 50/80/100%.
  dynamic "notification" {
    for_each = local.budget_notifications

    content {
      comparison_operator        = "GREATER_THAN"
      threshold                  = notification.value.threshold
      threshold_type             = "PERCENTAGE"
      notification_type          = "ACTUAL"
      subscriber_email_addresses = [notification.value.email]
    }
  }

  lifecycle {
    precondition {
      condition     = length(var.budget_alert_emails) > 0
      error_message = "Set budget_alert_emails to at least one address so the 50/80/100% alerts actually reach someone."
    }
  }
}
