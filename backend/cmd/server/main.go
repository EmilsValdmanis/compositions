package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	// "github.com/EmilsValdmanis/compositions/backend/internal/game"
)

type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	log.Println("client connected:", conn.LocalAddr())

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("client disconnected")
			break
		}

		var msg Message

		if err := json.Unmarshal(message, &msg); err != nil {
			log.Println("invalid json")
			continue
		}

		switch msg.Type {
		case "ping":
			response := Message{
				Type: "pong",
				Data: "hellow from server",
			}
			conn.WriteJSON(response)
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWS)
	log.Println("server running on :8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
