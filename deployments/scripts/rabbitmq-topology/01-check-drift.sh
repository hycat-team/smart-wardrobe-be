#!/usr/bin/env bash
# 01-check-drift.sh — compare the queue types the broker holds against the ones
# the registry declares. Read-only: this never mutates a queue.
#
# CI already runs the same check before every deploy (deploy-vps-services.yml /
# deploy-aws.yml). Run this when you want to look without waiting for CI, or to
# confirm a migration worked before deploying.
#
# Usage (on the box):
#   bash deployments/scripts/rabbitmq-topology/01-check-drift.sh [vps|aws]
#
# Exit codes: 0 = no drift (MISSING queues do not fail), 1 = drift, 2 = usage/setup error.
set -uo pipefail

cd "$(dirname "$0")"
# shellcheck source=common.sh
. "./common.sh"

TARGET="${1:-vps}"
case "$TARGET" in
  vps)
    require_box_env "$VPS_DIR" .env.production
    RABBITMQ_NETWORK="$VPS_NETWORK"
    cd "$VPS_DIR"
    ;;
  aws)
    require_box_env "$AWS_DIR" .env.aws
    cd "$AWS_DIR"
    ;;
  -h|--help)
    sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'
    exit 0
    ;;
  *)
    die "unknown target '$TARGET' (want vps or aws)"
    ;;
esac

ENV_FILE="$(ls .env.production .env.aws 2>/dev/null | head -1)"
[ -n "$ENV_FILE" ] || die "no .env file found in $(pwd)"

echo "==> Checking queue topology against the registry ($TARGET box)"
echo "    image=$TOOL_IMAGE env=$ENV_FILE"

# No `set -e` above: a drift exit code of 1 is a result to report, not a crash.
run_tool "$ENV_FILE" check
status=$?

case $status in
  0)
    echo
    echo "No drift. Every queue type matches the registry."
    ;;
  1)
    echo
    echo "Drift found. RabbitMQ cannot change x-queue-type on an existing queue,"
    echo "so a service declaring this topology will fail with 406 PRECONDITION_FAILED."
    echo "Follow 02-migrate-vps.sh (or 03-migrate-aws.sh) — migrate is never automatic."
    ;;
  *)
    echo
    echo "Could not complete the check (exit $status). See the error above." >&2
    ;;
esac
exit $status