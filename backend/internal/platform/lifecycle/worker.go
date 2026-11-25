package lifecycle

import (
	"context"
	"log/slog"
	"time"
)

type Worker struct {
	Interval time.Duration
	Logger   *slog.Logger
	Name     string
	OnTick   func(context.Context) error
}

func (w Worker) Run(ctx context.Context) error {
	logger := w.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	logger.Info("worker started", "worker", w.Name, "interval", w.Interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopped", "worker", w.Name)
			return nil
		case <-ticker.C:
			if w.OnTick == nil {
				logger.Debug("worker heartbeat", "worker", w.Name)
				continue
			}
			if err := w.OnTick(ctx); err != nil {
				logger.Error("worker tick failed", "worker", w.Name, "error", err)
			}
		}
	}
}
