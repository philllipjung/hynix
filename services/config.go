package services

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config - 설정 파일 구조체
type Config struct {
	ConfigSpecs []ConfigSpec `json:"config_specs"`
}

// ConfigSpec - 프로비저닝 설정
type ConfigSpec struct {
	ProvisionID         string             `json:"provision_id"`
	Enabled             string            `json:"enabled"`
	ResourceCalculation ResourceCalculation `json:"resource_calculation"`
	GangScheduling      GangScheduling      `json:"gang_scheduling"`
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

// CalculateQueue - 파일 크기에 따른 큐 계산
func CalculateQueue(minioPath string, threshold int64, minQueue, maxQueue string) (string, error) {
	// 파일 크기 확인
	fileSize, err := getFileSize(minioPath)
	if err != nil {
		// 파일이 없거나 읽기 실패 시 기본적으로 minQueue 반환
		return minQueue, nil
	}

	// threshold와 비교
	if fileSize < threshold {
		return minQueue, nil
	}
	return maxQueue, nil
}

// getFileSize - 파일 크기 가져오기 (MB 단위)
func getFileSize(path string) (int64, error) {
	// 경로에서 백슬래시를 슬래시로 변환
	cleanPath := path
	if len(path) > 0 && path[0] == '\\' {
		// Windows 경로 처리 (실제 환경에서는 적절히 변환 필요)
		cleanPath = "/" + path[1:]
	}
	cleanPath = convertWindowsPath(cleanPath)

	// 파일 정보 가져오기
	info, err := os.Stat(cleanPath)
	if err != nil {
		return 0, err
	}

	// 바이트를 MB로 변환
	sizeMB := info.Size() / (1024 * 1024)
	return sizeMB, nil
}

// convertWindowsPath - Windows 경로를 Unix 경로로 변환
func convertWindowsPath(path string) string {
	// 실제 구현에서는 환경에 따라 적절히 변환
	// 예: \root\hynix\kubernetes.zip -> /root/hynix/kubernetes.zip
	converted := path
	for i := 0; i < len(converted); i++ {
		if converted[i] == '\\' {
			converted = converted[:i] + "/" + converted[i+1:]
		}
	}
	return converted
}
