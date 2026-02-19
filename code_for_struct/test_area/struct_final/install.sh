#!/bin/bash

if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root (use sudo ./install.sh)"
  exit 1
fi

export PATH=$PATH:/usr/local/go/bin

echo ">>> Starting Struct Installation..."

# Find the exact absolute path to the go binary
GO_BIN=$(which go)

REAL_USER=$SUDO_USER
if [ -z "$REAL_USER" ]; then REAL_USER=$(whoami); fi
USER_HOME=$(getent passwd $REAL_USER | cut -d: -f6)

STRUCT_DIR="$USER_HOME/.struct/structures"

if [ ! -d "$STRUCT_DIR" ]; then
    echo "Creating directory: $STRUCT_DIR"
    mkdir -p "$STRUCT_DIR"
    chown -R $REAL_USER:$REAL_USER "$USER_HOME/.struct"
else
    echo "✅ Structure directory exists."
fi

echo "Compiling the application..."
sudo -u $REAL_USER $GO_BIN build -o struct .

if [ $? -ne 0 ]; then
    echo "❌ Compilation failed! Please check the error output above."
    exit 1
fi

echo "Installing 'struct' command to /bin/..."
mv struct /bin/

if [ $? -eq 0 ]; then
    echo ">>> 🎉 Installation Complete!"
    echo "You can now type 'struct' from anywhere in your terminal."
else
    echo "❌ Failed to install. Check your permissions."
fi