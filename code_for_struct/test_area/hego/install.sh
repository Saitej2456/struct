# #!/bin/bash

# echo ">>> Starting Struct Installation..."

# # 1. Create the hidden home directory structure
# STRUCT_HOME="$HOME/.struct/structures"
# if [ ! -d "$STRUCT_HOME" ]; then
#     echo "Creating directory: $STRUCT_HOME"
#     mkdir -p "$STRUCT_HOME"
# else
#     echo "Directory exists: $STRUCT_HOME"
# fi

# # 2. Compile the application
# echo "Compiling..."
# # Ensure Go tracks the dependencies
# if [ ! -f "go.mod" ]; then
#     go mod init struct_app
#     go get github.com/charmbracelet/bubbletea
#     go get github.com/charmbracelet/lipgloss
#     go get github.com/charmbracelet/bubbles/textinput
# fi

# go build -o struct .
# if [ $? -ne 0 ]; then
#     echo "Compilation failed! Check for errors."
#     exit 1
# fi
# echo "Compilation successful."

# # 3. Move executable to /bin
# # We use /bin as requested, but we need sudo permissions.
# echo "Installing 'struct' command to /bin/ (requires sudo)..."
# sudo mv struct /bin/

# if [ $? -eq 0 ]; then
#     echo ">>> Installation Complete!"
#     echo "You can now type 'struct' from anywhere in your terminal."
# else
#     echo "Failed to move the file. Installation incomplete."
# fi











#!/bin/bash

# Ensure the script is run as root (needed to move files to /bin)
if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root (use sudo ./install.sh)"
  exit 1
fi

echo ">>> Starting Struct Installation..."

# 1. Determine the Real User's Home Directory
# (Because $HOME usually points to /root when using sudo)
REAL_USER=$SUDO_USER
if [ -z "$REAL_USER" ]; then
  REAL_USER=$(whoami)
fi
USER_HOME=$(getent passwd $REAL_USER | cut -d: -f6)

STRUCT_DIR="$USER_HOME/.struct/structures"

# 2. Create the hidden directory structure
if [ ! -d "$STRUCT_DIR" ]; then
    echo "Creating directory: $STRUCT_DIR"
    mkdir -p "$STRUCT_DIR"
    # Fix ownership so the regular user can access it
    chown -R $REAL_USER:$REAL_USER "$USER_HOME/.struct"
else
    echo "Structure directory exists."
fi

# 3. Compile the application
echo "Compiling..."
# We run go build as the real user to keep the build artifacts clean
sudo -u $REAL_USER go build -o struct .

if [ $? -ne 0 ]; then
    echo "Compilation failed! Did you run ./pre-installer.sh?"
    exit 1
fi
echo "Compilation successful."

# 4. Install to /bin
echo "Installing 'struct' command to /bin/..."
mv struct /bin/

if [ $? -eq 0 ]; then
    echo ">>> Installation Complete!"
    echo "You can now type 'struct' from anywhere in your terminal."
else
    echo "Failed to install. Check your permissions."
fi