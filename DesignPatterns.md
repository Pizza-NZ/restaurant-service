# Design Patterns Used in Restaurant Service System

This document outlines the architectural patterns and design patterns implemented throughout our restaurant service system. Understanding these patterns helps maintain consistency and makes the codebase more maintainable.

## 1. Repository Pattern

**Location**: `internal/db/repository/`

**Purpose**: Encapsulates data access logic and provides a more object-oriented view of the persistence layer.

**Implementation**:
```go
type UserRepository struct {
    db *sqlx.DB
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
func (r *UserRepository) Create(ctx context.Context, user models.User) (*models.User, error)
```

**Benefits**:
- Separates business logic from data access logic
- Makes it easier to test business logic by mocking repositories
- Provides a consistent interface for data operations

## 2. Service Layer Pattern

**Location**: `internal/service/`

**Purpose**: Contains business logic and orchestrates operations between repositories and handlers.

**Implementation**:
```go
type MenuService struct {
    repos *repository.Repositories
}

func (s *MenuService) CreateItem(ctx context.Context, req models.MenuItemRequest) (*models.MenuItem, error)
```

**Benefits**:
- Centralizes business logic
- Promotes reusability across different handlers
- Easier to test business logic independently

## 3. Handler Pattern (HTTP)

**Location**: `internal/api/handler/`

**Purpose**: Handles HTTP requests and responses, delegating business logic to services.

**Implementation**:
```go
type MenuHandler struct {
    menuService *service.MenuService
    hub         *websocket.Hub
}

func (h *MenuHandler) HandleMenuCategories(w http.ResponseWriter, r *http.Request)
```

**Benefits**:
- Separates HTTP concerns from business logic
- Makes it easy to adapt the same business logic to different transport protocols

## 4. Factory/Registry Pattern

**Location**: `internal/db/repository/repositories.go`

**Purpose**: Provides a centralized way to create and access repository instances.

**Implementation**:
```go
type Repositories struct {
    User    *UserRepository
    Menu    *MenuRepository
    // ... etc
}

func NewRepositories(database *db.Postgres) *Repositories
```

**Benefits**:
- Centralizes repository creation
- Provides single point of access to all repositories
- Simplifies dependency injection

## 5. Hub Pattern (WebSocket)

**Location**: `internal/websocket/hub.go`

**Purpose**: Manages WebSocket connections and message broadcasting.

**Implementation**:
```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}
```

**Benefits**:
- Centralizes WebSocket connection management
- Enables efficient message broadcasting
- Simplifies client lifecycle management

## 6. Singleton Pattern

**Location**: WebSocket Hub, Database Connection

**Purpose**: Ensures only one instance of a resource exists throughout the application.

**Implementation**:
- Database connection created once in main.go
- WebSocket hub created once and shared across handlers

**Benefits**:
- Resource efficiency
- Consistent state management
- Prevents resource conflicts

## 7. Middleware Pattern

**Location**: `internal/middleware/`

**Purpose**: Provides cross-cutting concerns like authentication, logging, and request processing.

**Implementation**:
```go
func Auth(authService *service.AuthService) func(http.Handler) http.Handler
func Logger(next http.Handler) http.Handler
```

**Benefits**:
- Separates concerns (authentication, logging)
- Reusable across different routes
- Easy to add/remove functionality

## 8. Context Pattern

**Location**: Throughout the application

**Purpose**: Passes request-scoped values and cancellation signals through the call stack.

**Implementation**:
```go
ctx := context.WithValue(req.Context(), UserIDKey, userID)
userID, ok := middleware.GetUserID(ctx)
```

**Benefits**:
- Request-scoped data propagation
- Cancellation support
- Standard Go pattern for handling timeouts

## 9. Error Handling Pattern

**Location**: Throughout the application

**Purpose**: Consistent error handling and propagation.

**Implementation**:
```go
if err != nil {
    return nil, fmt.Errorf("failed to create user: %w", err)
}
```

**Benefits**:
- Error context preservation
- Consistent error messages
- Easy debugging with error wrapping

## 10. Configuration Pattern

**Location**: `internal/config/`

**Purpose**: Centralizes application configuration.

**Implementation**:
```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`
    JWT      JWTConfig     `yaml:"jwt"`
}
```

**Benefits**:
- Environment-specific configurations
- Type-safe configuration access
- Easy to modify without code changes

## 11. DTO (Data Transfer Object) Pattern

**Location**: `internal/models/`

**Purpose**: Separates internal domain models from API request/response structures.

**Implementation**:
```go
type User struct {           // Domain model
    ID           uuid.UUID
    PasswordHash string    // Never exposed
}

type UserRequest struct {    // DTO for requests
    Username string `json:"username"`
    Password string `json:"password"`
}
```

**Benefits**:
- API flexibility without affecting domain models
- Security (hiding sensitive fields)
- Validation at boundaries

## 12. Transaction Pattern

**Location**: Repository methods requiring multiple operations

**Purpose**: Ensures data consistency for operations that must succeed or fail together.

**Implementation**:
```go
tx, err := r.db.BeginTxx(ctx, nil)
if err != nil {
    return nil, err
}
defer func() {
    if err != nil {
        _ = tx.Rollback()
    }
}()
// ... operations
err = tx.Commit()
```

**Benefits**:
- Data integrity
- Atomic operations
- Rollback on failure

## 13. Observer Pattern (via WebSockets)

**Location**: WebSocket implementation for real-time updates

**Purpose**: Notifies multiple clients when state changes occur.

**Implementation**:
- Clients subscribe to WebSocket connections
- Server broadcasts updates when orders/menu items change
- Clients receive real-time notifications

**Benefits**:
- Real-time updates
- Decoupled communication
- Scalable notification system

## 14. Builder Pattern (Implicit)

**Location**: Order creation process

**Purpose**: Constructs complex objects step by step.

**Implementation**:
```go
// Order is built with items, modifiers, pricing calculations
order := models.Order{
    UserID:      userID,
    OrderNumber: orderNumber,
}
// Add items, calculate totals, apply routing
```

**Benefits**:
- Complex object construction
- Flexible order composition
- Step-by-step validation

## 15. Strategy Pattern (Routing)

**Location**: Order routing to stations

**Purpose**: Defines different algorithms for routing items to preparation stations.

**Implementation**:
- Each menu item has routing rules
- System selects appropriate station based on rules
- Fallback mechanisms for printer failures

**Benefits**:
- Flexible routing logic
- Easy to add new routing strategies
- Runtime algorithm selection

## Architecture Principles

### 1. Separation of Concerns
- Each package has a specific responsibility
- Clear boundaries between layers

### 2. Dependency Injection
- Dependencies passed through constructors
- Interfaces used for flexibility
- Easy to test with mocks

### 3. Error Propagation
- Errors wrapped with context
- Consistent error handling
- Clear error messages

### 4. Resource Management
- Proper closing of database connections
- Graceful shutdown handling
- Connection pooling for efficiency

### 5. Security by Design
- Authentication required by default
- Role-based access control
- Sensitive data never exposed

## Best Practices Implemented

1. **Context Usage**: All operations accept context for cancellation and timeout support
2. **Structured Logging**: Consistent log format and levels
3. **Graceful Shutdown**: Proper cleanup of resources on shutdown
4. **Configuration Management**: Environment-based configuration
5. **Database Migrations**: Version-controlled schema changes
6. **Real-time Updates**: WebSocket integration for live data
7. **Error Handling**: Consistent error wrapping and propagation
8. **Testing Structure**: Separation allows easy unit testing

## Future Considerations

1. **Circuit Breaker Pattern**: For printer operations
2. **Retry Pattern**: For network operations
3. **Caching Pattern**: For frequently accessed menu data
4. **Event Sourcing**: For order history and audit trails
5. **CQRS**: Separate read/write models for optimization