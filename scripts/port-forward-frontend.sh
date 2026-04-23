#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <droplet-ip>"
  exit 1
fi

exec ssh \
  -L 8081:127.0.0.1:8081 \
  root@"$1" \
  'kubectl -n tiny-llm port-forward svc/tiny-llm-frontend 8081:80 >/tmp/tiny-llm-frontend-portforward.log 2>&1 & wait'
