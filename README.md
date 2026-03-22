# go-connectrpc

A lightweight [Connect-RPC](https://connectrpc.com) server factory for Go. Handles HTTP/2, CORS, panic recovery, and structured logging out of the box — so your service setup stays minimal.

## Requirements

- Go 1.24+
- [connectrpc.com/connect v1.19+](https://github.com/connectrpc/connect-go)
- [golang.org/x/net](https://pkg.go.dev/golang.org/x/net)

## Installation
```sh
go get github.com/jayobado/go-connectrpc
```

---

## Quick start
```go
import (
    connectrpc "github.com/jayobado/go-connectrpc"
)

srv := connectrpc.NewServer(connectrpc.Config{
    Host: "0.0.0.0",
    Port: 8080,
    Registrars: []connectrpc.Registrar{
        &UserServiceHandler{},
        &OrderServiceHandler{},
    },
    CORS: connectrpc.CorsConfig{
        AllowedOrigins: []string{"https://myapp.com"},
    },
})

if err := srv.Serve(ctx); err != nil {
    log.Fatal(err)
}
```

---

## `Config`

| Field | Type | Description |
|---|---|---|
| `Host` | `string` | Bind address e.g. `"0.0.0.0"` or `"localhost"` |
| `Port` | `int` | Port to listen on |
| `Registrars` | `[]Registrar` | Services to register on the mux |
| `Interceptors` | `[]connect.Interceptor` | Additional interceptors — run innermost after built-ins |
| `CORS` | `CorsConfig` | CORS configuration — disabled if `AllowedOrigins` is empty |
| `ShutdownTimeout` | `time.Duration` | Graceful shutdown timeout — defaults to 10s |

---

## Registrar interface

Implement `Registrar` on your service handler to register it with the server:
```go
type Registrar interface {
    Register(mux *http.ServeMux, opts ...connect.HandlerOption)
}
```
```go
// example handler
type UserServiceHandler struct{}

func (h *UserServiceHandler) Register(mux *http.ServeMux, opts ...connect.HandlerOption) {
    path, handler := userv1connect.NewUserServiceHandler(h, opts...)
    mux.Handle(path, handler)
}
```

---

## Built-in interceptors

Two interceptors are applied automatically to every handler, in this order:
```
Request → Recovery → Logging → Handler → Logging → Recovery → Response
```

Recovery is outermost so it catches panics from all inner interceptors and the handler. Logging sits inside Recovery so it always records duration and error code even when recovery fires.

### Recovery

Catches panics in unary and streaming handlers. On panic it:

- Logs the procedure name, panic value, and full stack trace via `slog`
- Returns a `Connect CodeInternal` error to the client
```go
// automatically applied — no configuration needed
interceptor.Recovery()
```

### Logging

Logs every RPC call via `slog`. On success logs procedure and duration. On error also logs the Connect error code.
```go
// automatically applied — no configuration needed
interceptor.Logging()
```

Example log output:
```
INFO unary rpc procedure=/user.v1.UserService/GetUser duration=1.2ms
INFO unary rpc procedure=/user.v1.UserService/CreateUser duration=3.4ms code=already_exists error="..."
INFO streaming rpc procedure=/user.v1.UserService/WatchUsers duration=30.1s
```

### Custom interceptors

Additional interceptors are appended after the built-ins and run closest to the handler:
```go
srv := connectrpc.NewServer(connectrpc.Config{
    Interceptors: []connect.Interceptor{
        authInterceptor,
        rateLimitInterceptor,
    },
})
```

---

## CORS

CORS is disabled by default. Set `AllowedOrigins` to enable it:
```go
CORS: connectrpc.CorsConfig{
    AllowedOrigins: []string{"https://myapp.com", "https://admin.myapp.com"},
}
```

When `AllowedHeaders` or `MaxAge` are not set, sensible defaults are applied automatically.

### Default allowed headers
```
Content-Type
Connect-Protocol-Version
Connect-Timeout-Ms
Authorization
Cookie
X-Request-Id
```

### Default `MaxAge`

`7200` seconds (2 hours).

### Exposed headers

The following headers are always exposed to browser clients so Connect trailers are readable:
```
Content-Type
Connect-Protocol-Version
Grpc-Status
Grpc-Message
Grpc-Status-Details-Bin
```

### `CorsConfig` fields

| Field | Type | Description |
|---|---|---|
| `AllowedOrigins` | `[]string` | Allowed origins. Use `"*"` to allow all |
| `AllowedHeaders` | `[]string` | Allowed request headers — defaults applied if empty |
| `MaxAge` | `int` | Preflight cache duration in seconds — defaults to 7200 |

---

## Graceful shutdown

`Serve` accepts a `context.Context`. When the context is cancelled, the server shuts down gracefully waiting for in-flight requests to complete:
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

if err := srv.Serve(ctx); err != nil {
    log.Fatal(err)
}
```

The shutdown timeout controls how long the server waits for in-flight requests before forcefully closing:
```go
connectrpc.Config{
    ShutdownTimeout: 30 * time.Second, // default: 10s
}
```

---

## HTTP/2

The server uses `h2c` (HTTP/2 cleartext) so it works without TLS. Connect-RPC clients can use HTTP/2 framing over plain HTTP. To add TLS use `ListenAndServeTLS` via a custom `http.Server` wrapping the handler.

---

## Project structure
```
go-connectrpc/
├── main.go              # NewServer(), Server, Config, Registrar
├── cors.go              # corsMiddleware, CorsConfig, defaultCORSConfig
└── interceptors/
    ├── logging.go       # Logging() interceptor
    └── recovery.go      # Recovery() interceptor
```

## License

Copyright (C) 2026 Jeremy Obado. All rights reserved.