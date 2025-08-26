package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserId string
	conn   *websocket.Conn
}

type Message struct {
	Type   string          `json:"type"`
	Data   json.RawMessage `json:"data"`
	UserId string          `json:"userId"`
}

var (
	ClientConns   = map[string]*Client{}
	clientConnsMu sync.RWMutex
)

func generateNumericID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(999999))
	return fmt.Sprintf("%06d", n)
}

func (c *Client) run() {
	defer func() {
		c.conn.Close()
		clientConnsMu.Lock()
		delete(ClientConns, c.UserId)
		clientConnsMu.Unlock()
	}()

	clientConnsMu.Lock()
	ClientConns[c.UserId] = c
	clientConnsMu.Unlock()

	if err := c.conn.WriteJSON(Message{Type: "INITIAL_CONNECTION", Data: json.RawMessage([]byte(c.UserId))}); err != nil {
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
			continue
		}

		clientConnsMu.RLock()
		target := ClientConns[message.UserId]
		clientConnsMu.RUnlock()
		if target == nil {
			log.Printf("Target user not found: %s", message.UserId)
			continue
		}

		out := Message{
			Type:   message.Type,
			Data:   message.Data,
			UserId: c.UserId,
		}
		dataBytes, err := json.Marshal(out)
		if err != nil {
			log.Printf("Marshal error: %v", err)
			continue
		}

		if err := target.conn.WriteMessage(websocket.TextMessage, dataBytes); err != nil {
			log.Printf("Forward error to %s: %v", message.UserId, err)
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
