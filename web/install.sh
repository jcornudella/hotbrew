#!/bin/bash
# hotbrew installer - One command to brew them all
set -e

BOLD='\033[1m'
PINK='\033[38;5;205m'
CYAN='\033[38;5;117m'
RESET='\033[0m'

echo ""
echo -e "${PINK}    ) )${RESET}"
echo -e "${PINK}   ( (${RESET}"
echo -e "${PINK}    ) )${RESET}"
echo -e "${CYAN}   ______${RESET}"
echo -e "${CYAN}  |      |]${RESET}"
echo -e "${CYAN}  |      |${RESET}"
echo -e "${CYAN}   \\____/${RESET}"
echo ""
echo -e "${BOLD}☕ hotbrew installer${RESET}"
echo ""

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

# Install location
INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "$INSTALL_DIR"

echo "→ Downloading hotbrew..."

# For now, build from source (later: download binary)
if command -v go &> /dev/null; then
    go install github.com/jcornudella/hotbrew/cmd/hotbrew@latest
    INSTALL_DIR="$(go env GOPATH)/bin"
else
    echo "Go not found. Installing via binary..."
    # TODO: Add binary download URL
    echo "Please install Go first: https://go.dev/dl/"
    exit 1
fi

echo "→ Installed to $INSTALL_DIR/hotbrew"

# Check if in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "→ Add to your PATH by adding this to your shell rc:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

# Prompt for email (optional)
echo ""
read -p "→ Enter your email for updates (optional, press Enter to skip): " EMAIL

# Subscribe
if [ -n "$EMAIL" ]; then
    echo "→ Subscribing..."
    RESPONSE=$(curl -s -X POST https://hotbrew.dev/api/subscribe \
        -H "Content-Type: application/json" \
        -d "{\"email\": \"$EMAIL\"}" 2>/dev/null || echo '{"token":"local"}')

    TOKEN=$(echo $RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

    if [ -n "$TOKEN" ] && [ "$TOKEN" != "local" ]; then
        mkdir -p ~/.config/hotbrew
        echo "$TOKEN" > ~/.config/hotbrew/token
        echo "→ Subscribed! Token saved."
    fi
fi

# Setup shell
echo ""
echo "→ Setting up shell integration..."

SHELL_NAME=$(basename "$SHELL")
RC_FILE=""

case $SHELL_NAME in
    zsh)  RC_FILE="$HOME/.zshrc" ;;
    bash) RC_FILE="$HOME/.bashrc" ;;
    fish) RC_FILE="$HOME/.config/fish/config.fish" ;;
esac

if [ -n "$RC_FILE" ]; then
    read -p "→ Add hotbrew to $RC_FILE? [Y/n]: " ADD_TO_RC
    ADD_TO_RC=${ADD_TO_RC:-Y}

    if [[ $ADD_TO_RC =~ ^[Yy]$ ]]; then
        echo "" >> "$RC_FILE"
        echo "# hotbrew - Your morning, piping hot" >> "$RC_FILE"
        echo "command -v hotbrew &>/dev/null && hotbrew" >> "$RC_FILE"
        echo "→ Added to $RC_FILE"
    fi
fi

echo ""
echo -e "${BOLD}${PINK}☕ hotbrew is ready!${RESET}"
echo ""
echo "Run ${CYAN}hotbrew${RESET} to see your morning digest."
echo ""
