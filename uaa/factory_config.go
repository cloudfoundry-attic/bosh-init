package uaa

import (
	"crypto/x509"
	"encoding/pem"
	gonet "net"
	gourl "net/url"
	"strconv"
	"strings"

	bosherr "github.com/cloudfoundry/bosh-utils/errors"
)

type Config struct {
	Host string
	Port int

	Client       string
	ClientSecret string

	CACert string

	SkipSslValidation bool
}

func NewConfigFromURL(url string) (Config, error) {
	if len(url) == 0 {
		return Config{}, bosherr.Error("Expected non-empty UAA URL")
	}

	parsedURL, err := gourl.Parse(url)
	if err != nil {
		return Config{}, bosherr.WrapErrorf(err, "Parsing UAA URL '%s'", url)
	}

	host := parsedURL.Host
	port := 443

	if len(host) == 0 {
		host = url
	}

	if strings.Contains(host, ":") {
		var portStr string

		host, portStr, err = gonet.SplitHostPort(host)
		if err != nil {
			return Config{}, bosherr.WrapErrorf(
				err, "Extracting host/port from URL '%s'", url)
		}

		port, err = strconv.Atoi(portStr)
		if err != nil {
			return Config{}, bosherr.WrapErrorf(
				err, "Extracting port from URL '%s'", url)
		}
	}

	if len(host) == 0 {
		return Config{}, bosherr.Errorf("Expected to extract host from URL '%s'", url)
	}

	return Config{Host: host, Port: port}, nil
}

func (c Config) Validate() error {
	if len(c.Host) == 0 {
		return bosherr.Error("Missing 'Host'")
	}

	if c.Port == 0 {
		return bosherr.Error("Missing 'Port'")
	}

	if len(c.Client) == 0 {
		return bosherr.Error("Missing 'Client'")
	}

	if _, err := c.CACertPool(); err != nil {
		return err
	}

	return nil
}

func (c Config) CACertPool() (*x509.CertPool, error) {
	if len(c.CACert) == 0 {
		return nil, nil
	}

	certPool := x509.NewCertPool()

	block, _ := pem.Decode([]byte(c.CACert))
	if block == nil {
		return nil, bosherr.Error("Parsing CA certificate: Missing PEM block")
	}

	if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
		return nil, bosherr.Error("Parsing CA certificate: Not a certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, bosherr.WrapError(err, "Parsing CA certificate")
	}

	certPool.AddCert(cert)

	return certPool, nil
}
