# cloud

```bash
ansible-galaxy collection install community.general # for ufw
```

### Handy commands

#### Lint the Ansible files

```bash
ansible-lint ./playbook.yml
```

#### Deploy the Ansible's playbook

```bash
ansible-playbook \
  -i ./inventory.ini \
  ./playbook.yml \
  --user root \
  --private-key path/to/private/key
```

### Troubleshooting

#### Connect via SSH to the instance (and keep it running without freezing)

```bash
ssh -o "ServerAliveInterval 10" -o "TCPKeepAlive yes" -i path/to/private/key root@instance-address
```

#### Check the systemctl service status of the other's user 

```bash
systemctl --user --machine=appuser@ status gladiator.service
```

#### Impersonate as other user and run bash (su for users without shell configured)

```bash
sudo machinectl shell appuser@ /bin/bash
```

#### Check the Quadlet configuration (convert container definition to a systemd service)

```bash
/usr/lib/systemd/system-generators/podman-system-generator --user --dryrun
```
