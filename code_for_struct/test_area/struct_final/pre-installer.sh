#!/bin/bash

if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root (use sudo ./pre-installer.sh)"
  exit 1
fi

export PATH=$PATH:/usr/local/go/bin

echo ">>> 1. Checking System Dependencies..."
apt-get update -y

if ! command -v gcc &> /dev/null; then 
    apt-get install -y build-essential
else 
    echo "✅ GCC installed."
fi

if ! command -v go &> /dev/null; then 
    echo "❌ Go compiler not found. Please verify your Go installation."
    exit 1
else 
    echo "✅ Go installed: $(go version)"
fi

# Find the exact absolute path to the go binary
GO_BIN=$(which go)

echo ">>> 2. Setting up Project Dependencies..."
REAL_USER=$SUDO_USER
if [ -z "$REAL_USER" ]; then REAL_USER=$(whoami); fi

if [ ! -f "go.mod" ]; then
    sudo -u $REAL_USER $GO_BIN mod init struct_app
fi

echo "Downloading Go modules..."
sudo -u $REAL_USER $GO_BIN get github.com/charmbracelet/bubbletea@latest
sudo -u $REAL_USER $GO_BIN get github.com/charmbracelet/lipgloss@latest
sudo -u $REAL_USER $GO_BIN get github.com/charmbracelet/bubbles/textinput@latest
sudo -u $REAL_USER $GO_BIN get gopkg.in/yaml.v3@latest

sudo -u $REAL_USER $GO_BIN mod tidy

echo ">>> ✅ Pre-installation complete!"