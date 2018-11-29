package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/caarlos0/env"

	"github.com/CCI-MOC/obmd/internal/driver"
	"github.com/CCI-MOC/obmd/internal/driver/dummy"
	"github.com/CCI-MOC/obmd/internal/driver/ipmi"
	"github.com/CCI-MOC/obmd/internal/driver/mock"
	"github.com/CCI-MOC/obmd/token"
)

// Contents of the config file
type Config struct {
	DBType     string      `env:"DB_TYPE,required"`
	DBPath     string      `env:"DB_PATH,required"`
	ListenAddr string      `env:"LISTEN_ADDR,required"`
	AdminToken token.Token `env:"ADMIN_TOKEN,required"`
	Insecure   bool        `env:"INSECURE" envDefault:"false"`
	TLSCert    string      `env:"TLS_CERT"`
	TLSKey     string      `env:"TLS_KEY"`
}

var (
	genToken = flag.Bool("gen-token", false,
		"Generate a random token, instead of starting the daemon.")
)

// Exit with an error message if err != nil.
func chkfatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func getConfig() Config {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal("Parsing config from environment: ", err)
	}
	return cfg
}

func main() {
	flag.Parse()

	if *genToken {
		// The user passed -gen-token; generate a token and exit.
		tok, err := token.New()
		chkfatal(err)
		text, err := tok.MarshalText()
		chkfatal(err)
		fmt.Println(string(text))
		return
	}

	config := getConfig()

	// DB Types: sqlite3 or postgres
	db, err := sql.Open(config.DBType, config.DBPath)
	chkfatal(err)
	chkfatal(db.Ping())

	state, err := NewState(db, driver.Registry{
		"ipmi": ipmi.Driver,

		// TODO: maybe mask this behind a build tag, so it's not there
		// in production builds:
		"dummy": dummy.Driver,
		"mock":  mock.Driver,
	})
	chkfatal(err)
	srv := makeHandler(&config, NewDaemon(state))
	http.Handle("/", srv)

	host, _, err := net.SplitHostPort(config.ListenAddr)
	if err != nil {
		log.Fatal("Error parsing ListenAddr:", err)
	}

	hostIP := net.ParseIP(host)
	// if host was a hostname rather than an ip address, hostIP will be nil,
	// so this will correctly evaulate to false:
	isLoopback := hostIP.Equal(net.ParseIP("127.0.0.1")) || hostIP.Equal(net.ParseIP("::1"))

	haveCert := config.TLSCert != ""
	haveKey := config.TLSKey != ""

	if config.Insecure && haveCert {
		log.Fatal("Error: Do not specify TLS certificate file",
			" when Insecure is true.")
	}
	if config.Insecure && haveKey {
		log.Fatal("Error: Do not specify TLS key file",
			" when Insecure is true.")
	}

	if haveCert && !haveKey {
		log.Fatal("A TLS cert was specified without a key; you must",
			" specifiy both or neither.")
	}
	if haveKey && !haveCert {
		log.Fatal("A TLS key was specified without a cert; you must",
			" specifiy both or neither.")
	}

	if !config.Insecure && !haveKey && !isLoopback {
		msg := "Your configuration says to listen on a non-loopback" +
			" address, using plaintext HTTP. This is a bad idea." +
			" You should generate and specify a TLS keypair, or" +
			" only listen on the loopback interface (127.0.0.1, or" +
			" ::1 for ipv6). If you REALLY want to do this, you can" +
			" set the Insecure option to true."
		if host == "localhost" {
			msg += "\n\nNote that setting the host to \"localhost\"" +
				" is not sufficient; you must specify the" +
				" loopback ip address."
		}
		log.Fatal(msg)
	}

	if haveKey {
		chkfatal(http.ListenAndServeTLS(config.ListenAddr,
			config.TLSCert,
			config.TLSKey,
			nil))
	} else {
		chkfatal(http.ListenAndServe(config.ListenAddr, nil))
	}
}
