package main

import (
	"log"
	"os"
	"time"

	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/rlb/app/config"
	"github.com/umputun/rlb/app/picker"
	"github.com/umputun/rlb/app/server"
)

var opts struct {
	Conf     string `short:"c" long:"conf" env:"CONF" default:"rlb.yml" description:"configuration file"`
	Refresh  int    `short:"r" long:"refresh" env:"REFRESH" default:"30" description:"refresh interval (secs)"`
	TimeOut  int    `short:"t" long:"timeout" env:"TIMEOUT" default:"5" description:"HEAD/GET timeouts (secs)"`
	StatsURL string `short:"s" long:"stats" env:"STATS" default:"" description:"stats url"`
	Dbg      bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
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
		log.Fatalf("failed to open %s, %v", opts.Conf, err)
	}
	conf := config.NewConf(confReader)
	if err := confReader.Close(); err != nil {
		log.Printf("[WARN] failed to close %s, %s", opts.Conf, err.Error())
	}

	pck := picker.NewRandomWeighted(
		conf.Get(),
		time.Second*time.Duration(opts.Refresh),
		time.Second*time.Duration(opts.TimeOut),
	)

	server.NewRLBServer(pck, conf.NoNode.Message, opts.StatsURL, revision).Run()
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}
