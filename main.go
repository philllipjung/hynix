package main

import (
	"service-common/handlers"
	"service-common/logger"
	"service-common/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// 로거 초기화
	logger.Init()
	defer logger.Sync()

	logger.Logger.Info("서비스공통 마이크로서비스 시작",
		zap.String("port", "8080"),
	)

	r := gin.Default()

	// 미들웨어 설정
	r.Use(middleware.LoggingMiddleware())

	// 헬스체크 엔드포인트
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	// Prometheus 메트릭 엔드포인트
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API 라우터
	api := r.Group("/api/v1")
	{
		api.GET("/spark/reference", handlers.GetSparkReference)
		api.POST("/spark/create", handlers.CreateSparkApplication)
	}

	// 서버 시작
	if err := r.Run(":8080"); err != nil {
		logger.Logger.Fatal("서버 시작 실패",
			zap.Error(err),
		)
	}
}
