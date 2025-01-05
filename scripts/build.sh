#!/bin/bash

# Clean up previous build artifacts
rm -rf bin

# Create directories
mkdir -p bin/chunker
mkdir -p bin/worker

echo "Building chunker lambda..."
# Build the binary named 'bootstrap'
GOOS=linux GOARCH=amd64 go build -o bin/chunker/bootstrap src/chunker/main.go
# Create zip with bootstrap file
cd bin/chunker && zip ../chunker.zip bootstrap && cd ../..

echo "Building worker lambda..."
# Build the binary named 'bootstrap'
GOOS=linux GOARCH=amd64 go build -o bin/worker/bootstrap src/worker/main.go
# Create zip with bootstrap file
cd bin/worker && zip ../worker.zip bootstrap && cd ../..

echo "Build complete."