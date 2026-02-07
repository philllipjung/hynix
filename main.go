// Package main provides the Hynix microservice for Spark application management.
//
// This service exposes REST APIs for:
//   - Creating Spark applications with dynamic resource allocation
//   - Referencing Spark application configurations
//   - Integrating with Yunikorn for gang scheduling
//
// The service runs on port 8080 and supports:
//   - Health checks
//   - Prometheus metrics
//   - Structured logging
//
// Usage:
//
//	./hynix
//
// Environment:
//   PORT: Server port (default: 8080)
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

const (
	// ShutdownTimeout is the maximum time to wait for graceful shutdown
	ShutdownTimeout = 30 * time.Second
)

// getPort returns the port from environment variable or default
func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}

func main() {
	// Initialize logger
	logger.Init()
	defer logger.Sync()

	port := getPort()

	logger.Logger.Info("Starting Hynix microservice",
		zap.String("port", port),
		zap.String("version", "2.0"),
	)

	// Setup Gin router
	router := setupRouter()

	// Setup server
	server := &http.Server{
		Addr:    port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Logger.Info("Server listening", zap.String("addr", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	 GracefulShutdown(server)
}

// setupRouter configures and returns the Gin router with all routes and middleware
func setupRouter() *gin.Engine {
	router := gin.Default()

	// Middleware
	router.Use(middleware.LoggingMiddleware())

	// Health check endpoint
	router.GET("/health", handlers.HealthCheck)

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	setupAPIRoutes(router)

	return router
}

// setupAPIRoutes configures API v1 route group
func setupAPIRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		api.GET("/spark/reference", handlers.GetSparkReference)
		api.POST("/spark/create", handlers.CreateSparkApplication)
	}
}

// GracefulShutdown handles graceful server shutdown
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
