#!/bin/bash

# Quick start script for Network Monitor

echo "🌐 Network Monitor Quick Start"
echo "=============================="
echo ""

# Check if binary exists
if [ ! -f "./network-monitor" ]; then
    echo "Binary not found. Building..."
    ./build.sh
    if [ $? -ne 0 ]; then
        exit 1
    fi
    echo ""
fi

# Try to find ISP gateway
echo "Attempting to find your ISP gateway..."
if command -v traceroute &> /dev/null; then
    GATEWAY=$(traceroute -m 2 8.8.8.8 2>/dev/null | grep -E "^ 2" | awk '{print $2}')
    if [[ $GATEWAY =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "Found potential ISP gateway: $GATEWAY"
        echo ""
        echo "Starting monitor with targets: 8.8.8.8, 1.1.1.1, $GATEWAY"
        ./network-monitor -targets "8.8.8.8,1.1.1.1,$GATEWAY"
    else
        echo "Could not determine ISP gateway, using default targets"
        ./network-monitor
    fi
else
    echo "traceroute not found, using default targets"
    ./network-monitor
fi
