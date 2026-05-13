package main

import (
	"log"
	"log/slog"
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
	slog.Info("server starting", "addr", addr, "allowedOrigin", server.allowedOrigin)
	return listenAndServe(addr, server.routes())
}
