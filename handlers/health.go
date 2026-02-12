// Package handlers provides HTTP request handlers for the Hynix microservice.
//
// This package contains handlers for:
//   - Health checks
//   - Spark application creation
//   - Spark configuration reference
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns the health status of the service
// GET /health
//
// Response:
//   200: {"status": "healthy"}
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "hynix",
		"version": "2.0",
	})
}
