server-cert:
	mkcert \
		-key-file key.pem \
		-cert-file cert.pem \
		localhost

client-cert:
	mkcert \
		-client \
		-key-file client-key.pem \
		-cert-file client-cert.pem \
		localhost

link-ca-cert:
	#ln -s /root/.local/share/mkcert/rootCA.pem ca.pem
	cp "$(shell mkcert -CAROOT)/rootCA.pem" ca.pem

run:
	go run ./