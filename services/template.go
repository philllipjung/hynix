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

// ApplyBuildNumberToYAML - YAML 문자열에 빌드 번호 적용
// 템플릿 파일의 BUILD_NUMBER를 실제 빌드 번호로 교체
func ApplyBuildNumberToYAML(yamlStr string, buildNumber string) string {
	return strings.ReplaceAll(yamlStr, "BUILD_NUMBER", buildNumber)
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

// UpdateExecutorInstances - spec.executor.instances 업데이트
func UpdateExecutorInstances(yamlStr string, instances int) string {
	// YAML에서 "instances: <number>" 패턴을 찾아서 교체
	// 정확하게 executor 섹션의 instances만 교체하기 위해 더 구체적인 패턴 사용
	// 간단한 방법: 라인 단위로 처리하여 executor 섹션의 instances 찾기
	lines := strings.Split(yamlStr, "\n")
	inExecutorSection := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// executor 섹션 시작 확인
		if strings.HasPrefix(trimmed, "executor:") {
			inExecutorSection = true
			continue
		}

		// driver나 다른 섹션 시작하면 executor 섹션 종료
		if inExecutorSection && (strings.HasPrefix(trimmed, "driver:") || strings.HasPrefix(trimmed, "sparkConf:") || strings.HasPrefix(trimmed, "batchScheduler:")) {
			inExecutorSection = false
		}

		// executor 섹션 내의 instances 라인 찾기
		if inExecutorSection && strings.HasPrefix(trimmed, "instances:") {
			// 인덴트 유지하면서 값만 교체
			indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
			lines[i] = fmt.Sprintf("%sinstances: %d", indent, instances)
			break
		}
	}

	return strings.Join(lines, "\n")
}
