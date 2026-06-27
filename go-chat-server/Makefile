.PHONY: build run test test-cover vet fmt tidy clean all

BINARY := bin/server
PKG    := ./internal/chat

# ---- build & run ----

build:
	go build -o $(BINARY) ./cmd/server/

run: build
	./$(BINARY)

# ---- test ----

test:
	go test -v -count=1 $(PKG)

test-cover:
	go test -v -cover -coverprofile=coverage.out $(PKG)
	go tool cover -func=coverage.out

# ---- quality ----

vet:
	go vet ./...

fmt:
	gofmt -s -w .

tidy:
	go mod tidy

# ---- housekeeping ----

clean:
	rm -f $(BINARY) coverage.out

# ---- all ----

all: build vet test
