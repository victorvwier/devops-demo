#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${K3S_URL:-}" || -z "${K3S_TOKEN:-}" ]]; then
  echo "K3S_URL and K3S_TOKEN must be set"
  exit 1
fi

curl -sfL https://get.k3s.io | K3S_URL="$K3S_URL" K3S_TOKEN="$K3S_TOKEN" sh -
