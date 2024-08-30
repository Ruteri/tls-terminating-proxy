# tls-terminating-proxy

[![Goreport status](https://goreportcard.com/badge/github.com/ruteri/tls-terminating-proxy)](https://goreportcard.com/report/github.com/ruteri/tls-terminating-proxy)
[![Test status](https://github.com/ruteri/tls-terminating-proxy/workflows/Checks/badge.svg?branch=main)](https://github.com/ruteri/tls-terminating-proxy/actions?query=workflow%3A%22Checks%22)

Proxy that terminates tls and serves its tls certificate!  
Put the certificate API behind something like https://github.com/flashbots/cvm-reverse-proxy.  

---

**Generate certificate and key files**

```
openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 365 -key ca.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=Acme Root CA" -out ca.crt

openssl req -newkey rsa:2048 -nodes -keyout server.key -subj "/C=CN/ST=GD/L=SZ/O=Acme, Inc./CN=*.example.com" -out server.csr
openssl x509 -req -extfile <(printf "subjectAltName=DNS:example.com,DNS:www.example.com") -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

```

**Run the proxy**

```
go run ./cmd/proxy-server/main.go 
```

**Run the client (verification)**

Assumes `example.com` is your domain (put `127.0.0.1 example.com` in `/etc/hosts`).  
```
go run ./cmd/proxy-client/main.go --proxy-url https://example.com:8081
```

**Install dev dependencies**

```bash
go install mvdan.cc/gofumpt@v0.4.0
go install honnef.co/go/tools/cmd/staticcheck@v0.4.2
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3
go install go.uber.org/nilaway/cmd/nilaway@v0.0.0-20240821220108-c91e71c080b7
```

**Lint, test, format**

```bash
make lint
make test
make fmt
```
