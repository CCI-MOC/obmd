package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"

	"github.com/caarlos0/env"

	"github.com/CCI-MOC/obmd/internal/driver"
	"github.com/CCI-MOC/obmd/internal/driver/dummy"
	"github.com/CCI-MOC/obmd/internal/driver/ipmi"
	"github.com/CCI-MOC/obmd/internal/driver/mock"

	"github.com/CCI-MOC/obmd/httpserver"
	"github.com/CCI-MOC/obmd/token"
)

// Contents of the config file
type Config struct {
	DBType     string      `env:"DB_TYPE,required"`
	DBPath     string      `env:"DB_PATH,required"`
	AdminToken token.Token `env:"ADMIN_TOKEN,required"`
	ServerCfg  httpserver.Config
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
	if err := env.Parse(&cfg.ServerCfg); err != nil {
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

	if err := config.ServerCfg.Validate(); err != nil {
		log.Fatal(err)
	}

	chkfatal(httpserver.Run(&config.ServerCfg, nil))
}
