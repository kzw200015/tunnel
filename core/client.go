package core

import (
	"context"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/xtaci/smux"
	"io"
	"log"
	"net"
	"sync"
)

var dialer net.Dialer

type Client struct {
	ServerAddr string
	Mappers    []Mapper
}

func NewClient(serverAddr string, mappers []Mapper) *Client {
	return &Client{ServerAddr: serverAddr, Mappers: mappers}
}

type Mapper struct {
	RemotePort uint16
	TargetAddr string
}

func (c *Client) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, mapper := range c.Mappers {
		wg.Add(1)
		go startRelay(ctx, &wg, mapper, c.ServerAddr)
	}
	wg.Wait()
}

func startRelay(ctx context.Context, wg *sync.WaitGroup, mapper Mapper, serverAddr string) {
	defer func() {
		wg.Done()
	}()

	conn, err := dialer.DialContext(ctx, "tcp", serverAddr)
	if err != nil {
		log.Println(errors.Wrap(err, "连接服务器失败"))
	}
	defer CloseAndLog(conn)

	err = binary.Write(conn, binary.BigEndian, mapper.RemotePort)
	if err != nil {
		log.Println(err)
		return
	}

	session, err := smux.Client(conn, nil)
	if err != nil {
		log.Println(errors.Wrap(err, "创建隧道失败"))
		return
	}
	defer CloseAndLog(session)

	go func() {
		<-ctx.Done()
		CloseAndLog(session)
		CloseAndLog(conn)
	}()

	for {
		stream, err := session.AcceptStream()
		if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, smux.ErrTimeout) {
			log.Printf("%+v", errors.WithStack(err))
			break
		} else if err != nil {
			log.Printf("%+v", errors.WithStack(err))
			continue
		}
		go handleTargetConnection(ctx, stream, mapper.TargetAddr)
	}
}

func handleTargetConnection(ctx context.Context, stream *smux.Stream, targetAddr string) {
	defer CloseAndLog(stream)

	conn, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		log.Printf("%+v", errors.WithStack(err))
		return
	}
	defer CloseAndLog(conn)

	go CopyStream(conn, stream)
	CopyStream(stream, conn)
}
