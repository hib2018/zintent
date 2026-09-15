#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
prefix="${HOME}/.local"
stage="${repo_root}/zig-out/package"
force=0
dry_run=0

usage() {
  cat <<'EOF'
Usage: scripts/install.sh [--prefix ABSOLUTE_PATH] [--stage PATH] [--force] [--dry-run]

Install the zintent frontend and core using the global two-binary layout.
Existing files with different contents require --force.
EOF
}

while (($#)); do
  case "$1" in
    --prefix)
      [[ $# -ge 2 ]] || { echo "--prefix requires a value" >&2; exit 2; }
      prefix="$2"
      shift 2
      ;;
    --stage)
      [[ $# -ge 2 ]] || { echo "--stage requires a value" >&2; exit 2; }
      stage="$2"
      shift 2
      ;;
    --force)
      force=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ "${prefix}" != /* ]]; then
  echo "prefix must be an absolute path: ${prefix}" >&2
  exit 2
fi

frontend_source="${stage}/bin/zintent"
core_source="${stage}/libexec/zintent/zintent-core"
frontend_target="${prefix}/bin/zintent"
core_target="${prefix}/libexec/zintent/zintent-core"

for source in "${frontend_source}" "${core_source}"; do
  if [[ ! -f "${source}" || ! -x "${source}" ]]; then
    echo "missing executable: ${source}; run make package first" >&2
    exit 1
  fi
done

check_target() {
  local source="$1"
  local target="$2"
  if [[ -e "${target}" || -L "${target}" ]]; then
    if [[ ! -L "${target}" && -f "${target}" ]] && cmp -s "${source}" "${target}"; then
      return
    fi
    if ((force == 0)); then
      echo "refusing to overwrite existing file: ${target}" >&2
      echo "re-run with --force only after confirming it belongs to zintent" >&2
      exit 1
    fi
  fi
}

check_target "${frontend_source}" "${frontend_target}"
check_target "${core_source}" "${core_target}"

if ((dry_run == 1)); then
  printf 'Would install %s -> %s\n' "${frontend_source}" "${frontend_target}"
  printf 'Would install %s -> %s\n' "${core_source}" "${core_target}"
  exit 0
fi

install_one() {
  local source="$1"
  local target="$2"
  local target_dir
  local temp_dir
  if [[ ! -L "${target}" && -f "${target}" ]] && cmp -s "${source}" "${target}"; then
    printf 'Already installed %s\n' "${target}"
    return
  fi
  target_dir="$(dirname "${target}")"
  install -d "${target_dir}"
  temp_dir="$(mktemp -d "${target_dir}/.zintent-install.XXXXXX")"
  install -m 0755 "${source}" "${temp_dir}/payload"
  mv -f "${temp_dir}/payload" "${target}"
  rmdir "${temp_dir}"
}

install_one "${frontend_source}" "${frontend_target}"
install_one "${core_source}" "${core_target}"

printf 'Installed %s\n' "${frontend_target}"
printf 'Installed %s\n' "${core_target}"
