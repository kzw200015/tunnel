package core

import (
	"github.com/xtaci/smux"
	"net"
)

type SmuxListener struct {
	Session *smux.Session
}

func (s *SmuxListener) Accept() (net.Conn, error) {
	return s.Session.AcceptStream()
}

func (s *SmuxListener) Close() error {
	return s.Session.Close()
}

func (s *SmuxListener) Addr() net.Addr {
	return s.Session.LocalAddr()
}
