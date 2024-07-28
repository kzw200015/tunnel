package core

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/xtaci/smux"
	"io"
	log "log/slog"
	"net"
	"sync"
	"tailscale.com/net/socks5"
)

var dialer net.Dialer

type Client struct {
	ServerAddr string
	Mappers    []Mapper
	Proxies    []Proxy
}

func NewClient(serverAddr string, mappers []Mapper) *Client {
	return &Client{ServerAddr: serverAddr, Mappers: mappers}
}

type Mapper struct {
	RemotePort uint16
	TargetAddr string
}

type Proxy struct {
	RemotePort uint16
	ProxyPort  uint16
}

func (c *Client) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, mapper := range c.Mappers {
		wg.Add(1)
		go startRelay(ctx, &wg, mapper, c.ServerAddr)
	}
	wg.Wait()
	log.Info("客户端已关闭")
}

func startProxy() {
	server := socks5.Server{}
	listener, err := net.Listen("tcp", ":1080")
	if err != nil {
		log.Error("监听端口失败", "err", err)
		return
	}
	server.Serve(listener)
}

func startRelay(ctx context.Context, wg *sync.WaitGroup, mapper Mapper, serverAddr string) {
	defer func() {
		wg.Done()
	}()

	conn, err := dialer.DialContext(ctx, "tcp", serverAddr)
	if err != nil {
		log.Error("连接服务器失败", "err", err)
		return
	}
	defer CloseAndLog(conn)
	go func() {
		<-ctx.Done()
		CloseAndLog(conn)
		log.Info("关闭服务端连接")
	}()

	err = binary.Write(conn, binary.BigEndian, handshakePacket{
		Port:       mapper.RemotePort,
		ServerType: relayServerType,
	})
	if err != nil {
		log.Error("发送握手包失败", "err", err)
		return
	}

	session, err := smux.Client(conn, Config)
	if err != nil {
		log.Error("创建smux会话失败", "err", err)
		return
	}
	defer CloseAndLog(session)
	go func() {
		<-ctx.Done()
		CloseAndLog(session)
		log.Info("关闭smux会话")
	}()

	log.Info(fmt.Sprintf("映射端口 %d -> %s", mapper.RemotePort, mapper.TargetAddr))

	for {
		stream, err := session.AcceptStream()
		if errors.Is(err, io.ErrClosedPipe) {
			log.Error("会话已关闭", "err", err)
			return
		} else if errors.Is(err, smux.ErrTimeout) {
			log.Error("接受smux流超时", "err", err)
		} else if err != nil {
			log.Error("接受smux流失败", "err", err)
			continue
		}
		go handleTargetConnection(ctx, stream, mapper.TargetAddr)
	}
}

func handleTargetConnection(ctx context.Context, stream *smux.Stream, targetAddr string) {
	defer CloseAndLog(stream)
	go func() {
		<-ctx.Done()
		CloseAndLog(stream)
		log.Info("关闭目标连接")
	}()

	conn, err := dialer.DialContext(ctx, "tcp", targetAddr)
	if err != nil {
		log.Error("请求目标地址失败", "err", err)
		return
	}
	defer CloseAndLog(conn)
	go func() {
		<-ctx.Done()
		CloseAndLog(conn)
		log.Info("关闭目标连接")
	}()

	go CopyStream(conn, stream)
	CopyStream(stream, conn)
}
