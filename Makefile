all: test build

build:
	go build ./cmd/open-bastion

test:
	go test ./...
