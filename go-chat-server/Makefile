.PHONY: build run demo test test-cover vet fmt tidy clean all

BINARY := bin/server
PKG    := ./internal/chat

# ---- build & run ----

build:
	go build -o $(BINARY) ./cmd/server/

run: build
	./$(BINARY)

demo:
	@if lsof -i :8080 -sTCP:LISTEN >/dev/null 2>&1; then \
		echo "Server found on :8080, seeding demo data..."; \
		go run ./cmd/demo/; \
	else \
		echo "Server not running. Start it with 'make run' in another terminal,"; \
		echo "then re-run 'make demo'."; \
		exit 1; \
	fi

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
