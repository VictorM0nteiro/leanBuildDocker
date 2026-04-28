BINARY   := lbd
CMD_PATH := ./cmd/lbd
GO       := go

.PHONY: build test lint clean

build:
	$(GO) build -o $(BINARY) $(CMD_PATH)

test:
	go test ./...

test-verbose:
	go test -v ./...

test-cover:
	go test -cover ./...

lint:
	$(GO) vet ./...

clean:
	rm -f $(BINARY)
	$(GO) clean -testcache
