package main

import (
	"context"
	"time"
	"tunnel/core"
)

func main() {
	client := core.NewClient("127.0.0.1:8080", []core.Mapper{
		{12345, "himiku.com:80"},
		{443, "himiku.com:443"},
	})
	ctx, cancel := context.WithCancel(context.Background())
	after := time.After(time.Second * 5)
	go func() {
		<-after
		cancel()
	}()

	client.Start(ctx)

}
