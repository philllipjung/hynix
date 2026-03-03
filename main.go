package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"service-common/handlers"
	"service-common/logger"
	"service-common/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const ShutdownTimeout = 30 * time.Second

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}

func main() {
	logger.Init()
	defer logger.Sync()

	port := getPort()

	logger.Logger.Info("Starting Hynix microservice",
		zap.String("port", port),
		zap.String("version", "2.0"),
	)

	router := setupRouter()

	server := &http.Server{
		Addr:    port,
		Handler: router,
	}

	go func() {
		logger.Logger.Info("Server listening", zap.String("addr", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	GracefulShutdown(server)
}

func setupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(middleware.LoggingMiddleware())

	api := router.Group("/api/v1")
	{
		api.GET("/spark/reference", handlers.GetSparkReference)
	}

	return router
}

func GracefulShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	logger.Logger.Info("Shutdown signal received", zap.String("signal", sig.String()))

	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Logger.Info("Server exited")
}
