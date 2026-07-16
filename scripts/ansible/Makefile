SSH_USER ?= root
HOST    ?= your.server.ip
SSH_KEY ?= ~/.ssh/id_rsa
DOMAIN  ?=
EMAIL   ?=

lint:
	ansible-lint ./roles

fix-lint:
	ansible-lint ./roles --fix

# Deploy to a remote host — delegates to scripts/deploy.sh
deploy:
	HOST=$(HOST) \
	DOMAIN=$(DOMAIN) \
	EMAIL=$(EMAIL) \
	SSH_USER=$(SSH_USER) \
	SSH_KEY=$(SSH_KEY) \
	../deploy.sh

ssh:
	ssh -o "ServerAliveInterval 10" -o "TCPKeepAlive yes" -i $(SSH_KEY) $(SSH_USER)@$(HOST)
