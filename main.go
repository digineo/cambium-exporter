package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/digineo/cambium-exporter/exporter"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// DefaultConfigPath points to the default config file location.
// This might be overwritten at build time (using -ldflags).
var DefaultConfigPath = "./config.toml"

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	log.SetFlags(log.Lshortfile)

	listenAddress := kingpin.Flag(
		"web.listen-address",
		"Address on which to expose metrics and web interface.",
	).Default(":9836").String()

	configFile := kingpin.Flag(
		"config",
		"Path to configuration file.",
	).Default(DefaultConfigPath).String()

	versionFlag := kingpin.Flag(
		"version",
		"Print version information and exit.",
	).Short('v').Bool()

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

	client, err := exporter.LoadClientConfig(*configFile)
	if err != nil {
		log.Fatal(err.Error())
	}

	client.Start(*listenAddress, version)
}

func printVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	const l = "%-10s %-50s %s\n"
	fmt.Println("Dependencies\n------------")
	fmt.Printf(l, "main", info.Main.Path, version)
	for _, i := range info.Deps {
		if r := i.Replace; r != nil {
			fmt.Printf(l, "dep", r.Path, r.Version)
			fmt.Printf(l, "  replaces", i.Path, i.Version)
		} else {
			fmt.Printf(l, "dep", i.Path, i.Version)
		}
	}
}
