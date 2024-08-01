package core

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/xtaci/smux"
	log "log/slog"
	"net"
	"strconv"
	"time"
)

type Server struct {
	Token     string
	tokenHash [32]byte
}

func (s *Server) Start(addr string) {
	s.tokenHash = Hash(s.Token)
	listener, err := tls.Listen("tcp", addr, TlsConfig)
	if err != nil {
		log.Error("listen failed", "addr", addr, "err", err)
	}

	for {
		conn, err := listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			log.Error("listener closed", "err", err)
			return
		} else if err != nil {
			log.Warn("accept connection failed", "err", err)
			continue
		}
		go s.handleRelayConnection(conn)
	}
}

func (s *Server) handleRelayConnection(conn net.Conn) {
	defer CloseAndLog(conn)

	handshakePacket := HandshakePacket{}
	timeout := time.After(5 * time.Second)
	handshakeDone := make(chan error)
	go func() {
		err := binary.Read(conn, binary.BigEndian, &handshakePacket)
		if err != nil {
			handshakeDone <- err
			return
		}
		handshakeDone <- nil
	}()
	select {
	case <-timeout:
		log.Error("handshake timeout")
		return
	case err := <-handshakeDone:
		if err != nil {
			log.Error("handshake failed", "err", err)
			return
		}
	}

	if !bytes.Equal(s.tokenHash[:], handshakePacket.Token[:]) {
		log.Warn("validate token failed")
		return
	}

	session, err := smux.Server(conn, SmuxConfig)
	if err != nil {
		log.Error("create smux session failed", "err", err)
		return
	}
	defer CloseAndLog(session)
	closeChan := session.CloseChan()

	portInt := int(handshakePacket.Port)
	portStr := strconv.Itoa(portInt)
	listener, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		log.Error("listen remote port failed", "port", portInt, "err", err)
		return
	}
	log.Info("listen remote port", "port", portInt)
	defer CloseAndLog(listener)
	go func() {
		<-closeChan
		log.Info("smux session closed, stop listen remote port", "port", portInt)
		CloseAndLog(listener)
	}()

	for {
		c, err := listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			log.Info("remote port listener closed", "port", portInt, "err", err)
			return
		} else if err != nil {
			log.Warn("remote port accept connection failed", "port", portInt, "err", err)
			continue
		}
		log.Info("remote port accept connection", "port", portInt, "remoteAddr", c.RemoteAddr())
		go s.handleIncomingConnection(session, c)
	}
}

func (s *Server) handleIncomingConnection(session *smux.Session, conn net.Conn) {
	defer CloseAndLog(conn)

	stream, err := session.OpenStream()
	if err != nil {
		log.Error("open stream failed", "err", err)
		return
	}
	defer CloseAndLog(stream)

	go CopyStream(stream, conn)
	CopyStream(conn, stream)
}
