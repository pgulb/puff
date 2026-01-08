#!/usr/bin/env bash

set -euo pipefail

mkdir -p ~/.config/puff/bin
chmod -R 750 ~/.config/puff
curl -sSL https://github.com/pgulb/puff/releases/latest/download/puff -o ~/.config/puff/bin/puff
chmod +x ~/.config/puff/bin/puff

PUFF_PATH="export PATH=\$PATH:${HOME}/.config/puff/bin"

# Check and update .bashrc
if [[ -f ~/.bashrc ]]; then
    if ! grep -q "${HOME}/.config/puff/bin" ~/.bashrc; then
        echo "$PUFF_PATH" >> ~/.bashrc
        echo ".bashrc updated."
    else
        echo "PATH already set in .bashrc."
    fi
else
    echo ".bashrc not found, skipping."
fi

# Check and update .zshrc
if [[ -f ~/.zshrc ]]; then
    if ! grep -q "${HOME}/.config/puff/bin" ~/.zshrc; then
        echo "$PUFF_PATH" >> ~/.zshrc
        echo ".zshrc updated."
    else
        echo "PATH already set in .zshrc."
    fi
else
    echo ".zshrc not found, skipping."
fi

echo "Puff CLI installed. Restart your terminal or run 'source ~/.bashrc' or 'source ~/.zshrc'."
