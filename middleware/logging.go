package middleware

import (
	"service-common/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggingMiddleware - HTTP 요청 로깅 미들웨어
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 요청 처리 전
		c.Next()

		// 요청 처리 후 로깅
		logger.Logger.Info("HTTP 요청",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}
