# syntax=docker/dockerfile:1.7

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

FROM node:24-bookworm-slim AS studio-build
WORKDIR /src
RUN npm install --global pnpm@11.15.1
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY studio/package.json studio/package.json
RUN pnpm install --frozen-lockfile
COPY studio studio
RUN pnpm --filter @gyrifi/studio build

FROM ghcr.io/ggml-org/llama.cpp:server AS llama-runtime

FROM golang:1.24-bookworm AS runtime-build
ARG VERSION
ARG COMMIT
ARG BUILD_DATE
WORKDIR /src/runtime
COPY runtime/go.mod runtime/go.sum ./
RUN go mod download
COPY runtime ./
RUN rm -rf internal/interfaces/http/static && mkdir -p internal/interfaces/http/static
COPY --from=studio-build /src/studio/dist/ internal/interfaces/http/static/
RUN CGO_ENABLED=0 go test ./... \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w \
      -X github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo.Version=${VERSION} \
      -X github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo.Commit=${COMMIT} \
      -X github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo.Date=${BUILD_DATE}" \
      -o /out/gyrifi ./cmd/gyrifi

FROM ubuntu:24.04 AS runtime
ARG VERSION
ARG COMMIT
ARG BUILD_DATE
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.source="https://github.com/Gyrifi-Ai/Gyrif-context-ledger" \
      org.opencontainers.image.licenses="AGPL-3.0-only"
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates libgomp1 libssl3 libstdc++6 && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 10001 gyrifi \
    && useradd --system --uid 10001 --gid gyrifi --home-dir /data gyrifi \
    && mkdir -p /data /models \
    && chown -R gyrifi:gyrifi /data /models
COPY --from=runtime-build /out/gyrifi /usr/local/bin/gyrifi
COPY --from=llama-runtime /app/ /opt/llama/
USER gyrifi
ENV GYRIFI_HTTP_ADDRESS=:8080 \
    GYRIFI_DATA_DIR=/data \
    GYRIFI_EVALUATION_PROVIDER=disabled \
    GYRIFI_LLAMA_SERVER_PATH=/opt/llama/llama-server \
    GYRIFI_LLAMA_SERVER_PORT=8081 \
    GYRIFI_LOG_LEVEL=info \
    LD_LIBRARY_PATH=/opt/llama
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gyrifi"]
