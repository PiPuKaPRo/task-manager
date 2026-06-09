package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"task-manager/internal/config"
	"task-manager/internal/handler"
	"task-manager/internal/repository"
	"task-manager/internal/service"
	"task-manager/internal/worker"
	"task-manager/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env)

	log.Info("starting task manager api", "port", cfg.ServerPort)

	db, err := repository.NewDB(cfg)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	log.Info("database connected successfully")

	taskRepo := repository.NewTaskRepository(db)
	taskSvc := service.NewTaskService(taskRepo, log)
	taskHandler := handler.NewTaskHandler(taskSvc, log)

	notifyCh := make(chan worker.Notification, 100)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", taskHandler.Create)
	mux.HandleFunc("GET /tasks", taskHandler.List)
	mux.HandleFunc("GET /tasks/{id}", taskHandler.Get)
	mux.HandleFunc("PUT /tasks/{id}", taskHandler.Update)
	mux.HandleFunc("DELETE /tasks/{id}", taskHandler.Delete)
	mux.HandleFunc("POST /tasks/{id}/done", taskHandler.MarkDone)
	mux.HandleFunc("POST /tasks/{id}/undone", taskHandler.MarkUndone)
	mux.HandleFunc("POST /tasks/{id}/assign", taskHandler.Assign)
	mux.HandleFunc("GET /tasks/status/{status}", taskHandler.ListByStatus)
	mux.HandleFunc("GET /health", taskHandler.Health)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		log.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	g.Go(func() error {
		log.Info("notification worker started")
		return worker.Start(ctx, notifyCh, log)
	})

	g.Go(func() error {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Info("shutting down gracefully")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("server shutdown error", "error", err)
		}

		close(notifyCh)
		log.Info("graceful shutdown completed")
		return nil
	})

	if err := g.Wait(); err != nil {
		log.Error("application stopped with error", "error", err)
		os.Exit(1)
	}

	log.Info("application stopped successfully")
}
