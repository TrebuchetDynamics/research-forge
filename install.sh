#!/usr/bin/env bash
set -euo pipefail

go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest

echo "rforge installed. Run: rforge --version"
