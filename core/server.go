package core

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/xtaci/smux"
	log "log/slog"
	"net"
	"strconv"
	"time"
)

type Server struct {
}

func (s *Server) Start(addr string) {
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
		go handleRelayConnection(conn)
	}
}

func handleRelayConnection(conn net.Conn) {
	defer CloseAndLog(conn)

	var port uint16
	timeout := time.After(5 * time.Second)
	handshakeDone := make(chan error)
	go func() {
		err := binary.Read(conn, binary.BigEndian, &port)
		if err != nil {
			handshakeDone <- err
			return
		}
		handshakeDone <- nil
	}()
	select {
	case <-timeout:
		log.Error("读取端口超时")
		return
	case err := <-handshakeDone:
		if err != nil {
			log.Error("读取端口失败", "err", err)
			return
		} else {
			log.Info("读取端口成功", "port", port)
		}
	}

	session, err := smux.Server(conn, Config)
	if err != nil {
		log.Error("创建smux会话失败", "err", err)
		return
	}
	defer CloseAndLog(session)
	closeChan := session.CloseChan()

	portInt := int(port)
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
		go handleIncomingConnection(session, c)
	}
}

func handleIncomingConnection(session *smux.Session, conn net.Conn) {
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
