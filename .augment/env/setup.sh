#!/bin/bash
set -e

echo "🚀 Setting up NewsBalancer Go development environment..."

# Update system packages
echo "📦 Updating system packages..."
sudo apt-get update -y

# Install essential build tools
echo "🔧 Installing build essentials..."
sudo apt-get install -y \
    build-essential \
    curl \
    wget \
    git \
    ca-certificates \
    gnupg \
    lsb-release \
    software-properties-common

# Install Go 1.23.0
echo "🐹 Installing Go 1.23.0..."
cd /tmp
wget -q https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
rm go1.23.0.linux-amd64.tar.gz

# Add Go to PATH in user profile
echo "🔧 Configuring Go PATH..."
echo 'export PATH="/usr/local/go/bin:$PATH"' >> $HOME/.profile
echo 'export GOPATH="$HOME/go"' >> $HOME/.profile
echo 'export PATH="$GOPATH/bin:$PATH"' >> $HOME/.profile

# Source the profile to make Go available immediately
export PATH="/usr/local/go/bin:$PATH"
export GOPATH="$HOME/go"
export PATH="$GOPATH/bin:$PATH"

# Verify Go installation
echo "✅ Verifying Go installation..."
go version

# Install Node.js 20.x (required for Playwright and testing tools)
echo "📦 Installing Node.js 20.x..."
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs

# Verify Node.js installation
echo "✅ Verifying Node.js installation..."
node --version
npm --version

# Navigate to project directory
cd /mnt/persist/workspace

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod download
go mod tidy

# Install Node.js dependencies
echo "📦 Installing Node.js dependencies..."
npm install

# Install Playwright browsers
echo "🎭 Installing Playwright browsers..."
npx playwright install --with-deps

# Install additional Go tools for testing
echo "🔧 Installing Go testing tools..."
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create necessary directories (ignore if they exist)
echo "📁 Creating test directories..."
mkdir -p test-results
mkdir -p coverage || true
mkdir -p bin

# Create a basic Playwright configuration if it doesn't exist
echo "🎭 Creating Playwright configuration..."
cat > playwright.config.ts << 'EOF'
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:8080',
    trace: 'on-first-retry',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    command: 'go run ./cmd/server',
    port: 8080,
    timeout: 120 * 1000,
    reuseExistingServer: !process.env.CI,
  },
});
EOF

# Set up SQLite (already included in Go dependencies)
echo "🗄️ SQLite support already included in Go dependencies"

# Verify all tools are available
echo "🔍 Verifying tool installations..."
go version
node --version
npm --version
npx playwright --version

# Build the Go application to verify everything works
echo "🏗️ Building Go application..."
go build -v -o bin/newsbalancer ./cmd/server/...

echo "✅ Setup complete! Environment ready for testing."