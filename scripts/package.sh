#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
stage="${1:-${repo_root}/zig-out/package}"
frontend="${repo_root}/zig-out/bin/zintent"
core="${repo_root}/zig-out/bin/zintent-core"

if [[ ! -x "${frontend}" ]]; then
  echo "missing executable: ${frontend}; run make build first" >&2
  exit 1
fi
if [[ ! -x "${core}" ]]; then
  echo "missing executable: ${core}; run make build first" >&2
  exit 1
fi

install -d "${stage}/bin" "${stage}/libexec/zintent"
install -m 0755 "${frontend}" "${stage}/bin/zintent"
install -m 0755 "${core}" "${stage}/libexec/zintent/zintent-core"

printf 'Packaged zintent in %s\n' "${stage}"
