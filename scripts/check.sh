#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
python3 -m unittest discover -s mock -p 'test_*.py' -v
for component in gateway services/usuarios services/electrodomesticos services/consumo services/solar; do
 (cd "$component" && go test ./...)
done
