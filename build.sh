#!/bin/bash

echo "Building s3eker binaries..."

# Build all three binaries
go build -o s3eker ./cmd/launcher
go build -o s3eker-scraper ./cmd/scraper
go build -o s3eker-fbscan ./cmd/fbscan

echo "Build complete!"
echo ""
echo "You can now run:"
echo "  ./s3eker           # Main launcher"
echo "  ./s3eker-scraper   # Direct scraper access"
echo "  ./s3eker-fbscan    # Direct Firebase scanner access"
