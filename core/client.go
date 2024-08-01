package core

import (
	"context"
	"crypto/tls"
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

var tlsDialer = tls.Dialer{
	Config: TlsConfig,
}
var dialer net.Dialer

type Client struct {
	ServerAddr string
	Token      string
	Ctx        context.Context
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

func (c *Client) Start() {
	var wg sync.WaitGroup
	if c.Relays != nil {
		for _, mapper := range c.Relays {
			wg.Add(1)
			go c.startRelay(&wg, mapper)
		}
	}
	if c.Proxies != nil {
		for _, proxy := range c.Proxies {
			wg.Add(1)
			go c.startProxy(&wg, proxy)
		}
	}
	wg.Wait()
	log.Info("client stopped")
}

func (c *Client) createSession(remotePort uint16) (*smux.Session, net.Conn, error) {
	conn, err := tlsDialer.DialContext(c.Ctx, "tcp", c.ServerAddr)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	go func() {
		<-c.Ctx.Done()
		CloseAndLog(conn)
		log.Info("close session connection")
	}()

	err = binary.Write(conn, binary.BigEndian, HandshakePacket{
		Port:  remotePort,
		Token: Hash(c.Token),
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "handshake failed")
	}

	session, err := smux.Client(conn, SmuxConfig)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}

	go func() {
		<-c.Ctx.Done()
		CloseAndLog(session)
		log.Info("close session")
	}()

	return session, conn, err
}

func (c *Client) startProxy(wg *sync.WaitGroup, proxy Proxy) {
	defer func() {
		wg.Done()
	}()

	session, conn, err := c.createSession(proxy.RemotePort)
	if err != nil {
		log.Info("create session failed", "err", err)
		return
	}
	defer CloseAndLog(conn)
	defer CloseAndLog(session)

	socks5Server := socks5.Server{}
	log.Info(fmt.Sprintf("socks server start at %d", proxy.RemotePort))
	err = socks5Server.Serve(&SmuxListener{Session: session})
	if err != nil {
		log.Error("sock5 server serve failed", "err", err)
		return
	}
}

func (c *Client) startRelay(wg *sync.WaitGroup, relay Relay) {
	defer func() {
		wg.Done()
	}()

	session, conn, err := c.createSession(relay.RemotePort)
	if err != nil {
		log.Info("create session failed", "err", err)
		return
	}
	defer CloseAndLog(conn)
	defer CloseAndLog(session)

	log.Info(fmt.Sprintf("relay server start at %d -> %s", relay.RemotePort, relay.TargetAddr))

	for {
		stream, err := session.AcceptStream()
		if errors.Is(err, io.ErrClosedPipe) {
			log.Error("session closed", "err", err)
			return
		} else if errors.Is(err, smux.ErrTimeout) {
			log.Error("accept stream timeout", "err", err)
		} else if err != nil {
			log.Error("accept stream failed", "err", err)
			continue
		}
		go c.handleTargetConnection(stream, relay.TargetAddr)
	}
}

func (c *Client) handleTargetConnection(stream *smux.Stream, targetAddr string) {
	defer CloseAndLog(stream)
	go func() {
		<-c.Ctx.Done()
		CloseAndLog(stream)
		log.Info("close target connection")
	}()

	conn, err := dialer.DialContext(c.Ctx, "tcp", targetAddr)
	if err != nil {
		log.Error("dial target failed", "err", err)
		return
	}
	defer CloseAndLog(conn)
	go func() {
		<-c.Ctx.Done()
		CloseAndLog(conn)
		log.Info("close target connection")
	}()

	go CopyStream(conn, stream)
	CopyStream(stream, conn)
}
