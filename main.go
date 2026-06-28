package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	slot    int
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func (c *Client) writeJSON(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteJSON(v)
}

func (c *Client) writeMessage(messageType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(messageType, data)
}

type Message struct {
	Type   string          `json:"type"`
	Data   json.RawMessage `json:"data"`
	UserId string          `json:"userId"`
}

var (
	clients        [2]*Client
	mu             sync.Mutex
	connectedCount int
)

func (c *Client) run() {
	userID := fmt.Sprintf("user%d", c.slot)
	partnerID := fmt.Sprintf("user%d", 1-c.slot)

	defer func() {
		mu.Lock()
		clients[c.slot] = nil
		connectedCount--
		partner := clients[1-c.slot]
		mu.Unlock()
		if partner != nil {
			partner.writeJSON(Message{Type: "PEER_DISCONNECTED"})
		}
		c.conn.Close()
	}()

	// Check if both peers are now present. If this is slot 1 joining,
	// tell slot 0 (the initiator) to start the call, passing slot 1's ID.
	// If this is slot 0 and slot 1 was already waiting, tell slot 0 to start.
	mu.Lock()
	partner := clients[1-c.slot]
	mu.Unlock()

	if partner != nil {
		// Both peers are connected. The initiator is always slot 0.
		// Tell slot 0 to kick off the offer, with slot 1's ID as the target.
		initiator := clients[0]
		peerOfInitiator := fmt.Sprintf("user%d", 1) // always user1
		if c.slot == 1 {
			// This client just joined as slot 1 — tell slot 0 to start.
			peerIDBytes, _ := json.Marshal(peerOfInitiator)
			initiator.writeJSON(Message{Type: "START_CALL", Data: json.RawMessage(peerIDBytes)})
		} else {
			// This client is slot 0 and slot 1 was already here — tell slot 0 to start.
			peerIDBytes, _ := json.Marshal(partnerID)
			c.writeJSON(Message{Type: "START_CALL", Data: json.RawMessage(peerIDBytes)})
		}
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

		mu.Lock()
		partner := clients[1-c.slot]
		mu.Unlock()
		if partner == nil {
			continue
		}

		out := Message{
			Type:   message.Type,
			Data:   message.Data,
			UserId: userID,
		}
		dataBytes, err := json.Marshal(out)
		if err != nil {
			log.Printf("Marshal error: %v", err)
			continue
		}

		if err := partner.writeMessage(websocket.TextMessage, dataBytes); err != nil {
			log.Printf("Forward error to %s: %v", fmt.Sprintf("user%d", 1-c.slot), err)
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

	mu.Lock()
	if connectedCount >= 2 {
		mu.Unlock()
		conn.WriteJSON(Message{Type: "ERROR", Data: json.RawMessage(`"Server full"`)})
		conn.Close()
		return
	}
	slot := -1
	for i, c := range clients {
		if c == nil {
			slot = i
			clients[i] = &Client{slot: i, conn: conn}
			connectedCount++
			break
		}
	}
	mu.Unlock()

	go clients[slot].run()
}

func main() {
	server := http.FileServer(http.Dir("public"))
	http.Handle("/", http.StripPrefix("/", server))
	http.HandleFunc("/ws", handleWSConnections)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server started on port %v", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil); err != nil {
		log.Fatal(err)
	}
}
