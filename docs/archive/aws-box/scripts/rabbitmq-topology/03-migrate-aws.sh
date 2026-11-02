#!/usr/bin/env bash
# 03-migrate-aws.sh — bring drifted AWS box queues to the declared type.
#
# Same drain/delete/redeclare/replay as 02-migrate-vps.sh, but the broker is
# CloudAMQP: an external managed service, not a container on this box. That
# changes three things:
#
#   - no --network flag: the tool reaches CloudAMQP over the public internet
#     (amqps + management TLS, both from .env.aws)
#   - only the -worker containers consume queues; the -http ones never do
#   - connection budget: the plan caps total connections, so the tool prints the
#     current count before draining
#
# Usage (on the AWS box):
#   bash deployments/scripts/rabbitmq-topology/03-migrate-aws.sh [--yes] [--queue <name>]
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
    -h|--help) sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) die "unknown argument '$1' (want --yes or --queue <name>)" ;;
  esac
  shift
done

require_box_env "$AWS_DIR" .env.aws
cd "$AWS_DIR"
ENV_FILE=.env.aws

docker pull "$TOOL_IMAGE"

echo "==> Step 1/4 — show current drift"
run_tool "$ENV_FILE" check "${QUEUE_ARGS[@]}"
check_status=$?
if [ "$check_status" -eq 2 ]; then
  echo "The check itself failed. Common causes: RABBITMQ_VHOST wrong, or" >&2
  echo "credentials without access to it (CloudAMQP returns 403 for a bad vhost)." >&2
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
  - stop the workers:  $AWS_SERVICES
  - drain each drifted queue to $SPILL_HOST_DIR
  - delete and recreate those queues on CloudAMQP with the declared type
  - replay the drained messages

Re-run with --yes to proceed, or --queue <name> to migrate one queue only.
EOF
  exit 0
fi

echo "==> Step 2/4 — stop the workers ($AWS_SERVICES)"
$AWS_COMPOSE stop $AWS_SERVICES
echo "    (only -worker containers consume queues on this box)"

echo
echo "==> Step 3/4 — drain, recreate and replay"
run_tool "$ENV_FILE" migrate --yes "${QUEUE_ARGS[@]}"
migrate_status=$?

echo
echo "==> Step 4/4 — re-check"
run_tool "$ENV_FILE" check
check_again=$?

echo
echo "Restart the workers when ready (the -http containers were left running):"
echo "  cd $AWS_DIR && $AWS_COMPOSE up -d $AWS_SERVICES"
echo
if [ "$migrate_status" -ne 0 ]; then
  echo "Migration FAILED (exit $migrate_status). Do not restart the workers." >&2
  echo "Drained messages, if any, are in $SPILL_HOST_DIR — replay after fixing." >&2
  exit "$migrate_status"
fi
if [ "$check_again" -ne 0 ]; then
  echo "Migration ran but the queue types still do not match — inspect before restarting." >&2
  exit 1
fi
echo "Migration complete and verified."