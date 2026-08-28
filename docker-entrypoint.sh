#!/bin/sh
set -e

# EVO reads /app/config.yml at startup. The image does not ship one, so build it
# here from the environment. Without this the app silently falls back to SQLite.
# Full config.yml, base64-encoded (Railway variable). A bad decode must not leave
# an empty config behind: EVO would fall back to SQLite without saying so.
if [ -n "$MEDIAX_CONFIG" ] && echo "$MEDIAX_CONFIG" | base64 -d > /tmp/config.yml 2>/dev/null && grep -q "^[[:space:]]*Type:" /tmp/config.yml; then
  mv /tmp/config.yml /app/config.yml
else
  if [ -n "$MEDIAX_CONFIG" ]; then
    echo "MEDIAX_CONFIG is not a usable config.yml, falling back to env vars" >&2
  fi
  : "${PORT:=8080}"
  : "${DATABASE_TYPE:=mysql}"
  : "${DATABASE_DATABASE:=mediax}"
  : "${DATABASE_USERNAME:=root}"

  cat > /app/config.yml <<EOF
Database:
  Enabled: true
  Type: ${DATABASE_TYPE}
  Server: ${DATABASE_SERVER}
  Database: ${DATABASE_DATABASE}
  Username: ${DATABASE_USERNAME}
  Password: "${DATABASE_PASSWORD}"
  Cache: "false"
  ConnMaxLifTime: 1h
  Debug: "3"
  MaxIdleConns: 10
  MaxOpenConns: 100
  Params: "parseTime=true"
  SSLMode: false
  SlowQueryThreshold: 500ms

HTTP:
  Host: 0.0.0.0
  Port: ${PORT}
  BodyLimit: ${HTTP_BODY_LIMIT:-25mb}
  CaseSensitive: false
  CompressedFileSuffix: .evo.gz
  Concurrency: 1024
  DisableDefaultContentType: false
  DisableDefaultDate: false
  DisableHeaderNormalizing: false
  DisableKeepalive: false
  ETag: false
  GETOnly: false
  IdleTimeout: 0
  Immutable: false
  Network: ""
  Prefork: false
  ProxyHeader: X-Forwarded-For
  ReadBufferSize: 10mb
  ReadTimeout: ${HTTP_READ_TIMEOUT:-30s}
  ReduceMemoryUsage: false
  ServerHeader: EVO
  StrictRouting: false
  UnescapePath: false
  EnablePrintRoutes: false
  WriteBufferSize: 4kb
  WriteTimeout: ${HTTP_WRITE_TIMEOUT:-30s}
EOF
fi

exec /app/mediax "$@"
