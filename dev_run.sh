#!/usr/bin/env bash
set -euo pipefail

export DEBUG=1
echo "Starting 0xRootShell (DEBUG=1)..."
go run ./cmd/rootsh
