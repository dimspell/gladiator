all: compile

mig_db_url ?= sqlite://database.sqlite
mig_dir ?= internal/console/database/migrations
mig_name ?= create_users_table
mig_version ?= VERSION

join_id ?= 2

build:
	go build -race -v -o /dev/null ./

serve:
	 go run -v ./ serve --backend-addr=127.0.0.1:6112 --console-addr=127.0.0.1:2137
	#go run ./ serve --backend-addr=0.0.0.0:6112 --console-addr=0.0.0.0:2137
	#(go build -v); (.\gladiator.exe serve --backend-addr=0.0.0.0:6112 --console-addr=0.0.0.0:2137)

test:
	go test -v --race ./...

console: clear
	go run -v ./ console --console-addr=127.0.0.1:2137
	#go run ./ console --console-addr=0.0.0.0:2137
	#(go build -v);; (.\gladiator.exe console --console-addr=0.0.0.0:2137)

backend:
	go run -v ./ backend --backend-addr=127.0.0.1:6112 --console-addr=192.168.121.212:2137

tools-install:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags sqlite github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	brew install bufbuild/buf/buf

gen-sqlc:
	sqlc generate

mig-create:
	migrate create -ext sql -dir $(mig_dir) -seq $(mig_name)

mig-up:
	migrate -database $(mig_db_url) -path $(mig_dir) up

mig-force:
	migrate -path $(mig_dir) -database $(mig_db_url) force $(mig_version)

grpc-test:
	buf curl \
		--schema proto \
		--data '{"pet_type": "PET_TYPE_SNAKE", "name": "Ekans"}' \
		http://localhost:8080/multi.v1.PetStoreService/PutPet

gen-grpc:
	buf generate proto

p2p-join:
	go run ./cmd/p2p-join -name="user2" -id=2

p2p-join3:
	go run ./cmd/p2p-join -name="user3" -id=3

p2p-host:
	go run ./cmd/p2p-host

clear:
	clear

relay-host: clear
	go run ./cmd/relay-host

relay-join: clear
	go run ./cmd/relay-join -player=$(join_id)
