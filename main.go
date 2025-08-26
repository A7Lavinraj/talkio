package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
}

type Message struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

var ClientConns = map[string]*Client{}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func generateNumericID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(999999))
	return fmt.Sprintf("%06d", n)
}

func handleWSConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	var uuid = generateNumericID()
	ClientConns[uuid] = &Client{conn: conn}

	if err := conn.WriteJSON(Message{Type: "INITIAL_CONNECTION", Data: uuid}); err != nil {
		log.Println("Unable to send initial message")
	}

	for {
		mt, msg, err := conn.ReadMessage()
		if err != nil || mt == websocket.CloseMessage {
			break
		}

		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}

	conn.Close()
}

func main() {
	server := http.FileServer(http.Dir("public"))

	http.Handle("/", http.StripPrefix("/", server))
	http.HandleFunc("/ws", handleWSConnections)

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
