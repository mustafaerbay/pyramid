#!/bin/sh
# The bcrypt hash contains `$`, which docker compose would interpolate if it
# were passed as-is through .env, so it arrives base64-encoded.
set -eu
: "${BASIC_AUTH_USER:?BASIC_AUTH_USER must be set}"
: "${BASIC_AUTH_HASH_B64:?BASIC_AUTH_HASH_B64 must be set}"
BASIC_AUTH_HASH="$(printf '%s' "$BASIC_AUTH_HASH_B64" | base64 -d)"
export BASIC_AUTH_HASH
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
