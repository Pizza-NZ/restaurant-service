package router

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/pizza-nz/restaurant-service/internal/api"
	"github.com/pizza-nz/restaurant-service/internal/api/handler"
	"github.com/pizza-nz/restaurant-service/internal/db/repository"
	"github.com/pizza-nz/restaurant-service/internal/messaging"
	"github.com/pizza-nz/restaurant-service/internal/middleware"
	"github.com/pizza-nz/restaurant-service/internal/models"
	"github.com/pizza-nz/restaurant-service/internal/service"
	"github.com/pizza-nz/restaurant-service/internal/websockets"
)

// New creates and configures the main application router and its middleware.
// It returns a http.Handler, making it compatible with the standard net/http server.
func New(repos *repository.Repositories, authService *service.AuthService, hub *websockets.Hub) http.Handler {
	mux := http.NewServeMux()

	// --- Initialize Event Publisher ---
	publisher, err := messaging.NewEventPublisher()
	if err != nil {
		log.Fatalf("Failed to create event publisher: %v", err)
	}

	// --- Initialize Services (with specific repository interfaces) ---
	// Each service now only gets the repository it needs, satisfying the new interfaces.
	menuService := service.NewMenuService(repos.Menu, repos.Station)
	printerService := service.NewPrinterService(repos.Printer, repos.Printer, repos.Order, repos.Station)
	orderService := service.NewOrderService(repos.Order, publisher)
	stationService := service.NewStationService(repos.Station, repos.Printer, repos.Printer)

	// --- Initialize Handlers ---
	// The presentation layer that handles HTTP requests
	menuHandler := handler.NewMenuHandler(menuService, hub)
	orderHandler := handler.NewOrderHandler(orderService, hub)
	stationHandler := handler.NewStationHandler(stationService, hub)
	printerHandler := handler.NewPrinterHandler(printerService, hub)
	userHandler := handler.NewUserHandler(authService)
	webSocketHandler := handler.NewWebSocketHandler(hub)

	// --- Define Middleware Chains ---
	authMiddleware := middleware.Auth(authService)

	// Admin-only access chain
	adminChain := func(h http.Handler) http.Handler {
		return authMiddleware(middleware.RequireRole(models.RoleAdmin)(h))
	}

	// Manager or Admin access chain
	managerChain := func(h http.Handler) http.Handler {
		return authMiddleware(middleware.RequireRole(models.RoleAdmin, models.RoleManager)(h))
	}

	// --- Public Routes (No Authentication Required) ---
	mux.HandleFunc("POST /api/auth/login", handleLogin(authService))
	mux.Handle("/ws", http.HandlerFunc(webSocketHandler.ServeHTTP))
	mux.Handle("/metrics", promhttp.Handler())

	// --- Protected Routes (Authentication Required) ---

	// Order routes (accessible by cashiers, managers, admins)
	orderChain := authMiddleware(middleware.RequireRole(models.RoleAdmin, models.RoleManager, models.RoleCashier)(http.HandlerFunc(orderHandler.HandleOrders)))
	mux.Handle("/api/orders", orderChain)
	mux.Handle("/api/orders/", orderChain)

	orderItemChain := authMiddleware(middleware.RequireRole(models.RoleAdmin, models.RoleManager, models.RoleCashier, models.RoleKitchen)(http.HandlerFunc(orderHandler.HandleOrderItems)))
	mux.Handle("/api/order-items/", orderItemChain)

	// Station routes
	mux.Handle("/api/stations", authMiddleware(http.HandlerFunc(stationHandler.HandleStations)))
	mux.Handle("/api/stations/", authMiddleware(http.HandlerFunc(stationHandler.HandleStations)))
	mux.Handle("GET /api/stations/{id}/items", authMiddleware(http.HandlerFunc(orderHandler.HandleStationItems))) // Note: PathValue based routing
	mux.Handle("/api/routing/", managerChain(http.HandlerFunc(stationHandler.HandleRoutingRules)))

	// Menu routes
	mux.Handle("/api/menu/categories", authMiddleware(http.HandlerFunc(menuHandler.HandleMenuCategories)))
	mux.Handle("/api/menu/categories/", authMiddleware(http.HandlerFunc(menuHandler.HandleMenuCategories)))
	mux.Handle("/api/menu/items", authMiddleware(http.HandlerFunc(menuHandler.HandleMenuItems)))
	mux.Handle("/api/menu/items/", authMiddleware(http.HandlerFunc(menuHandler.HandleMenuItems)))
	mux.Handle("/api/modifiers", managerChain(http.HandlerFunc(menuHandler.HandleModifiers)))
	mux.Handle("/api/modifiers/", managerChain(http.HandlerFunc(menuHandler.HandleModifiers)))

	// Hardware Management routes (Printers, Displays) - requires manager access
	mux.Handle("/api/printers", managerChain(http.HandlerFunc(printerHandler.HandlePrinters)))
	mux.Handle("/api/printers/", managerChain(http.HandlerFunc(printerHandler.HandlePrinters)))
	mux.Handle("/api/displays", managerChain(http.HandlerFunc(printerHandler.HandleDisplays)))
	mux.Handle("/api/displays/", managerChain(http.HandlerFunc(printerHandler.HandleDisplays)))

	// User Management routes (Admin only)
	userChain := adminChain(http.HandlerFunc(userHandler.HandleUsers))
	mux.Handle("/api/users", userChain)
	mux.Handle("/api/users/", userChain)

	// Wrap the entire mux with the request logger middleware. This ensures all requests are logged.
	return middleware.Tracing(middleware.Metrics(middleware.Logger(mux)))
}

// handleLogin is a closure that handles the user login process.
func handleLogin(authService *service.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
			api.Error(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		token, user, err := authService.Login(r.Context(), loginReq.Username, loginReq.Password)
		if err != nil {
			api.Error(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		response := struct {
			Token string       `json:"token"`
			User  *models.User `json:"user"`
		}{
			Token: token,
			User:  user,
		}

		api.JSON(w, http.StatusOK, response)
	}
}
