#!/bin/bash

# Build script for Tab'd Native Host

set -e

echo "Building Tab'd Native Host..."

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Build for current platform
echo "Building for $(go env GOOS)/$(go env GOARCH)..."
go build -o tabd-native-host main.go

# Make executable
chmod +x tabd-native-host

echo "Build complete: tabd-native-host"

# Optionally build for other platforms
if [ "$1" = "all" ]; then
    echo "Building for all platforms..."
    
    # macOS
    GOOS=darwin GOARCH=amd64 go build -o tabd-native-host-darwin-amd64 main.go
    GOOS=darwin GOARCH=arm64 go build -o tabd-native-host-darwin-arm64 main.go
    
    # Linux
    GOOS=linux GOARCH=amd64 go build -o tabd-native-host-linux-amd64 main.go
    GOOS=linux GOARCH=arm64 go build -o tabd-native-host-linux-arm64 main.go
    GOOS=linux GOARCH=386 go build -o tabd-native-host-linux-386 main.go
    GOOS=linux GOARCH=arm go build -o tabd-native-host-linux-arm main.go
    
    # Windows
    GOOS=windows GOARCH=amd64 go build -o tabd-native-host-windows-amd64.exe main.go
    GOOS=windows GOARCH=386 go build -o tabd-native-host-windows-386.exe main.go
    GOOS=windows GOARCH=arm64 go build -o tabd-native-host-windows-arm64.exe main.go
    
    # FreeBSD
    GOOS=freebsd GOARCH=amd64 go build -o tabd-native-host-freebsd-amd64 main.go
    GOOS=freebsd GOARCH=386 go build -o tabd-native-host-freebsd-386 main.go
    
    # OpenBSD
    GOOS=openbsd GOARCH=amd64 go build -o tabd-native-host-openbsd-amd64 main.go
    GOOS=openbsd GOARCH=386 go build -o tabd-native-host-openbsd-386 main.go
    
    # NetBSD
    GOOS=netbsd GOARCH=amd64 go build -o tabd-native-host-netbsd-amd64 main.go
    GOOS=netbsd GOARCH=386 go build -o tabd-native-host-netbsd-386 main.go
    
    echo "Cross-platform builds complete"
fi

echo "Done!"
