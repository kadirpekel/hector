#!/bin/bash
# Hector Uninstallation Script

set -e

echo "🗑️  Uninstalling Hector..."
echo ""

# Check common install locations
LOCATIONS=(
    "/usr/local/bin/hector"
    "$HOME/.local/bin/hector"
    "$HOME/bin/hector"
)

REMOVED=false

for location in "${LOCATIONS[@]}"; do
    if [ -f "$location" ]; then
        echo "Removing: $location"
        rm -f "$location"
        REMOVED=true
    fi
done

if [ "$REMOVED" = true ]; then
    echo ""
    echo "✅ Hector uninstalled successfully!"
else
    echo "❌ Hector not found in standard locations"
    echo ""
    echo "Checked:"
    for location in "${LOCATIONS[@]}"; do
        echo "  - $location"
    done
fi

