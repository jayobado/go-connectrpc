package connectrpc

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	interceptor "github.com/jayobado/go-connectrpc/interceptors"
)

type Registrar interface {
	Register(mux *http.ServeMux, opts ...connect.HandlerOption)
}

type Config struct {
	Host            string
	Port            int
	Registrars      []Registrar
	Interceptors    []connect.Interceptor
	CORS            CorsConfig
	ShutdownTimeout time.Duration // defaults to 10s if zero
}

type Server struct {
	handler http.Handler
	host    string
	port    int
	timeout time.Duration
}

func NewServer(cfg Config) *Server {
	interceptors := []connect.Interceptor{
		interceptor.Logging(),
		interceptor.Recovery(),
	}
	interceptors = append(interceptors, cfg.Interceptors...)

	mux := http.NewServeMux()
	opts := []connect.HandlerOption{
		connect.WithInterceptors(interceptors...),
	}
	for _, r := range cfg.Registrars {
		r.Register(mux, opts...)
	}

	var handler http.Handler = mux
	if len(cfg.CORS.AllowedOrigins) > 0 {
		cors := cfg.CORS
		if len(cors.AllowedHeaders) == 0 {
			cors.AllowedHeaders = defaultCORSConfig(cors.AllowedOrigins).AllowedHeaders
		}
		if cors.MaxAge == 0 {
			cors.MaxAge = defaultCORSConfig(cors.AllowedOrigins).MaxAge
		}
		handler = corsMiddleware(cors)(mux)
	}

	timeout := cfg.ShutdownTimeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Server{
		handler: handler,
		host:    cfg.Host,
		port:    cfg.Port,
		timeout: timeout,
	}
}

func (svr *Server) Serve(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", svr.host, svr.port)
	server := &http.Server{
		Addr:    address,
		Handler: h2c.NewHandler(svr.handler, &http2.Server{}),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), svr.timeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	}()

	slog.Info("Connect RPC server listening", "address", address)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}