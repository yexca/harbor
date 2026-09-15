FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY go.mod ./
COPY server ./server
RUN GOMAXPROCS=2 CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -p 2 -trimpath -ldflags="-s -w" -o /harbor ./server

FROM alpine:3.23
RUN apk add --no-cache ca-certificates \
    && addgroup -g 10001 harbor \
    && adduser -D -H -u 10001 -G harbor harbor \
    && mkdir /data && chown harbor:harbor /data
COPY --from=build /harbor /harbor
USER harbor:harbor
ENV HARBOR_ADDR=:8080 HARBOR_DATA_DIR=/data
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/harbor", "healthcheck"]
ENTRYPOINT ["/harbor"]
