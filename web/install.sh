
#!/bin/bash
# hotbrew installer - One command to brew them all
set -e

MODE="remote"
if [[ "$1" == "--local" ]]; then
    MODE="local"
fi

BOLD='[1m'
PINK='[38;5;205m'
CYAN='[38;5;117m'
RESET='[0m'

printf "
${PINK}    ) )${RESET}
"
printf "${PINK}   ( (${RESET}
"
printf "${PINK}    ) )${RESET}
"
printf "${CYAN}   ______${RESET}
"
printf "${CYAN}  |      |]${RESET}
"
printf "${CYAN}  |      |${RESET}
"
printf "${CYAN}   \____/${RESET}

"
printf "${BOLD}☕ hotbrew installer${RESET}

"

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

echo "→ Downloading hotbrew..."
if [[ "$MODE" == "local" ]]; then
    (cd "$REPO_ROOT" && go install ./cmd/hotbrew)
else
    go install github.com/jcornudella/hotbrew/cmd/hotbrew@latest
fi
INSTALL_DIR="$(go env GOPATH)/bin"
echo "→ Installed to $INSTALL_DIR/hotbrew"

SHELL_NAME=$(basename "$SHELL")
RC_FILE=""
case $SHELL_NAME in
    zsh)  RC_FILE="$HOME/.zshrc" ;;
    bash) RC_FILE="$HOME/.bashrc" ;;
    fish) RC_FILE="$HOME/.config/fish/config.fish" ;;
    *)    RC_FILE="" ;;
esac

ADDED_PATH=false
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    if [[ -n "$RC_FILE" ]]; then
        if ! grep -q "$INSTALL_DIR" "$RC_FILE" 2>/dev/null; then
            printf '
# hotbrew binary
export PATH="\$PATH:%s"
' "$INSTALL_DIR" >> "$RC_FILE"
            echo "→ Added $INSTALL_DIR to PATH via $RC_FILE"
            ADDED_PATH=true
        fi
    else
        echo "→ Add to your PATH by adding this to your shell rc:"
        echo "  export PATH="\$PATH:$INSTALL_DIR""
    fi
fi

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
        PAYLOAD="{"email":"$ESCAPED_EMAIL"}"
        RESPONSE=$(curl -s -X POST https://hotbrew.dev/api/subscribe             -H "Content-Type: application/json"             --data-binary "$PAYLOAD" 2>/dev/null || echo '{"token":"local"}')

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
if [[ -n "$RC_FILE" ]]; then
    read -p "→ Add hotbrew autorun to $RC_FILE? [Y/n]: " ADD_TO_RC
    ADD_TO_RC=${ADD_TO_RC:-Y}
    if [[ $ADD_TO_RC =~ ^[Yy]$ ]]; then
        cat <<'EOAUTORUN' >> "$RC_FILE"

# hotbrew - Your morning, piping hot
command -v hotbrew &>/dev/null && hotbrew
EOAUTORUN
        echo "→ Added autorun snippet to $RC_FILE"
    fi
fi

echo ""
echo -e "${BOLD}${PINK}☕ hotbrew is ready!${RESET}"
echo ""
if [[ "$ADDED_PATH" == true ]]; then
    echo "Run 'source $RC_FILE' or restart your terminal to use hotbrew immediately."
fi
echo "Run ${CYAN}hotbrew${RESET} to see your morning digest."
echo ""
