package main

import (
	"context"
	"os"
	"os/signal"
	"tunnel/core"
)

func main() {
	client := core.NewClient("127.0.0.1:8080", []core.Mapper{
		{12345, "himiku.com:80"},
		{443, "himiku.com:443"},
	})
	ctx, cancel := context.WithCancel(context.Background())
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	go func() {
		<-interrupt
		cancel()
	}()
	client.Start(ctx)

}
