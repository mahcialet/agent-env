package auth

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io"
	"os"
)

func Fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:])
}
func ReadFingerprint(path string) (string, error) {
	b, e := read(path)
	if e != nil {
		return "", e
	}
	block, _ := pem.Decode(b)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", errors.New("certificate PEM required")
	}
	c, e := x509.ParseCertificate(block.Bytes)
	if e != nil {
		return "", e
	}
	return Fingerprint(c), nil
}
func read(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil {
		return nil, e
	}
	if len(b) > 1<<20 {
		return nil, errors.New("TLS material exceeds size limit")
	}
	return b, nil
}
func LoadTLS(ca, cert, key string, server bool) (*tls.Config, error) {
	caBytes, e := read(ca)
	if e != nil {
		return nil, e
	}
	certBytes, e := read(cert)
	if e != nil {
		return nil, e
	}
	keyBytes, e := read(key)
	if e != nil {
		return nil, e
	}
	pair, e := tls.X509KeyPair(certBytes, keyBytes)
	if e != nil {
		return nil, e
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, errors.New("CA certificate PEM required")
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{pair}, RootCAs: pool}
	if server {
		cfg.ClientCAs = pool
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return cfg, nil
}
