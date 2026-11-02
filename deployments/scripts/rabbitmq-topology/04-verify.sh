#!/usr/bin/env bash
# 04-verify.sh — confirm every queue matches the registry, and show DLQ depth.
#
# Run after 02 to close the loop, or on its own as a health look. Unlike
# 01-check-drift.sh this also surfaces dead-letter backlog, which is the other
# thing worth knowing after a topology change: a queue can have the right type
# and still be silently failing into its DLQ.
#
# Usage (on the VPS):
#   bash deployments/scripts/rabbitmq-topology/04-verify.sh [vps]
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
  -h|--help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
  *) die "unknown target '$TARGET' (want vps)" ;;
esac

ENV_FILE=.env.production
[ -f "$ENV_FILE" ] || die "no $ENV_FILE found in $(pwd)"

docker pull "$TOOL_IMAGE"

echo "==> Queue topology"
run_tool "$ENV_FILE" check
status=$?

echo
echo "==> DLQ depth"
# Read through the management API rather than rabbitmqctl: the broker is a
# container on the compose network, and the HTTP API works the same from here.
mgmt_port=15672
mgmt_scheme="http"
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