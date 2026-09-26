// Command api is the entry point for membership-service — see
// library-docs/09-microservices/services/03-membership-service/README.md.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/code-corhuila/lms-membership-api/internal/application/usecase"
	"github.com/code-corhuila/lms-membership-api/internal/config"
	circulationclient "github.com/code-corhuila/lms-membership-api/internal/infrastructure/circulation"
	httpserver "github.com/code-corhuila/lms-membership-api/internal/infrastructure/http"
	"github.com/code-corhuila/lms-membership-api/internal/infrastructure/http/handler"
	"github.com/code-corhuila/lms-membership-api/internal/infrastructure/logger"
	"github.com/code-corhuila/lms-membership-api/internal/infrastructure/postgres"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		println("config error:", err.Error())
		return err
	}

	zapLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		return err
	}
	defer func() { _ = zapLog.Sync() }()
	log := zapLog.Sugar()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DSN())
	if err != nil {
		log.Errorw("failed to connect to database", "error", err)
		return err
	}
	defer pool.Close()

	studentRepo := postgres.NewStudentRepository(pool)
	circulationClient := circulationclient.NewClient(cfg.CirculationServiceURL, cfg.JWTSecret)

	createStudentUseCase := usecase.NewCreateStudent(studentRepo)
	getStudentUseCase := usecase.NewGetStudent(studentRepo)
	updateStudentUseCase := usecase.NewUpdateStudent(studentRepo)
	deactivateStudentUseCase := usecase.NewDeactivateStudent(studentRepo, circulationClient)
	searchStudentsUseCase := usecase.NewSearchStudents(studentRepo)
	suspendStudentUseCase := usecase.NewSuspendStudent(studentRepo)
	studentHandler := handler.NewStudentHandler(createStudentUseCase, getStudentUseCase, updateStudentUseCase, deactivateStudentUseCase, searchStudentsUseCase, suspendStudentUseCase)

	router := httpserver.NewRouter(httpserver.RouterConfig{
		DB:         pool,
		JWTSecret:  cfg.JWTSecret,
		CORSOrigin: cfg.CORSOrigin,
		Students:   studentHandler,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Infow("membership-service listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorw("server error", "error", err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
