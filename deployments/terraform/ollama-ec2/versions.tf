# Feature 032 - Ollama CPU trial, Terraform skeleton (Phase 1)
#
# Terraform provider constraints for the throw-away EC2 host described in
# specs/032-ollama-lightsail-trial/plan.md section 4.
#
# STATE POLICY: local state only. There is intentionally NO backend block, no S3 and
# no DynamoDB. The state file lives on the operator machine, outside the repository.
# .gitignore blocks .terraform/, *.tfstate* and *.tfvars; never commit them.
#
# The aws provider block lives here, not in main.tf (T019). Do not declare a second
# provider "aws" block in main.tf.

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.60"
    }
  }
}

provider "aws" {
  region = var.aws_region

  # Feature 032 requires these tags on every resource (FR-028). Resources declared in
  # main.tf (T019) inherit them through default_tags.
  default_tags {
    tags = merge(var.tags, {
      Project     = var.project
      Feature     = var.feature
      Environment = var.environment
      Owner       = var.owner
      ExpiresOn   = var.expires_on
      ManagedBy   = "terraform"
    })
  }
}
