#!/bin/bash
# DDALAB Launcher macOS Installation Helper
# This script helps users install and configure the DDALAB Launcher on macOS

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LAUNCHER_DIR="$(dirname "$SCRIPT_DIR")"

echo "🚀 DDALAB Launcher macOS Installation Helper"
echo "=============================================="
echo ""

# Check if we're on macOS
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "❌ This script is only for macOS systems."
    exit 1
fi

# Find launcher binary
LAUNCHER_BINARY=""
for file in "$LAUNCHER_DIR"/*.command; do
    if [[ -f "$file" ]]; then
        LAUNCHER_BINARY="$file"
        break
    fi
done

# If no .command file found, look for regular binary
if [[ -z "$LAUNCHER_BINARY" ]]; then
    for file in "$LAUNCHER_DIR"/ddalab-launcher*; do
        if [[ -f "$file" && -x "$file" ]]; then
            LAUNCHER_BINARY="$file"
            break
        fi
    done
fi

if [[ -z "$LAUNCHER_BINARY" ]]; then
    echo "❌ No DDALAB Launcher binary found in $LAUNCHER_DIR"
    echo "   Please ensure you've extracted the release archive properly."
    exit 1
fi

echo "📦 Found launcher binary: $(basename "$LAUNCHER_BINARY")"
echo ""

# Check if quarantine flag is present
if xattr -l "$LAUNCHER_BINARY" 2>/dev/null | grep -q "com.apple.quarantine"; then
    echo "🔒 Quarantine flag detected on launcher binary."
    echo "   This prevents the launcher from running normally."
    echo ""
    echo "🛠️  Removing quarantine flag..."
    
    if sudo xattr -rd com.apple.quarantine "$LAUNCHER_BINARY"; then
        echo "✅ Quarantine flag removed successfully!"
    else
        echo "❌ Failed to remove quarantine flag."
        echo "   You may need to run this manually:"
        echo "   sudo xattr -rd com.apple.quarantine \"$LAUNCHER_BINARY\""
        exit 1
    fi
else
    echo "✅ No quarantine flag detected - binary should run normally."
fi

# Set executable permissions just in case
echo ""
echo "🔧 Ensuring executable permissions..."
chmod +x "$LAUNCHER_BINARY"
echo "✅ Executable permissions set."

echo ""
echo "🎉 Installation complete!"
echo ""
echo "You can now run the launcher in any of these ways:"
echo "   • Double-click: $(basename "$LAUNCHER_BINARY")"
echo "   • Terminal: \"$LAUNCHER_BINARY\""
echo ""
echo "If you encounter any issues, please report them at:"
echo "https://github.com/sdraeger/DDALAB-launcher/issues"