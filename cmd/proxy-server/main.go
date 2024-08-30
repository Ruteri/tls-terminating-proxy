package main

import (
	"context"
	"crypto/tls"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/ruteri/tls-terminating-proxy/common"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

var flags []cli.Flag = []cli.Flag{
	&cli.StringFlag{
		Name:  "cert-service-listen-addr",
		Value: "0.0.0.0:8080",
		Usage: "address to serve certificate on",
	},
	&cli.StringFlag{
		Name:  "proxy-listen-addr",
		Value: "0.0.0.0:8081",
		Usage: "address proxy should listen on",
	},
	&cli.StringFlag{
		Name:  "proxy-target-addr",
		Value: "http://127.0.0.1:8082",
		Usage: "address proxy should forward to",
	},
	&cli.StringFlag{
		Name:  "certificate-file",
		Value: "server.crt",
		Usage: "Certificate to present (PEM)",
	},
	&cli.StringFlag{
		Name:  "ca-certificate-file",
		Value: "ca.crt",
		Usage: "Certificate to present (PEM)",
	},
	&cli.StringFlag{
		Name:  "private-key-file",
		Value: "server.key",
		Usage: "Private key for the certificate (PEM)",
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
		Value: "proxy",
		Usage: "add 'service' tag to logs",
	},
	/*
		&cli.BoolFlag{
			Name:  "pprof",
			Value: false,
			Usage: "enable pprof debug endpoint",
		},
	*/
}

func main() {
	app := &cli.App{
		Name:  "httpserver",
		Usage: "Serve API, and metrics",
		Flags: flags,
		Action: func(cCtx *cli.Context) error {
			apiListenAddr := cCtx.String("cert-service-listen-addr")
			listenAddr := cCtx.String("proxy-listen-addr")
			targetAddr := cCtx.String("proxy-target-addr")
			certFile := cCtx.String("certificate-file")
			caCertFile := cCtx.String("ca-certificate-file")
			keyFile := cCtx.String("private-key-file")
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

			caCertData, err := os.ReadFile(caCertFile)
			if err != nil {
				log.Error("could not read cert data", "err", err)
				return err
			}

			apiSrv := &http.Server{
				Addr:              apiListenAddr,
				Handler:           &DummyHandler{log: log, certData: caCertData},
				ReadHeaderTimeout: 200 * time.Millisecond,
			}

			targetURL, err := url.Parse(targetAddr)
			if err != nil {
				log.Error("could not parse target address", "err", err)
				return err
			}

			proxy := httputil.NewSingleHostReverseProxy(targetURL)

			srv := &http.Server{
				Addr:              listenAddr,
				Handler:           proxy,
				ReadHeaderTimeout: time.Second,
				TLSConfig: &tls.Config{
					MinVersion:               tls.VersionTLS13,
					PreferServerCipherSuites: true,
				},
			}

			exit := make(chan os.Signal, 3)

			go func() {
				log.Info("Starting proxy server", "addr", listenAddr)
				if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil {
					log.Error("proxy exited", "err", err)
				}
				exit <- os.Interrupt
			}()

			go func() {
				log.Info("Starting API server", "addr", apiListenAddr)
				if err := apiSrv.ListenAndServe(); err != nil {
					log.Error("api existed", "err", err)
				}
				exit <- os.Interrupt
			}()

			signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
			<-exit

			// Shutdown server once termination signal is received
			_ = srv.Shutdown(context.Background())
			_ = apiSrv.Shutdown(context.Background())
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

type DummyHandler struct {
	log      *slog.Logger
	certData []byte
}

func (d *DummyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write(d.certData)
	if err != nil {
		d.log.Error("could not respond with cert data", "err", err)
	}
}
