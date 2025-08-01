# Web Service Example

This example shows how to implement comprehensive logging in a web service using zlog's signal-based approach.

## Complete Web Service Setup

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/gorilla/mux"
    "github.com/zoobzio/zlog"
)

// Define our application signals
const (
    // HTTP request lifecycle
    HTTP_REQUEST_STARTED   = "HTTP_REQUEST_STARTED"
    HTTP_REQUEST_COMPLETED = "HTTP_REQUEST_COMPLETED"
    HTTP_REQUEST_FAILED    = "HTTP_REQUEST_FAILED"
    
    // Business events
    USER_REGISTERED = "USER_REGISTERED"
    USER_LOGIN      = "USER_LOGIN"
    USER_LOGOUT     = "USER_LOGOUT"
    
    // Application lifecycle
    SERVER_STARTING = "SERVER_STARTING"
    SERVER_READY    = "SERVER_READY"
    SERVER_SHUTDOWN = "SERVER_SHUTDOWN"
    
    // Error conditions
    DATABASE_ERROR       = "DATABASE_ERROR"
    VALIDATION_ERROR     = "VALIDATION_ERROR"
    AUTHENTICATION_ERROR = "AUTHENTICATION_ERROR"
)

// User represents a user in our system
type User struct {
    ID       string `json:"id"`
    Username string `json:"username"`
    Email    string `json:"email"`
}

// Request/Response types
type RegisterRequest struct {
    Username string `json:"username"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

func main() {
    // Setup logging based on environment
    setupLogging()
    
    // Initialize the server
    server := NewServer()
    
    // Start server
    port := getEnvOrDefault("PORT", "8080")
    zlog.Emit(SERVER_STARTING, "Starting web server",
        zlog.String("port", port),
        zlog.String("env", getEnvOrDefault("ENV", "development")))
    
    if err := server.Start(":" + port); err != nil {
        zlog.Fatal("Server failed to start",
            zlog.Err(err),
            zlog.String("port", port))
    }
}

func setupLogging() {
    env := getEnvOrDefault("ENV", "development")
    
    switch env {
    case "production":
        setupProductionLogging()
    case "staging":
        setupStagingLogging()
    default:
        setupDevelopmentLogging()
    }
}

func setupDevelopmentLogging() {
    // Development: console output with debug level
    zlog.EnableStandardLogging(zlog.DEBUG)
    
    // Pretty console sink for development
    consoleSink := zlog.NewSink("console", func(ctx context.Context, event zlog.Event) error {
        timestamp := event.Time.Format("15:04:05")
        fmt.Printf("= [%s] %s: %s\n", timestamp, event.Signal, event.Message)
        
        for _, field := range event.Fields {
            if field.Key != "timestamp" {
                fmt.Printf("   %s: %v\n", field.Key, field.Value)
            }
        }
        return nil
    })
    
    // Route all business events to console
    zlog.RouteSignal(HTTP_REQUEST_STARTED, consoleSink)
    zlog.RouteSignal(HTTP_REQUEST_COMPLETED, consoleSink)
    zlog.RouteSignal(USER_REGISTERED, consoleSink)
    zlog.RouteSignal(USER_LOGIN, consoleSink)
}

func setupProductionLogging() {
    // Production: structured JSON to stderr
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Setup metrics collection
    setupMetrics()
    
    // Setup audit logging
    setupAuditLogging()
    
    // Setup error alerting
    setupErrorAlerting()
}

// Server represents our web server
type Server struct {
    router *mux.Router
    users  map[string]*User // Simple in-memory store for demo
}

func NewServer() *Server {
    s := &Server{
        router: mux.NewRouter(),
        users:  make(map[string]*User),
    }
    
    s.setupRoutes()
    s.setupMiddleware()
    
    return s
}

func (s *Server) setupRoutes() {
    // API routes
    api := s.router.PathPrefix("/api/v1").Subrouter()
    api.HandleFunc("/register", s.handleRegister).Methods("POST")
    api.HandleFunc("/login", s.handleLogin).Methods("POST")
    api.HandleFunc("/logout", s.handleLogout).Methods("POST")
    api.HandleFunc("/users/{id}", s.handleGetUser).Methods("GET")
    
    // Health check
    s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
}

func (s *Server) setupMiddleware() {
    // Logging middleware
    s.router.Use(s.loggingMiddleware)
    
    // Recovery middleware
    s.router.Use(s.recoveryMiddleware)
}

func (s *Server) Start(addr string) error {
    zlog.Emit(SERVER_READY, "Server ready to accept connections",
        zlog.String("addr", addr))
    
    return http.ListenAndServe(addr, s.router)
}

// Middleware for request logging
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        requestID := generateRequestID()
        
        // Add request ID to context
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        r = r.WithContext(ctx)
        
        // Log request start
        zlog.Emit(HTTP_REQUEST_STARTED, "HTTP request started",
            zlog.String("request_id", requestID),
            zlog.String("method", r.Method),
            zlog.String("path", r.URL.Path),
            zlog.String("remote_addr", r.RemoteAddr),
            zlog.String("user_agent", r.UserAgent()))
        
        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        // Process request
        next.ServeHTTP(wrapped, r)
        
        // Log request completion
        duration := time.Since(start)
        
        if wrapped.statusCode >= 400 {
            zlog.Emit(HTTP_REQUEST_FAILED, "HTTP request failed",
                zlog.String("request_id", requestID),
                zlog.String("method", r.Method),
                zlog.String("path", r.URL.Path),
                zlog.Int("status", wrapped.statusCode),
                zlog.Duration("duration", duration))
        } else {
            zlog.Emit(HTTP_REQUEST_COMPLETED, "HTTP request completed",
                zlog.String("request_id", requestID),
                zlog.String("method", r.Method),
                zlog.String("path", r.URL.Path),
                zlog.Int("status", wrapped.statusCode),
                zlog.Duration("duration", duration))
        }
    })
}

// HTTP Handlers

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.respondWithError(w, r, http.StatusBadRequest, "Invalid JSON", err)
        return
    }
    
    // Validate request
    if req.Username == "" || req.Email == "" || req.Password == "" {
        zlog.Emit(VALIDATION_ERROR, "Registration validation failed",
            zlog.String("request_id", getRequestID(r.Context())),
            zlog.String("reason", "missing_required_fields"))
        
        s.respondWithError(w, r, http.StatusBadRequest, "Missing required fields", nil)
        return
    }
    
    // Check if user already exists
    if _, exists := s.users[req.Username]; exists {
        zlog.Emit(VALIDATION_ERROR, "Registration failed - user exists",
            zlog.String("request_id", getRequestID(r.Context())),
            zlog.String("username", req.Username))
        
        s.respondWithError(w, r, http.StatusConflict, "Username already exists", nil)
        return
    }
    
    // Create user
    user := &User{
        ID:       generateUserID(),
        Username: req.Username,
        Email:    req.Email,
    }
    
    s.users[req.Username] = user
    
    // Log successful registration
    zlog.Emit(USER_REGISTERED, "User registered successfully",
        zlog.String("request_id", getRequestID(r.Context())),
        zlog.String("user_id", user.ID),
        zlog.String("username", user.Username),
        zlog.String("email", user.Email))
    
    s.respondWithData(w, user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.respondWithError(w, r, http.StatusBadRequest, "Invalid JSON", err)
        return
    }
    
    // Find user
    user, exists := s.users[req.Username]
    if !exists {
        zlog.Emit(AUTHENTICATION_ERROR, "Login failed - user not found",
            zlog.String("request_id", getRequestID(r.Context())),
            zlog.String("username", req.Username),
            zlog.String("remote_addr", r.RemoteAddr))
        
        s.respondWithError(w, r, http.StatusUnauthorized, "Invalid credentials", nil)
        return
    }
    
    // Log successful login
    zlog.Emit(USER_LOGIN, "User logged in successfully",
        zlog.String("request_id", getRequestID(r.Context())),
        zlog.String("user_id", user.ID),
        zlog.String("username", user.Username),
        zlog.String("remote_addr", r.RemoteAddr))
    
    s.respondWithData(w, user)
}

// Response helpers

func (s *Server) respondWithData(w http.ResponseWriter, data interface{}) {
    response := APIResponse{
        Success: true,
        Data:    data,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (s *Server) respondWithError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
    if err != nil {
        zlog.Error("HTTP request error",
            zlog.String("request_id", getRequestID(r.Context())),
            zlog.String("method", r.Method),
            zlog.String("path", r.URL.Path),
            zlog.Int("status", status),
            zlog.String("error", message),
            zlog.Err(err))
    }
    
    response := APIResponse{
        Success: false,
        Error:   message,
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(response)
}

// Utility functions

type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}

func getRequestID(ctx context.Context) string {
    if id, ok := ctx.Value("request_id").(string); ok {
        return id
    }
    return "unknown"
}

func generateRequestID() string {
    return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func generateUserID() string {
    return fmt.Sprintf("usr_%d", time.Now().UnixNano())
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// Logging setup helpers

func setupMetrics() {
    metricsSink := zlog.NewSink("metrics", func(ctx context.Context, event zlog.Event) error {
        // Route business events to metrics system
        return nil
    })
    
    zlog.RouteSignal(USER_REGISTERED, metricsSink)
    zlog.RouteSignal(USER_LOGIN, metricsSink)
    zlog.RouteSignal(HTTP_REQUEST_COMPLETED, metricsSink)
}

func setupAuditLogging() {
    auditSink := zlog.NewSink("audit", func(ctx context.Context, event zlog.Event) error {
        // Write to audit log
        return nil
    })
    
    zlog.RouteSignal(USER_REGISTERED, auditSink)
    zlog.RouteSignal(USER_LOGIN, auditSink)
    zlog.RouteSignal(AUTHENTICATION_ERROR, auditSink)
}

func setupErrorAlerting() {
    alertSink := zlog.NewSink("alerts", func(ctx context.Context, event zlog.Event) error {
        // Send alerts for critical events
        return nil
    })
    
    zlog.RouteSignal(zlog.ERROR, alertSink)
    zlog.RouteSignal(zlog.FATAL, alertSink)
}
```

## Example Usage

Start the server:

```bash
# Development mode
ENV=development go run main.go

# Production mode  
ENV=production PORT=8080 go run main.go
```

Test the endpoints:

```bash
# Register a user
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"secret123"}'

# Login
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"secret123"}'
```

## Sample Log Output

Development mode:
```
= [14:30:15] SERVER_STARTING: Starting web server
   port: 8080
   env: development

= [14:30:20] USER_REGISTERED: User registered successfully
   request_id: req_1640123420123456789
   user_id: usr_1640123420123456790
   username: alice
   email: alice@example.com
```

Production mode (JSON):
```json
{"time":"2023-10-20T14:30:15Z","signal":"SERVER_STARTING","message":"Starting web server","port":"8080","env":"production"}
{"time":"2023-10-20T14:30:20Z","signal":"USER_REGISTERED","message":"User registered successfully","request_id":"req_1640123420123456789","user_id":"usr_1640123420123456790","username":"alice","email":"alice@example.com"}
```

This example demonstrates signal-based logging with context propagation, environment-specific configuration, and multiple event destinations.