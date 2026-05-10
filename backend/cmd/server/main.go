package main

import (
	"log"
	"net/http"
)

var listenAndServe = http.ListenAndServe
var fatalOnRunError = log.Fatal

func main() {
	if err := runServer(":8080"); err != nil {
		fatalOnRunError(err)
	}
}

func runServer(addr string) error {
	server, err := newConfiguredWSServer()
	if err != nil {
		return err
	}
	log.Printf("server running on %s", addr)
	return listenAndServe(addr, server.routes())
}
