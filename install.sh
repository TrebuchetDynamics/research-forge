#!/usr/bin/env bash
set -euo pipefail

git clone https://github.com/TrebuchetDynamics/research-forge
cd research-forge
go test ./...
go build -o bin/rforge ./cmd/rforge

echo "rforge built at bin/rforge"
