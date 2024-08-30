# go-template

[![Goreport status](https://goreportcard.com/badge/github.com/flashbots/go-template)](https://goreportcard.com/report/github.com/flashbots/go-template)
[![Test status](https://github.com/flashbots/go-template/workflows/Checks/badge.svg?branch=main)](https://github.com/flashbots/go-template/actions?query=workflow%3A%22Checks%22)

Proxy that terminates tls and serves its tls certificate!  
Put the certificate API behind something like https://github.com/flashbots/cvm-reverse-proxy.  

---

**Generate certificate and key files**

```
openssl req -new -subj "/CN=localhost/" -newkey rsa:2048 -nodes -keyout key.pem -out localhost.csr
openssl x509 -req -days 365 -in localhost.csr -signkey key.pem -out cert.pem
```

**Run the proxy**

```
go run ./cmd/proxy-server/main.go 
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
