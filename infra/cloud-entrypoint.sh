#!/bin/sh
set -eu
ADDR=127.0.0.1:8081 /consumo &
consumo_pid=$!
ADDR=0.0.0.0:${PORT:-10000} /gateway &
gateway_pid=$!
trap 'kill "$consumo_pid" "$gateway_pid" 2>/dev/null || true; wait || true' TERM INT EXIT
# BusyBox wait -n finaliza si cualquiera de los procesos deja de servir.
wait -n
exit 1
