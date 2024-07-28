package core

import (
	"io"
	log "log/slog"
)

func CloseAndLog(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Debug("close error", "err", err)
	}
}

func CopyStream(dst io.Writer, src io.Reader) {
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Warn("copy error", "err", err)
	}
}

var relayServerType = uint8(0)
var proxyServerType = uint8(0)

type handshakePacket struct {
	Port       uint16
	ServerType uint8
}
