package cert

import _ "embed"

const ServerName = "tunnel"

//go:embed cert.pem
var Cert []byte

//go:embed key.pem
var Key []byte
