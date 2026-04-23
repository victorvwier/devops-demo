#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <droplet-ip>"
  exit 1
fi

exec ssh \
  -L 3000:127.0.0.1:3000 \
  root@"$1" \
  'kubectl -n observability port-forward svc/grafana 3000:3000 >/tmp/grafana-portforward.log 2>&1 & wait'
