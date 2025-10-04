#!/bin/bash
# Hector Installation Script

set -e

echo "🚀 Installing Hector..."
echo ""

# Build the binary
echo "📦 Building Hector..."
go build -o hector cmd/hector/main.go

# Determine install location
if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
elif [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
else
    INSTALL_DIR="$HOME/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "📁 Installing to: $INSTALL_DIR"

# Copy binary
cp hector "$INSTALL_DIR/hector"
chmod +x "$INSTALL_DIR/hector"

echo ""
echo "✅ Hector installed successfully!"
echo ""
echo "Usage:"
echo "  hector              # Default config"
echo "  hector coding       # Developer mode"
echo "  hector --help       # Show help"
echo ""

# Check if directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "⚠️  Note: $INSTALL_DIR is not in your PATH"
    echo ""
    echo "Add this to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

echo "🎉 Ready to use: hector coding"

