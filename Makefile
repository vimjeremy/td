.PHONY: build test install
build:
	go build -o td .
test:
	go vet ./... && go test ./...
install:
	go build -o $(HOME)/.local/bin/td .
