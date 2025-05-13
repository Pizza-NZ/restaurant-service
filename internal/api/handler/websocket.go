package handler

import (
	"net/http"

	"github.com/gorilla/websocket"

	"github.com/pizza-nz/restaurant-service/internal/api"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

type WebSocketHandler struct {
	hub *websockets.Hub
}

func NewWebSocketHandler(hub *websockets.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// TODO: Change for Prod
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		api.BadRequest(w, "user_id is required")
		return
	}

	clientTypeStr := r.URL.Query().Get("client_type")
	if clientTypeStr == "" {
		api.BadRequest(w, "client_type is required")
		return
	}

	clientType := websockets.ClientType(clientTypeStr)

	switch clientType {
	case websockets.ClientTypePOS, websockets.ClientTypeKDS, websockets.ClientTypeAdmin,
		websockets.ClientTypeDisplay, websockets.ClientTypePrinter:
		// Valid client type
	default:
		api.BadRequest(w, "invalid client_type")
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// If upgrading fails, the upgrader has already written the error to the response
		return
	}

	websockets.ServeWs(h.hub, conn, userID, clientType)
}
