#!/bin/sh
# Genera una credencial efímera sin escribirla en archivos ni imprimirla.
set -eu
cd "$(dirname "$0")/.."
if [ -z "${HEALTHCHECK_TOKEN:-}" ]; then
    HEALTHCHECK_TOKEN=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
fi
if [ "${#HEALTHCHECK_TOKEN}" -lt 32 ]; then
    echo 'HEALTHCHECK_TOKEN debe tener al menos 32 caracteres' >&2
    exit 1
fi
export HEALTHCHECK_TOKEN
exec docker compose "$@"
