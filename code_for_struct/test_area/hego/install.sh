#!/bin/bash
echo ">>> Starting <struct/> Compilation & Installation..."

# Ensure the local structures directory exists
mkdir -p ~/.struct/structures

# Initialize the Go module if it doesn't already exist
if [ ! -f go.mod ]; then
    echo ">>> Initializing Go module..."
    go mod init struct_app
fi

echo ">>> Fetching required Go libraries..."
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles/textinput
go get gopkg.in/yaml.v3
go get github.com/psanford/wormhole-william/wormhole
go mod tidy

echo ">>> Compiling the application (Linking C and Go)..."
# Build the application. CGO_ENABLED=1 is required for backend.c to compile
CGO_ENABLED=1 go build -o struct_cli

if [ $? -eq 0 ]; then
    echo ">>> Build successful! Installing to /usr/local/bin..."
    sudo mv struct_cli /usr/local/bin/struct
    sudo chmod +x /usr/local/bin/struct
    echo ">>> Installation complete! You can now run the app by typing: struct"
else
    echo ">>> Build failed. Please check the compiler errors above."
    exit 1
fi