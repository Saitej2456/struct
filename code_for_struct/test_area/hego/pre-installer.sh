#!/bin/bash

# Ensure the script is run as root
if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root (use sudo ./pre-installer.sh)"
  exit 1
fi

echo ">>> 1. Checking System Dependencies..."

# Update Package Lists
echo "Updating apt..."
apt-get update -y

# --- INSTALL GCC ---
if ! command -v gcc &> /dev/null; then
    echo "GCC not found. Installing build-essential..."
    apt-get install -y build-essential
else
    echo "GCC is already installed."
fi

# --- INSTALL GO ---
if ! command -v go &> /dev/null; then
    echo "Go not found. Installing golang..."
    apt-get install -y golang
else
    echo "Go is already installed."
fi

echo ">>> 2. Setting up Project Dependencies..."

# Detect the real user (who called sudo) to avoid creating root-owned files in your dev folder
REAL_USER=$SUDO_USER
if [ -z "$REAL_USER" ]; then
  REAL_USER=$(whoami)
fi

# Initialize Go Module if missing
if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    # Run as the real user, not root
    sudo -u $REAL_USER go mod init struct_app
fi

# Download the required libraries
echo "Downloading Bubble Tea, Lip Gloss & Text Input..."
sudo -u $REAL_USER go get github.com/charmbracelet/bubbletea
sudo -u $REAL_USER go get github.com/charmbracelet/lipgloss
sudo -u $REAL_USER go get github.com/charmbracelet/bubbles/textinput

echo ">>> Pre-installation complete!"
echo "You can now run: sudo ./install.sh"