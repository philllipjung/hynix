package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config - 설정 파일 구조체
type Config struct {
	ConfigSpecs []ConfigSpec `json:"config_specs"`
}

// ConfigSpec - 프로비저닝 설정
type ConfigSpec struct {
	ProvisionID         string              `json:"provision_id"`
	Enabled             string              `json:"enabled"`
	ResourceCalculation ResourceCalculation `json:"resource_calculation"`
	GangScheduling      GangScheduling      `json:"gang_scheduling"`
	BuildNumber         BuildNumber         `json:"build_number"`
}

// ResourceCalculation - 리소스 계산 설정
type ResourceCalculation struct {
	Minio     string `json:"minio"`
	Threshold int64  `json:"threshold"`
	MinQueue  string `json:"min_queue"`
	MaxQueue  string `json:"max_queue"`
}

// GangScheduling - Gang Scheduling 설정
type GangScheduling struct {
	CPU      string `json:"cpu"`
	Memory   string `json:"memory"`
	Executor string `json:"executor"`
}

// BuildNumber - 빌드 번호 설정
type BuildNumber struct {
	Number string `json:"number"`
}

// MinIOMetadata - MinIO 객체 메타데이터
type MinIOMetadata struct {
	Path          string    `json:"path"`
	Size          int64     `json:"size"`
	ETag          string    `json:"etag"`
	LastModified  time.Time `json:"last_modified"`
	ContentType   string    `json:"content_type"`
	StorageClass  string    `json:"storage_class"`
	UserMetadata  map[string]string `json:"user_metadata"`
}

// MinIOConfig - MinIO 연결 설정
type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

// LoadConfig - config.json 로드
func LoadConfig() (*Config, error) {
	configPath := "./config/config.json"

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config.json 파일 읽기 실패: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("config.json 파싱 실패: %w", err)
	}

	return &config, nil
}

// FindProvisionConfig - 프로비저닝 ID에 해당하는 설정 찾기
func FindProvisionConfig(config *Config, provisionID string) (*ConfigSpec, error) {
	for i := range config.ConfigSpecs {
		if config.ConfigSpecs[i].ProvisionID == provisionID {
			return &config.ConfigSpecs[i], nil
		}
	}
	return nil, fmt.Errorf("프로비저닝 ID %s를 찾을 수 없음", provisionID)
}

// IsProvisionEnabled - 프로비저닝이 활성화되어 있는지 확인
func IsProvisionEnabled(spec *ConfigSpec) bool {
	return spec.Enabled == "true"
}

// BuildMinioPath - MinIO 경로 생성: {minio_base_path}/{service_id}
func BuildMinioPath(minioBasePath, serviceID string) string {
	return fmt.Sprintf("%s/%s", minioBasePath, serviceID)
}

// CalculateQueueWithMetadata - MinIO 파일 크기에 따른 큐 계산 및 메타데이터 반환
func CalculateQueueWithMetadata(minioBasePath, serviceID string, threshold int64, minQueue, maxQueue string) (string, int64, *MinIOMetadata, error) {
	// MinIO 경로 생성: {base_path}/{service_id}
	minioPath := BuildMinioPath(minioBasePath, serviceID)

	// MinIO에서 파일 크기 및 메타데이터 확인
	metadata, err := getMinIOMetadata(minioPath)
	if err != nil {
		// 파일이 없거나 읽기 실패 시 기본적으로 minQueue 반환
		return minQueue, 0, nil, fmt.Errorf("MinIO 파일 크기 확인 실패: %w (기본값: %s 사용)", err, minQueue)
	}

	// threshold와 비교 (threshold는 바이트 단위)
	if metadata.Size < threshold {
		return minQueue, metadata.Size, metadata, nil
	}
	return maxQueue, metadata.Size, metadata, nil
}

// CalculateQueue - MinIO 파일 크기에 따른 큐 계산
func CalculateQueue(minioPath string, threshold int64, minQueue, maxQueue string) (string, int64, error) {
	// MinIO에서 파일 크기 확인 (메타데이터만)
	fileSize, err := getMinIOObjectSize(minioPath)
	if err != nil {
		// 파일이 없거나 읽기 실패 시 기본적으로 minQueue 반환
		return minQueue, 0, fmt.Errorf("MinIO 파일 크기 확인 실패: %w (기본값: %s 사용)", err, minQueue)
	}

	// threshold와 비교 (threshold는 바이트 단위)
	if fileSize < threshold {
		return minQueue, fileSize, nil
	}
	return maxQueue, fileSize, nil
}

// getMinIOMetadata - MinIO에서 객체 메타데이터 가져오기 (다운로드 없이 메타데이터만)
func getMinIOMetadata(minioPath string) (*MinIOMetadata, error) {
	// MinIO 연결 설정 (환경 변수에서 읽기)
	accessKey := os.Getenv("MINIO_ROOT_USER")
	secretKey := os.Getenv("MINIO_ROOT_PASSWORD")

	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("MinIO 환경 변수 설정 안됨 (MINIO_ROOT_USER, MINIO_ROOT_PASSWORD)")
	}

	// MinIO 클라이언트 초기화 (로컬호스트)
	minioClient, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("MinIO 클라이언트 초기화 실패: %w", err)
	}

	// minioPath에서 버킷과 객체 이름 파싱 (예: "bucket/object")
	bucket, object, err := parseMinioPath(minioPath)
	if err != nil {
		return nil, fmt.Errorf("MinIO 경로 파싱 실패: %w", err)
	}

	// 객체 메타데이터만 가져오기 (StatObject - 다운로드 없음)
	ctx := context.Background()
	objInfo, err := minioClient.StatObject(ctx, bucket, object, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("MinIO 객체 메타데이터 조회 실패: %w", err)
	}

	// 메타데이터 구조체로 변환
	metadata := &MinIOMetadata{
		Path:         minioPath,
		Size:         objInfo.Size,
		ETag:         objInfo.ETag,
		LastModified: objInfo.LastModified,
		ContentType:  objInfo.ContentType,
		StorageClass: objInfo.StorageClass,
		UserMetadata: objInfo.UserMetadata,
	}

	return metadata, nil
}

// getMinIOObjectSize - MinIO에서 객체 크기만 가져오기 (다운로드 없이 메타데이터만)
func getMinIOObjectSize(minioPath string) (int64, error) {
	metadata, err := getMinIOMetadata(minioPath)
	if err != nil {
		return 0, err
	}
	return metadata.Size, nil
}

// parseMinioPath - MinIO 경로를 버킷과 객체로 파싱
func parseMinioPath(path string) (string, string, error) {
	// "bucket/object/path" 형식에서 버킷과 객체 경로 분리
	parts := splitPath(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("잘못된 MinIO 경로 형식: %s (예: bucket/path/to/file)", path)
	}

	bucket := parts[0]
	object := joinPath(parts[1:], "/")

	return bucket, object, nil
}

// splitPath - 경로 분할 헬퍼 함수
func splitPath(s, delimiter string) []string {
	if s == "" {
		return []string{}
	}

	var parts []string
	start := 0

	for i := 0; i < len(s); i++ {
		if i+len(delimiter) <= len(s) && s[i:i+len(delimiter)] == delimiter {
			parts = append(parts, s[start:i])
			start = i + len(delimiter)
			i += len(delimiter) - 1
		}
	}

	parts = append(parts, s[start:])
	return parts
}

// joinPath - 경로 결합 헬퍼 함수
func joinPath(parts []string, delimiter string) string {
	if len(parts) == 0 {
		return ""
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += delimiter + parts[i]
	}
	return result
}

// GetExecutorInt - executor 문자열을 정수로 변환
func GetExecutorInt(executorStr string) (int, error) {
	return strconv.Atoi(executorStr)
}

// FormatBytes - 바이트를 사람이 읽기 쉬운 형식으로 변환
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
