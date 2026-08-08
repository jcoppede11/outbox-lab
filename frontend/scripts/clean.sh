#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "Removing dependencies..."
rm -rf node_modules pnpm-lock.yaml .pnpm-store package-lock.json

echo "Removing build output..."
rm -rf dist out-tsc tmp bazel-out

echo "Removing Angular and test artifacts..."
rm -rf .angular/cache coverage __screenshots__

echo "Removing TypeScript build info..."
find . -name "*.tsbuildinfo" -delete 2>/dev/null || true

echo "Clean complete. Run: pnpm install"
