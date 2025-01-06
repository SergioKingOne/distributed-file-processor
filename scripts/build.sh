#!/bin/bash

# Clean up previous build artifacts
rm -rf bin

# Create directories
mkdir -p bin/chunker
mkdir -p bin/worker

echo "Building chunker lambda..."
# Build using Amazon Linux 2 Docker image
docker run --rm \
    -v "$PWD":/app \
    -w /app \
    public.ecr.aws/amazonlinux/amazonlinux:2 \
    bash -c "yum install -y golang && go build -o bin/chunker/bootstrap src/chunker/main.go"

# Create zip with bootstrap file
cd bin/chunker && zip ../chunker.zip bootstrap && cd ../..

echo "Building worker lambda..."
# Build using Amazon Linux 2 Docker image
docker run --rm \
    -v "$PWD":/app \
    -w /app \
    public.ecr.aws/amazonlinux/amazonlinux:2 \
    bash -c "yum install -y golang && go build -o bin/worker/bootstrap src/worker/main.go"

# Create zip with bootstrap file
cd bin/worker && zip ../worker.zip bootstrap && cd ../..

echo "Build complete."