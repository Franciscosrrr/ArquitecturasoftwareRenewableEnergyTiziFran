#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
sh scripts/compose.sh up -d --build
for service in verificar-go verificar-real verificar-e2e; do
  sh scripts/compose.sh --profile verificar run --build --rm "$service"
done
