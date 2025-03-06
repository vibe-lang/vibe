#!/bin/bash
set -e

echo "Building Vibe language interpreter..."
go build -o vibe cmd/vibe/main.go

echo "Build complete. You can now run a Vibe script with:"
echo "./vibe run examples/hello.vb"