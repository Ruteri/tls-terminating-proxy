# syntax=docker/dockerfile:1
FROM golang:1.21 as builder
ARG VERSION
WORKDIR /build
ADD . /build/
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux \
    go build \
        -trimpath \
        -ldflags "-s -X main.version=${VERSION}" \
        -v \
        -o tls-proxy-server \
    cmd/proxy-server/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/tls-proxy-server /app/tls-proxy-server
ENV CERT_SERVICE_LISTEN_ADDR=":8080"
ENV PROXY_LISTEN_ADDR=":8081"
EXPOSE 8080
EXPOSE 8081
CMD ["/app/tls-proxy-server"]
