#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <droplet-ip>"
  exit 1
fi

droplet_ip="$1"
script_dir="$(cd "$(dirname "$0")" && pwd)"

"${script_dir}/port-forward-argocd.sh" "${droplet_ip}" &
argocd_pid=$!

"${script_dir}/port-forward-grafana.sh" "${droplet_ip}" &
grafana_pid=$!

trap 'kill ${argocd_pid} ${grafana_pid} 2>/dev/null || true' EXIT
wait ${argocd_pid} ${grafana_pid}
