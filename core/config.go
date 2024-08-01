package core

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/xtaci/smux"
	"time"
	"tunnel/cert"
)

var SmuxConfig = smux.DefaultConfig()
var TlsConfig = &tls.Config{
	MinVersion: tls.VersionTLS13, // 限制最低支持的 TLS 版本
	MaxVersion: tls.VersionTLS13,
}

func init() {
	SmuxConfig.KeepAliveInterval = 1 * time.Second
	SmuxConfig.KeepAliveTimeout = 5 * time.Second
	initTlsConfig()
}

func initTlsConfig() {
	keyPair, err := tls.X509KeyPair(cert.Cert, cert.Key)
	if err != nil {
		panic(err)
	}

	TlsConfig.Certificates = []tls.Certificate{keyPair}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(cert.Cert)
	TlsConfig.RootCAs = certPool
	TlsConfig.ServerName = cert.ServerName
}
