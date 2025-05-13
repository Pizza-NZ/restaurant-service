package websockets

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Upgrader is the WebSocket upgrader configuration
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin controls cross-origin requests
	CheckOrigin: func(r *http.Request) bool {
		// For development, allow all origins
		// In production, you should restrict this to your specific domain(s)
		// For a Raspberry Pi deployment on local network, you might want to:
		// - Allow all origins on the local network
		// - Or configure specific allowed origins based on your setup

		// Example for production:
		// origin := r.Header.Get("Origin")
		// return origin == "http://localhost:3000" ||
		//        origin == "http://192.168.1.100:3000" ||
		//        strings.HasPrefix(origin, "http://192.168.")

		return true // Allow all origins for now
	},
	// Error handler for upgrade failures
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		// Log the error for debugging
		http.Error(w, reason.Error(), status)
	},
}

// Additional configuration options that can be set if needed:

// SetBufferSizes updates the read and write buffer sizes
func SetBufferSizes(readBufferSize, writeBufferSize int) {
	Upgrader.ReadBufferSize = readBufferSize
	Upgrader.WriteBufferSize = writeBufferSize
}

// SetCheckOrigin updates the CheckOrigin function
func SetCheckOrigin(checkOrigin func(r *http.Request) bool) {
	Upgrader.CheckOrigin = checkOrigin
}

// EnableCompression enables message compression
func EnableCompression() {
	Upgrader.EnableCompression = true
}
