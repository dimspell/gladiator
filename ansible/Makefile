SSH_USER ?= root
HOST_ADDR ?= your.server.ip

PATH_TO_PRIVATE_KEY ?= path/to/private/key

lint:
	ansible-lint ./roles

fix-lint:
	ansible-lint ./roles --fix

playbook:
	ansible-playbook \
		-i ./inventory.ini \
		./playbook.yml \
		--user $(SSH_USER) \
		--private-key $(PATH_TO_PRIVATE_KEY)

ssh:
	ssh -o "ServerAliveInterval 10" -o "TCPKeepAlive yes" -i $(PATH_TO_PRIVATE_KEY) $(SSH_USER)@$(HOST_ADDR)
