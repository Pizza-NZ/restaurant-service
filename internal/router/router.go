// internal/router/router.go
package router

import (
	"encoding/json"
	"net/http"

	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/middleware"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/pizza-nz/restaurant-service/internal/service"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

// Router handles HTTP routing
type Router struct {
	mux      *http.ServeMux
	repos    *repository.Repositories
	auth     *service.AuthService
	hub      *websockets.Hub
	notFound http.Handler
}

// New creates a new router
func New(repos *repository.Repositories, auth *service.AuthService, hub *websockets.Hub) *Router {
	r := &Router{
		mux:      http.NewServeMux(),
		repos:    repos,
		auth:     auth,
		hub:      hub,
		notFound: http.NotFoundHandler(),
	}

	// Set up routes
	r.setupRoutes()

	return r
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// setupRoutes sets up the routes for the router
func (r *Router) setupRoutes() {
	// Public routes
	r.mux.Handle("/api/auth/login", http.HandlerFunc(r.handleLogin))
	r.mux.Handle("/ws", http.HandlerFunc(r.handleWebSocket))

	// Protected routes
	apiHandler := http.NewServeMux()
	// apiHandler.Handle("/users", r.requireRole(models.RoleAdmin, http.HandlerFunc(r.handleUsers)))
	// apiHandler.Handle("/menu/categories", http.HandlerFunc(r.handleMenuCategories))
	// apiHandler.Handle("/menu/items", http.HandlerFunc(r.handleMenuItems))
	// apiHandler.Handle("/orders", http.HandlerFunc(r.handleOrders))
	// apiHandler.Handle("/stations", http.HandlerFunc(r.handleStations))
	// apiHandler.Handle("/printers", http.HandlerFunc(r.handlePrinters))

	// Apply middleware to protected routes
	apiChain := middleware.Logger(
		middleware.Auth(r.auth)(
			apiHandler,
		),
	)

	r.mux.Handle("/api/", http.StripPrefix("/api", apiChain))
}

// requireRole creates a middleware that checks if the user has the required role
func (r *Router) requireRole(role models.UserRole, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		userRole, ok := middleware.GetUserRole(req.Context())
		if !ok || userRole != role {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, req)
	})
}

// handleLogin handles user login
func (r *Router) handleLogin(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	// Decode the request body
	if err := json.NewDecoder(req.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Attempt to login
	token, user, err := r.auth.Login(req.Context(), loginReq.Username, loginReq.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Return the token and user info
	response := struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	}{
		Token: token,
		User:  *user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleWebSocket handles WebSocket connections
func (r *Router) handleWebSocket(w http.ResponseWriter, req *http.Request) {
	// Get user ID and client type from the request
	userID := req.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	clientTypeStr := req.URL.Query().Get("client_type")
	if clientTypeStr == "" {
		http.Error(w, "client_type is required", http.StatusBadRequest)
		return
	}

	clientType := websockets.ClientType(clientTypeStr)

	// Validate client type
	switch clientType {
	case websockets.ClientTypePOS, websockets.ClientTypeKDS, websockets.ClientTypeAdmin,
		websockets.ClientTypeDisplay, websockets.ClientTypePrinter:
		// Valid client type
	default:
		http.Error(w, "invalid client_type", http.StatusBadRequest)
		return
	}

	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := websockets.Upgrader.Upgrade(w, req, nil)
	if err != nil {
		// If upgrading fails, the upgrader has already written the error to the response
		return
	}

	// Handle the WebSocket connection
	websockets.ServeWs(r.hub, conn, userID, clientType)
}

// The following handler functions would be implemented based on your needs:
// handleUsers
// handleMenuCategories
// handleMenuItems
// handleOrders
// handleStations
// handlePrinters
