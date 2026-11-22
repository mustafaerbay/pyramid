FROM caddy:2-alpine
COPY --chmod=0644 Caddyfile /etc/caddy/Caddyfile
COPY --chmod=0755 entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]
