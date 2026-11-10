#!/bin/sh
set -eu
export COMPOSE_BAKE="${COMPOSE_BAKE:-false}"
export COMPOSE_PARALLEL_LIMIT="${COMPOSE_PARALLEL_LIMIT:-1}"
cd "$(dirname "$0")/.."
if [ ! -f .env ]; then
  umask 077
  for key in HEALTHCHECK_TOKEN INTERNAL_TOKEN M2M_API_KEY M2M_DENIED_KEY MYSQL_ROOT_PASSWORD MYSQL_PASSWORD; do
    value=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
    printf '%s=%s\n' "$key" "$value" >> .env
  done
  printf '%s\n' 'Credenciales locales generadas en .env (no se muestran ni se versionan).'
fi
if ! grep -q '^PANEL_PROVIDER_API_KEY=' .env; then
  umask 077
  value=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
  printf '\nPANEL_PROVIDER_API_KEY=%s\n' "$value" >> .env
fi
for key in USUARIOS_DB_PASSWORD USUARIOS_DB_ROOT_PASSWORD CONSUMO_DB_PASSWORD CONSUMO_DB_ROOT_PASSWORD JWT_SECRET ADMIN_PASSWORD MONGO_ROOT_PASSWORD MONGO_SOLAR_PASSWORD RABBITMQ_PASSWORD VALKEY_PASSWORD SEARCH_API_KEY GRAFANA_ADMIN_PASSWORD; do
  if ! grep -q "^${key}=" .env; then
    umask 077
    value=$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')
    printf '\n%s=%s\n' "$key" "$value" >> .env
  fi
done
umask 077
mkdir -p .runtime
sed -n 's/^HEALTHCHECK_TOKEN=//p' .env | tr -d '\r\n' > .runtime/metrics-token
if ! grep -q '^RABBITMQ_HOSTNAME=' .env; then
  broker_id=$(docker compose ps --all --quiet rabbitmq)
  broker_hostname=renewable-broker
  if [ -n "$broker_id" ]; then
    broker_hostname=$(docker inspect --format '{{.Config.Hostname}}' "$broker_id")
    case "$broker_hostname" in ''|*[!a-zA-Z0-9.-]*) echo 'Identidad del broker inválida' >&2; exit 1;; esac
  fi
  printf '\nRABBITMQ_HOSTNAME=%s\n' "$broker_hostname" >> .env
fi
if [ "${1:-}" = "build" ] || { [ "${1:-}" = "up" ] && printf '%s\n' "$@" | grep -qx -- '--build'; }; then
  for service in consumo electrodomesticos usuarios solar solar-2 proveedor-paneles-sim gateway; do
    docker compose build "$service"
  done
  if [ "$1" = "build" ]; then exit 0; fi
  for arg in "$@"; do shift; if [ "$arg" != '--build' ]; then set -- "$@" "$arg"; fi; done
fi
exec docker compose "$@"
