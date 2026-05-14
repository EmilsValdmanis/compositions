package main

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
)

var listenAndServe = http.ListenAndServe
var fatalOnRunError = log.Fatal

func main() {
	configureLogger(os.Stdout)
	if err := runServer(":8080"); err != nil {
		fatalOnRunError(err)
	}
}

func configureLogger(output io.Writer) {
	slog.SetDefault(slog.New(slog.NewTextHandler(output, nil)))
}

func runServer(addr string) error {
	server, err := newConfiguredWSServer()
	if err != nil {
		return err
	}
	slog.Info("server starting", "addr", addr, "allowedOrigin", server.allowedOrigin)
	return listenAndServe(addr, server.routes())
}
