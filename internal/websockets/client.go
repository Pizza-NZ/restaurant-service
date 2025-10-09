package websockets

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10

	maxMessageSize = 1024 * 1024 // 1MB
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

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
		if err := c.conn.Close(); err != nil {
				logger.Error("Failed to close connection", "error", err)
			}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logger.Error("Failed to set read deadline", "error", err)
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			logger.Error("Failed to set read deadline", "error", err)
		}
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("websocket error", "error", err)
			}
			break
		}

		// Process
		var wsMessage Message
		if err := json.Unmarshal(message, &wsMessage); err != nil {
			logger.Error("Error unmarshaling message", "error", err)
			continue
		}

		// Handler
		switch wsMessage.Type {
		case TypeDisplayRegister:
			var registerData struct {
				StationID string `json:"station_id"`
			}
			if err := json.Unmarshal(wsMessage.Data, &registerData); err != nil {
				logger.Error("Error unmarshaling register data", "error", err)
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
				logger.Error("Error unmarshaling printer status", "error", err)
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
		if err := c.conn.Close(); err != nil {
			logger.Error("Failed to close connection", "error", err)
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Error("Failed to set write deadline", "error", err)
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logger.Error("Failed to write close message", "error", err)
				}
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				logger.Error("Failed to write message", "error", err)
			}

			n := len(c.send)
			for range n {
				if _, err := w.Write([]byte{'\n'}); err != nil {
					logger.Error("Failed to write newline", "error", err)
				}
				if _, err := w.Write(<-c.send); err != nil {
					logger.Error("Failed to write message from send channel", "error", err)
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Error("Failed to set write deadline", "error", err)
			}
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
