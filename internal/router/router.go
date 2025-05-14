package router

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pizza-nz/restaurant-service/internal/api/handler"
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
	// Initialize services
	menuService := service.NewMenuService(r.repos)
	printerService := service.NewPrinterService(r.repos)
	orderService := service.NewOrderService(r.repos, printerService)
	stationService := service.NewStationService(r.repos)

	// Initialize handlers
	menuHandler := handler.NewMenuHandler(menuService, r.hub)
	orderHandler := handler.NewOrderHandler(orderService, r.hub)
	stationHandler := handler.NewStationHandler(stationService, r.hub)
	printerHandler := handler.NewPrinterHandler(printerService, r.hub)
	userHandler := handler.NewUserHandler(r.auth)

	// Public routes
	r.mux.HandleFunc("/api/auth/login", r.handleLogin)
	r.mux.HandleFunc("/ws", r.handleWebSocket)

	// Protected routes
	r.mux.HandleFunc("/api/menu/categories", r.withAuth(menuHandler.HandleMenuCategories))
	r.mux.HandleFunc("/api/menu/categories/", r.withAuth(menuHandler.HandleMenuCategories))
	r.mux.HandleFunc("/api/menu/items", r.withAuth(menuHandler.HandleMenuItems))
	r.mux.HandleFunc("/api/menu/items/", r.withAuth(menuHandler.HandleMenuItems))
	r.mux.HandleFunc("/api/modifiers", r.withAuth(menuHandler.HandleModifiers))
	r.mux.HandleFunc("/api/modifiers/", r.withAuth(menuHandler.HandleModifiers))

	r.mux.HandleFunc("/api/orders", r.withAuth(orderHandler.HandleOrders))
	r.mux.HandleFunc("/api/orders/", r.withAuth(orderHandler.HandleOrders))
	r.mux.HandleFunc("/api/order-items/", r.withAuth(orderHandler.HandleOrderItems))
	r.mux.HandleFunc("/api/stations", r.withAuth(stationHandler.HandleStations))
	r.mux.HandleFunc("/api/stations/", r.withAuth(stationHandler.HandleStations))
	r.mux.HandleFunc("/api/stations/", r.withAuth(orderHandler.HandleStationItems))
	r.mux.HandleFunc("/api/routing", r.withAuth(stationHandler.HandleRoutingRules))
	r.mux.HandleFunc("/api/routing/", r.withAuth(stationHandler.HandleRoutingRules))

	r.mux.HandleFunc("/api/printers", r.withAuth(printerHandler.HandlePrinters))
	r.mux.HandleFunc("/api/printers/", r.withAuth(printerHandler.HandlePrinters))
	r.mux.HandleFunc("/api/displays", r.withAuth(printerHandler.HandleDisplays))
	r.mux.HandleFunc("/api/displays/", r.withAuth(printerHandler.HandleDisplays))

	// Admin-only routes
	r.mux.HandleFunc("/api/users", r.withAuth(r.withRole(models.RoleAdmin, userHandler.HandleUsers)))
	r.mux.HandleFunc("/api/users/", r.withAuth(r.withRole(models.RoleAdmin, userHandler.HandleUsers)))
}

// withAuth is middleware for authentication
func (r *Router) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Get the Authorization header
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		// Get the token
		tokenString := parts[1]

		// Validate the token
		claims, err := r.auth.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Parse the user ID
		userID := claims.UserID
		userRole := claims.Role

		// Add user info to context
		ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
		ctx = context.WithValue(ctx, middleware.UserRoleKey, userRole)

		// Call the next handler with the updated context
		next(w, req.WithContext(ctx))
	}
}

// withRole is middleware for role-based authorization
func (r *Router) withRole(role models.UserRole, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// Get the role from context
		roleValue := req.Context().Value(middleware.UserRoleKey)
		if roleValue == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userRole := models.UserRole(roleValue.(string))

		// Check if the role is allowed
		if userRole != role {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Call the next handler
		next(w, req)
	}
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
