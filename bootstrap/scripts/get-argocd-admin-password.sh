#!/usr/bin/env bash
set -euo pipefail

for _ in $(seq 1 300); do
  if kubectl -n argocd get secret argocd-initial-admin-secret >/dev/null 2>&1; then
    kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d
    printf '\n'
    exit 0
  fi
  sleep 2
done

echo "argocd-initial-admin-secret not found"
exit 1
