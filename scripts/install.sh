#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        arm64) ARCH="arm64" ;;
        *) 
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    echo "${OS}-${ARCH}"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to download file with progress
download_file() {
    local url="$1"
    local output="$2"
    
    if command_exists curl; then
        curl -L -o "$output" "$url"
    elif command_exists wget; then
        wget -O "$output" "$url"
    else
        print_error "Neither curl nor wget is installed. Please install one of them."
        exit 1
    fi
}

# Function to verify checksum
verify_checksum() {
    local file="$1"
    local expected_checksum="$2"
    
    if command_exists sha256sum; then
        local actual_checksum=$(sha256sum "$file" | cut -d' ' -f1)
    elif command_exists shasum; then
        local actual_checksum=$(shasum -a 256 "$file" | cut -d' ' -f1)
    else
        print_warning "Could not verify checksum - checksum tools not available"
        return 0
    fi
    
    if [ "$actual_checksum" = "$expected_checksum" ]; then
        print_success "Checksum verification passed"
        return 0
    else
        print_error "Checksum verification failed"
        return 1
    fi
}

# Main installation function
main() {
    print_status "Starting MCPHost installation..."
    
    # Detect platform
    PLATFORM=$(detect_platform)
    print_status "Detected platform: $PLATFORM"
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        print_warning "Running as root. This is not recommended for security reasons."
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installation cancelled."
            exit 1
        fi
    fi
    
    # Get latest version
    print_status "Fetching latest version..."
    LATEST_VERSION=$(curl -s https://api.github.com/repos/your-username/mcphost/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$LATEST_VERSION" ]; then
        print_error "Could not fetch latest version. Using default version."
        LATEST_VERSION="v1.0.0"
    fi
    
    print_status "Latest version: $LATEST_VERSION"
    
    # Set download URL
    DOWNLOAD_URL="https://github.com/your-username/mcphost/releases/download/${LATEST_VERSION}/mcphost-${PLATFORM}"
    
    if [ "$OS" = "windows" ]; then
        DOWNLOAD_URL="${DOWNLOAD_URL}.exe"
    fi
    
    print_status "Download URL: $DOWNLOAD_URL"
    
    # Create temporary directory
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"
    
    # Download binary
    print_status "Downloading MCPHost binary..."
    download_file "$DOWNLOAD_URL" "mcphost"
    
    # Make executable
    chmod +x mcphost
    
    # Test the binary
    print_status "Testing binary..."
    if ./mcphost --version >/dev/null 2>&1; then
        print_success "Binary test passed"
    else
        print_error "Binary test failed"
        exit 1
    fi
    
    # Determine installation directory
    if [ "$EUID" -eq 0 ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="$HOME/.local/bin"
        # Create directory if it doesn't exist
        mkdir -p "$INSTALL_DIR"
    fi
    
    # Check if mcphost is already installed
    if [ -f "$INSTALL_DIR/mcphost" ]; then
        print_warning "MCPHost is already installed at $INSTALL_DIR/mcphost"
        read -p "Overwrite existing installation? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installation cancelled."
            exit 1
        fi
    fi
    
    # Install binary
    print_status "Installing to $INSTALL_DIR..."
    if [ "$EUID" -eq 0 ]; then
        cp mcphost "$INSTALL_DIR/"
    else
        cp mcphost "$INSTALL_DIR/"
    fi
    
    # Verify installation
    if [ -f "$INSTALL_DIR/mcphost" ]; then
        print_success "MCPHost installed successfully!"
    else
        print_error "Installation failed"
        exit 1
    fi
    
    # Add to PATH if not already there
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        print_warning "Adding $INSTALL_DIR to PATH..."
        
        if [ -f "$HOME/.bashrc" ]; then
            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$HOME/.bashrc"
            print_status "Added to ~/.bashrc"
        fi
        
        if [ -f "$HOME/.zshrc" ]; then
            echo "export PATH=\"$INSTALL_DIR:\$PATH\"" >> "$HOME/.zshrc"
            print_status "Added to ~/.zshrc"
        fi
        
        print_warning "Please restart your terminal or run 'source ~/.bashrc' (or ~/.zshrc) to update PATH"
    fi
    
    # Clean up
    cd /
    rm -rf "$TEMP_DIR"
    
    # Display success message
    print_success "MCPHost installation completed!"
    echo
    echo "Next steps:"
    echo "1. Install Ollama: https://ollama.ai"
    echo "2. Pull a model: ollama pull qwen2.5:7b"
    echo "3. Create config file: cp examples/configs/local.json ."
    echo "4. Test installation: mcphost --help"
    echo
    echo "Documentation: https://github.com/your-username/mcphost"
    echo "Issues: https://github.com/your-username/mcphost/issues"
}

# Handle script arguments
case "${1:-}" in
    --version)
        echo "MCPHost Installer v1.0.0"
        exit 0
        ;;
    --help)
        echo "Usage: $0 [OPTIONS]"
        echo
        echo "Options:"
        echo "  --version    Show version"
        echo "  --help       Show this help message"
        echo
        echo "This script installs MCPHost, an AI-powered Artifactory management tool."
        exit 0
        ;;
    "")
        main
        ;;
    *)
        print_error "Unknown option: $1"
        echo "Use --help for usage information"
        exit 1
        ;;
esac
