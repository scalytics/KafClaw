#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

go test -count=1 -run TestBundledArtifactsPresent ./internal/skills

echo "Bundled skills artifact validation passed."
