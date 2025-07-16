#!/bin/bash
set -e

echo "Setting up Go development environment..."

# Update package lists
sudo apt-get update

# Install required system packages
sudo apt-get install -y wget curl git build-essential

# Install Go 1.23.0
GO_VERSION="1.23.0"
GO_TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TARBALL}"

echo "Downloading Go ${GO_VERSION}..."
wget -q "${GO_URL}" -O "/tmp/${GO_TARBALL}"

echo "Installing Go..."
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "/tmp/${GO_TARBALL}"

# Add Go to PATH in user's profile
echo 'export PATH="/usr/local/go/bin:$PATH"' >> "$HOME/.profile"
echo 'export GOPATH="$HOME/go"' >> "$HOME/.profile"
echo 'export PATH="$GOPATH/bin:$PATH"' >> "$HOME/.profile"

# Set environment for current session
export PATH="/usr/local/go/bin:$PATH"
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"

# Verify Go installation
go version

# Navigate to workspace
cd /mnt/persist/workspace

# Download Go dependencies
echo "Downloading Go dependencies..."
go mod download

# Verify dependencies are available
go mod verify

# Set environment variables for testing
export NO_AUTO_ANALYZE=true
export NO_DOCKER=true
export ENABLE_RACE_DETECTION=false
export CGO_ENABLED=0

echo "Go environment setup complete!"
echo "Go version: $(go version)"
echo "GOPATH: $GOPATH"
echo "Workspace: $(pwd)"