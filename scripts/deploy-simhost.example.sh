#!/usr/bin/env bash
# Example deploy helper for simhost. Copy to a local ignored script first:
#   cp scripts/deploy-simhost.example.sh scripts/deploy-simhost.sh
#   chmod +x scripts/deploy-simhost.sh
# then edit defaults or export:
#   SIMHOST_HOST=YOUR_EC2_IP
#   SIMHOST_USER=ubuntu
#   SIMHOST_KEY=/path/to/your-key.pem
#   SIMHOST_SKIP_BUILD=1
#   SIMHOST_SKIP_SMOKE=1
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOST="${SIMHOST_HOST:?set SIMHOST_HOST to your server IP or hostname}"
USER_NAME="${SIMHOST_USER:-ubuntu}"
KEY="${SIMHOST_KEY:?set SIMHOST_KEY to your SSH private key path}"
REMOTE_DIR="${SIMHOST_REMOTE_DIR:-/opt/casters-simhost}"
TARBALL_NAME="casters-simhost-linux-amd64.tar.gz"
LOCAL_TARBALL="${ROOT}/dist/${TARBALL_NAME}"
SSH_OPTS=(-i "${KEY}" -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new)

# Expand a leading ~/ in SIMHOST_KEY.
KEY="${KEY/#\~/$HOME}"

if [[ ! -f "${KEY}" ]]; then
	echo "deploy-simhost: SSH key not found: ${KEY}" >&2
	exit 1
fi
chmod 400 "${KEY}" 2>/dev/null || true

if [[ "${SIMHOST_SKIP_BUILD:-}" != "1" ]]; then
	"${ROOT}/scripts/package-simhost.sh"
fi

if [[ ! -f "${LOCAL_TARBALL}" ]]; then
	echo "deploy-simhost: missing ${LOCAL_TARBALL}" >&2
	echo "Run ./scripts/package-simhost.sh first, or unset SIMHOST_SKIP_BUILD." >&2
	exit 1
fi

echo "copying ${TARBALL_NAME} to ${USER_NAME}@${HOST}:~/"
scp "${SSH_OPTS[@]}" "${LOCAL_TARBALL}" "${USER_NAME}@${HOST}:~/${TARBALL_NAME}"

echo "installing on ${HOST}..."
ssh "${SSH_OPTS[@]}" "${USER_NAME}@${HOST}" bash -s <<EOF
set -euo pipefail
sudo mkdir -p '${REMOTE_DIR}'
sudo systemctl stop casters-simhost 2>/dev/null || true
sudo tar -xzf \$HOME/${TARBALL_NAME} -C '${REMOTE_DIR}' --strip-components=1
sudo cp '${REMOTE_DIR}/simhost.service' /etc/systemd/system/casters-simhost.service
sudo systemctl daemon-reload
sudo systemctl enable casters-simhost
sudo systemctl restart casters-simhost
sudo systemctl --no-pager --full status casters-simhost
EOF

echo "deployed. connect: ws://${HOST}:7474/"

if [[ "${SIMHOST_SKIP_SMOKE:-}" != "1" ]]; then
	echo "running simsmoke..."
	(
		cd "${ROOT}"
		go run ./cmd/simsmoke -url "ws://${HOST}:7474/"
	)
fi
