#!/usr/bin/env bash

set -euo pipefail

mkdir -p ~/.config/puff/bin
chmod -R 750 ~/.config/puff
curl -sSL https://github.com/pgulb/puff/releases/latest/download/puff -o ~/.config/puff/bin/puff
chmod +x ~/.config/puff/bin/puff

echo "export PATH=\$PATH:${HOME}/.config/puff/bin" >> ~/.bashrc
echo "export PATH=\$PATH:${HOME}/.config/puff/bin" >> ~/.zshrc
echo "Puff CLI installed. Restart your terminal or run 'source ~/.bashrc' or 'source ~/.zshrc'."
