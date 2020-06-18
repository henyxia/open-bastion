all: test build

build:
	go build $$(pwd)/cmd/open-bastion

test:
	go test ./...