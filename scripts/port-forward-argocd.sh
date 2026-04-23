#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <droplet-ip>"
  exit 1
fi

droplet_ip="$1"

exec ssh \
  -L 8080:127.0.0.1:8080 \
  root@"${droplet_ip}" \
  'kubectl -n argocd port-forward svc/argocd-server 8080:443 >/tmp/argocd-portforward.log 2>&1 & wait'
