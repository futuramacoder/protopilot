.PHONY: build run test lint clean

build:
	CGO_ENABLED=0 go build -o ./bin/protopilot ./cmd/protopilot/

run:
	go run ./cmd/protopilot/ $(ARGS)

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf ./bin/
