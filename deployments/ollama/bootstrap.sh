#!/usr/bin/env bash
# Feature 032 - idempotent bootstrap for the Ollama CPU trial host.
#
# Contract: specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md section 2
# Runbook:  docs/operations/ollama-trial-runbook.md sections 3 and 4
#
# This is the Phase 1 skeleton (task T007); tasks T021/T022 finish the proxy config and
# the bootstrap logic. Every TODO marker below names the task that owns the real work.
#
# Rules:
#   - No secret is ever written to this file, to a log line, or to the repository.
#   - The Bearer token is read from Secrets Manager at boot and written to
#     /etc/ollama-trial/proxy-bearer with 0600 permissions.
#   - The script must be safe to re-run (idempotent): no duplicate fstab entries, no
#     duplicate nginx servers, no silent model re-pull.
#
# Usage: sudo /opt/ollama-trial/bootstrap.sh
# Requires: trial.env rendered by cloud-init at /etc/ollama-trial/trial.env

set -euo pipefail

readonly TRIAL_DIR="/etc/ollama-trial"
readonly ENV_FILE="${TRIAL_DIR}/trial.env"
readonly TEMPLATE_DIR="/opt/ollama-trial"
readonly NGINX_CONF="/etc/nginx/conf.d/ollama.conf"
readonly BEARER_FILE="${TRIAL_DIR}/proxy-bearer"
readonly OLLAMA_DATA_DIR="/var/lib/ollama"
readonly ACME_WEBROOT="/var/www/certbot"
readonly LOG_PREFIX="[032-bootstrap]"

log() {
  printf '%s %s\n' "${LOG_PREFIX}" "$*"
}

fail() {
  printf '%s ERROR: %s\n' "${LOG_PREFIX}" "$*" >&2
  exit 1
}

require_env() {
  local name="$1"
  local value="${!name:-}"
  if [[ -z "${value}" ]]; then
    fail "required variable ${name} is empty; check ${ENV_FILE} / Terraform inputs"
  fi
}

# TODO(T022): keep this list in sync with contracts/env-ai-trial.md section 2.
load_env() {
  [[ -r "${ENV_FILE}" ]] || fail "missing ${ENV_FILE}; cloud-init did not render the trial env"
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  require_env OLLAMA_HOST
  require_env OLLAMA_MODEL
  require_env OLLAMA_MODEL_DIGEST
  require_env OLLAMA_VERSION
  require_env OLLAMA_NUM_PARALLEL
  require_env OLLAMA_MAX_QUEUE
  require_env OLLAMA_KEEP_ALIVE
  require_env OLLAMA_NUM_GPU
  require_env PROXY_HTTPS_PORT
  require_env PROXY_ACME_PORT
  require_env PROXY_ALLOWED_CIDR
  require_env PROXY_HOSTNAME
  require_env PROXY_ACME_EMAIL
  require_env PROXY_BEARER_SECRET_ARN
}

# Mount the dedicated gp3 volume on /var/lib/ollama without losing data on re-run.
# TODO(T022): implement with blkid/UUID and a managed /etc/fstab entry; this skeleton
# only validates the prerequisites so a misconfigured host fails loudly.
prepare_data_volume() {
  log "checking data volume ${TRIAL_DATA_DEVICE:-/dev/sdf} for ${OLLAMA_DATA_DIR}"
  if [[ ! -b "${TRIAL_DATA_DEVICE:-/dev/sdf}" ]]; then
    fail "data device ${TRIAL_DATA_DEVICE:-/dev/sdf} not present; check the Terraform EBS attachment"
  fi
  # TODO(T022): mkfs.ext4 only when the volume has no filesystem, mount by UUID,
  # chown ollama:ollama, then write the fstab line exactly once.
  install -d -m 0755 "${OLLAMA_DATA_DIR}"
}

install_ollama() {
  if command -v ollama >/dev/null 2>&1; then
    log "ollama already installed: $(ollama --version 2>/dev/null | head -n1)"
  else
    log "installing pinned Ollama ${OLLAMA_VERSION}"
    # TODO(T022): pin the installer URL and verify a checksum/release signature; do not
    # follow "latest". OLLAMA_VERSION must equal the version recorded in Phase A.
    curl -fsSL https://ollama.com/install.sh | OLLAMA_VERSION="${OLLAMA_VERSION}" sh
  fi

  install -d -m 0755 /etc/systemd/system/ollama.service.d
  cat >/etc/systemd/system/ollama.service.d/10-trial.conf <<EOF
[Service]
Environment="OLLAMA_HOST=${OLLAMA_HOST}"
Environment="OLLAMA_NUM_PARALLEL=${OLLAMA_NUM_PARALLEL}"
Environment="OLLAMA_MAX_QUEUE=${OLLAMA_MAX_QUEUE}"
Environment="OLLAMA_KEEP_ALIVE=${OLLAMA_KEEP_ALIVE}"
Environment="OLLAMA_NUM_GPU=${OLLAMA_NUM_GPU}"
Environment="OLLAMA_MODELS=${OLLAMA_DATA_DIR}/models"
EOF

  systemctl daemon-reload
  systemctl enable ollama
  systemctl restart ollama
  log "waiting for Ollama on ${OLLAMA_HOST}"
  for _ in $(seq 1 30); do
    if curl -fsS "http://${OLLAMA_HOST}/api/version" >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
  curl -fsS "http://${OLLAMA_HOST}/api/version" >/dev/null || fail "Ollama did not become ready"
}

pull_model() {
  log "pulling ${OLLAMA_MODEL} (pinned digest ${OLLAMA_MODEL_DIGEST})"
  ollama pull "${OLLAMA_MODEL}"

  # TODO(T022): confirm the exact digest command against the pinned Ollama version.
  # `ollama list` prints the digest prefix in the ID column; compare it case-insensitively
  # after stripping the "sha256:" prefix. A mismatch must abort the bootstrap.
  local expected="${OLLAMA_MODEL_DIGEST#sha256:}"
  local actual
  actual="$(ollama list | awk -v model="${OLLAMA_MODEL}" '$1 == model { print $2; exit }')"
  [[ -n "${actual}" ]] || fail "model ${OLLAMA_MODEL} not found after pull"
  case "${expected}" in
    "${actual}"*)
      log "digest prefix matches: ${actual}"
      ;;
    *)
      fail "digest mismatch: expected ${OLLAMA_MODEL_DIGEST}, got ${actual}; stopping before measurement"
      ;;
  esac
}

fetch_bearer_token() {
  log "reading proxy Bearer token from Secrets Manager"
  command -v aws >/dev/null 2>&1 || fail "aws CLI is missing; the instance profile path cannot read the secret without it"
  # TODO(T022): add retry/backoff and a clear error when the instance profile lacks
  # secretsmanager:GetSecretValue on this ARN.
  local token
  token="$(aws secretsmanager get-secret-value \
    --region "${AWS_REGION:-ap-southeast-1}" \
    --secret-id "${PROXY_BEARER_SECRET_ARN}" \
    --query SecretString --output text)"
  [[ -n "${token}" && "${token}" != "None" ]] || fail "empty Bearer token from Secrets Manager"

  # Never log the token. Write it to a root-only file consumed by the nginx renderer.
  umask 077
  printf '%s' "${token}" >"${BEARER_FILE}"
  chmod 0600 "${BEARER_FILE}"
}

render_nginx_config() {
  local template="${TEMPLATE_DIR}/nginx-ollama.conf.template"
  [[ -r "${template}" ]] || fail "missing ${template}; main.tf must upload it (TODO T019)"

  # TODO(T021/T022): keep the placeholder names in sync with the template. This sed
  # renderer is a skeleton; replace it with a renderer that cannot break on special
  # characters in the token, and never echo the rendered file to stdout.
  local token tls_cert tls_key
  token="$(cat "${BEARER_FILE}")"
  tls_cert="/etc/letsencrypt/live/${PROXY_HOSTNAME}/fullchain.pem"
  tls_key="/etc/letsencrypt/live/${PROXY_HOSTNAME}/privkey.pem"

  local escaped_token
  escaped_token="$(printf '%s' "${token}" | sed -e 's/[\/&|]/\\&/g')"

  sed \
    -e "s|__PROXY_HOSTNAME__|${PROXY_HOSTNAME}|g" \
    -e "s|__PROXY_ALLOWED_CIDR__|${PROXY_ALLOWED_CIDR}|g" \
    -e "s|__PROXY_HTTPS_PORT__|${PROXY_HTTPS_PORT}|g" \
    -e "s|__PROXY_ACME_PORT__|${PROXY_ACME_PORT}|g" \
    -e "s|__PROXY_TLS_CERT_PATH__|${tls_cert}|g" \
    -e "s|__PROXY_TLS_KEY_PATH__|${tls_key}|g" \
    -e "s|__PROXY_BEARER_TOKEN__|${escaped_token}|g" \
    "${template}" >"${NGINX_CONF}"
  chmod 0640 "${NGINX_CONF}"
  nginx -t
}

issue_certificate() {
  install -d -m 0755 "${ACME_WEBROOT}"
  # TODO(T021/T022): bootstrap the ACME server block first (plain HTTP challenge), then
  # run certbot, then reload with the TLS config. A single "certbot --nginx" call is not
  # enough when the TLS server block already references a certificate that does not exist.
  if [[ ! -d "/etc/letsencrypt/live/${PROXY_HOSTNAME}" ]]; then
    log "certificate not present yet; TODO(T021) must complete the ACME challenge flow"
  fi
}

prewarm_and_assert_thinking_off() {
  log "pre-warming ${OLLAMA_MODEL} and asserting thinking is disabled"
  # TODO(T022): send reasoning_effort=none, then fail the bootstrap if the response has
  # reasoning_content/thinking, if the JSON body is not valid, or if thinking_tokens > 0.
  curl -fsS "http://${OLLAMA_HOST}/v1/chat/completions" \
    -H 'Content-Type: application/json' \
    -d "{\"model\":\"${OLLAMA_MODEL}\",\"messages\":[{\"role\":\"user\",\"content\":\"ping\"}],\"stream\":false,\"max_tokens\":16,\"reasoning_effort\":\"none\"}" \
    >/dev/null || fail "pre-warm request failed"
}

verify_isolation() {
  log "verifying Ollama listens on loopback only"
  if ss -lntp | grep -E ':11434\b' | grep -v '127\.0\.0\.1:11434'; then
    fail "Ollama is listening beyond 127.0.0.1:11434; refusing to mark the host trial-ready"
  fi
  log "allowlist in nginx: ${PROXY_ALLOWED_CIDR} (security group is the outer guard)"
}

main() {
  load_env
  prepare_data_volume
  install_ollama
  pull_model
  fetch_bearer_token
  # ACME must complete before the TLS server block is rendered: the template references
  # a certificate path that only exists after certbot succeeded.
  issue_certificate
  render_nginx_config
  prewarm_and_assert_thinking_off
  verify_isolation
  log "bootstrap finished; run the runbook section B2 checks before enabling the trial"
}

main "$@"
