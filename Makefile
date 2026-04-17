APP_NAME=pingtower

.PHONY: build test run docker-build

build:
	go build -o dist/$(APP_NAME) ./cmd/server

test:
	go test ./...

run:
	go run ./cmd/server

docker-build:
	docker build -t $(APP_NAME):latest .
