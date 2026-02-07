package handlers

import (
	"encoding/json"
	"fmt"
	"service-common/logger"
	"service-common/metrics"
	"service-common/services"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// LogFieldKeys for structured logging
	LogFieldEndpoint     = "endpoint"
	LogFieldProvisionID  = "provision_id"
	LogFieldServiceID    = "service_id"
	LogFieldCategory     = "category"
	LogFieldRegion       = "region"
	LogFieldNamespace    = "namespace"
	LogFieldResourceName = "resource_name"
	LogFieldEnabled      = "enabled"
	LogFieldReason       = "reason"
	LogFieldDurationMs   = "duration_ms"

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

	// 1. 클라이언트 입력 로그
	logClientInput(&req)

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

	// 4. config.json에서 읽은 값 로그
	logConfigValues(provisionConfig)

	// 5. 템플릿 YAML 로드
	yamlTemplate, err := services.LoadTemplateRaw(req.ProvisionID)
	if err != nil {
		handleTemplateLoadError(c, startTime, &req, err)
		return
	}

	// 6. enabled 확인 및 처리
	if !services.IsProvisionEnabled(provisionConfig) {
		handleDisabledProvision(c, startTime, &req, provisionConfig, yamlTemplate)
		return
	}

	// 7. 활성화 모드 처리
	handleEnabledProvision(c, startTime, &req, provisionConfig, yamlTemplate)
}

// validateRequest validates required fields
func validateRequest(req *CreateRequest) error {
	if req.ProvisionID == "" || req.ServiceID == "" || req.Category == "" || req.Region == "" {
		return fmt.Errorf("필수 필드가 누락되었습니다. provision_id, service_id, category, region이 모두 필요합니다")
	}
	return nil
}

// logClientInput - 클라이언트 입력 로그 (1번째 로그)
func logClientInput(req *CreateRequest) {
	inputLog := map[string]interface{}{
		"log_type":      "client_input",
		"endpoint":      "create",
		"provision_id":  req.ProvisionID,
		"service_id":    req.ServiceID,
		"category":      req.Category,
		"region":        req.Region,
		"received_at":   time.Now().Format(time.RFC3339),
	}

	logJSON, _ := json.Marshal(inputLog)
	logger.Logger.Info(string(logJSON))
}

// logConfigValues - config.json에서 읽은 값 로그 (2번째 로그)
func logConfigValues(config *services.ConfigSpec) {
	configLog := map[string]interface{}{
		"log_type":    "config_values",
		"provision_id": config.ProvisionID,
		"enabled":      config.Enabled,
		"resource_calculation": map[string]interface{}{
			"minio":     config.ResourceCalculation.Minio,
			"threshold": config.ResourceCalculation.Threshold,
			"min_queue": config.ResourceCalculation.MinQueue,
			"max_queue": config.ResourceCalculation.MaxQueue,
		},
		"gang_scheduling": map[string]interface{}{
			"cpu":      config.GangScheduling.CPU,
			"memory":   config.GangScheduling.Memory,
			"executor": config.GangScheduling.Executor,
		},
		"build_number": map[string]interface{}{
			"number": config.BuildNumber.Number,
		},
	}

	logJSON, _ := json.Marshal(configLog)
	logger.Logger.Info(string(logJSON))
}

// logMinIOResourceCalculation - MinIO 리소스 계산 결과 로그 (3번째 로그)
func logMinIOResourceCalculation(req *CreateRequest, config *services.ConfigSpec, queue string, fileSize int64) {
	resourceLog := map[string]interface{}{
		"log_type":     "minio_resource_calculation",
		"endpoint":     "create",
		"provision_id": req.ProvisionID,
		"service_id":   req.ServiceID,
		"minio_path":   config.ResourceCalculation.Minio,
		"file_size":    fileSize,
		"threshold":    config.ResourceCalculation.Threshold,
		"selected_queue": queue,
		"calculated_at": time.Now().Format(time.RFC3339),
	}

	logJSON, _ := json.Marshal(resourceLog)
	logger.Logger.Info(string(logJSON))
}

// logFinalYAML - 결과 YAML 로그 (4번째 로그) - YAML을 문자열로 유지
func logFinalYAML(yamlStr string) string {
	finalLog := map[string]interface{}{
		"log_type":     "final_yaml_result",
		"content":      yamlStr,
		"generated_at": time.Now().Format(time.RFC3339),
	}

	logJSON, _ := json.Marshal(finalLog)
	logger.Logger.Info(string(logJSON))

	return yamlStr
}

// logMinIOMetadata - MinIO 파일 메타데이터 로그 (5번째 로그)
func logMinIOMetadata(req *CreateRequest, metadata *services.MinIOMetadata) {
	metadataLog := map[string]interface{}{
		"log_type":        "minio_metadata",
		"endpoint":        "create",
		"provision_id":    req.ProvisionID,
		"service_id":      req.ServiceID,
		"minio_path":      metadata.Path,
		"size_bytes":      metadata.Size,
		"size_formatted":  services.FormatBytes(metadata.Size),
		"etag":            metadata.ETag,
		"last_modified":    metadata.LastModified.Format(time.RFC3339),
		"content_type":    metadata.ContentType,
		"storage_class":   metadata.StorageClass,
		"user_metadata":   metadata.UserMetadata,
		"fetched_at":      time.Now().Format(time.RFC3339),
	}

	logJSON, _ := json.Marshal(metadataLog)
	logger.Logger.Info(string(logJSON))
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

	// BUILD_NUMBER 적용
	yamlTemplate = services.ApplyBuildNumberToYAML(yamlTemplate, provisionConfig.BuildNumber.Number)

	// 4. 최종 YAML 로그 출력
	logFinalYAML(yamlTemplate)

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

	// 리소스 계산 수행 (MinIO에서 파일 크기 및 메타데이터 확인)
	// MinIO 경로: {resource_calculation.minio}/{service_id}
	queue, fileSize, metadata, err := services.CalculateQueueWithMetadata(
		provisionConfig.ResourceCalculation.Minio,
		req.ServiceID,
		provisionConfig.ResourceCalculation.Threshold,
		provisionConfig.ResourceCalculation.MinQueue,
		provisionConfig.ResourceCalculation.MaxQueue,
	)

	// 3. MinIO 리소스 계산 결과 로그 출력
	logMinIOResourceCalculation(req, provisionConfig, queue, fileSize)

	// 5. MinIO 메타데이터 로그 출력
	if metadata != nil {
		logMinIOMetadata(req, metadata)
	}

	if err != nil {
		// MinIO 오류는 경고로 처리하고 계속 진행 (기본값 사용)
		logger.Logger.Warn("MinIO 리소스 계산 경고",
			zap.String(LogFieldEndpoint, "create"),
			zap.String(LogFieldProvisionID, req.ProvisionID),
			zap.Error(err),
		)
	}

	logResourceCalculation(req, provisionConfig, queue, fileSize)
	metrics.QueueSelection.WithLabelValues(req.ProvisionID, queue).Inc()

	// 큐 설정 적용
	yamlTemplate = updateQueueInYAML(yamlTemplate, queue)

	// Template 처리 로직 1: config.json의 gang_scheduling.executor를 task-groups의 executor minMember에 대입
	executorMinMember, err := services.GetExecutorInt(provisionConfig.GangScheduling.Executor)
	if err != nil {
		handleExecutorConfigError(c, startTime, req, err)
		return
	}

	logGangSchedulingConfig(req, provisionConfig, executorMinMember)
	recordGangSchedulingMetrics(req.ProvisionID, provisionConfig, executorMinMember)

	// task-groups의 executor minMember 업데이트
	yamlTemplate = updateExecutorMinMemberInYAML(yamlTemplate, executorMinMember)

	// Template 처리 로직 2: config.json의 gang_scheduling.executor를 spec.executor.instances에 대입
	yamlTemplate = services.UpdateExecutorInstances(yamlTemplate, executorMinMember)

	// Template 처리 로직 3: config.json의 build_number.number를 BUILD_NUMBER에 대입
	yamlTemplate = services.ApplyBuildNumberToYAML(yamlTemplate, provisionConfig.BuildNumber.Number)

	// 서비스 ID 라벨 적용
	yamlTemplate = services.ApplyServiceIDLabelsToYAML(yamlTemplate, req.ServiceID)

	// 4. 최종 YAML 로그 출력
	logFinalYAML(yamlTemplate)

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
func logResourceCalculation(req *CreateRequest, config *services.ConfigSpec, queue string, fileSize int64) {
	logger.Logger.Info("리소스 계산 완료",
		zap.String(LogFieldEndpoint, "create"),
		zap.String(LogFieldProvisionID, req.ProvisionID),
		zap.String(LogFieldServiceID, req.ServiceID),
		zap.String(LogFieldCategory, req.Category),
		zap.String("file_path", config.ResourceCalculation.Minio),
		zap.Int64("file_size_bytes", fileSize),
		zap.Int64("threshold_bytes", config.ResourceCalculation.Threshold),
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

// updateQueueInYAML updates the queue value in YAML
func updateQueueInYAML(yamlStr string, queue string) string {
	// batchSchedulerOptions.queue 값 교체
	lines := strings.Split(yamlStr, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "queue:") {
			indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
			lines[i] = fmt.Sprintf("%squeue: root.%s", indent, queue)
			break
		}
	}
	return strings.Join(lines, "\n")
}

// updateExecutorMinMemberInYAML updates executor minMember in task-groups annotation
func updateExecutorMinMemberInYAML(yamlStr string, minMember int) string {
	lines := strings.Split(yamlStr, "\n")
	inTaskGroups := false
	taskGroupStarted := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// task-groups 배열 시작 확인
		if strings.Contains(trimmed, "yunikorn.apache.org/task-groups:") {
			inTaskGroups = true
			continue
		}

		// task-groups 섹션 종료 확인
		if inTaskGroups && (strings.HasPrefix(trimmed, "serviceAccount:") || !strings.HasPrefix(trimmed, "|") && !strings.HasPrefix(trimmed, "[") && trimmed != "" && !strings.HasPrefix(trimmed, "-") && !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "}")) {
			inTaskGroups = false
			taskGroupStarted = false
		}

		// executor task-group 찾기
		if inTaskGroups && strings.Contains(trimmed, `"name": "spark-executor"`) {
			taskGroupStarted = true
		}

		// executor의 minMember 업데이트
		if taskGroupStarted && strings.Contains(trimmed, `"minMember":`) {
			indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
			lines[i] = fmt.Sprintf("%s\"minMember\": %d,", indent, minMember)
			taskGroupStarted = false
			break
		}
	}

	return strings.Join(lines, "\n")
}
