#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <droplet-ip>"
  exit 1
fi

exec ssh -N -L 8080:127.0.0.1:30080 root@"$1"
