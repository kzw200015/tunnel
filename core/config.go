package core

import (
	"github.com/xtaci/smux"
	"time"
)

var Config = smux.DefaultConfig()

func init() {
	Config.KeepAliveInterval = 1 * time.Second
	Config.KeepAliveTimeout = 5 * time.Second
}
