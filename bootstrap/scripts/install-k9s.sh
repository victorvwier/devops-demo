#!/usr/bin/env bash
set -euo pipefail

version="${K9S_VERSION:-latest}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

if [[ "${version}" == "latest" ]]; then
  url="https://github.com/derailed/k9s/releases/latest/download/k9s_Linux_amd64.tar.gz"
else
  url="https://github.com/derailed/k9s/releases/download/${version}/k9s_Linux_amd64.tar.gz"
fi

curl -fsSL "${url}" -o "${tmpdir}/k9s.tar.gz"
tar -xzf "${tmpdir}/k9s.tar.gz" -C "${tmpdir}"
install -m 0755 "${tmpdir}/k9s" /usr/local/bin/k9s
