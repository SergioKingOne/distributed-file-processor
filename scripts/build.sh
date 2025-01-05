#!/bin/bash

echo "Building chunker lambda..."
GOOS=linux GOARCH=amd64 go build -o bin/chunker src/chunker/main.go

echo "Creating chunker.zip..."
zip -j bin/chunker.zip bin/chunker

echo "Building worker lambda..."
GOOS=linux GOARCH=amd64 go build -o bin/worker src/worker/main.go

echo "Build complete."