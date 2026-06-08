package worker

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"task-manager/internal/domain"
)

type Notification struct {
	TaskID    domain.TaskID
	Action    string
	Timestamp time.Time
}

func Start(ctx context.Context, ch <-chan Notification, logger *slog.Logger) error {
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < 3; i++ {
		g.Go(func() error {
			return worker(ctx, ch, logger)
		})
	}

	return g.Wait()
}

func worker(ctx context.Context, ch <-chan Notification, logger *slog.Logger) error {
	for {
		select {
		case <-ctx.Done():
			logger.Info("worker stopping due to context cancellation")
			return nil
		case notif, ok := <-ch:
			if !ok {
				logger.Info("notification channel closed")
				return nil
			}
			// Эмуляция отправки уведомления
			logger.Info("sending notification",
				"task_id", notif.TaskID,
				"action", notif.Action,
			)
			time.Sleep(50 * time.Millisecond)
		}
	}
}
