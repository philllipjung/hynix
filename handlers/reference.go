package handlers

import (
	"fmt"
	"service-common/logger"
	"service-common/services"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetSparkReference - Reference 엔드포인트 핸들러
// GET /api/v1/spark/reference?provision_id=0001-wfbm
func GetSparkReference(c *gin.Context) {
	startTime := time.Now()
	provisionID := c.Query("provision_id")

	if provisionID == "" {
		logger.Logger.Error("필수 파라미터 누락", zap.String("endpoint", "reference"))
		c.JSON(400, gin.H{"error": "필수 파라미터가 누락되었습니다. provision_id가 필요합니다"})
		return
	}

	logger.Logger.Info("Reference 요청 수신", zap.String("endpoint", "reference"), zap.String("provision_id", provisionID))

	yamlTemplate, err := services.LoadTemplateRaw(provisionID)
	if err != nil {
		logger.Logger.Error("템플릿 로드 실패", zap.String("endpoint", "reference"), zap.String("provision_id", provisionID), zap.Error(err))
		c.JSON(404, gin.H{"error": fmt.Sprintf("템플릿 로드 실패: %v", err)})
		return
	}

	c.Header("Content-Type", "application/x-yaml")
	c.String(200, yamlTemplate)

	logger.Logger.Info("Reference YAML 반환 완료",
		zap.String("endpoint", "reference"),
		zap.String("provision_id", provisionID),
		zap.Float64("duration_ms", float64(time.Since(startTime).Milliseconds())),
	)
}
