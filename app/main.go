package main

import (
	"os"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/rlb/app/config"
	"github.com/umputun/rlb/app/picker"
	"github.com/umputun/rlb/app/server"
)

var opts struct {
	Port     int           `short:"p" long:"port" env:"PORT" default:"7070" description:"port"`
	Conf     string        `short:"c" long:"conf" env:"CONF" default:"rlb.yml" description:"configuration file"`
	Refresh  time.Duration `short:"r" long:"refresh" env:"REFRESH" default:"30" description:"refresh interval"`
	TimeOut  time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"5" description:"HEAD/GET timeouts"`
	StatsURL string        `short:"s" long:"stats" env:"STATS" default:"" description:"stats url"`
	Dbg      bool          `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	log.Printf("RLB - %s", revision)
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	setupLog(opts.Dbg)

	confReader, err := os.Open(opts.Conf)
	if err != nil {
		log.Fatalf("[PANIC] failed to open %s, %v", opts.Conf, err)
	}

	conf := config.NewConf(confReader)
	if err := confReader.Close(); err != nil {
		log.Printf("[WARN] failed to close %s, %s", opts.Conf, err.Error())
	}

	pck := picker.NewRandomWeighted(conf.Get(), opts.Refresh, opts.TimeOut)
	server.NewRLBServer(pck, conf.NoNode.Message, opts.StatsURL, opts.Port, revision).Run()
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
