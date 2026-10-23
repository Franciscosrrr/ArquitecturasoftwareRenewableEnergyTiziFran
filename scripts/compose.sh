#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ ! -f .env ]; then
  umask 077
  for key in HEALTHCHECK_TOKEN INTERNAL_TOKEN M2M_API_KEY M2M_DENIED_KEY MYSQL_ROOT_PASSWORD MYSQL_PASSWORD; do
    value=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
    printf '%s=%s\n' "$key" "$value" >> .env
  done
  printf '%s\n' 'Credenciales locales generadas en .env (no se muestran ni se versionan).'
fi
exec docker compose "$@"
