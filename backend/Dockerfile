# syntax=docker/dockerfile:1

# ---- build stage ----
FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache git

# Copy the whole module and resolve dependencies. types.gen.go is committed,
# so no code generation is needed at build time.
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# ---- runtime stage ----
FROM alpine:3.20 AS run
RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server /app/server
# Demo data loader: /app/seed --wipe (deletes everything but sign-in accounts).
COPY --from=build /out/seed /app/seed
# Attached files live here (a volume in compose); owned by app so a fresh
# named volume inherits a writable directory.
RUN mkdir -p /data/uploads && chown app:app /data/uploads
ENV UPLOAD_DIR=/data/uploads
USER app

EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 \
  CMD wget -qO- http://127.0.0.1:8080/health >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/server"]
