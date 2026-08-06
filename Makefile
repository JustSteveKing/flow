GO_DIRS := ./internal ./cmd/flow ./cmd/flow-desktop

.PHONY: all cli desktop-frontend desktop test vet fmt fmt-check check clean

all: cli

# Build the CLI.
cli:
	go build -o bin/flow ./cmd/flow

# Build the React overlay frontend into cmd/flow-desktop/frontend/dist.
desktop-frontend:
	cd cmd/flow-desktop/frontend && npm install && npm run build

# Build the desktop binary (frontend must be built first; it is go:embed'd).
desktop: desktop-frontend
	go build -o bin/flow-desktop ./cmd/flow-desktop

test:
	go test -race ./internal/... ./cmd/flow

vet:
	go vet ./internal/... ./cmd/flow

fmt:
	gofmt -w $(GO_DIRS)

fmt-check:
	@unformatted="$$(gofmt -l $(GO_DIRS))"; \
	if [ -n "$$unformatted" ]; then \
		echo "These files need gofmt:"; echo "$$unformatted"; exit 1; \
	fi

# What CI runs on the pure-Go surface.
check: fmt-check vet test

clean:
	rm -rf bin cmd/flow-desktop/frontend/dist/assets cmd/flow-desktop/frontend/dist/index.html
