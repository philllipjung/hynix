package services

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LoadTemplateRaw - 프로비저닝 ID에 해당하는 템플릿 YAML 로드 (문자열)
func LoadTemplateRaw(provisionID string) (string, error) {
	// 프로비저닝 ID의 하이픈을 언더스코어로 변환
	filename := strings.ReplaceAll(provisionID, "-", "_")
	filePath := fmt.Sprintf("./template/%s.yaml", filename)

	// YAML 파일 읽기
	data, err := ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("템플릿 파일 읽기 실패: %w", err)
	}

	return string(data), nil
}

// ApplyServiceIDLabelsToYAML - YAML 문자열에 서비스 ID 라벨 적용
// 템플릿 파일의 SERVICE_ID_PLACEHOLDER를 실제 서비스 ID로 교체
func ApplyServiceIDLabelsToYAML(yamlStr string, serviceID string) string {
	return strings.ReplaceAll(yamlStr, "SERVICE_ID_PLACEHOLDER", serviceID)
}

// UpdateExecutorMinMember - task-groups annotation의 executor minMember 업데이트
func UpdateExecutorMinMember(taskGroupsStr string, minMember int) (string, error) {
	var taskGroups []map[string]interface{}
	if err := json.Unmarshal([]byte(taskGroupsStr), &taskGroups); err != nil {
		return "", fmt.Errorf("task-groups JSON 파싱 실패: %w", err)
	}

	// executor 그룹 찾아서 minMember 업데이트
	for i := range taskGroups {
		if taskGroups[i]["name"] == "spark-executor" {
			taskGroups[i]["minMember"] = minMember
			break
		}
	}

	// 다시 JSON으로 변환
	updated, err := json.MarshalIndent(taskGroups, "", "        ")
	if err != nil {
		return "", fmt.Errorf("task-groups JSON 변환 실패: %w", err)
	}

	return string(updated), nil
}
