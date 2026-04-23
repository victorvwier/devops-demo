#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <droplet-ip>"
  exit 1
fi

droplet_ip="$1"
script_dir="$(cd "$(dirname "$0")" && pwd)"

"${script_dir}/port-forward-frontend.sh" "${droplet_ip}" &
frontend_pid=$!

"${script_dir}/port-forward-argocd.sh" "${droplet_ip}" &
argocd_pid=$!

"${script_dir}/port-forward-grafana.sh" "${droplet_ip}" &
grafana_pid=$!

printf '%s\n' "Frontend: http://localhost:8081" "Argo CD: http://localhost:8080" "Grafana: http://localhost:3000"

trap 'kill ${frontend_pid} ${argocd_pid} ${grafana_pid} 2>/dev/null || true' EXIT
wait ${frontend_pid} ${argocd_pid} ${grafana_pid}
