# Postgres with the schema baked in, so the Coolify deployment needs no bind
# mount: schema.sql runs once, on first start with an empty volume.
FROM postgres:16-alpine
COPY --chmod=0644 schema.sql /docker-entrypoint-initdb.d/01-schema.sql
