#!/usr/bin/env bash
# Shared settings for the rabbitmq-topology scripts.
#
# Source this, do not run it. It resolves where the box keeps its env files and
# which image the tool runs as. Both VPS and AWS are covered here because the
# tool itself is identical — only the box layout and the broker location differ.
#
# Override any of these from the environment before sourcing.

# ---------------------------------------------------------------------------
# Tool image
# ---------------------------------------------------------------------------
# The deploy workflows run a topology check against the image they just built,
# pinned by digest. Paste that ref here (it is printed in the workflow log) or
# export TOOL_IMAGE before sourcing. `latest` is acceptable when the tool has not
# changed recently, but a digest is what CI actually verified.
TOOL_IMAGE="${TOOL_IMAGE:-ghcr.io/hycat/rabbitmq-topology:latest}"

# Where the drain spill files go. Mounted into the container so drained messages
# survive the container exiting — without this they would be lost with it.
SPILL_HOST_DIR="${SPILL_HOST_DIR:-/opt/closy-rabbitmq-drain}"

# ---------------------------------------------------------------------------
# VPS box — self-hosted broker in the compose stack
# ---------------------------------------------------------------------------
VPS_DIR="${VPS_DIR:-/opt/closy}"
VPS_COMPOSE="docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml"
# Each VPS container runs both http and worker, so all three consume queues.
VPS_SERVICES="account commerce ai"
# The broker is a container on this network; the tool joins it to reach
# rabbitmq:15672 and rabbitmq:5672 by service name.
VPS_NETWORK="closy-prod_data"
# Set by 01/02 (VPS scripts) to join the broker network; left empty on AWS.
RABBITMQ_NETWORK=""

# ---------------------------------------------------------------------------
# AWS box — CloudAMQP, no broker container
# ---------------------------------------------------------------------------
AWS_DIR="${AWS_DIR:-/opt/closy-aws}"
AWS_COMPOSE="docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml"
# On AWS only the -worker containers consume queues; the -http ones do not.
AWS_SERVICES="account-worker commerce-worker ai-worker"
# No network flag: CloudAMQP is reached over the public internet via amqps.

die() {
  printf '\033[31mERROR\033[0m %s\n' "$*" >&2
  exit 1
}

# require_box_env verifies the box directory and its env file exist before any
# destructive step, so a typo in DEPLOY_DIR fails fast and read-only.
require_box_env() {
  local dir="$1" env_file="$2"
  [ -d "$dir" ] || die "box directory not found: $dir (override with DEPLOY_DIR)"
  [ -f "$dir/$env_file" ] || die "env file not found: $dir/$env_file (check VPS_DIR / AWS_DIR)"
}

# run_tool invokes the reconcile tool. The spill directory is only mounted for
# migrate; check never writes anything.
run_tool() {
  local env_file="$1" mode="$2"
  shift 2
  local args=(--rm --env-file "$env_file")
  if [ -n "$RABBITMQ_NETWORK" ]; then
    args+=(--network "$RABBITMQ_NETWORK")
  fi
  if [ "$mode" = "migrate" ]; then
    # Drained messages are the only copy of that traffic until replay
    # finishes, so they must survive the container: the tool is told to write
    # to the mounted /spill, and an unwritable host dir fails here rather than
    # mid-drain, after messages have left the queue.
    mkdir -p "$SPILL_HOST_DIR" 2>/dev/null || true
    [ -d "$SPILL_HOST_DIR" ] && [ -w "$SPILL_HOST_DIR" ] ||
      die "spill dir $SPILL_HOST_DIR is missing or not writable (create it writable, or set SPILL_HOST_DIR)"
    args+=(-v "$SPILL_HOST_DIR:/spill")
    set -- --spill-dir /spill "$@"
  fi
  docker run "${args[@]}" "$TOOL_IMAGE" --mode "$mode" "$@"
}