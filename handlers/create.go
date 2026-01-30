package handlers

import (
	"fmt"
	"service-common/logger"
	"service-common/metrics"
	"service-common/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// LogFieldKeys for structured logging
	LogFieldEndpoint   = "endpoint"
	LogFieldProvisionID = "provision_id"
	LogFieldServiceID  = "service_id"
	LogFieldCategory   = "category"
	LogFieldRegion     = "region"
	LogFieldNamespace  = "namespace"
	LogFieldResourceName = "resource_name"
	LogFieldEnabled    = "enabled"
	LogFieldReason     = "reason"
	LogFieldDurationMs = "duration_ms"

	// Status values
	StatusSuccess = "success"
	StatusError   = "error"
)

// CreateSparkApplication - Create 엔드포인트 핸들러
// POST /api/v1/spark/create
// Request Body: {"provision_id": "0001-wfbm", "service_id": "1234-wfbm", "category": "tttm", "region": "ic"}
func CreateSparkApplication(c *gin.Context) {
	// 요청 시작 시간 기록
	startTime := time.Now()

	// 요청 바디 파싱
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		handleRequestError(c, startTime, "", "요청 파싱 실패", err)
		return
	}

	// 필수 필드 검증
	if err := validateRequest(&req); err != nil {
		handleValidationError(c, startTime, &req, err.Error())
		return
	}

	logRequestReceived(&req)

	// 1. 템플릿 YAML 로드
	yamlTemplate, err := services.LoadTemplateRaw(req.ProvisionID)
	if err != nil {
		handleTemplateLoadError(c, startTime, &req, err)
		return
	}

	// 2. config.json 로드
	config, err := services.LoadConfig()
	if err != nil {
		handleConfigLoadError(c, startTime, &req, err)
		return
	}

	// 3. 프로비저닝 ID에 해당하는 설정 찾기
	provisionConfig, err := services.FindProvisionConfig(config, req.ProvisionID)
	if err != nil {
		handleProvisionConfigError(c, startTime, &req, err)
		return
	}

	// 4. enabled 확인 및 처리
	if !services.IsProvisionEnabled(provisionConfig) {
		handleDisabledProvision(c, startTime, &req, provisionConfig, yamlTemplate)
		return
	}

	// 5. 활성화 모드 처리
	handleEnabledProvision(c, startTime, &req, provisionConfig, yamlTemplate)
}

// validateRequest validates required fields
func validateRequest(req *CreateRequest) error {
	if req.ProvisionID == "" || req.ServiceID == "" || req.Category == "" || req.Region == "" {
		return fmt.Errorf("필수 필드가 누락되었습니다. provision_id, service_id, category, region이 모두 필요합니다")
	}
	return nil
}

// logRequestReceived logs incoming request
func logRequestReceived(req *CreateRequest) {
	logger.Logger.Info("Create 요청 수신",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String(LogFieldRegion, req.Region),
	)
}

// handleRequestError handles request parsing errors
func handleRequestError(c *gin.Context, startTime time.Time, provisionID, message string, err error) {
	logger.Logger.Error(message,
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(provisionID, "create", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(provisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(400, gin.H{
		"error": fmt.Sprintf("%s: %v", message, err),
	})
}

// handleValidationError handles validation errors
func handleValidationError(c *gin.Context, startTime time.Time, req *CreateRequest, message string) {
	logger.Logger.Error("필수 필드 누락",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String(LogFieldRegion, req.Region),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "create", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(400, gin.H{
		"error": message,
	})
}

// handleTemplateLoadError handles template loading errors
func handleTemplateLoadError(c *gin.Context, startTime time.Time, req *CreateRequest, err error) {
	logger.Logger.Error("템플릿 로드 실패",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "create", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(404, gin.H{
		"error": fmt.Sprintf("템플릿 로드 실패: %v", err),
	})
}

// handleConfigLoadError handles config loading errors
func handleConfigLoadError(c *gin.Context, startTime time.Time, req *CreateRequest, err error) {
	logger.Logger.Error("설정 로드 실패",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "create", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(500, gin.H{
		"error": fmt.Sprintf("설정 로드 실패: %v", err),
	})
}

// handleProvisionConfigError handles provision config errors
func handleProvisionConfigError(c *gin.Context, startTime time.Time, req *CreateRequest, err error) {
	logger.Logger.Error("프로비저닝 설정 찾기 실패",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "create", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(404, gin.H{
		"error": fmt.Sprintf("프로비저닝 설정 찾기 실패: %v", err),
	})
}

// handleDisabledProvision handles disabled provision mode
func handleDisabledProvision(c *gin.Context, startTime time.Time, req *CreateRequest, provisionConfig *services.ConfigSpec, yamlTemplate string) {
	logger.Logger.Info("프로비저닝 비활성화 모드",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String(LogFieldRegion, req.Region),
		zap.String(LogFieldEnabled, provisionConfig.Enabled),
		zap.String(LogFieldReason, "disabled"),
	)

	// 메트릭 기록
	metrics.ProvisionMode.WithLabelValues(req.ProvisionID, "false").Inc()
	metrics.ResourceCalculationSkipped.WithLabelValues(req.ProvisionID, "disabled").Inc()

	// 서비스 ID 라벨만 적용
	yamlTemplate = services.ApplyServiceIDLabelsToYAML(yamlTemplate, req.ServiceID)

	// Kubernetes API 서버로 SparkApplication CR 생성 요청
	result, err := services.CreateSparkApplicationCRFromYAML(yamlTemplate)
	if err != nil {
		handleK8sError(c, startTime, req, err, result)
		return
	}

	logCreationSuccess(req, result.Namespace, result.Name, startTime)
	recordSuccessMetrics(req.ProvisionID, result.Namespace, "create", startTime)

	// 비활성화 모드 응답
	c.JSON(201, gin.H{
		"message":      "SparkApplication CR 생성 성공 (비활성화 모드)",
		"provision_id": req.ProvisionID,
		"service_id":   req.ServiceID,
		"category":     req.Category,
		"region":       req.Region,
		"result":       result,
	})
}

// handleEnabledProvision handles enabled provision mode
func handleEnabledProvision(c *gin.Context, startTime time.Time, req *CreateRequest, provisionConfig *services.ConfigSpec, yamlTemplate string) {
	logger.Logger.Info("프로비저닝 활성화 모드",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String(LogFieldRegion, req.Region),
		zap.String(LogFieldEnabled, provisionConfig.Enabled),
	)

	// 메트릭 기록
	metrics.ProvisionMode.WithLabelValues(req.ProvisionID, "true").Inc()

	// 리소스 계산 수행
	queue, err := services.CalculateQueue(
		provisionConfig.ResourceCalculation.Minio,
		provisionConfig.ResourceCalculation.Threshold,
		provisionConfig.ResourceCalculation.MinQueue,
		provisionConfig.ResourceCalculation.MaxQueue,
	)
	if err != nil {
		handleCalculationError(c, startTime, req, err)
		return
	}

	logResourceCalculation(req, provisionConfig, queue)
	metrics.QueueSelection.WithLabelValues(req.ProvisionID, queue).Inc()

	// 큐 설정 적용
	yamlTemplate = updateQueueInYAML(yamlTemplate, queue)

	// Gang Scheduling 설정 적용
	executorMinMember, err := strconv.Atoi(provisionConfig.GangScheduling.Executor)
	if err != nil {
		handleExecutorConfigError(c, startTime, req, err)
		return
	}

	logGangSchedulingConfig(req, provisionConfig, executorMinMember)
	recordGangSchedulingMetrics(req.ProvisionID, provisionConfig, executorMinMember)

	// task-groups의 executor minMember 업데이트
	yamlTemplate = updateExecutorMinMemberInYAML(yamlTemplate, executorMinMember)

	// 서비스 ID 라벨 적용
	yamlTemplate = services.ApplyServiceIDLabelsToYAML(yamlTemplate, req.ServiceID)

	// Kubernetes API 서버로 SparkApplication CR 생성 요청
	result, err := services.CreateSparkApplicationCRFromYAML(yamlTemplate)
	if err != nil {
		handleK8sError(c, startTime, req, err, result)
		return
	}

	logCreationSuccess(req, result.Namespace, result.Name, startTime)
	recordSuccessMetrics(req.ProvisionID, result.Namespace, "create", startTime)

	// 활성화 모드 응답
	c.JSON(201, gin.H{
		"message":      "SparkApplication CR 생성 성공",
		"provision_id": req.ProvisionID,
		"service_id":   req.ServiceID,
		"category":     req.Category,
		"region":       req.Region,
		"result":       result,
	})
}

// handleCalculationError handles resource calculation errors
func handleCalculationError(c *gin.Context, startTime time.Time, req *CreateRequest, err error) {
	logger.Logger.Error("리소스 계산 실패",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "create", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(500, gin.H{
		"error": fmt.Sprintf("리소스 계산 실패: %v", err),
	})
}

// handleExecutorConfigError handles executor config errors
func handleExecutorConfigError(c *gin.Context, startTime time.Time, req *CreateRequest, err error) {
	logger.Logger.Error("executor 설정 변환 실패",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "create", StatusError).Inc()
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(500, gin.H{
		"error": fmt.Sprintf("executor 설정 변환 실패: %v", err),
	})
}

// handleK8sError handles Kubernetes API errors
func handleK8sError(c *gin.Context, startTime time.Time, req *CreateRequest, err error, result *services.CreateResult) {
	logger.Logger.Error("Kubernetes API 요청 실패",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Error(err),
	)
	metrics.RequestsTotal.WithLabelValues(req.ProvisionID, "create", StatusError).Inc()
	if result != nil {
		metrics.K8sCreation.WithLabelValues(req.ProvisionID, result.Namespace, StatusError).Inc()
	}
	metrics.RequestDuration.WithLabelValues(req.ProvisionID, "create").Observe(time.Since(startTime).Seconds())
	c.JSON(500, gin.H{
		"error": fmt.Sprintf("Kubernetes API 요청 실패: %v", err),
	})
}

// logResourceCalculation logs resource calculation results
func logResourceCalculation(req *CreateRequest, config *services.ConfigSpec, queue string) {
	logger.Logger.Info("리소스 계산 완료",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String("file_path", config.ResourceCalculation.Minio),
		zap.Float64("threshold_mb", float64(config.ResourceCalculation.Threshold)),
		zap.String("selected_queue", queue),
	)
}

// logGangSchedulingConfig logs gang scheduling configuration
func logGangSchedulingConfig(req *CreateRequest, config *services.ConfigSpec, executorMinMember int) {
	logger.Logger.Info("Gang Scheduling 구성",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.Int("executor_min_member", executorMinMember),
		zap.String("cpu", config.GangScheduling.CPU),
		zap.String("memory", config.GangScheduling.Memory),
	)
}

// recordGangSchedulingMetrics records gang scheduling metrics
func recordGangSchedulingMetrics(provisionID string, config *services.ConfigSpec, executorMinMember int) {
	metrics.ExecutorMinMember.WithLabelValues(provisionID).Set(float64(executorMinMember))

	cpuValue, _ := strconv.ParseFloat(config.GangScheduling.CPU, 64)
	metrics.GangSchedulingResources.WithLabelValues(provisionID, "cpu").Set(cpuValue)

	memoryValue, _ := strconv.ParseFloat(config.GangScheduling.Memory, 64)
	metrics.GangSchedulingResources.WithLabelValues(provisionID, "memory").Set(memoryValue)
}

// logCreationSuccess logs successful CR creation
func logCreationSuccess(req *CreateRequest, namespace, name string, startTime time.Time) {
	logger.Logger.Info("SparkApplication CR 생성 성공",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String(LogFieldRegion, req.Region),
		zap.String(LogFieldNamespace, namespace),
		zap.String(LogFieldResourceName, name),
		zap.Float64(LogFieldDurationMs, float64(time.Since(startTime).Milliseconds())),
	)
}

// recordSuccessMetrics records success metrics
func recordSuccessMetrics(provisionID, namespace, endpoint string, startTime time.Time) {
	metrics.RequestsTotal.WithLabelValues(provisionID, endpoint, StatusSuccess).Inc()
	metrics.K8sCreation.WithLabelValues(provisionID, namespace, StatusSuccess).Inc()
	metrics.RequestDuration.WithLabelValues(provisionID, endpoint).Observe(time.Since(startTime).Seconds())
}
