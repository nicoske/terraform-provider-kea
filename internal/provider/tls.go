package provider

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
)

func newTLSHTTPClient(caFile, clientCertFile, clientKeyFile, serverName string) (*http.Client, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ServerName: serverName,
	}

	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("reading TLS CA file: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no PEM certificates found in TLS CA file %s", caFile)
		}
		config.RootCAs = pool
	}

	switch {
	case clientCertFile != "" && clientKeyFile == "":
		return nil, errors.New("tls_client_cert_file is set without tls_client_key_file")
	case clientCertFile == "" && clientKeyFile != "":
		return nil, errors.New("tls_client_key_file is set without tls_client_cert_file")
	case clientCertFile != "":
		cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("loading TLS client certificate: %w", err)
		}
		config.Certificates = []tls.Certificate{cert}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = config

	return &http.Client{Transport: transport}, nil
}
