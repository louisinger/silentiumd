package grpcservice

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/louisinger/silentiumd/internal/application"
	"golang.org/x/crypto/acme/autocert"
)

type Config struct {
	Port       uint32
	AppService application.SilentiumService
	NoTLS      bool
	HostName   string
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
	return c.NoTLS
}

func (c Config) address() string {
	return fmt.Sprintf(":%d", c.Port)
}

func (c Config) gatewayAddress() string {
	return fmt.Sprintf("localhost:%d", c.Port)
}

func (c Config) tlsConfig() (*tls.Config, error) {
	m := autocert.Manager{
		Prompt: autocert.AcceptTOS,
	}

	return m.TLSConfig(), nil
}
