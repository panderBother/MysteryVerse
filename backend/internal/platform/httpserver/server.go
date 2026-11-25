package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mysteryverse/mysteryverse/backend/internal/platform/apperror"
	"github.com/mysteryverse/mysteryverse/backend/internal/platform/health"
	"github.com/mysteryverse/mysteryverse/backend/pkg/version"
)

type Server struct {
	httpServer      *http.Server
	logger          *slog.Logger
	shutdownTimeout time.Duration
}

type successEnvelope[T any] struct {
	Data T    `json:"data"`
	Meta meta `json:"meta"`
}

type meta struct {
	APIVersion string    `json:"api_version"`
	RequestID  string    `json:"request_id"`
	ServerTime time.Time `json:"server_time"`
}

func New(addr string, logger *slog.Logger, shutdownTimeout time.Duration, healthService *health.Service) *Server {
	return NewWithHandler(addr, logger, shutdownTimeout, NewHandler(logger, healthService))
}

func NewWithHandler(
	addr string,
	logger *slog.Logger,
	shutdownTimeout time.Duration,
	handler http.Handler,
) *Server {
	return &Server{
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
	}
}

func NewHandler(logger *slog.Logger, healthService *health.Service) http.Handler {
	return NewRootHandler(logger, healthService)
}

func NewRootHandler(logger *slog.Logger, healthService *health.Service) *chi.Mux {
	if logger == nil {
		logger = slog.Default()
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(recoverer(logger))
	router.Use(accessLog(logger))

	router.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, r, healthService.Live())
	})
	router.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, r, healthService.Ready())
	})
	router.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, r, healthService.Ready())
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		apperror.Write(w, http.StatusNotFound, apperror.Error{
			Code:      "NOT_FOUND",
			Message:   "The requested endpoint does not exist.",
			RequestID: middleware.GetReqID(r.Context()),
			Retryable: false,
		})
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		apperror.Write(w, http.StatusMethodNotAllowed, apperror.Error{
			Code:      "METHOD_NOT_ALLOWED",
			Message:   "The requested method is not allowed for this endpoint.",
			RequestID: middleware.GetReqID(r.Context()),
			Retryable: false,
		})
	})

	return router
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("http server starting", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		s.logger.Info("http server stopping")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err, ok := <-errCh:
		if ok {
			return err
		}
		return nil
	}
}

func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "request_id", middleware.GetReqID(r.Context()), "panic", recovered)
					apperror.Write(w, http.StatusInternalServerError, apperror.Error{
						Code:      "INTERNAL_ERROR",
						Message:   "An internal error occurred.",
						RequestID: middleware.GetReqID(r.Context()),
						Retryable: true,
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func accessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapped, r)
			logger.Info(
				"http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.Status(),
				"duration_ms", time.Since(started).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}

func writeSuccess[T any](w http.ResponseWriter, r *http.Request, data T) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(successEnvelope[T]{
		Data: data,
		Meta: meta{
			APIVersion: version.ContractAPI,
			RequestID:  middleware.GetReqID(r.Context()),
			ServerTime: time.Now().UTC(),
		},
	})
}
