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
	Relays     []Relay
	Proxies    []Proxy
}

type Relay struct {
	RemotePort uint16
	TargetAddr string
}

type Proxy struct {
	RemotePort uint16
}

func (c *Client) Start(ctx context.Context) {
	var wg sync.WaitGroup
	if c.Relays != nil {
		for _, mapper := range c.Relays {
			wg.Add(1)
			go startRelay(ctx, &wg, mapper, c.ServerAddr)
		}
	}
	if c.Proxies != nil {
		for _, proxy := range c.Proxies {
			wg.Add(1)
			go startProxy(ctx, &wg, proxy, c.ServerAddr)
		}
	}
	wg.Wait()
	log.Info("客户端已关闭")
}

func createSession(ctx context.Context, remotePort uint16, serverAddr string) (*smux.Session, net.Conn, error) {
	conn, err := dialer.DialContext(ctx, "tcp", serverAddr)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	go func() {
		<-ctx.Done()
		CloseAndLog(conn)
		log.Info("关闭服务端连接")
	}()

	err = binary.Write(conn, binary.BigEndian, remotePort)
	if err != nil {
		return nil, nil, errors.Wrap(err, "发送握手包失败")
	}

	session, err := smux.Client(conn, Config)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	go func() {
		<-ctx.Done()
		CloseAndLog(session)
		log.Info("关闭smux会话")
	}()

	return session, conn, err
}

func startProxy(ctx context.Context, wg *sync.WaitGroup, proxy Proxy, serverAddr string) {
	defer func() {
		wg.Done()
	}()

	session, conn, err := createSession(ctx, proxy.RemotePort, serverAddr)
	if err != nil {
		log.Info("创建smux会话失败", "err", err)
		return
	}
	defer CloseAndLog(conn)
	defer CloseAndLog(session)

	socks5Server := socks5.Server{}
	err = socks5Server.Serve(&SmuxListener{Session: session})
	if err != nil {
		log.Error("启动socks5服务失败", "err", err)
		return
	}
}

func startRelay(ctx context.Context, wg *sync.WaitGroup, relay Relay, serverAddr string) {
	defer func() {
		wg.Done()
	}()

	session, conn, err := createSession(ctx, relay.RemotePort, serverAddr)
	if err != nil {
		log.Info("创建smux会话失败", "err", err)
		return
	}
	defer CloseAndLog(conn)
	defer CloseAndLog(session)

	log.Info(fmt.Sprintf("映射端口 %d -> %s", relay.RemotePort, relay.TargetAddr))

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
		go handleTargetConnection(ctx, stream, relay.TargetAddr)
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
