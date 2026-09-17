package server

import (
	"context"
	"errors"
	"net/http"
	"time"
)

// New uses finite connection timeouts so slow clients cannot keep server resources indefinitely.
func New(address string, handler http.Handler) *http.Server {
	return &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
}

// Run serves until the context is canceled or the listener fails, using a bounded shutdown context
// so in-flight requests can finish without delaying process termination indefinitely.
func Run(ctx context.Context, srv *http.Server) error {
	errorsCh := make(chan error, 1)
	go func() { errorsCh <- srv.ListenAndServe() }()
	select {
	case err := <-errorsCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
