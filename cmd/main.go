package main

import (
	"context"
	"flag"
	"fmt"
	log "log/slog"
	"os"
	"os/signal"
	"strings"
	"tunnel/core"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("expected 'client' or 'server' subcommands")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "client":
		client()
	case "server":
		server()
	default:
		fmt.Printf("expected 'client' or 'server' subcommands, got '%s'\n", os.Args[1])
		os.Exit(1)
	}
}

func server() {
	flagSet := flag.NewFlagSet("server", flag.ExitOnError)
	l := flagSet.String("l", "", "listen address")
	t := flagSet.String("t", "", "token")
	parseFlags(flagSet)
	s := core.Server{Token: *t}
	s.Start(*l)
}
func client() {
	flagSet := flag.NewFlagSet("client", flag.ExitOnError)
	s := flagSet.String("s", "", "server address")
	t := flagSet.String("t", "", "token")
	r := flagSet.String("r", "", "relays")
	p := flagSet.String("p", "", "proxies")
	parseFlags(flagSet)

	c := core.Client{
		ServerAddr: *s,
		Token:      *t,
	}

	if *r != "" {
		var relays []core.Relay
		entries := strings.Split(*r, ",")
		for _, entry := range entries {
			relay, err := processRelayEntry(entry)
			if err != nil {
				log.Error("error parsing relay entry", "err", err)
				return
			}
			relays = append(relays, relay)
		}
		c.Relays = relays
	}

	if *p != "" {
		var proxies []core.Proxy
		entries := strings.Split(*p, ",")
		for _, entry := range entries {
			remotePort, err := core.ParsePort(entry)
			if err != nil {
				log.Error("error parsing proxy entry", "err", err)
				return
			}
			proxies = append(proxies, core.Proxy{
				RemotePort: remotePort,
			})
		}
		c.Proxies = proxies
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Ctx = ctx

	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt, os.Kill)
	go func() {
		<-interrupt
		cancel()
	}()
	c.Start()
}

func parseFlags(set *flag.FlagSet) {
	err := set.Parse(os.Args[2:])
	if err != nil {
		log.Error("error parsing flags", "err", err)
		return
	}
}

func processRelayEntry(entry string) (core.Relay, error) {
	relayArgs := strings.Split(entry, "/")
	remotePort := relayArgs[0]
	targetAddr := relayArgs[1]
	port, err := core.ParsePort(remotePort)
	if err != nil {
		return core.Relay{}, err
	}
	return core.Relay{
		RemotePort: port,
		TargetAddr: targetAddr,
	}, nil
}
