package core

import (
	"bytes"
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
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Error("监听失败", "addr", addr, "err", err)
	}

	for {
		conn, err := listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			log.Error("监听器关闭", "err", err)
			return
		} else if err != nil {
			log.Warn("接受连接失败", "err", err)
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
		log.Error("握手口超时")
		return
	case err := <-handshakeDone:
		if err != nil {
			log.Error("握手失败", "err", err)
			return
		}
	}

	if !bytes.Equal(s.tokenHash[:], handshakePacket.Token[:]) {
		log.Warn("鉴权失败")
		return
	}

	session, err := smux.Server(conn, SmuxConfig)
	if err != nil {
		log.Error("创建smux会话失败", "err", err)
		return
	}
	defer CloseAndLog(session)
	closeChan := session.CloseChan()

	portInt := int(handshakePacket.Port)
	portStr := strconv.Itoa(portInt)
	listener, err := net.Listen("tcp", ":"+portStr)
	if err != nil {
		log.Error("监听映射端口失败", "port", portInt, "err", err)
		return
	}
	log.Info("监听映射端口", "port", portInt)
	defer CloseAndLog(listener)
	go func() {
		<-closeChan
		log.Info("smux会话已关闭，停止监听映射端口", "port", portInt)
		CloseAndLog(listener)
	}()

	for {
		c, err := listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			log.Info("映射端口已关闭", "port", portInt, "err", err)
			return
		} else if err != nil {
			log.Warn("映射端口接受连接失败", "port", portInt, "err", err)
			continue
		}
		log.Info("映射端口接受连接", "port", portInt, "remoteAddr", c.RemoteAddr())
		go s.handleIncomingConnection(session, c)
	}
}

func (s *Server) handleIncomingConnection(session *smux.Session, conn net.Conn) {
	defer CloseAndLog(conn)

	stream, err := session.OpenStream()
	if err != nil {
		log.Error("创建smux流失败", "err", err)
		return
	}
	defer CloseAndLog(stream)

	go CopyStream(stream, conn)
	CopyStream(conn, stream)
}
