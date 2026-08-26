# syntax=docker/dockerfile:1
# ---- build stage -----------------------------------------------------------
FROM golang:1.23-alpine AS build

WORKDIR /src

# Cache module metadata first (the project is standard-library only today,
# but this keeps the layer reusable if dependencies are added later).
COPY go.mod ./
COPY . .

ARG CGO_ENABLED=0
ARG GOOS=linux
ARG GOARCH=amd64

RUN CGO_ENABLED=${CGO_ENABLED} GOOS=${GOOS} GOARCH=${GOARCH} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/cleanroom-environment-monitor-service .

# ---- runtime stage ---------------------------------------------------------
FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app
COPY --from=build /out/cleanroom-environment-monitor-service /app/cleanroom-environment-monitor-service

# The binary is static (CGO_ENABLED=0), so it runs as the unprivileged user.
USER app

ENV PORT=8080
EXPOSE 8080

# busybox wget is present in alpine:3.20 and keeps the image small.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/healthz" || exit 1

ENTRYPOINT ["/app/cleanroom-environment-monitor-service"]
