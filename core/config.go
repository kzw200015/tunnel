package core

import (
	"github.com/xtaci/smux"
	"time"
)

var SmuxConfig = smux.DefaultConfig()

func init() {
	SmuxConfig.KeepAliveInterval = 1 * time.Second
	SmuxConfig.KeepAliveTimeout = 5 * time.Second
}
