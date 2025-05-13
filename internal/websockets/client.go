package websockets

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 1024 * 1024 // 1MB
)

type MessageType string

const (
	TypeOrderNew        MessageType = "order.new"
	TypeOrderUpdate     MessageType = "order.update"
	TypeItemUpdate      MessageType = "item.update"
	TypeMenuUpdate      MessageType = "menu.update"
	TypeStationItems    MessageType = "station.items"
	TypeDisplayRegister MessageType = "display.register"
	TypePrinterStatus   MessageType = "printer.status"
	TypeError           MessageType = "error"
	TypePing            MessageType = "ping"
	TypePong            MessageType = "pong"
)

type ClientType string

const (
	ClientTypePOS     ClientType = "pos"
	ClientTypeKDS     ClientType = "kds"
	ClientTypeAdmin   ClientType = "admin"
	ClientTypeDisplay ClientType = "display"
	ClientTypePrinter ClientType = "printer"
)

type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data"`
	StationID string          `json:"station_id,omitempty"`
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	userID string

	clientType ClientType

	stationID string
}

func NewClient(hub *Hub, conn *websocket.Conn, userID string, clientType ClientType) *Client {
	return &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 256),
		userID:     userID,
		clientType: clientType,
	}
}

func (c *Client) SetStationID(stationID string) {
	c.stationID = stationID
	if stationID != "" {
		c.hub.RegisterStationClient(c, stationID)
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		// Process
		var wsMessage Message
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Handler
		switch wsMessage.Type {
		case TypeDisplayRegister:
			var registerData struct {
				StationID string `json:"station_id"`
			}
			if err := json.Unmarshal(wsMessage.Data, &registerData); err != nil {
				log.Printf("Error unmarshaling register data: %v", err)
				continue
			}
			c.SetStationID(registerData.StationID)

		case TypePrinterStatus:
			// Handle printer
			var statusData struct {
				PrinterID string `json:"printer_id"`
				Status    string `json:"status"`
				Error     string `json:"error,omitempty"`
			}
			if err := json.Unmarshal(wsMessage.Data, &statusData); err != nil {
				log.Printf("Error unmarshaling printer status: %v", err)
				continue
			}
			statusMsg, _ := json.Marshal(wsMessage)
			c.hub.broadcast <- statusMsg

		case TypePing:
			pongMsg, _ := json.Marshal(Message{Type: TypePong})
			c.send <- pongMsg

		default:
			// For other messages, just broadcast to all clients
			// In a production system, you'd have more sophisticated message routing
			c.hub.broadcast <- message
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for range n {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, conn *websocket.Conn, userID string, clientType ClientType) {
	client := NewClient(hub, conn, userID, clientType)

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
