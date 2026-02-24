.PHONY: deps test coverage bench run docker-up docker-down

deps:
	go mod download

test:
	go test -v ./...

coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

bench:
	go test -bench=. -benchmem -run=NONE ./...

run: deps test
	go run ./cmd/api/main.go

docker-up:
	docker compose up --build

docker-down:
	docker compose down
