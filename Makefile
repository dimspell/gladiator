all: compile

mig_db_url ?= sqlite://database.sqlite
mig_dir ?= console/database/migrations
mig_name ?= create_users_table
mig_version ?= VERSION

build:
	go build -v -o /dev/null ./

serve:
	 go run ./ serve --backend-addr=127.0.0.1:6112 --console-addr=127.0.0.1:2137
	#go run ./ serve --backend-addr=0.0.0.0:6112 --console-addr=0.0.0.0:2137
	#(go build -v); (.\gladiator.exe serve --backend-addr=0.0.0.0:6112 --console-addr=0.0.0.0:2137)

console:
	go run ./ console --console-addr=127.0.0.1:2137
	#go run ./ console --console-addr=0.0.0.0:2137
	#(go build -v);; (.\gladiator.exe console --console-addr=0.0.0.0:2137)

backend:
	go run ./ backend --backend-addr=127.0.0.1:6112 --console-addr=192.168.121.212:2137

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
