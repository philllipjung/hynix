package services

import (
	"fmt"
	"os"
	"strings"
)

// LoadTemplateRaw - 프로비저닝 ID에 해당하는 템플릿 YAML 로드
func LoadTemplateRaw(provisionID string) (string, error) {
	filename := strings.ReplaceAll(provisionID, "-", "_")
	filePath := fmt.Sprintf("./template/%s.yaml", filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("템플릿 파일 읽기 실패: %w", err)
	}

	return string(data), nil
}
