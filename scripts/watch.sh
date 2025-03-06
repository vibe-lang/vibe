#!/bin/bash

# Check if fswatch is installed
if ! command -v fswatch &> /dev/null; then
    echo "❌ fswatch is not installed. Please install it with:"
    echo "    brew install fswatch"
    exit 1
fi

echo "🔍 Watching for changes in pkg/ and cmd/ directories..."
echo "   Press Ctrl+C to stop watching."
echo ""

# Run tests once at the beginning
./scripts/test.sh

# Watch for changes in Go files and run tests when changes are detected
fswatch -o pkg cmd | while read; do
    echo "\n📂 Changes detected, running tests..."
    ./scripts/test.sh
    echo "\n🔍 Watching for more changes..."
done