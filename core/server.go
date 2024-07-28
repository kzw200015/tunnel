package core

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/xtaci/smux"
	"log"
	"net"
	"strconv"
)

type Server struct {
	Addr     string
	Sessions []*smux.Session
}

func NewServer(addr string) *Server {
	s := &Server{
		Addr: addr,
	}

	return s
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println()
			continue
		}
		go handleRelayConnection(conn)
	}
}

func handleRelayConnection(conn net.Conn) {
	defer CloseAndLog(conn)

	var port uint16
	err := binary.Read(conn, binary.BigEndian, &port)
	if err != nil {
		log.Println(errors.WithStack(err))
		return
	}

	session, err := smux.Server(conn, nil)
	if err != nil {

	}
	defer CloseAndLog(session)

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(int(port)))
	if err != nil {
		log.Println(errors.Wrap(err, "监听服务端口失败"))
		return
	}

	closeChan := session.CloseChan()
	go func() {
		<-closeChan
		CloseAndLog(listener)
	}()

	for {
		c, err := listener.Accept()
		if err != nil {
			log.Println(errors.Wrap(err, "服务端口接受连接失败"))
			continue
		}
		go handleIncomingConnection(session, c)
	}
}

func handleIncomingConnection(session *smux.Session, conn net.Conn) {
	defer CloseAndLog(conn)
	stream, err := session.OpenStream()
	if err != nil {
		log.Println(errors.Wrap(err, "打开smux会话失败"))
		return
	}

	go CopyStream(stream, conn)
	CopyStream(conn, stream)
}
