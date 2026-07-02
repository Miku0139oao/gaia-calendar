.PHONY: dev build test ent frontend

dev:
	go run ./cmd/server

build:
	bun --cwd frontend run build
	go build -o bin/gaia-calendar ./cmd/server

test:
	go test ./...

ent:
	go run entgo.io/ent/cmd/ent generate ./ent/schema

frontend:
	bun --cwd frontend install
	bun --cwd frontend run build
