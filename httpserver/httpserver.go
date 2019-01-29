// Package httpserver handles the logic of configuring an HTTP server with TLS
// certs, choosing a port to listen on, etc, which is orthogonal to the business
// logic of the actual service.
package httpserver

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

// Config captures the http server related configuration from the environment.
type Config struct {
	ListenAddr string `env:"LISTEN_ADDR,required"`
	Insecure   bool   `env:"INSECURE" envDefault:"false"`
	TLSCert    string `env:"TLS_CERT"`
	TLSKey     string `env:"TLS_KEY"`
}

// Validate the config, returning an error describing any problems which occur.
func (config *Config) Validate() error {
	host, _, err := net.SplitHostPort(config.ListenAddr)
	if err != nil {
		return fmt.Errorf("Error parsing LISTEN_ADDR: %v", err)
	}

	hostIP := net.ParseIP(host)
	// if host was a hostname rather than an ip address, hostIP will be nil,
	// so this will correctly evaulate to false:
	isLoopback := hostIP.Equal(net.ParseIP("127.0.0.1")) || hostIP.Equal(net.ParseIP("::1"))

	haveCert := config.TLSCert != ""
	haveKey := config.TLSKey != ""

	if config.Insecure && haveCert {
		return errors.New("Error: Do not specify TLS certificate file" +
			" when INSECURE is true (unset the TLS_CERT environment variable).")
	}
	if config.Insecure && haveKey {
		return errors.New("Error: Do not specify TLS key file" +
			" when INSECURE is true (unset the TLS_KEY environment variable).")
	}

	if haveCert && !haveKey {
		return errors.New("A TLS cert was specified without a key; you must" +
			" specify both the environment variables TLS_CERT and TLS_KEY," +
			" or neither.")
	}
	if haveKey && !haveCert {
		return errors.New("A TLS key was specified without a cert; you must" +
			" specify both the environment variables TLS_CERT and TLS_KEY," +
			" or neither.")
	}

	if !config.Insecure && !haveKey && !isLoopback {
		msg := "Your configuration says to listen on a non-loopback" +
			" address, using plaintext HTTP. This is a bad idea." +
			" You should generate and specify a TLS keypair, or" +
			" only listen on the loopback interface (127.0.0.1, or" +
			" ::1 for ipv6). If you REALLY want to do this, you can" +
			" set the INSECURE environment variable to true."
		if host == "localhost" {
			msg += "\n\nNote that setting the host to \"localhost\"" +
				" is not sufficient; you must specify the" +
				" loopback ip address."
		}
		return errors.New(msg)
	}
	return nil
}

// Run an http server based on the config, using the handler to service requests.
// if the handler is nil, http.DefaultServeMux is used. If Run returns, the error
// will be non-nil.
func Run(config *Config, handler http.Handler) error {
	if config.TLSKey == "" {
		return http.ListenAndServe(config.ListenAddr, handler)
	} else {
		return http.ListenAndServeTLS(
			config.ListenAddr,
			config.TLSCert,
			config.TLSKey,
			handler,
		)
	}
}
