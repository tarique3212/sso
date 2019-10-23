package httpserver

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/buzzfeed/sso/internal/pkg/logging"
)

// Run runs an http server and ensures that it is shut down gracefully within
// the given shutdown timeout, allowing all in-flight requests to complete.
func Run(srv *http.Server, shutdownTimeout time.Duration, logger *logging.LogEntry) error {
	// shutdownCh triggers graceful shutdown on SIGINT or SIGTERM
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	// exitCh will be closed when it is safe to exit, after graceful shutdown
	exitCh := make(chan struct{})

	go func() {
		sig := <-shutdownCh
		logger.Info("shutdown started by signal: ", sig)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		if err := srv.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("shutdown error")
		}

		close(exitCh)
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-exitCh
	logger.Info("shutdown finished")

	return nil
}
