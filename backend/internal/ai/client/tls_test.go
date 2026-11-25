package client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type testCertificateAuthority struct {
	certificate *x509.Certificate
	key         *rsa.PrivateKey
	pool        *x509.CertPool
}

func newTestCertificateAuthority(t *testing.T) testCertificateAuthority {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate CA key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "MysteryVerse Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create CA certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse CA certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(certificate)
	return testCertificateAuthority{
		certificate: certificate,
		key:         key,
		pool:        pool,
	}
}

func issueTestCertificate(
	t *testing.T,
	ca testCertificateAuthority,
	commonName string,
	dnsNames []string,
	ipAddresses []net.IP,
	usage x509.ExtKeyUsage,
) tls.Certificate {
	return issueTestCertificateWithWindow(
		t,
		ca,
		commonName,
		dnsNames,
		ipAddresses,
		usage,
		time.Now().Add(-time.Hour),
		time.Now().Add(time.Hour),
	)
}

func issueTestCertificateWithWindow(
	t *testing.T,
	ca testCertificateAuthority,
	commonName string,
	dnsNames []string,
	ipAddresses []net.IP,
	usage x509.ExtKeyUsage,
	notBefore time.Time,
	notAfter time.Time,
) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate certificate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		DNSNames:     dnsNames,
		IPAddresses:  ipAddresses,
	}
	der, err := x509.CreateCertificate(
		rand.Reader,
		template,
		ca.certificate,
		&key.PublicKey,
		ca.key,
	)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)},
	)
	certificate, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("load certificate key pair: %v", err)
	}
	return certificate
}

func TestClientConnectsWithVerifiedMutualTLS(t *testing.T) {
	ca := newTestCertificateAuthority(t)
	serverCertificate := issueTestCertificate(
		t,
		ca,
		"ai.internal.test",
		[]string{"localhost"},
		[]net.IP{net.ParseIP("127.0.0.1")},
		x509.ExtKeyUsageServerAuth,
	)
	clientCertificate := issueTestCertificate(
		t,
		ca,
		"go-core",
		nil,
		nil,
		x509.ExtKeyUsageClientAuth,
	)

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			t.Fatal("server did not receive a client certificate")
		}
		if r.TLS.PeerCertificates[0].Subject.CommonName != "go-core" {
			t.Fatalf("client certificate = %q", r.TLS.PeerCertificates[0].Subject.CommonName)
		}
		_ = json.NewEncoder(w).Encode(newAgentResponse())
	}))
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCertificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    ca.pool,
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	client, err := New(Config{
		BaseURL:      server.URL,
		ServiceID:    "go-core",
		ServiceToken: "service-token",
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			RootCAs:      ca.pool,
			Certificates: []tls.Certificate{clientCertificate},
			ServerName:   "localhost",
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.Run(context.Background(), newAgentRequest(), "mtls-good"); err != nil {
		t.Fatalf("Run() with mTLS error = %v", err)
	}
}

func TestClientFailsClosedWithoutClientCertificate(t *testing.T) {
	ca := newTestCertificateAuthority(t)
	serverCertificate := issueTestCertificate(
		t,
		ca,
		"ai.internal.test",
		[]string{"localhost"},
		[]net.IP{net.ParseIP("127.0.0.1")},
		x509.ExtKeyUsageServerAuth,
	)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(newAgentResponse())
	}))
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCertificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    ca.pool,
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	client, err := New(Config{
		BaseURL:      server.URL,
		ServiceID:    "go-core",
		ServiceToken: "service-token",
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    ca.pool,
			ServerName: "localhost",
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.Run(context.Background(), newAgentRequest(), "mtls-missing"); err == nil {
		t.Fatal("Run() succeeded without a client certificate")
	}
}

func TestClientFailsClosedWithExpiredClientCertificate(t *testing.T) {
	ca := newTestCertificateAuthority(t)
	serverCertificate := issueTestCertificate(
		t,
		ca,
		"ai.internal.test",
		[]string{"localhost"},
		[]net.IP{net.ParseIP("127.0.0.1")},
		x509.ExtKeyUsageServerAuth,
	)
	expiredClientCertificate := issueTestCertificateWithWindow(
		t,
		ca,
		"go-core",
		nil,
		nil,
		x509.ExtKeyUsageClientAuth,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(-time.Hour),
	)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(newAgentResponse())
	}))
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCertificate},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    ca.pool,
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	client, err := New(Config{
		BaseURL:      server.URL,
		ServiceID:    "go-core",
		ServiceToken: "service-token",
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			RootCAs:      ca.pool,
			Certificates: []tls.Certificate{expiredClientCertificate},
			ServerName:   "localhost",
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if _, err := client.Run(context.Background(), newAgentRequest(), "mtls-expired"); err == nil {
		t.Fatal("Run() succeeded with an expired client certificate")
	}
}
