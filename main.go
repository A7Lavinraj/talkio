package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserId string
	conn   *websocket.Conn
}

type Message struct {
	Type   string `json:"type"`
	Data   any    `json:"data"`
	UserId string `json:"userId"`
}

var ClientConns = map[string]*Client{}

func generateNumericID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(999999))
	return fmt.Sprintf("%06d", n)
}

func (c *Client) run() {
	defer c.conn.Close()

	ClientConns[c.UserId] = c

	if err := c.conn.WriteJSON(Message{Type: "INITIAL_CONNECTION", Data: c.UserId}); err != nil {
		log.Println("Unable to send initial message")
	}

	for {
		mt, msg, err := c.conn.ReadMessage()
		if err != nil || mt == websocket.CloseMessage {
			break
		}

		var message Message
		if err := json.Unmarshal(msg, &message); err != nil {
			log.Printf("JSON parse error: %v", err)
		} else {
			if message.Type == "PEER_CONNECTION_REQUEST" {
				Client := ClientConns[message.UserId]
				dataBytes, err := json.Marshal(Message{Type: "PEER_CONNECTION_REQUEST", Data: message.Data, UserId: c.UserId})
				if err != nil {
					fmt.Printf("Failed to marshal message.Data: %v\n", err)
					continue
				}

				if err := Client.conn.WriteMessage(websocket.TextMessage, dataBytes); err != nil {
					fmt.Printf("Failed to transfer PEER_CONNECTION_REQUEST to userId: %s\n", message.UserId)
				}
			} else if message.Type == "PEER_CONNECTION_RESPONSE" {
				Client := ClientConns[message.UserId]
				dataBytes, err := json.Marshal(Message{Type: "PEER_CONNECTION_RESPONSE", Data: message.Data, UserId: c.UserId})
				if err != nil {
					fmt.Printf("Failed to marshal message.Data: %v\n", err)
					continue
				}

				if err := Client.conn.WriteMessage(websocket.TextMessage, dataBytes); err != nil {
					fmt.Printf("Failed to transfer PEER_CONNECTION_REQUEST to userId: %s\n", message.UserId)
				}
			} else {
				fmt.Printf("Invalid message type %s", message.Type)
			}
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func handleWSConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	go (&Client{UserId: generateNumericID(), conn: conn}).run()
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
