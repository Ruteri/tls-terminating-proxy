package main

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	tdx_abi "github.com/google/go-tdx-guest/abi"
	tdx_pb "github.com/google/go-tdx-guest/proto/tdx"
	"github.com/google/go-tdx-guest/verify"
	"github.com/google/uuid"
	"github.com/ruteri/tls-terminating-proxy/common"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

var flags []cli.Flag = []cli.Flag{
	&cli.StringFlag{
		Name:  "cert-service",
		Value: "http://127.0.0.1:8080",
		Usage: "certificate service url",
	},
	&cli.StringFlag{
		Name:  "proxy-url",
		Value: "https://127.0.0.1:8081",
		Usage: "proxy url",
	},
	&cli.BoolFlag{
		Name:  "log-json",
		Value: false,
		Usage: "log in JSON format",
	},
	&cli.BoolFlag{
		Name:  "log-debug",
		Value: false,
		Usage: "log debug messages",
	},
	&cli.BoolFlag{
		Name:  "log-uid",
		Value: false,
		Usage: "generate a uuid and add to all log messages",
	},
	&cli.StringFlag{
		Name:  "log-service",
		Value: "proxy-client",
		Usage: "add 'service' tag to logs",
	},
}

func main() {
	app := &cli.App{
		Name:  "tls proxy client",
		Usage: "Check tls connectivity",
		Flags: flags,
		Action: func(cCtx *cli.Context) error {
			apiAddr := cCtx.String("cert-service")
			proxyAddr := cCtx.String("proxy-url")
			logJSON := cCtx.Bool("log-json")
			logDebug := cCtx.Bool("log-debug")
			logUID := cCtx.Bool("log-uid")
			logService := cCtx.String("log-service")
			// enablePprof := cCtx.Bool("pprof")

			log := common.SetupLogger(&common.LoggingOpts{
				Debug:   logDebug,
				JSON:    logJSON,
				Service: logService,
				Version: common.Version,
			})

			if logUID {
				id := uuid.Must(uuid.NewRandom())
				log = log.With("uid", id.String())
			}

			//nolint:gosec
			resp, err := http.Get(apiAddr)
			if err != nil {
				log.Error("could not request cert data", "err", err)
				return err
			}
			defer resp.Body.Close()

			certAndQuoteData, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error("could not read cert data", "err", err)
				return err
			}

			var caCertResponseData struct {
				Cert  []byte `json:"cert"`
				Quote []byte `json:"quote"`
			}

			if err := json.Unmarshal(certAndQuoteData, &caCertResponseData); err != nil {
				log.Error("could not unmarshal cert data", "err", err)
				return err
			}

			v4Quote, err := verifyQuote(log, caCertResponseData.Quote)
			if err != nil {
				log.Error("could not validate cert quote", "err", err)
				return err
			}

			// TODO: validate quote matches the expected measurement!
			caCertAppData := sha256.New().Sum(caCertResponseData.Cert)
			if !bytes.Equal(v4Quote.Header.UserData, caCertAppData) {
				log.Error("unexpected user data in quote", "expected", hex.EncodeToString(caCertAppData), "actual", hex.EncodeToString(v4Quote.Header.UserData))
				return errors.New("user data mismatch")
			}

			roots := x509.NewCertPool()
			ok := roots.AppendCertsFromPEM(caCertResponseData.Cert)
			if !ok {
				log.Error("invalid certificate received", "cert", string(caCertResponseData.Cert))
				return errors.New("invalid certificate")
			}

			/* TCP example
			conn, err := tls.Dial("tcp", proxyAddr, &tls.Config{
				RootCAs: roots,
			})
			if err != nil {
				log.Error("could not open connection", "err", err)
				return err
			}
			*/

			/* HTTP-OVER-TLS example */

			client := &http.Client{
				//nolint:gosec
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: roots,
					},
				},
			}

			proxyResp, err := client.Get(proxyAddr)
			if err != nil {
				log.Error("could not get proxied service", "err", err)
				return err
			}
			defer proxyResp.Body.Close()

			respBody, err := io.ReadAll(proxyResp.Body)
			if err != nil {
				log.Error("could not read proxied service body", "err", err)
				return err
			}

			log.Info("Received", "resp", string(respBody))
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func verifyQuote(log *slog.Logger, rawQuote []byte) (*tdx_pb.QuoteV4, error) {
	protoQuote, err := tdx_abi.QuoteToProto(rawQuote[:])
	if err != nil {
		log.Error("invalid quote format", "quote", rawQuote)
		return nil, err
	}

	v4Quote, err := func() (*tdx_pb.QuoteV4, error) {
		switch q := protoQuote.(type) {
		case *tdx_pb.QuoteV4:
			return q, nil
		default:
			return nil, fmt.Errorf("unsupported quote type: %T", q)
		}
	}()
	if err != nil {
		return nil, err
	}

	log.Info("quote", "quote", v4Quote)

	options := verify.DefaultOptions()
	// TODO: fetch collateral before verifying to distinguish the error better
	err = verify.TdxQuote(protoQuote, options)
	if err != nil {
		return nil, err
	}

	return v4Quote, nil
}
