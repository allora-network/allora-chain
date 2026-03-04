#!/bin/bash
set -eu

# PURPOSE
# Restore a local Allora node data directory from a remote snapshot archive URL.
# This script is "restore-only": it DOES NOT start `allorad` or `cosmovisor`.
#
# WHAT IT DOES
# 1) Downloads the snapshot archive from SNAPSHOT_URL into a temporary file.
# 2) Optionally verifies SHA-256 when SNAPSHOT_SHA256 is provided.
# 3) Replaces APP_HOME/data entirely (deletes old data and re-extracts snapshot).
# 4) Extracts archive based on URL extension: .tar.zst, .tar.gz/.tgz, or .tar.
# 5) Writes APP_HOME/snapshot-restored-from-url.txt with URL/checksum/timestamp.
#
# END STATE
# - Restored chain data exists under APP_HOME/data.
# - No node process is started; no background service is left running by this script.
# - You must run `allorad start ...` or `cosmovisor run start ...` separately.
#
# REQUIRED INPUTS / OPTIONS
# - SNAPSHOT_URL (required): direct URL to snapshot archive.
# - SNAPSHOT_SHA256 (optional but recommended): checksum validation.
# - APP_HOME (optional): target node home dir; defaults to $HOME/.allorad.
#
# EXAMPLES
# 1) Safe/verified restore (recommended):
# SNAPSHOT_URL="https://example.com/allora-testnet-1-1234567.tar.zst" \
# SNAPSHOT_SHA256="abcdef123456..." \
# APP_HOME="$HOME/.allorad" \
# bash scripts/restore_snapshot_from_url.sh
#
# 2) Quick restore without checksum (faster, less safe):
# SNAPSHOT_URL="https://example.com/allora-testnet-1-1234567.tar.zst" \
# APP_HOME="$HOME/.allorad" \
# bash scripts/restore_snapshot_from_url.sh
#
# RUNNING A NODE AFTER RESTORE (separate step)
# - Binary mode:
#   allorad --home "$APP_HOME" start
# - Cosmovisor mode:
#   DAEMON_HOME="$APP_HOME" DAEMON_NAME=allorad cosmovisor run start --home "$APP_HOME"

if [ -z "${SNAPSHOT_URL:-}" ]; then
  echo "SNAPSHOT_URL is required"
  exit 1
fi

APP_HOME="${APP_HOME:-$HOME/.allorad}"
SNAPSHOT_SHA256="${SNAPSHOT_SHA256:-}"
DATA_DIR="${APP_HOME}/data"
RESTORE_FLAG="${APP_HOME}/snapshot-restored-from-url.txt"

mkdir -p "${APP_HOME}"

tmp_archive="$(mktemp)"
cleanup() {
  rm -f "${tmp_archive}"
}
trap cleanup EXIT

echo "Downloading snapshot from ${SNAPSHOT_URL}"
curl -fL "${SNAPSHOT_URL}" -o "${tmp_archive}"

if [ -n "${SNAPSHOT_SHA256}" ]; then
  echo "Verifying snapshot checksum"
  if command -v sha256sum >/dev/null 2>&1; then
    actual_sha="$(sha256sum "${tmp_archive}" | awk '{print $1}')"
  else
    actual_sha="$(shasum -a 256 "${tmp_archive}" | awk '{print $1}')"
  fi

  if [ "${actual_sha}" != "${SNAPSHOT_SHA256}" ]; then
    echo "Checksum mismatch"
    echo "expected: ${SNAPSHOT_SHA256}"
    echo "actual:   ${actual_sha}"
    exit 1
  fi
fi

echo "Resetting ${DATA_DIR}"
rm -rf "${DATA_DIR}"
mkdir -p "${DATA_DIR}"

case "${SNAPSHOT_URL}" in
  *.tar.zst)
    echo "Extracting .tar.zst snapshot"
    tar --zstd -xvf "${tmp_archive}" -C "${DATA_DIR}"
    ;;
  *.tar.gz|*.tgz)
    echo "Extracting .tar.gz snapshot"
    tar -xzvf "${tmp_archive}" -C "${DATA_DIR}"
    ;;
  *.tar)
    echo "Extracting .tar snapshot"
    tar -xvf "${tmp_archive}" -C "${DATA_DIR}"
    ;;
  *)
    echo "Unsupported snapshot archive extension in URL: ${SNAPSHOT_URL}"
    echo "Supported: .tar.zst, .tar.gz, .tgz, .tar"
    exit 1
    ;;
esac

echo "url=${SNAPSHOT_URL}" > "${RESTORE_FLAG}"
echo "sha256=${SNAPSHOT_SHA256}" >> "${RESTORE_FLAG}"
echo "restored_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")" >> "${RESTORE_FLAG}"
echo "Snapshot restore completed into ${DATA_DIR}"
