#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <droplet-ip>"
  exit 1
fi

droplet_ip="$1"
frontend_url="http://${droplet_ip}:30081"
argocd_url="https://${droplet_ip}:30080"
grafana_url="http://${droplet_ip}:30030"

if command -v xdg-open >/dev/null 2>&1; then
  xdg-open "${frontend_url}" >/dev/null 2>&1 || true
  xdg-open "${argocd_url}" >/dev/null 2>&1 || true
  xdg-open "${grafana_url}" >/dev/null 2>&1 || true
elif command -v open >/dev/null 2>&1; then
  open "${frontend_url}" >/dev/null 2>&1 || true
  open "${argocd_url}" >/dev/null 2>&1 || true
  open "${grafana_url}" >/dev/null 2>&1 || true
fi

printf '%s\n' "Frontend: ${frontend_url}" "Argo CD: ${argocd_url}" "Grafana: ${grafana_url}"
