#!/bin/bash
# Script to run backend tests using Docker

set -e

echo "Building test container..."
docker build -f backend/Dockerfile.test -t backend-tests . -q

echo "Running tests..."
docker run --rm backend-tests go test ./pkg/validation ./internal/config -v -cover

echo ""
echo "Tests completed successfully!"
