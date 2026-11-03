# Feature 032 - Terraform outputs (hand-off contract).
#
# Source of truth: specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md section 3.
# These seven outputs are what the runbook and the Lightsail .env.aws consume.
#
# No secret is exposed here: secret_arn is an identifier only. The Bearer value is
# never printed, never in state output, never in CI logs.
#
# `elastic_ip` is the only output needed by the first apply step
# (`terraform apply -target=aws_eip.ollama`); everything else appears after the
# full apply.

output "instance_id" {
  description = "EC2 instance id used by SSM Session Manager."
  value       = aws_instance.ollama.id
}

output "elastic_ip" {
  description = "Elastic IP to point the manual A record at (never the API domain)."
  value       = aws_eip.ollama.public_ip
}

output "proxy_hostname" {
  description = "Proxy hostname that the Lightsail *_TEXT_PRIMARY_ENDPOINT uses."
  value       = var.proxy_hostname
}

output "secret_arn" {
  description = "Secrets Manager ARN holding the proxy Bearer token (verify read access)."
  value       = aws_secretsmanager_secret.proxy_bearer.arn
}

output "ollama_version" {
  description = "Pinned Ollama runtime version, to compare against the host."
  value       = var.ollama_version
}

output "model_digest" {
  description = "Pinned model digest, to compare local and EC2."
  value       = var.model_digest
}

output "security_group_id" {
  description = "Security group id, to verify the Lightsail /32 allowlist."
  value       = aws_security_group.ollama.id
}
