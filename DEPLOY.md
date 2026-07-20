# Deploy

## Prerequisites

- **Ansible** installed locally (`brew install ansible` or `pip install ansible`)
- **SSH key** with root (or sudo-capable user) access to the target server
- A **domain name** pointing to the server's public IP

## Quick start

```bash
HOST=1.2.3.4 \
DOMAIN=gladiator.example.com \
EMAIL=admin@example.com \
./scripts/deploy.sh
```

That's it. The script will:

1. Install the required Ansible collection (`community.general`)
2. Run the playbook against the server

You can also go through the Makefile:

```bash
make deploy HOST=1.2.3.4 DOMAIN=gladiator.example.com EMAIL=admin@example.com
```

### Optional variables

| Variable | Default | Description |
|---|---|---|
| `SSH_USER` | `root` | SSH user on the target |
| `SSH_KEY` | *(none)* | Path to your SSH private key |
| `ANSIBLE_EXTRA_VARS` | *(none)* | Additional vars forwarded to Ansible (`k=v k=v`) |

## What gets installed

| Role | What it does |
|---|---|
| **podman** | Installs Podman, enables auto-update timer |
| **security** | Hardens SSH (key-only), enables UFW (ports 22/80/443/9999), installs fail2ban & unattended-upgrades |
| **appuser** | Creates unprivileged `appuser`, enables systemd lingering, sets file limits |
| **gladiator** | Deploys the container as a Quadlet systemd service via Podman |
| **nginx** | Installs nginx + certbot, provisions a Let's Encrypt certificate, configures reverse proxy |

## Building the container image locally

```bash
make docker-build
# or
docker compose build
```

This compiles the Go binary inside a multi-stage Docker build and produces the `gladiator` image. Run it locally:

```bash
docker compose up -d
```

## SSH reference

```bash
make ssh HOST=1.2.3.4 SSH_KEY=~/.ssh/id_ed25519
```

## Per-server notes

- The container runs rootless under `appuser` via Quadlet (Podman's systemd integration)
- Container auto-updates: `podman-auto-update.timer` checks for new images on a system timer
- The SQLite database persists in `/var/lib/gladiator/` on the host
- UDP port 9999 is the game relay; TCP 2137 is the HTTP console (proxied through nginx on 443)
