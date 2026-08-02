.PHONY: run build dev clean db-create db-migrate db-seed

run:
	go run ./cmd/server/

build:
	go build -o build/investo.exe ./cmd/server/

dev:
	air

clean:
	rm -rf build/

db-create:
	mysql -u root -e "CREATE DATABASE IF NOT EXISTS investo CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

db-migrate: db-create
	go run ./cmd/server/ --migrate

db-seed:
	go run ./cmd/server/ --seed

deps:
	go mod tidy
	go mod download

all: deps build
