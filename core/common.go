package core

import (
	"io"
	log "log/slog"
	"strconv"
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

type HandshakePacket struct {
	Token [32]byte
	Port  uint16
}

func ParsePort(s string) (uint16, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return uint16(i), err
}
