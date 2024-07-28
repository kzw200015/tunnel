package main

import (
	"log"
	"tunnel/core"
)

func main() {
	server := core.NewServer(":8080")
	err := server.Start()
	if err != nil {
		log.Fatalln(err)
	}
}
