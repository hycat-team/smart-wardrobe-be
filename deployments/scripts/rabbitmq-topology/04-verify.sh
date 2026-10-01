#!/usr/bin/env bash
# 04-verify.sh — confirm every queue matches the registry, and show DLQ depth.
#
# Run after 02/03 to close the loop, or on its own as a health look. Unlike
# 01-check-drift.sh this also surfaces dead-letter backlog, which is the other
# thing worth knowing after a topology change: a queue can have the right type
# and still be silently failing into its DLQ.
#
# Usage (on the box):
#   bash deployments/scripts/rabbitmq-topology/04-verify.sh [vps|aws]
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
  -h|--help) sed -n '2,12p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
  *) die "unknown target '$TARGET' (want vps or aws)" ;;
esac

ENV_FILE="$(ls .env.production .env.aws 2>/dev/null | head -1)"
[ -n "$ENV_FILE" ] || die "no .env file found in $(pwd)"

docker pull "$TOOL_IMAGE"

echo "==> Queue topology"
run_tool "$ENV_FILE" check
status=$?

echo
echo "==> DLQ depth"
# Read through the management API rather than rabbitmqctl: on the AWS box there
# is no broker container to exec into, and one code path serves both.
mgmt_port=15672
mgmt_scheme="http"
if [ "$TARGET" = "aws" ]; then
  mgmt_scheme="https"
fi
host="$(grep -E '^RABBITMQ_HOST=' "$ENV_FILE" | cut -d= -f2-)"
user="$(grep -E '^RABBITMQ_USER=' "$ENV_FILE" | cut -d= -f2-)"
pass="$(grep -E '^RABBITMQ_PASSWORD=' "$ENV_FILE" | cut -d= -f2-)"

if [ -n "$host" ] && [ -n "$user" ]; then
  curl -sS -u "$user:$pass" \
    "${mgmt_scheme}://${host}:${mgmt_port}/api/queues" \
    | tr ',' '\n' \
    | grep -E '"(name|type|messages)"' \
    | sed 's/^ *//' \
    | paste - - - \
    | grep 'dlq' \
    | sed 's/"name"://; s/"type"://; s/"messages"://' \
    | awk -F'\t' '{printf "  %-46s %-8s %s message(s)\n", $1, $2, $3}' || echo "  (no DLQ found)"
else
  echo "  (could not read RABBITMQ_HOST/USER from $ENV_FILE — skipped)"
fi

echo
if [ "$status" -eq 0 ]; then
  echo "Topology verified. A non-zero message count above means DLQ backlog, not a type problem."
else
  echo "Topology still does not match — see the DRIFT lines above." >&2
fi
exit $status