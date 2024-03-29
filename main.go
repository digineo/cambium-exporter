package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/digineo/cambium-exporter/auth"
	"github.com/digineo/cambium-exporter/exporter"

	kingpin "github.com/alecthomas/kingpin/v2"
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

	listenAddress := kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9836").String()
	configFile := kingpin.Flag("config", "Path to configuration file.").Default(DefaultConfigPath).String()
	performLogin := kingpin.Flag("login", "Perform login test, and dump session cookie.").Bool()
	loginTimeout := kingpin.Flag("login.timeout", "Timeout for login and session refresh.").Default("5m").Short('t').Duration()
	verbose := kingpin.Flag("verbose", "Increase log verbosity.").Short('V').Bool()
	versionFlag := kingpin.Flag("version", "Print version information and exit.").Short('v').Bool()
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

	if headless := os.Getenv("HEADLESS"); headless == "0" {
		auth.SetHeadless(false)
	}
	if binary := os.Getenv("CHROME_BINARY"); binary != "" {
		auth.SetExecPath(binary)
	}
	if *loginTimeout > time.Second {
		auth.SetLoginTimeout(*loginTimeout)
	}

	client, err := exporter.LoadClientConfig(*configFile, *verbose)
	if err != nil {
		log.Fatal(err.Error())
	}

	if *performLogin {
		info, err := auth.Login(client.Username, client.Password, *verbose)
		if err != nil {
			log.Fatalf("login failed: %v", err)
			return
		}
		log.Printf("login succeeded: %+v", info)
		return
	}

	log.Fatal(client.Start(*listenAddress, Version()))
}

func printVersion() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	const l = "%-10s %-50s %s\n"
	fmt.Println("Dependencies\n------------")
	fmt.Printf(l, "main", info.Main.Path, Version())
	for _, i := range info.Deps {
		if r := i.Replace; r != nil {
			fmt.Printf(l, "dep", r.Path, r.Version)
			fmt.Printf(l, "  replaces", i.Path, i.Version)
		} else {
			fmt.Printf(l, "dep", i.Path, i.Version)
		}
	}
}

func Version() string {
	if commit == "" && date == "" {
		return version
	}
	return fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
}
