# Simulator host (AWS / VPS)

Authoritative WebSocket match host. Clients connect with room codes; the host
starts a match from the decks each player submits on create/join.

Default port: **7474**  
Connect URL: `ws://YOUR_PUBLIC_IP:7474/`

## One-command deploy (recommended)

`scripts/deploy-simhost.sh` is gitignored so personal host/key paths stay local.
Copy the example once, edit defaults if you want, then run it:

```bash
cp scripts/deploy-simhost.example.sh scripts/deploy-simhost.sh
chmod +x scripts/deploy-simhost.sh
SIMHOST_HOST=YOUR_EC2_IP SIMHOST_KEY=~/path/to/key.pem ./scripts/deploy-simhost.sh
```

Optional: `SIMHOST_USER`, `SIMHOST_SKIP_BUILD=1`, `SIMHOST_SKIP_SMOKE=1`.

## Option A — binary (manual)

### 1. Build the release tarball (on your laptop)

From the repo root:

```bash
./scripts/package-simhost.sh
```

This writes `dist/casters-simhost-linux-amd64.tar.gz` containing:

- `simhost` (Linux amd64 binary)
- `data/cards.json`
- `simhost.service` (systemd unit)
- this README

### 2. Copy to the server

```bash
scp dist/casters-simhost-linux-amd64.tar.gz ubuntu@YOUR_EC2_IP:~/
```

### 3. On the EC2 instance

```bash
sudo mkdir -p /opt/casters-simhost
sudo tar -xzf ~/casters-simhost-linux-amd64.tar.gz -C /opt/casters-simhost
cd /opt/casters-simhost

# quick foreground test
./simhost -addr :7474 -catalog data/cards.json
# Ctrl+C when it looks good

# install as a service
sudo cp simhost.service /etc/systemd/system/casters-simhost.service
sudo systemctl daemon-reload
sudo systemctl enable --now casters-simhost
sudo systemctl status casters-simhost
```

### 4. Security group / firewall

Inbound **TCP 7474** from the IPs that should play (or `0.0.0.0/0` for a public
playtest). No other ports are required for simhost itself.

SSH remains on 22 as usual.

### 5. Smoke test

From your laptop (with the repo):

```bash
go run ./cmd/simsmoke -url ws://YOUR_PUBLIC_IP:7474/
```

You should see a room code, guest join, and `status=Setup`.

## Option B — Docker

On a machine with Docker (build locally or on the instance):

```bash
docker build -f Dockerfile.simhost -t casters-simhost .
docker run -d --name casters-simhost --restart unless-stopped -p 7474:7474 casters-simhost
```

Same security-group rule: TCP **7474**.

## Notes

- Hotseat in the desktop app is unchanged; networked Fyne Host/Join UI is separate.
- TLS (`wss://`) is not included yet — put nginx/Caddy in front later if you need it.
- Each filled room starts from the two decks players submit on create/join
  (protocol version 3). Redeploy `simhost` whenever the desktop app protocol
  version advances.
