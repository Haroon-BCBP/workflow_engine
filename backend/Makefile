.PHONY: api worker build vendor test

api:
	go run ./cmd/api/main.go

worker:
	go run ./cmd/worker/main.go

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

vendor:
	go mod tidy && go mod vendor

test:
	go test ./...

dev:
	@echo "Run 'make watch-api' and 'make watch-worker' in separate terminals"
	@echo "Then run 'cd ../UI && npm run dev' for the frontend"

watch-api:
	clear && air -c .air.toml

watch-worker:
	clear && air -c .air.worker.toml
