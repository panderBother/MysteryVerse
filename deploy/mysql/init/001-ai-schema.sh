#!/usr/bin/env bash
set -euo pipefail

mysql --protocol=socket -uroot -p"${MYSQL_ROOT_PASSWORD}" <<-EOSQL
CREATE DATABASE IF NOT EXISTS mv_ai
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
GRANT ALL PRIVILEGES ON mv_ai.* TO '${MYSQL_USER}'@'%';
FLUSH PRIVILEGES;
EOSQL
