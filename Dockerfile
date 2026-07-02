FROM oven/bun:1.3.14 AS frontend
WORKDIR /src/frontend
COPY frontend/package.json frontend/bun.lock ./
RUN bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/frontend/dist ./frontend/dist
RUN go build -o /out/gaia-calendar ./cmd/server

FROM alpine:3.22
WORKDIR /app
RUN adduser -D -H appuser
COPY --from=backend /out/gaia-calendar /app/gaia-calendar
COPY --from=frontend /src/frontend/dist /app/frontend/dist
RUN mkdir -p /app/data && chown -R appuser:appuser /app
ENV APP_ADDR=:8080
ENV FRONTEND_DIR=/app/frontend/dist
USER appuser
EXPOSE 8080
CMD ["/app/gaia-calendar"]
