#!/usr/bin/env bash
set -euo pipefail

OUT=dist
BINNAME=0xrootshell.exe

echo "Cleaning old dist..."
rm -rf $OUT
mkdir -p $OUT

echo "Tidy modules..."
go mod tidy

echo "Building windows/amd64 binary..."
env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o $OUT/$BINNAME ./cmd/rootsh

echo "Built: $OUT/$BINNAME"
