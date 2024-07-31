package main

import (
	"context"
	"flag"
	log "log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"tunnel/core"
)

func main() {
	listenAddr := flag.String("s", "", "监听地址")
	serverAddr := flag.String("c", "", "服务端地址")
	proxyPortStr := flag.String("p", "", "远程代理端口")
	relayEntryStr := flag.String("r", "", "转发配置")

	flag.Parse()
	if *listenAddr != "" {
		server := &core.Server{
			Token: "200015",
		}
		server.Start(*listenAddr)
	} else if *serverAddr != "" {
		var proxies []core.Proxy
		if *proxyPortStr != "" {
			for _, port := range strings.Split(*proxyPortStr, ",") {
				portInt, err := strconv.Atoi(port)
				if err != nil {
					log.Error("parse port error", "port", port, "err", err)
					return
				}
				proxies = append(proxies, core.Proxy{RemotePort: uint16(portInt)})
			}
		}

		var relays []core.Relay
		if *relayEntryStr != "" {
			for _, entry := range strings.Split(*relayEntryStr, ",") {
				entryArgs := strings.SplitN(entry, ":", 2)
				port, err := strconv.Atoi(entryArgs[0])
				if err != nil {
					log.Error("parse port error", "port", port, "err", err)
					return
				}
				relays = append(relays, core.Relay{
					RemotePort: uint16(port),
					TargetAddr: entryArgs[1],
				})
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		client := &core.Client{
			ServerAddr: *serverAddr,
			Token:      "200015",
			Ctx:        ctx,
			Relays:     relays,
			Proxies:    proxies,
		}
		interrupt := make(chan os.Signal)
		signal.Notify(interrupt, os.Interrupt)
		go func() {
			<-interrupt
			cancel()
		}()
		client.Start()
	}

}
