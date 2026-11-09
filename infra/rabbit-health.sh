#!/bin/sh
set -eu
# Evita arrancar otra VM Erlang en cada chequeo y no imprime credenciales.
authorization=$(printf '%s:%s' "$RABBITMQ_DEFAULT_USER" "$RABBITMQ_DEFAULT_PASS" | base64 | tr -d '\n')
for endpoint in ready-to-serve-clients local-alarms; do
  wget -q -T 3 -O /dev/null --header="Authorization: Basic $authorization" "http://127.0.0.1:15672/api/health/checks/$endpoint" >/dev/null 2>&1 || exit 1
done
