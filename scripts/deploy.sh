#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────
# one-command deploy: install + configure a server
# ──────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ANSIBLE_DIR="${SCRIPT_DIR}/ansible"

# ── defaults ──────────────────────────────────
SSH_USER="${SSH_USER:-root}"
SSH_KEY="${SSH_KEY:-}"
HOST="${HOST:-}"
DOMAIN="${DOMAIN:-}"
EMAIL="${EMAIL:-}"

# ── usage ─────────────────────────────────────
usage() {
  cat >&2 <<EOF
Deploy the server to a remote host.

Usage:
  HOST=<ip-or-hostname>          (required)
  DOMAIN=<fqdn>                  (required — e.g. mygame.example.com)
  EMAIL=<email>                  (required — for Let's Encrypt)
  SSH_USER=<user>                (default: root)
  SSH_KEY=<path>                 (default: none — uses your default SSH key)
  ANSIBLE_EXTRA_VARS="k=v k=v"  (optional — passed through to ansible-playbook)
  $0

Example:
  HOST=1.2.3.4 \\
  DOMAIN=gladiator.example.com \\
  EMAIL=admin@example.com \\
  SSH_KEY=~/.ssh/id_ed25519 \\
  ./scripts/deploy.sh
EOF
  exit 1
}

# ── validate ──────────────────────────────────
[ -z "$HOST" ] && usage
[ -z "$DOMAIN" ] && usage
[ -z "$EMAIL" ] && usage

# ── galaxy deps ───────────────────────────────
echo "==> Installing Ansible collection dependencies..."
ansible-galaxy collection install -r "${ANSIBLE_DIR}/requirements.yml"

# ── run playbook ──────────────────────────────
echo "==> Deploying to ${HOST} as ${SSH_USER} ..."
set -x
ansible-playbook \
  -i "${HOST}," \
  "${ANSIBLE_DIR}/playbook.yml" \
  --user "${SSH_USER}" \
  ${SSH_KEY:+--private-key "${SSH_KEY}"} \
  --extra-vars "nginx_domain_fqdn=${DOMAIN} nginx_certbot_email=${EMAIL} ${ANSIBLE_EXTRA_VARS:-}"
set +x

echo ""
echo "==> Done. Server is live at https://${DOMAIN}"
