#!/bin/bash
# Script to run backend tests using Docker

set -e

echo "Building test container..."
docker build -f backend/Dockerfile.test -t backend-tests . -q

echo "Running tests..."
echo "==============================================="
echo "Phase 1: Validation and Config Tests"
echo "==============================================="
docker run --rm backend-tests go test ./pkg/validation ./internal/config -v -cover

echo ""
echo "==============================================="
echo "Phase 2: Service Layer Tests"
echo "==============================================="
docker run --rm backend-tests go test ./internal/service/conversation ./internal/service/summary -v -cover

echo ""
echo "==============================================="
echo "Phase 3: ChatService Tests"
echo "==============================================="
docker run --rm backend-tests go test ./internal/service/chat -v -cover

echo ""
echo "==============================================="
echo "All Tests Summary"
echo "==============================================="
docker run --rm backend-tests go test ./pkg/validation ./internal/config ./internal/service/conversation ./internal/service/summary ./internal/service/chat -cover

echo ""
echo "Tests completed successfully!"
