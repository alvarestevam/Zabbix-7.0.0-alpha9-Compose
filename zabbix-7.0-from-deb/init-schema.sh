#!/bin/bash
set -e

: "${DB_HOST:?Missing DB_HOST}"
: "${DB_NAME:?Missing DB_NAME}"
: "${DB_USER:?Missing DB_USER}"
: "${DB_PASSWORD:?Missing DB_PASSWORD}"

echo "[INIT] Esperando banco de dados..."
until mysql -h"${DB_HOST}" -u"${DB_USER}" -p"${DB_PASSWORD}" -e "SELECT 1;" "${DB_NAME}" &>/dev/null; do
  sleep 2
done

echo "[INIT] Importando schema..."
mysql -h"${DB_HOST}" -u"${DB_USER}" -p"${DB_PASSWORD}" "${DB_NAME}" < /sql/schema.sql
mysql -h"${DB_HOST}" -u"${DB_USER}" -p"${DB_PASSWORD}" "${DB_NAME}" < /sql/images.sql
mysql -h"${DB_HOST}" -u"${DB_USER}" -p"${DB_PASSWORD}" "${DB_NAME}" < /sql/data.sql

echo "[INIT] Iniciando Zabbix Server..."
exec /usr/sbin/zabbix_server -f
