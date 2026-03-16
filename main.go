package connectrpc

import (
	"context"
	"fmt"
	"net/http"
	"log/slog"
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
	Host 		string
	Port		int
	Registrars	[]Registrar
	Interceptors []connect.Interceptor
	CORS		CorsConfig
}

type Server struct {
	handler http.Handler
	host	string
	port 	int
}

func NewServer(cfg Config) *Server {
	interceptors := []connect.Interceptor{
        interceptor.Recovery(),
        interceptor.Logging(),
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
		handler = corsMiddleware(cfg.CORS)(mux)
	}

	return &Server{
		handler:	handler,
		host:	cfg.Host,
		port:	cfg.Port,
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
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        if err := server.Shutdown(shutdownCtx); err != nil {
            slog.Error("shutdown error", slog.Any("error", err))
        }
    }()

	slog.Info("Connect RPC server listening", slog.String("address", address))

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}