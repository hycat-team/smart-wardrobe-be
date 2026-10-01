#!/usr/bin/env bash
# 02-migrate-vps.sh — bring drifted VPS queues to the type the registry declares.
#
# This is the destructive one. For each drifted queue it drains the contents to
# a spill file, deletes the queue, redeclares it with the correct type, then
# replays the drained messages. RabbitMQ offers no in-place conversion between
# classic and quorum, so this drain/delete/redeclare sequence is the only route.
#
# It refuses to run while workers are attached: deleting a queue under a live
# consumer races the delivery in flight, so the script stops the workers first.
#
# Usage (on the VPS box):
#   bash deployments/scripts/rabbitmq-topology/02-migrate-vps.sh [--yes] [--queue <name>]
#
# Services are NOT restarted by this script — bring them back yourself, so you
# decide when traffic resumes.
set -uo pipefail

cd "$(dirname "$0")"
# shellcheck source=common.sh
. "./common.sh"

ASSUME_YES=0
QUEUE_ARGS=()

while [ $# -gt 0 ]; do
  case "$1" in
    --yes|-y) ASSUME_YES=1 ;;
    --queue) shift; QUEUE_ARGS=(--queue "$1") ;;
    -h|--help) sed -n '2,18p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) die "unknown argument '$1' (want --yes or --queue <name>)" ;;
  esac
  shift
done

require_box_env "$VPS_DIR" .env.production
cd "$VPS_DIR"
RABBITMQ_NETWORK="$VPS_NETWORK"
ENV_FILE=.env.production

docker pull "$TOOL_IMAGE"

echo "==> Step 1/4 — show current drift"
# Exit 1 here means drift EXISTS, which is the precondition for migrating — not
# an error. Exit 2 would be a real failure (unreachable broker, bad env).
run_tool "$ENV_FILE" check "${QUEUE_ARGS[@]}"
check_status=$?
if [ "$check_status" -eq 2 ]; then
  echo "The check itself failed (could not reach the broker). Stopping." >&2
  exit 2
fi
if [ "$check_status" -eq 0 ]; then
  echo
  echo "Nothing is drifted — no migration needed."
  exit 0
fi

echo
if [ "$ASSUME_YES" -ne 1 ]; then
  cat <<EOF

Drift is present (shown above). Migrating will:
  - stop the workers:      $VPS_SERVICES
  - drain each drifted queue to $SPILL_HOST_DIR
  - delete and recreate those queues with the declared type
  - replay the drained messages
  - change no other queue

Re-run with --yes to proceed, or --queue <name> to migrate one queue only.
EOF
  exit 0
fi

echo "==> Step 2/4 — stop the workers ($VPS_SERVICES)"
$VPS_COMPOSE stop $VPS_SERVICES
echo "    (each VPS container runs both http and worker, so all three are stopped)"

echo
echo "==> Step 3/4 — drain, recreate and replay"
run_tool "$ENV_FILE" migrate --yes "${QUEUE_ARGS[@]}"
migrate_status=$?

echo
echo "==> Step 4/4 — re-check"
run_tool "$ENV_FILE" check
check_again=$?

echo
echo "Restart the services when ready:"
echo "  cd $VPS_DIR && $VPS_COMPOSE up -d $VPS_SERVICES"
echo
if [ "$migrate_status" -ne 0 ]; then
  echo "Migration FAILED (exit $migrate_status). Do not restart the services." >&2
  echo "Drained messages, if any, are in $SPILL_HOST_DIR — replay after fixing." >&2
  exit "$migrate_status"
fi
if [ "$check_again" -ne 0 ]; then
  echo "Migration ran but the queue types still do not match — inspect before restarting." >&2
  exit 1
fi
echo "Migration complete and verified."