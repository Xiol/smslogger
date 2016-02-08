package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	log "github.com/Sirupsen/logrus"
)

type Formatter struct{}

// Format a log entry in a text-only format.
func (f *Formatter) Format(entry *log.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("%s [%s] %s\n", entry.Time.Format("2006-01-02 15:04:05.000"), entry.Level.String(), entry.Message)), nil
}

func main() {
	var bind string
	var port int
	var logLevel string
	var assets string
	var dbDump bool

	flag.BoolVar(&dbDump, "dbdump", false, "Dump database queries")
	flag.StringVar(&logLevel, "loglevel", "info", "Logging level")
	flag.StringVar(&bind, "bind", "127.0.0.1", "Bind to this IP address")
	flag.StringVar(&assets, "assets", "./static", "Path to static assets")
	flag.IntVar(&port, "port", 9345, "Server on this port")
	flag.Parse()

	runtime.GOMAXPROCS(runtime.NumCPU())

	lvl, err := log.ParseLevel(logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Log level not recognised, must be one of: debug, info, warning, error, fatal, panic")
		os.Exit(1)
	}

	log.SetFormatter(&Formatter{})
	log.SetLevel(lvl)
	log.SetOutput(os.Stderr)

	log.Warningf("SMS Logger starting up on %s:%d", bind, port)

	engine := InitWeb(logLevel, assets)
	db := InitDB()

	if dbDump {
		db.LogMode(true)
	}

	InitApi(engine, db)

	defer func() {
		db.Close()
	}()

	engine.Run(fmt.Sprintf("%s:%d", bind, port))
}
