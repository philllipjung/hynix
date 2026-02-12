package handlers

import (
	"encoding/json"
	"fmt"
	"service-common/logger"
	"service-common/metrics"
	"service-common/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ReferenceRequest - Reference 엔드포인트 요청 파라미터
type ReferenceRequest struct {
	ProvisionID string
	ServiceID   string
	Category    string
	UID         string
}

// GetSparkReference - Reference 엔드포인트 핸들러
// GET /api/v1/spark/reference?provision_id=0001-wfbm&service_id=123456&category=tttm
func GetSparkReference(c *gin.Context) {
	// 요청 시작 시간 기록
	startTime := time.Now()

	// 쿼리 파라미터 추출
	req := parseReferenceRequest(c)

	// 필수 파라미터 검증
	if err := validateReferenceRequest(&req); err != nil {
		handleReferenceValidationError(c, startTime, &req, err.Error())
		return
	}

	logReferenceRequestReceived(&req)

	// 1. 템플릿 YAML 로드
	yamlTemplate, err := services.LoadTemplateRaw(req.ProvisionID)
	if err != nil {
		handleReferenceTemplateError(c, startTime, &req, err)
		return
	}

	// 2. config.json 로드
	config, err := services.LoadConfig()
	if err != nil {
		handleReferenceConfigError(c, startTime, &req, err)
		return
	}

	// 3. 프로비저닝 ID에 해당하는 설정 찾기
	provisionConfig, err := services.FindProvisionConfig(config, req.ProvisionID)
	if err != nil {
		handleReferenceProvisionError(c, startTime, &req, err)
		return
	}

	// 4. enabled 확인 및 처리
	if !services.IsProvisionEnabled(provisionConfig) {
		handleReferenceDisabled(c, startTime, &req, provisionConfig, yamlTemplate)
		return
	}

	// 5. 활성화 모드 처리
	handleReferenceEnabled(c, startTime, &req, provisionConfig, yamlTemplate)
}

// parseReferenceRequest extracts request parameters from query string
func parseReferenceRequest(c *gin.Context) ReferenceRequest {
	return ReferenceRequest{
		ProvisionID: c.Query("provision_id"),
		ServiceID:   c.Query("service_id"),
		Category:    c.Query("category"),
		UID:         c.Query("uid"),
	}
}

// validateReferenceRequest validates reference request parameters
func validateReferenceRequest(req *ReferenceRequest) error {
	if req.ProvisionID == "" || req.ServiceID == "" || req.Category == "" || req.UID == "" {
		return fmt.Errorf("필수 파라미터가 누락되었습니다. provision_id, service_id, category, uid가 모두 필요합니다")
	}
	return nil
}

// logReferenceRequestReceived logs incoming reference request
func logReferenceRequestReceived(req *ReferenceRequest) {
	logger.Logger.Info("Reference 요청 수신",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
	)
}

// handleReferenceValidationError handles validation errors
func handleReferenceValidationError(c *gin.Context, startTime time.Time, req *ReferenceRequest, message string) {
	logger.Logger.Error("필수 파라미터 누락",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "reference", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "reference").Observe(time.Since(startTime).Seconds())
	c.JSON(400, gin.H{
		"error": message,
	})
}

// handleReferenceTemplateError handles template loading errors
func handleReferenceTemplateError(c *gin.Context, startTime time.Time, req *ReferenceRequest, err error) {
	logger.Logger.Error("템플릿 로드 실패",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "reference", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "reference").Observe(time.Since(startTime).Seconds())
	c.JSON(404, gin.H{
		"error": fmt.Sprintf("템플릿 로드 실패: %v", err),
	})
}

// handleReferenceConfigError handles config loading errors
func handleReferenceConfigError(c *gin.Context, startTime time.Time, req *ReferenceRequest, err error) {
	logger.Logger.Error("설정 로드 실패",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "reference", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "reference").Observe(time.Since(startTime).Seconds())
	c.JSON(500, gin.H{
		"error": fmt.Sprintf("설정 로드 실패: %v", err),
	})
}

// handleReferenceProvisionError handles provision config errors
func handleReferenceProvisionError(c *gin.Context, startTime time.Time, req *ReferenceRequest, err error) {
	logger.Logger.Error("프로비저닝 설정 찾기 실패",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "reference", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "reference").Observe(time.Since(startTime).Seconds())
	c.JSON(404, gin.H{
		"error": fmt.Sprintf("프로비저닝 설정 찾기 실패: %v", err),
	})
}

// handleReferenceDisabled handles disabled provision mode for reference
func handleReferenceDisabled(c *gin.Context, startTime time.Time, req *ReferenceRequest, provisionConfig *services.ConfigSpec, yamlTemplate string) {
	logger.Logger.Info("프로비저닝 비활성화 모드",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String(LogFieldEnabled, provisionConfig.Enabled),
		zap.String(LogFieldReason, "disabled"),
	)

	// 메트릭 기록
	metrics.ProvisionMode.WithLabelValues(req.ProvisionID, "false").Inc()
	metrics.ResourceCalculationSkipped.WithLabelValues(req.ProvisionID, "disabled").Inc()

	// build_number 적용
	yamlTemplate = services.ApplyBuildNumberToYAML(yamlTemplate, provisionConfig.BuildNumber.Number)

	// 서비스 ID 라벨 적용
	yamlOutput := services.ApplyServiceIDLabelsToYAML(yamlTemplate, req.ServiceID)

	logReferenceYAMLComplete(req, yamlOutput, startTime, false)
	recordReferenceSuccessMetrics(req.ProvisionID, startTime)

	// 클라이언트에게 YAML 응답
	sendYAMLResponse(c, yamlOutput)
}

// handleReferenceEnabled handles enabled provision mode for reference
func handleReferenceEnabled(c *gin.Context, startTime time.Time, req *ReferenceRequest, provisionConfig *services.ConfigSpec, yamlTemplate string) {
	logger.Logger.Info("프로비저닝 활성화 모드",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String(LogFieldEnabled, provisionConfig.Enabled),
	)

	// 메트릭 기록
	metrics.ProvisionMode.WithLabelValues(req.ProvisionID, "true").Inc()

	// 리소스 계산 수행 (MinIO에서 파일 크기 및 메타데이터 확인)
	// MinIO 경로: config의 resource_calculation.minio 값에서 <<service_id>>를 service_id로 치환
	queue, fileSize, metadata, count, err := services.CalculateQueueWithMetadata(
		provisionConfig.ResourceCalculation.Minio,
		req.ServiceID,
		provisionConfig.ResourceCalculation.Threshold,
		provisionConfig.ResourceCalculation.MinQueue,
		provisionConfig.ResourceCalculation.MaxQueue,
	)

	logResourceCalculationReference(req, provisionConfig, queue, fileSize)

	// 5. MinIO 메타데이터 로그 출력
	if metadata != nil {
		logMinIOMetadataReference(req, metadata)
	}

	if err != nil {
		// MinIO 오류는 경고로 처리하고 계속 진행 (기본값 사용)
		logger.Logger.Warn("MinIO 리소스 계산 경고",
			zap.String(LogFieldEndpoint, "reference"),
			zap.String(LogFieldProvisionID, req.ProvisionID),
			zap.Error(err),
		)
	}

	metrics.QueueSelection.WithLabelValues(req.ProvisionID, queue).Inc()

	// 폴더인 경우 spark.file.count 추가 (count > 0)
	if count > 0 {
		yamlTemplate = services.ApplySparkFileCountToYAML(yamlTemplate, count)
	}

	// 큐 설정 적용
	yamlTemplate = updateQueueInYAML(yamlTemplate, queue)

	// Gang Scheduling 설정 적용
	executorMinMember, err := strconv.Atoi(provisionConfig.GangScheduling.Executor)
	if err != nil {
		handleReferenceExecutorError(c, startTime, req, err)
		return
	}

	logGangSchedulingConfigReference(req, provisionConfig, executorMinMember)
	recordGangSchedulingMetrics(req.ProvisionID, provisionConfig, executorMinMember)

	// task-groups의 executor minMember 업데이트
	yamlTemplate = updateExecutorMinMemberInYAML(yamlTemplate, executorMinMember)

	// Template 처리 로직 2: config.json의 gang_scheduling.executor를 spec.executor.instances에 대입
	yamlTemplate = services.UpdateExecutorInstances(yamlTemplate, executorMinMember)

	// Template 처리 로직 3: config.json의 build_number.number를 BUILD_NUMBER에 대입
	yamlTemplate = services.ApplyBuildNumberToYAML(yamlTemplate, provisionConfig.BuildNumber.Number)

	// 서비스 ID 라벨 적용 (UID 포함)
	yamlOutput := services.ApplyServiceIDLabelsWithUIDToYAML(yamlTemplate, req.ServiceID, req.Category, req.UID)

	logReferenceYAMLComplete(req, yamlOutput, startTime, true)
	recordReferenceSuccessMetrics(req.ProvisionID, startTime)

	// 클라이언트에게 YAML 응답
	sendYAMLResponse(c, yamlOutput)
}

// handleReferenceCalculationError handles resource calculation errors
func handleReferenceCalculationError(c *gin.Context, startTime time.Time, req *ReferenceRequest, err error) {
	logger.Logger.Error("리소스 계산 실패",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "reference", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "reference").Observe(time.Since(startTime).Seconds())
	c.JSON(500, gin.H{
		"error": fmt.Sprintf("리소스 계산 실패: %v", err),
	})
}

// handleReferenceExecutorError handles executor config errors
func handleReferenceExecutorError(c *gin.Context, startTime time.Time, req *ReferenceRequest, err error) {
	logger.Logger.Error("executor 설정 변환 실패",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "reference", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "reference").Observe(time.Since(startTime).Seconds())
	c.JSON(500, gin.H{
		"error": fmt.Sprintf("executor 설정 변환 실패: %v", err),
	})
}

// logResourceCalculationReference logs resource calculation for reference
func logResourceCalculationReference(req *ReferenceRequest, config *services.ConfigSpec, queue string, fileSize int64) {
	logger.Logger.Info("리소스 계산 완료",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String("file_path", config.ResourceCalculation.Minio),
		zap.Int64("file_size_bytes", fileSize),
		zap.Int64("threshold_bytes", config.ResourceCalculation.Threshold),
		zap.String("selected_queue", queue),
	)
}

// logGangSchedulingConfigReference logs gang scheduling config for reference
func logGangSchedulingConfigReference(req *ReferenceRequest, config *services.ConfigSpec, executorMinMember int) {
	logger.Logger.Info("Gang Scheduling 구성",
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Int("executor_min_member", executorMinMember),
		zap.String("cpu", config.GangScheduling.CPU),
		zap.String("memory", config.GangScheduling.Memory),
	)
}

// logReferenceYAMLComplete logs YAML completion with full YAML content
func logReferenceYAMLComplete(req *ReferenceRequest, yamlOutput string, startTime time.Time, enabled bool) {
	mode := "비활성화 모드"
	if enabled {
		mode = "활성화 모드"
	}

	logger.Logger.Info(fmt.Sprintf("YAML 반환 완료 (%s)", mode),
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Float64(LogFieldDurationMs, float64(time.Since(startTime).Milliseconds())),
	)

	// YAML 내용을 로그에 출력
	logger.Logger.Info(fmt.Sprintf("생성된 YAML (%s)", mode),
		zap.String(LogFieldEndpoint, "reference"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String("content", yamlOutput),
	)
}

// recordReferenceSuccessMetrics records success metrics for reference
func recordReferenceSuccessMetrics(provisionID string, startTime time.Time) {
	metrics.RequestsTotal.WithLabelValues(provisionID, "reference", StatusSuccess).Inc()
	metrics.RequestDuration.WithLabelValues(provisionID, "reference").Observe(time.Since(startTime).Seconds())
}

// sendYAMLResponse sends YAML response to client
func sendYAMLResponse(c *gin.Context, yamlOutput string) {
	c.Header("Content-Type", "application/x-yaml")
	c.String(200, yamlOutput)
}

// logMinIOMetadataReference - MinIO 파일 메타데이터 로그 (5번째 로그)
func logMinIOMetadataReference(req *ReferenceRequest, metadata *services.MinIOMetadata) {
	metadataLog := map[string]interface{}{
		"log_type":       "minio_metadata",
		"endpoint":       "reference",
		"provision_id":   req.ProvisionID,
		"service_id":     req.ServiceID,
		"minio_path":     metadata.Path,
		"size_bytes":     metadata.Size,
		"size_formatted": services.FormatBytes(metadata.Size),
		"etag":           metadata.ETag,
		"last_modified":   metadata.LastModified.Format(time.RFC3339),
		"content_type":   metadata.ContentType,
		"storage_class":  metadata.StorageClass,
		"user_metadata":  metadata.UserMetadata,
		"fetched_at":     time.Now().Format(time.RFC3339),
	}

	logJSON, _ := json.Marshal(metadataLog)
	logger.Logger.Info(string(logJSON))
}

