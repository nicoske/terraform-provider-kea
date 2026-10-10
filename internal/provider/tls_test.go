package provider

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testCert struct {
	cert     *x509.Certificate
	key      *ecdsa.PrivateKey
	certFile string
	keyFile  string
}

func newTestCert(t *testing.T, template *x509.Certificate, parent *testCert) *testCert {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template.NotBefore = time.Now().Add(-time.Hour)
	template.NotAfter = time.Now().Add(time.Hour)

	issuer, issuerKey := template, key
	if parent != nil {
		issuer, issuerKey = parent.cert, parent.key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, &key.PublicKey, issuerKey)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	c := &testCert{
		cert:     cert,
		key:      key,
		certFile: filepath.Join(dir, "cert.pem"),
		keyFile:  filepath.Join(dir, "key.pem"),
	}
	writePEM(t, c.certFile, "CERTIFICATE", der)
	writePEM(t, c.keyFile, "PRIVATE KEY", keyDER)
	return c
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()

	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}

type testPKI struct {
	ca     *testCert
	server *testCert
	client *testCert
}

func newTestPKI(t *testing.T) testPKI {
	t.Helper()

	ca := newTestCert(t, &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}, nil)
	server := newTestCert(t, &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "kea.test"},
		DNSNames:     []string{"kea.test"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}, ca)
	client := newTestCert(t, &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "terraform"},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, ca)
	return testPKI{ca: ca, server: server, client: client}
}

func TestNewTLSHTTPClient(t *testing.T) {
	pki := newTestPKI(t)

	notPEM := filepath.Join(t.TempDir(), "not.pem")
	if err := os.WriteFile(notPEM, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		caFile     string
		certFile   string
		keyFile    string
		serverName string
		wantErr    string
	}{
		{name: "system roots"},
		{name: "ca", caFile: pki.ca.certFile},
		{name: "mutual tls", caFile: pki.ca.certFile, certFile: pki.client.certFile, keyFile: pki.client.keyFile},
		{name: "server name", caFile: pki.ca.certFile, serverName: "kea.test"},
		{name: "missing ca file", caFile: filepath.Join(t.TempDir(), "missing.pem"), wantErr: "reading TLS CA file"},
		{name: "ca file without certificates", caFile: notPEM, wantErr: "no PEM certificates found"},
		{name: "cert without key", certFile: pki.client.certFile, wantErr: "tls_client_cert_file is set without tls_client_key_file"},
		{name: "key without cert", keyFile: pki.client.keyFile, wantErr: "tls_client_key_file is set without tls_client_cert_file"},
		{name: "mismatched key", certFile: pki.client.certFile, keyFile: pki.server.keyFile, wantErr: "loading TLS client certificate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newTLSHTTPClient(tt.caFile, tt.certFile, tt.keyFile, tt.serverName)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			config := client.Transport.(*http.Transport).TLSClientConfig
			if config.MinVersion != tls.VersionTLS12 {
				t.Errorf("expected minimum version TLS 1.2, got %x", config.MinVersion)
			}
			if config.ServerName != tt.serverName {
				t.Errorf("expected server name %q, got %q", tt.serverName, config.ServerName)
			}
			if (config.RootCAs != nil) != (tt.caFile != "") {
				t.Errorf("expected custom roots %t, got %t", tt.caFile != "", config.RootCAs != nil)
			}
			if len(config.Certificates) > 0 != (tt.certFile != "") {
				t.Errorf("expected client certificate %t, got %d certificates", tt.certFile != "", len(config.Certificates))
			}
		})
	}
}

func TestNewTLSHTTPClient_handshake(t *testing.T) {
	pki := newTestPKI(t)

	clientCAs := x509.NewCertPool()
	clientCAs.AddCert(pki.ca.cert)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{pki.server.cert.Raw},
			PrivateKey:  pki.server.key,
		}},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientCAs,
	}
	server.StartTLS()
	defer server.Close()

	tests := []struct {
		name       string
		certFile   string
		keyFile    string
		serverName string
		wantErr    bool
	}{
		{name: "mutual tls", certFile: pki.client.certFile, keyFile: pki.client.keyFile},
		{name: "server name", certFile: pki.client.certFile, keyFile: pki.client.keyFile, serverName: "kea.test"},
		{name: "wrong server name", certFile: pki.client.certFile, keyFile: pki.client.keyFile, serverName: "other.test", wantErr: true},
		{name: "no client certificate", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newTLSHTTPClient(pki.ca.certFile, tt.certFile, tt.keyFile, tt.serverName)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := client.Get(server.URL)
			if err == nil {
				_ = resp.Body.Close()
			}
			if tt.wantErr != (err != nil) {
				t.Fatalf("expected error %t, got %v", tt.wantErr, err)
			}
		})
	}
}
