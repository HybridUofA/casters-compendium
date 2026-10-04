#!/usr/bin/env bash
# Build a Linux amd64 tarball for AWS/VPS deploy of cmd/simhost.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="${ROOT}/dist"
STAGE="${DIST}/casters-simhost-linux-amd64"
OUT="${DIST}/casters-simhost-linux-amd64.tar.gz"

mkdir -p "${STAGE}/data"
rm -f "${OUT}"

echo "building linux/amd64 simhost..."
(cd "${ROOT}" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "${STAGE}/simhost" ./cmd/simhost)

cp "${ROOT}/data/cards.json" "${STAGE}/data/cards.json"
cp "${ROOT}/deploy/simhost/simhost.service" "${STAGE}/simhost.service"
cp "${ROOT}/deploy/simhost/README.md" "${STAGE}/README.md"

tar -czf "${OUT}" -C "${DIST}" casters-simhost-linux-amd64
echo "wrote ${OUT}"
ls -lh "${OUT}"
