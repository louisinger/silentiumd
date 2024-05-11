package grpcservice

import (
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"github.com/louisinger/silentiumd/internal/application"
	"golang.org/x/net/http2"
)

type Config struct {
	Port       uint32
	AppService application.SilentiumService
	TlsKey     string
	TlsCert    string
}

func (c Config) Validate() error {
	lis, err := net.Listen("tcp", c.address())
	if err != nil {
		return fmt.Errorf("invalid port: %s", err)
	}
	defer lis.Close()

	return nil
}

func (c Config) insecure() bool {
	return c.TlsKey == "" || c.TlsCert == ""
}

func (c Config) address() string {
	return fmt.Sprintf(":%d", c.Port)
}

func (c Config) gatewayAddress() string {
	return fmt.Sprintf("localhost:%d", c.Port)
}

func (c Config) tlsConfig() (*tls.Config, error) {
	if c.TlsKey == "" || c.TlsCert == "" {
		return nil, errors.New("tls_key and tls_cert both needs to be provided")
	}

	certificate, err := tls.LoadX509KeyPair(c.TlsCert, c.TlsKey)
	if err != nil {
		return nil, err
	}

	config := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
		Certificates: []tls.Certificate{certificate},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}
	config.Rand = rand.Reader

	return config, nil
}
