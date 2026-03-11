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
else#!/bin/bash
echo ">>> Starting pre-installation system setup..."

# 1. Update package lists and install C compiler / build tools
sudo apt-get update
sudo apt-get install -y build-essential gcc curl wget git

# 2. Check for Go installation. If missing, download and install it.
if ! command -v go &> /dev/null
then
    echo ">>> Go is not installed. Fetching the official Go binary..."
    wget https://go.dev/dl/go1.22.1.linux-amd64.tar.gz
    
    echo ">>> Extracting Go to /usr/local..."
    sudo tar -C /usr/local -xzf go1.22.1.linux-amd64.tar.gz
    rm go1.22.1.linux-amd64.tar.gz
    
    # Add Go to the environment paths
    echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.profile
    echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.bashrc
    export PATH=$PATH:/usr/local/go/bin
    
    echo ">>> Go installed successfully."
else
    echo ">>> Go is already installed: $(go version)"
fi

echo ""
echo ">>> Pre-installation complete!"
echo ">>> IMPORTANT: Please run 'source ~/.bashrc' or restart your terminal before running install.sh."
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