.PHONY: build test clean watch

# Default target
all: build

# Build the vibe interpreter
build:
	go build -o vibe cmd/vibe/main.go

# Run the tests once
test:
	./scripts/test.sh

# Clean build artifacts
clean:
	rm -f vibe

# Watch for changes and run tests automatically
watch:
	./scripts/watch.sh

# Install dependencies (like fswatch)
install-deps:
	@echo "Installing dependencies..."
	@if ! command -v fswatch &> /dev/null; then \
		echo "Installing fswatch..."; \
		brew install fswatch; \
	else \
		echo "fswatch already installed."; \
	fi