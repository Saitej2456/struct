#!/bin/bash

# --- STYLING ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# --- HELPER FUNCTIONS ---
print_step() { echo -e "${BLUE}>>> $1${NC}"; }
print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warn() { echo -e "${YELLOW}! $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }

# Interactive confirmation prompt
ask_confirm() {
    read -r -p "$(echo -e "${YELLOW}? $1 [Y/n]: ${NC}")" response
    case "$response" in
        [nN][oO]|[nN]) return 1 ;;
        *) return 0 ;; # Default to Yes
    esac
}

# --- PRE-FLIGHT CHECKS ---
if [ "$EUID" -eq 0 ]; then
    print_error "Please do not run this entire script as root or with sudo."
    print_warn "Run it as your normal user. The script will automatically ask for sudo permissions only when necessary."
    exit 1
fi

print_step "Starting <struct/> Interactive Setup Wizard..."

# --- 1. SYSTEM DEPENDENCIES (CGO) ---
if ! command -v gcc &> /dev/null; then
    print_warn "GCC (C Compiler) is missing. This is required for the CGO backend."
    if ask_confirm "Do you want to install build-essential via apt?"; then
        sudo apt-get update
        sudo apt-get install -y build-essential gcc
        print_success "GCC installed."
    else
        print_error "Cannot proceed without a C compiler. Exiting."
        exit 1
    fi
else
    print_success "GCC is already installed."
fi

# --- 2. GO ENVIRONMENT CHECK ---
NEEDS_GO=false
GO_TARGET_VERSION="1.22.1"

if ! command -v go &> /dev/null; then
    print_warn "Go is not installed or not in your PATH."
    NEEDS_GO=true
else
    # Parse Go version (e.g., "go1.18.1" -> "1.18.1" -> "1 18")
    GO_VERSION_STR=$(go version | awk '{print $3}' | sed 's/go//')
    IFS='.' read -ra V_PARTS <<< "$GO_VERSION_STR"
    
    if [ "${V_PARTS[0]}" -lt 1 ] || ( [ "${V_PARTS[0]}" -eq 1 ] && [ "${V_PARTS[1]}" -lt 22 ] ); then
        print_warn "Your Go version ($GO_VERSION_STR) is too old. <struct/> requires Go 1.22+."
        NEEDS_GO=true
    else
        print_success "Modern Go version ($GO_VERSION_STR) detected."
        GO_BIN_PATH=$(which go)
    fi
fi

if [ "$NEEDS_GO" = true ]; then
    if ask_confirm "Do you want to automatically download and install Go $GO_TARGET_VERSION?"; then
        # OS & Architecture detection
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        ARCH=$(uname -m)
        if [ "$ARCH" = "x86_64" ]; then GOARCH="amd64"; elif [ "$ARCH" = "aarch64" ]; then GOARCH="arm64"; else GOARCH=""; fi
        
        if [ -z "$GOARCH" ]; then
            print_error "Unsupported architecture: $ARCH. Please install Go manually."
            exit 1
        fi

        print_step "Removing any existing outdated Go installations..."
        sudo apt-get remove -y golang-go golang-src 2>/dev/null
        sudo rm -rf /usr/local/go

        GO_URL="https://go.dev/dl/go${GO_TARGET_VERSION}.${OS}-${GOARCH}.tar.gz"
        print_step "Downloading Go from $GO_URL ..."
        wget -q -O /tmp/go.tar.gz "$GO_URL"
        
        print_step "Extracting Go to /usr/local..."
        sudo tar -C /usr/local -xzf /tmp/go.tar.gz
        rm /tmp/go.tar.gz

        GO_BIN_PATH="/usr/local/go/bin/go"
        
        # Check Profile path injections
        PROFILE_FILE="$HOME/.bashrc"
        if [[ $SHELL == *"zsh"* ]]; then PROFILE_FILE="$HOME/.zshrc"; fi

        if ! grep -q "/usr/local/go/bin" "$PROFILE_FILE"; then
            if ask_confirm "Add Go to your PATH in $PROFILE_FILE?"; then
                echo -e '\n# Go Environment for struct' >> "$PROFILE_FILE"
                echo 'export PATH=$PATH:/usr/local/go/bin' >> "$PROFILE_FILE"
                print_success "Added to $PROFILE_FILE"
                export PATH=$PATH:/usr/local/go/bin
            fi
        fi
        print_success "Go $GO_TARGET_VERSION installed successfully."
    else
        print_error "Cannot proceed without a modern Go environment. Exiting."
        exit 1
    fi
fi

# --- 3. PROJECT DEPENDENCIES ---
print_step "Setting up project modules..."
if [ -f go.mod ]; then
    if ask_confirm "Existing go.mod found. Do you want to reset it to ensure clean dependencies?"; then
        rm -f go.mod go.sum
        "$GO_BIN_PATH" mod init struct_app
        print_success "go.mod reset."
    fi
else
    "$GO_BIN_PATH" mod init struct_app
fi

print_step "Fetching libraries..."
"$GO_BIN_PATH" get github.com/charmbracelet/bubbletea
"$GO_BIN_PATH" get github.com/charmbracelet/lipgloss
"$GO_BIN_PATH" get github.com/charmbracelet/bubbles/textinput
"$GO_BIN_PATH" get gopkg.in/yaml.v3
"$GO_BIN_PATH" get github.com/psanford/wormhole-william/wormhole
"$GO_BIN_PATH" mod tidy
print_success "Libraries downloaded."

# --- 4. COMPILATION ---
print_step "Compiling <struct/>..."
# CGO_ENABLED is explicitly set to link backend.c
CGO_ENABLED=1 "$GO_BIN_PATH" build -o struct_cli

if [ $? -eq 0 ]; then
    print_success "Compilation successful."
    
    if ask_confirm "Install 'struct' globally to /usr/local/bin? (Requires sudo)"; then
        sudo mv struct_cli /usr/local/bin/struct
        sudo chmod +x /usr/local/bin/struct
        mkdir -p ~/.struct/structures
        print_success "<struct/> is fully installed!"
        echo -e "\n${GREEN}To get started, simply type: ${BLUE}struct${NC}"
        
        # Reminder for path updates
        if [ "$NEEDS_GO" = true ]; then
            echo -e "${YELLOW}Note: You may need to run 'source ~/.bashrc' or restart your terminal first for the Go paths to update.${NC}"
        fi
    else
        print_success "Skipped global installation. You can run it locally with: ./struct_cli"
    fi
else
    print_error "Build failed. Please check the compiler output above."
    exit 1
fi