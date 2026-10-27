#!/usr/bin/env bash
# Feature 032 - idempotent bootstrap for the Ollama CPU trial host.
#
# Contract: specs/032-ollama-lightsail-trial/contracts/env-ai-trial.md sections 2 and 3
# Runbook:  docs/operations/ollama-trial-runbook.md sections 3 and 4
#
# Responsibilities
#   1. Mount the dedicated gp3 volume on /var/lib/ollama (idempotent fstab entry).
#   2. Install the PINNED Ollama version and refuse to continue on a mismatch.
#   3. Pull the model, verify its digest and stop before measurement on mismatch.
#   4. Read the proxy Bearer token from Secrets Manager through the instance role
#      and never log it.
#   5. Serve the ACME challenge, obtain the certificate, then render the TLS/Bearer
#      nginx config. If DNS is not ready yet the script does not hard-fail: it
#      leaves the HTTP config, writes TLS_PENDING and exits 0.
#   6. Pre-warm the model and PROVE thinking is off (no reasoning content, zero
#      reasoning tokens, valid JSON).
#   7. Assert Ollama listens on loopback only, then write TRIAL_READY.
#
# Rules:
#   - No secret is ever written to this file, to a log line, or to the repository.
#   - The script must be safe to re-run: no duplicate fstab entries, no duplicate
#     model pull, no certificate re-issue while a valid certificate exists.
#
# Usage: sudo /opt/ollama-trial/bootstrap.sh
# Requires: /etc/ollama-trial/trial.env rendered by cloud-init.

set -euo pipefail

readonly TRIAL_DIR="/etc/ollama-trial"
readonly ENV_FILE="${TRIAL_DIR}/trial.env"
readonly TEMPLATE_DIR="/opt/ollama-trial"
readonly NGINX_CONF="/etc/nginx/conf.d/ollama.conf"
readonly BEARER_FILE="${TRIAL_DIR}/proxy-bearer"
readonly READY_FILE="${TRIAL_DIR}/TRIAL_READY"
readonly TLS_PENDING_FILE="${TRIAL_DIR}/TLS_PENDING"
readonly OLLAMA_DATA_DIR="/var/lib/ollama"
readonly ACME_WEBROOT="/var/www/certbot"
readonly LOG_PREFIX="[032-bootstrap]"

log() {
  printf '%s %s\n' "${LOG_PREFIX}" "$*"
}

warn() {
  printf '%s WARNING: %s\n' "${LOG_PREFIX}" "$*" >&2
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

# ------------------------------------------------------------------------------
# Configuration
# ------------------------------------------------------------------------------
# Keep this list in sync with contracts/env-ai-trial.md section 2.
load_env() {
  [[ -r "${ENV_FILE}" ]] || fail "missing ${ENV_FILE}; cloud-init did not render the trial env"
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  require_env AWS_REGION
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

  # Loopback-only inference is a hard invariant, not a preference.
  [[ "${OLLAMA_HOST}" == "127.0.0.1:11434" ]] ||
    fail "OLLAMA_HOST is ${OLLAMA_HOST}; the trial requires 127.0.0.1:11434 so 11434 never listens publicly"
}

# ------------------------------------------------------------------------------
# Data volume
# ------------------------------------------------------------------------------
# Ubuntu 24.04 exposes EBS volumes as nvme devices, so the configured /dev/sdf may
# resolve to /dev/nvme1n1. Resolve it and never touch the root device.
resolve_data_device() {
  local configured="${TRIAL_DATA_DEVICE:-/dev/sdf}"
  local resolved root_source attempt

  for attempt in $(seq 1 30); do
    if [[ -b "${configured}" ]]; then
      break
    fi
    sleep 1
  done
  [[ -b "${configured}" ]] || fail "data device ${configured} not present; check the Terraform EBS attachment"

  resolved="$(readlink -f "${configured}" 2>/dev/null || true)"
  [[ -n "${resolved}" && -b "${resolved}" ]] || fail "cannot resolve ${configured} to a block device"

  root_source="$(findmnt -n -o SOURCE / 2>/dev/null || true)"
  if [[ -n "${root_source}" && "${resolved}" == "${root_source}"* ]]; then
    fail "${configured} resolved to the root device (${resolved}); refusing to format it"
  fi
  case "${resolved}" in
    /dev/nvme0n1 | /dev/xvda | /dev/sda)
      fail "${configured} resolved to the root disk (${resolved}); refusing to format it"
      ;;
  esac

  printf '%s' "${resolved}"
}

# Mount the dedicated gp3 volume on /var/lib/ollama without losing data on re-run.
prepare_data_volume() {
  local device fs_uuid fstab_line
  device="$(resolve_data_device)"
  log "using data device ${device} for ${OLLAMA_DATA_DIR}"

  install -d -m 0755 "${OLLAMA_DATA_DIR}"

  if blkid "${device}" >/dev/null 2>&1; then
    log "data device already has a filesystem; keeping it"
  else
    log "formatting ${device} as ext4 (first boot only)"
    mkfs.ext4 -L ollama-models -q "${device}"
  fi

  fs_uuid="$(blkid -s UUID -o value "${device}")"
  [[ -n "${fs_uuid}" ]] || fail "cannot read the filesystem UUID of ${device}"

  if ! grep -qs "UUID=${fs_uuid}" /etc/fstab; then
    fstab_line="UUID=${fs_uuid} ${OLLAMA_DATA_DIR} ext4 defaults,nofail 0 2"
    printf '%s\n' "${fstab_line}" >>/etc/fstab
    log "added the fstab entry for UUID=${fs_uuid}"
  else
    log "fstab entry already present for UUID=${fs_uuid}"
  fi

  if mountpoint -q "${OLLAMA_DATA_DIR}"; then
    log "${OLLAMA_DATA_DIR} is already mounted"
  else
    mount "${OLLAMA_DATA_DIR}"
  fi

  install -d -m 0755 "${OLLAMA_DATA_DIR}/models"
  chown -R ollama:ollama "${OLLAMA_DATA_DIR}" 2>/dev/null || true
}

# ------------------------------------------------------------------------------
# Ollama runtime
# ------------------------------------------------------------------------------
install_ollama() {
  if command -v ollama >/dev/null 2>&1 && ollama --version 2>/dev/null | grep -q "${OLLAMA_VERSION}"; then
    log "pinned Ollama ${OLLAMA_VERSION} already installed"
  else
    log "installing pinned Ollama ${OLLAMA_VERSION} (never 'latest')"
    # The official installer honours OLLAMA_VERSION. It is the only download this
    # script performs; a failure is fatal because an unpinned runtime invalidates
    # the trial evidence.
    curl -fsSL https://ollama.com/install.sh | OLLAMA_VERSION="${OLLAMA_VERSION}" sh ||
      fail "Ollama ${OLLAMA_VERSION} install failed; do not continue on an unpinned runtime"
  fi

  local installed
  installed="$(ollama --version 2>/dev/null | head -n1 || true)"
  [[ "${installed}" == *"${OLLAMA_VERSION}"* ]] ||
    fail "installed Ollama (${installed}) does not match pinned ${OLLAMA_VERSION}; refusing to measure"

  # Lifecycle/daemon management stays with the operator (new feature 032 rule):
  # this drop-in only sets the trial limits and the loopback bind.
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

  chown -R ollama:ollama "${OLLAMA_DATA_DIR}" 2>/dev/null || true

  systemctl daemon-reload
  systemctl enable ollama
  systemctl restart ollama

  log "waiting for Ollama on ${OLLAMA_HOST}"
  local attempt
  for attempt in $(seq 1 60); do
    if curl -fsS "http://${OLLAMA_HOST}/api/version" >/dev/null 2>&1; then
      break
    fi
    sleep 2
  done
  curl -fsS "http://${OLLAMA_HOST}/api/version" >/dev/null ||
    fail "Ollama did not become ready on ${OLLAMA_HOST}; check journalctl -u ollama"
}

pull_model() {
  log "pulling ${OLLAMA_MODEL} (pinned digest ${OLLAMA_MODEL_DIGEST})"
  ollama pull "${OLLAMA_MODEL}"

  local expected actual
  expected="$(printf '%s' "${OLLAMA_MODEL_DIGEST#sha256:}" | tr '[:upper:]' '[:lower:]' | cut -c1-12)"
  # `ollama list` prints the 12-character digest prefix in the ID column.
  # `|| true` keeps a harmless SIGPIPE from aborting the script (pipefail is on).
  actual="$(ollama list 2>/dev/null | awk -v model="${OLLAMA_MODEL}" '$1 == model { print $2; exit }' | tr '[:upper:]' '[:lower:]' | cut -c1-12 || true)"
  [[ -n "${actual}" ]] || fail "model ${OLLAMA_MODEL} not found after pull"

  if [[ "${expected}" != "${actual}" ]]; then
    fail "digest mismatch: pinned ${OLLAMA_MODEL_DIGEST}, host has ${actual}; stopping before measurement"
  fi
  log "model digest matches the pinned prefix ${actual}"
}

# ------------------------------------------------------------------------------
# Secret
# ------------------------------------------------------------------------------
fetch_bearer_token() {
  log "reading the proxy Bearer token from Secrets Manager"
  command -v aws >/dev/null 2>&1 ||
    fail "aws CLI is missing; the instance profile path cannot read the secret without it"

  local attempt token=""
  for attempt in $(seq 1 5); do
    if token="$(aws secretsmanager get-secret-value \
      --region "${AWS_REGION}" \
      --secret-id "${PROXY_BEARER_SECRET_ARN}" \
      --query SecretString --output text 2>/dev/null)"; then
      if [[ -n "${token}" && "${token}" != "None" ]]; then
        break
      fi
    fi
    token=""
    warn "secret read attempt ${attempt} failed; retrying in 5s"
    sleep 5
  done

  [[ -n "${token}" ]] ||
    fail "cannot read the proxy Bearer token from ${PROXY_BEARER_SECRET_ARN}. Check that the secret has a version (TF_VAR_proxy_bearer_token) and that the instance profile can call secretsmanager:GetSecretValue."

  # Never log the token. Write it to a root-only file consumed by the renderer.
  umask 077
  printf '%s' "${token}" >"${BEARER_FILE}"
  chmod 0600 "${BEARER_FILE}"
}

# ------------------------------------------------------------------------------
# nginx: ACME first, TLS second
# ------------------------------------------------------------------------------
ensure_acme_webroot() {
  install -d -m 0755 "${ACME_WEBROOT}"
}

# HTTP-only config used before a certificate exists. It serves the ACME challenge
# and nothing else, so a host that booted before DNS was ready is still usable and
# can obtain the certificate once the A record resolves.
render_acme_only_config() {
  ensure_acme_webroot
  log "rendering the ACME/HTTP-only config (certificate not issued yet)"
  cat >"${NGINX_CONF}" <<EOF
# Feature 032 - temporary ACME-only config, generated by bootstrap.sh.
# Replaced by the full TLS/Bearer config once certbot succeeds.
server {
    listen ${PROXY_ACME_PORT} default_server;
    listen [::]:${PROXY_ACME_PORT} default_server;
    server_name ${PROXY_HOSTNAME};

    server_tokens off;
    access_log off;

    location /.well-known/acme-challenge/ {
        root ${ACME_WEBROOT};
        default_type "text/plain";
        try_files \$uri =404;
    }

    location / {
        add_header Content-Type text/plain;
        return 503 "ollama trial host is not trial-ready: certificate pending\n";
    }
}
EOF
  chmod 0644 "${NGINX_CONF}"
  reload_nginx
}

reload_nginx() {
  nginx -t
  if systemctl is-active --quiet nginx; then
    systemctl reload nginx
  else
    systemctl enable nginx
    systemctl restart nginx
  fi
}

# Returns 0 when a certificate is available, 1 when it could not be issued yet.
# A failure here is not fatal for the host: the caller keeps the ACME config and
# leaves TLS_PENDING for the operator.
issue_certificate() {
  ensure_acme_webroot

  if [[ -f "/etc/letsencrypt/live/${PROXY_HOSTNAME}/fullchain.pem" ]]; then
    log "certificate already present for ${PROXY_HOSTNAME}"
    return 0
  fi

  log "requesting a Let's Encrypt certificate for ${PROXY_HOSTNAME} (HTTP-01)"
  if certbot certonly --webroot -w "${ACME_WEBROOT}" -d "${PROXY_HOSTNAME}" \
    --email "${PROXY_ACME_EMAIL}" --agree-tos --non-interactive --keep-until-expiring; then
    return 0
  fi

  warn "certbot did not obtain a certificate (usually the A record does not resolve yet)"
  return 1
}

# Render the TLS/Bearer template. The token is substituted through a sed escape
# helper so characters such as &, / and | cannot break the render.
render_nginx_config() {
  local template="${TEMPLATE_DIR}/nginx-ollama.conf.template"
  [[ -r "${template}" ]] || fail "missing ${template}; cloud-init must upload it"

  local token tls_cert tls_key
  token="$(cat "${BEARER_FILE}")"
  [[ -n "${token}" ]] || fail "${BEARER_FILE} is empty; the Bearer token was not fetched"
  tls_cert="/etc/letsencrypt/live/${PROXY_HOSTNAME}/fullchain.pem"
  tls_key="/etc/letsencrypt/live/${PROXY_HOSTNAME}/privkey.pem"
  [[ -r "${tls_cert}" && -r "${tls_key}" ]] || fail "certificate files are missing for ${PROXY_HOSTNAME}"

  local escaped_token escaped_cidr escaped_hostname
  escaped_token="$(printf '%s' "${token}" | sed -e 's/[\\&|]/\\&/g')"
  escaped_cidr="$(printf '%s' "${PROXY_ALLOWED_CIDR}" | sed -e 's/[\\&|]/\\&/g')"
  escaped_hostname="$(printf '%s' "${PROXY_HOSTNAME}" | sed -e 's/[\\&|]/\\&/g')"

  umask 027
  sed \
    -e "s|__PROXY_HOSTNAME__|${escaped_hostname}|g" \
    -e "s|__PROXY_ALLOWED_CIDR__|${escaped_cidr}|g" \
    -e "s|__PROXY_HTTPS_PORT__|${PROXY_HTTPS_PORT}|g" \
    -e "s|__PROXY_ACME_PORT__|${PROXY_ACME_PORT}|g" \
    -e "s|__PROXY_TLS_CERT_PATH__|${tls_cert}|g" \
    -e "s|__PROXY_TLS_KEY_PATH__|${tls_key}|g" \
    -e "s|__PROXY_BEARER_TOKEN__|${escaped_token}|g" \
    "${template}" >"${NGINX_CONF}"
  chmod 0640 "${NGINX_CONF}"
  log "rendered the TLS/Bearer nginx config"
}

# ------------------------------------------------------------------------------
# Verification
# ------------------------------------------------------------------------------
prewarm_and_assert_thinking_off() {
  log "pre-warming ${OLLAMA_MODEL} and asserting thinking is off"
  local payload response
  payload="$(printf '{"model":"%s","messages":[{"role":"user","content":"ping"}],"stream":false,"max_tokens":16,"temperature":0}' "${OLLAMA_MODEL}")"

  response="$(curl -fsS --max-time 300 "http://${OLLAMA_HOST}/v1/chat/completions" \
    -H 'Content-Type: application/json' \
    -d "${payload}")" || fail "pre-warm request failed; the model did not answer"

  command -v jq >/dev/null 2>&1 || fail "jq is missing; cannot verify the pre-warm response"

  printf '%s' "${response}" | jq -e . >/dev/null 2>&1 ||
    fail "pre-warm response is not valid JSON; do not start measuring"

  printf '%s' "${response}" | jq -e '.choices[0].message.content | type == "string" and length > 0' >/dev/null ||
    fail "pre-warm response has no content"

  if printf '%s' "${response}" | jq -e '((.choices[0].message.reasoning_content // "") | length > 0) or ((.choices[0].message.thinking // "") | length > 0)' >/dev/null; then
    fail "thinking is ON: the pre-warm response carried reasoning content. Change the pinned model/version before measuring."
  fi

  if printf '%s' "${response}" | jq -e '(.usage.completion_tokens_details.reasoning_tokens // 0) > 0' >/dev/null; then
    fail "thinking is ON: usage reports reasoning tokens. Do not start the trial."
  fi

  log "thinking off verified: no reasoning content, zero reasoning tokens"
}

verify_isolation() {
  log "verifying Ollama listens on 127.0.0.1:11434 only"
  if ! command -v ss >/dev/null 2>&1; then
    warn "ss is missing; cannot verify the listening sockets. Install iproute2 and re-run."
    return 0
  fi

  local offending
  offending="$(ss -lntH 2>/dev/null | awk '{print $4}' | grep -E ':11434$' | grep -v '^127\.0\.0\.1:11434$' || true)"
  if [[ -n "${offending}" ]]; then
    fail "Ollama is listening beyond 127.0.0.1:11434 (${offending}); refusing to mark the host trial-ready"
  fi

  log "allowlist in nginx: ${PROXY_ALLOWED_CIDR} (the security group is the outer guard)"
}

mark_ready() {
  {
    printf 'trial_ready_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'ollama_version=%s\n' "${OLLAMA_VERSION}"
    printf 'model=%s\n' "${OLLAMA_MODEL}"
    printf 'model_digest=%s\n' "${OLLAMA_MODEL_DIGEST}"
    printf 'proxy_hostname=%s\n' "${PROXY_HOSTNAME}"
    printf 'allowed_cidr=%s\n' "${PROXY_ALLOWED_CIDR}"
  } >"${READY_FILE}"
  chmod 0644 "${READY_FILE}"
  rm -f "${TLS_PENDING_FILE}"
}

# ------------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------------
main() {
  load_env
  prepare_data_volume
  install_ollama
  pull_model
  fetch_bearer_token
  prewarm_and_assert_thinking_off
  verify_isolation

  # ACME must be reachable before the TLS server block is rendered: the template
  # references certificate paths that only exist after certbot succeeded.
  render_acme_only_config

  if issue_certificate; then
    render_nginx_config
    reload_nginx
    mark_ready
    log "bootstrap finished: host is trial-ready (${READY_FILE})"
    return 0
  fi

  # Not trial-ready, but NOT a hard failure: cloud-init must finish so the operator
  # can create the A record and re-run this script over SSM.
  rm -f "${READY_FILE}"
  {
    printf 'certificate_pending_at=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf 'proxy_hostname=%s\n' "${PROXY_HOSTNAME}"
    printf 'next_step=%s\n' "point the A record for ${PROXY_HOSTNAME} at the Elastic IP (terraform output elastic_ip), wait for DNS, then re-run sudo /opt/ollama-trial/bootstrap.sh"
  } >"${TLS_PENDING_FILE}"
  chmod 0644 "${TLS_PENDING_FILE}"
  warn "bootstrap finished WITHOUT TLS: the host is NOT trial-ready. See ${TLS_PENDING_FILE}, finish the DNS step, then re-run this script."
  printf '%s\n' "${LOG_PREFIX} NOT TRIAL-READY: certificate pending"
  return 0
}

main "$@"
