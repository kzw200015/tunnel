package main

import (
	"tunnel/core"
)

func main() {
	server := core.NewServer()
	server.Start(":8080")
}
