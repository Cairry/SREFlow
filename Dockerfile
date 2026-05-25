# Stage 1: Build frontend
FROM docker.1ms.run/node:22-alpine AS frontend-builder

RUN corepack enable && corepack prepare yarn@latest --activate

WORKDIR /app/web

COPY web/ ./

RUN yarn install && yarn build

# Stage 2: Build backend
FROM docker.1ms.run/golang:1.26.0 AS backend-builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /root

COPY . /root

RUN CGO_ENABLED=0 go build -o sreflow ./cmd/server.go \
    && chmod +x sreflow

# Stage 3: Production image
FROM docker.1ms.run/node:22-alpine

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /root/sreflow /app/sreflow

# Copy Next.js standalone output
COPY --from=frontend-builder /app/web/.next/standalone /app/web
COPY --from=frontend-builder /app/web/.next/static /app/web/.next/static
COPY --from=frontend-builder /app/web/public /app/web/public

COPY <<'EOF' /app/start.sh
#!/bin/sh
# Start Go backend on port 9001
./sreflow &
# Start Next.js frontend on port 8080 with API proxy to backend
cd /app/web && PORT=8080 node server.js
EOF

RUN chmod +x /app/start.sh

EXPOSE 8080

ENTRYPOINT ["/app/start.sh"]
