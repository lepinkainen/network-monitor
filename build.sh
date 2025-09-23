#!/bin/bash

# Build script for Network Monitor

echo "Building Network Monitor..."

# Get dependencies
echo "Downloading dependencies..."
go mod download

# Build the binary
echo "Building binary..."
go build -o network-monitor *.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "To run the monitor:"
    echo "  ./network-monitor"
    echo ""
    echo "To run with custom targets:"
    echo "  ./network-monitor -targets \"8.8.8.8,1.1.1.1,YOUR_ISP_GATEWAY\""
    echo ""
    echo "Dashboard will be available at: http://localhost:8080"
else
    echo "❌ Build failed. Please check for errors above."
    exit 1
fi
