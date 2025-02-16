VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null || echo no-version)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GOLDFLAGS=-w -X main.version=$(VERSION) -X main.commit=$(GIT_COMMIT)

OUTDIR ?= out

.PHONY: lint
lint:
	go mod tidy
	go fmt ./...
	go vet ./...
	golangci-lint run

.PHONY: test
test:
	go test ./...
	go test -race ./...

.PHONY: build
build:
	go build -ldflags "$(GOLDFLAGS)" -o $(OUTDIR)/kubectl-evict

.PHONY: install
install:
	go install -ldflags "$(GOLDFLAGS)"

clean:
	-rm -r $(OUTDIR)
