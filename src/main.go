package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	flags "github.com/jessevdk/go-flags"
)

// Embed version info at build time.
// e.g. `go build cmd/pong -ldflags "-X main.version=1.0.0"`
var version = ""

func getVersion() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(version unknown)"
	}
	return info.Main.Version
}

// ref. https://pkg.go.dev/github.com/jessevdk/go-flags
type Options struct {
	Help       bool   `short:"h" long:"help" description:"print help and exit"`
	Version    bool   `short:"v" long:"version" description:"print version and exit"`
	VersionAll bool   `short:"V" long:"version-all" hidden:"yes"`
	Count      int    `short:"c" long:"count" value-name:"<count>" default:"4" description:"stop after <count> replies"`
	TimeToLive int    `short:"t" long:"ttl" value-name:"<ttl>" default:"64" description:"define time to live"`
	Padding    string `short:"p" long:"padding" value-name:"<pattern>" description:"contents of padding byte"`
	Args       struct {
		Destination string `positional-arg-name:"<destination>" description:"dns name or ip address"`
	} `positional-args:"yes" required:"yes"`
}

var opts Options

func main() {

	parser := flags.NewParser(&opts, flags.PassDoubleDash)
	parser.Usage = "[options]"
	_, err := parser.Parse()
	if opts.Version {
		// e.g "pong 0.0.1 Linux/amd64 (go1.17.7)"
		fmt.Printf("%s %s %s/%s (%s)\n",
			filepath.Base(os.Args[0]),
			getVersion(),
			runtime.GOOS,
			runtime.GOARCH,
			runtime.Version())
		os.Exit(0)
	}
	if opts.VersionAll {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Println("build info not found.")
			os.Exit(1)
		}
		fmt.Printf("%s %s\n", info.Main.Path, info.Main.Version)
		for _, m := range info.Deps {
			fmt.Printf("%s %s\n", m.Path, m.Version)
		}
		os.Exit(0)
	}
	if opts.Help {
		parser.WriteHelp(os.Stderr)
		os.Exit(0)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	pong()
}
