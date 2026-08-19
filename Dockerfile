# syntax=docker/dockerfile:1.7

ARG VERSION=dev

FROM --platform=$BUILDPLATFORM node:24-alpine AS web-build

WORKDIR /src

COPY package.json package-lock.json ./
COPY web/package.json web/package.json
RUN npm ci

COPY web web
RUN npm run build:web

FROM --platform=$BUILDPLATFORM golang:1.26.6-alpine AS go-build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd cmd
COPY internal internal
COPY --from=web-build /src/internal/webui/dist internal/webui/dist

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -tags production \
    -trimpath \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o /out/iiiu-nav \
    ./cmd/iiiu-nav
RUN mkdir -p /out/data

FROM go-build AS go-test

RUN CGO_ENABLED=0 go test ./cmd/... ./internal/...

FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION

LABEL org.opencontainers.image.title="iiiu-nav" \
      org.opencontainers.image.description="Lightweight single-user navigation" \
      org.opencontainers.image.version=$VERSION

COPY --from=go-build /out/iiiu-nav /iiiu-nav
COPY --from=go-build --chown=65532:65532 /out/data /data

ENV IIU_NAV_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]

ENTRYPOINT ["/iiiu-nav"]
