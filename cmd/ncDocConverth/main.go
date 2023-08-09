package main

import (
	"crypto/tls"
	"net/http"
	"time"

	"git.rpjosh.de/RPJosh/go-logger"
	"git.rpjosh.de/ncDocConverter/internal/models"
	"git.rpjosh.de/ncDocConverter/internal/ncworker"
)

var version string

type WebApplication struct {
	logger *logger.Logger
	config *models.WebConfig
}

func main() {
	defer logger.CloseFile()

	config, err := models.SetConfig(version)
	if err != nil {
		logger.Error(err.Error())
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		MinVersion:       tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	webApp := WebApplication{
		logger: logger.GetGlobalLogger(),
		config: config,
	}

	srv := &http.Server{
		Addr:         config.Server.Address,
		ErrorLog:     nil,
		Handler:      webApp.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	ncConvertUsers, err := models.ParseConvertUsers(webApp.config.Server.JobFile)
	if err != nil {
		logger.Error("Unable to parse the file %s: %s", "dd", err)
	}
	ncworker.NewScheduler(ncConvertUsers, config)

	if 1 == 1 {
		// Webserver for BookStack hooks currently not implemented
		return
	}

	logger.Info("Server started on %s", config.Server.Address)
	var errw error
	if config.Server.Certificate == "" {
		errw = srv.ListenAndServe()
	} else {
		errw = srv.ListenAndServeTLS(config.Server.Certificate+"cert.pem", config.Server.Certificate+"key.pem")
	}

	logger.Error("Failed to run the HTTP Server: %s", errw)
}
