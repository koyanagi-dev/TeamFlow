#!/bin/bash
set -e

echo "Setting up pre-commit hooks..."

# Check if pre-commit is installed
if ! command -v pre-commit >/dev/null 2>&1; then
    echo "Installing pre-commit..."
    if command -v brew >/dev/null 2>&1; then
        brew install pre-commit
    elif command -v pip3 >/dev/null 2>&1; then
        pip3 install pre-commit
    else
        echo "Error: Cannot install pre-commit. Please install manually."
        exit 1
    fi
fi

# Install pre-commit hooks
pre-commit install

echo "✓ Pre-commit hooks installed successfully"
echo ""
echo "To run manually: pre-commit run --all-files"
echo "To bypass: git commit --no-verify (use only when necessary)"
