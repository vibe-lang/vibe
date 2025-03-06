#!/bin/bash

# Run all tests with verbose output and color
# The -v flag enables verbose output
# The -p=1 flag disables parallel test execution to make output clearer

echo "Running Vibe tests..."
go test -v -p=1 ./pkg/... ./cmd/...

# Check if any tests failed
if [ $? -eq 0 ]; then
    echo "✅ All tests passed!"
else
    echo "❌ Some tests failed."
fi