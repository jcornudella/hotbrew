#!/bin/bash
# hotbrew installer - One command to brew them all
set -e

MODE="remote"
if [[ "$1" == "--local" ]]; then
    MODE="local"
fi

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

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) ARCH="amd64" ;;
esac

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "$SCRIPT_DIR/.." && pwd)

ensure_go() {
    if command -v go >/dev/null 2>&1; then
        return
    fi

    echo "→ Go not found. Installing..."
    case "$OS" in
        darwin)
            if command -v brew >/dev/null 2>&1; then
                brew install go
            else
                echo "Homebrew not detected. Install Go manually: https://go.dev/dl/"
                exit 1
            fi
            ;;
        linux)
            if command -v apt-get >/dev/null 2>&1; then
                sudo apt-get update && sudo apt-get install -y golang
            elif command -v yum >/dev/null 2>&1; then
                sudo yum install -y golang
            elif command -v pacman >/dev/null 2>&1; then
                sudo pacman -Sy --noconfirm go
            else
                echo "Unsupported package manager. Install Go manually: https://go.dev/dl/"
                exit 1
            fi
            ;;
        *)
            echo "Unsupported OS. Install Go manually: https://go.dev/dl/"
            exit 1
            ;;
    esac

    if ! command -v go >/dev/null 2>&1; then
        echo "Go installation failed. Please install from https://go.dev/dl/"
        exit 1
    fi
}

ensure_go

if [[ "$MODE" == "local" ]]; then
    if [[ ! -f "$REPO_ROOT/go.mod" ]]; then
        echo "--local must be run from the cloned hotbrew repo"
        exit 1
    fi
fi

INSTALL_DIR="${HOME}/.local/bin"
mkdir -p "$INSTALL_DIR"

echo "→ Downloading hotbrew..."
if [[ "$MODE" == "local" ]]; then
    (cd "$REPO_ROOT" && go install ./cmd/hotbrew)
else
    go install github.com/jcornudella/hotbrew/cmd/hotbrew@latest
fi
INSTALL_DIR="$(go env GOPATH)/bin"

echo "→ Installed to $INSTALL_DIR/hotbrew"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "→ Add to your PATH by adding this to your shell rc:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
read -p "→ Enter your email for updates (optional, press Enter to skip): " EMAIL

json_escape() {
    local s="$1"
    s="${s//\\/\\\\}"
    s="${s//\"/\\\"}"
    s="${s//$'\n'/\\n}"
    s="${s//$'\r'/\\r}"
    s="${s//$'\t'/\\t}"
    printf '%s' "$s"
}

if [ -n "$EMAIL" ]; then
    if ! [[ "$EMAIL" =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
        echo "Invalid email format. Skipping subscription."
    else
        echo "→ Subscribing..."
        ESCAPED_EMAIL=$(json_escape "$EMAIL")
        PAYLOAD="{\"email\":\"$ESCAPED_EMAIL\"}"
        RESPONSE=$(curl -s -X POST https://hotbrew.dev/api/subscribe \
            -H "Content-Type: application/json" \
            --data-binary "$PAYLOAD" 2>/dev/null || echo '{"token":"local"}')

        TOKEN=$(echo "$RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

        if [ -n "$TOKEN" ] && [ "$TOKEN" != "local" ]; then
            mkdir -p -m 700 ~/.config/hotbrew
            OLD_UMASK=$(umask)
            umask 077
            printf '%s' "$TOKEN" > ~/.config/hotbrew/token
            umask "$OLD_UMASK"
            echo "→ Subscribed! Token saved."
        fi
    fi
fi

echo ""
echo "→ Setting up shell integration..."
SHELL_NAME=$(basename "$SHELL")
RC_FILE=""
case $SHELL_NAME in
    zsh)  RC_FILE="$HOME/.zshrc" ;;
    bash) RC_FILE="$HOME/.bashrc" ;;
    fish) RC_FILE="$HOME/.config/fish/config.fish" ;;
    *)    RC_FILE="" ;;
esac

if [ -n "$RC_FILE" ]; then
    read -p "→ Add hotbrew to $RC_FILE? [Y/n]: " ADD_TO_RC
    ADD_TO_RC=${ADD_TO_RC:-Y}
    if [[ $ADD_TO_RC =~ ^[Yy]$ ]]; then
        {
            echo ""
            echo "# hotbrew - Your morning, piping hot"
            echo "command -v hotbrew &>/dev/null && hotbrew"
        } >> "$RC_FILE"
        echo "→ Added to $RC_FILE"
    fi
fi

echo ""
echo -e "${BOLD}${PINK}☕ hotbrew is ready!${RESET}"
echo ""
echo "Run ${CYAN}hotbrew${RESET} to see your morning digest."
echo ""
