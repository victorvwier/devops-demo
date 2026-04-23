#!/usr/bin/env bash
set -euo pipefail

curl -sfL https://get.k3s.io | sh -s - server \
  --write-kubeconfig-mode 644 \
  --tls-san "${K3S_TLS_SAN:-127.0.0.1}"
