package httpclient

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// TLSConfig holds TLS configuration options
type TLSConfig struct {
	InsecureSkipVerify bool   // Skip TLS certificate verification (dev/test only)
	CACertificate      string // Path to custom CA certificate file
}

// ConfigureTLS creates an http.Transport with TLS configuration
func ConfigureTLS(config *TLSConfig) (*http.Transport, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	// Handle custom CA certificate
	if config != nil && config.CACertificate != "" {
		caCert, err := os.ReadFile(config.CACertificate)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate from %s: %w", config.CACertificate, err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate from %s", config.CACertificate)
		}

		transport.TLSClientConfig.RootCAs = caCertPool
	}

	// Handle insecure skip verify (dev/test only)
	if config != nil && config.InsecureSkipVerify {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	return transport, nil
}

// WithTLSConfig is an option for creating HTTP clients with TLS configuration
func WithTLSConfig(config *TLSConfig) Option {
	return func(c *Client) {
		if config != nil {
			transport, err := ConfigureTLS(config)
			if err != nil {
				// Log warning but don't fail - use default transport
				fmt.Printf("Warning: Failed to configure TLS: %v\n", err)
				return
			}

			// Update the HTTP client's transport
			// If client already exists, update its transport
			// Otherwise, create a new client with the transport
			if c.client != nil {
				c.client.Transport = transport
			} else {
				// Client will be created later, store transport for later use
				// This is handled by creating a client with the transport
				c.client = &http.Client{
					Transport: transport,
					Timeout:   60 * time.Second, // Default timeout
				}
			}
		}
	}
}
