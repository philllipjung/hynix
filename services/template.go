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
// Kubernetes 리소스 이름에는 사용할 수 없는 문자 제거 (트레일링 슬래시)
// UID가 있는 경우: SERVICE_ID_PLACEHOLDER-category-uid 형식
// UID가 없는 경우: SERVICE_ID_PLACEHOLDER-category 형식
func ApplyServiceIDLabelsToYAML(yamlStr string, serviceID string) string {
	// Kubernetes 리소스 이름용: 트레일링 슬래시 제거
	k8sSafeName := strings.TrimRight(serviceID, "/")
	return strings.ReplaceAll(yamlStr, "SERVICE_ID_PLACEHOLDER", k8sSafeName)
}

// ApplyServiceIDLabelsWithUIDToYAML - YAML 문자열에 서비스 ID 라벨 적용 (UID 포함)
// UID가 있는 경우: SERVICE_ID_PLACEHOLDER-category-uid 형식
// UID가 없는 경우: SERVICE_ID_PLACEHOLDER-category 형식
func ApplyServiceIDLabelsWithUIDToYAML(yamlStr string, serviceID string, category string, uid string) string {
	// 트레일링 슬래시 제거
	k8sSafeName := strings.TrimRight(serviceID, "/")

	// UID와 category에 따라 포맷 결정
	var replacement string
	if uid != "" {
		replacement = fmt.Sprintf("%s-%s-%s", k8sSafeName, category, uid)
	} else {
		replacement = fmt.Sprintf("%s-%s", k8sSafeName, category)
	}

	// SERVICE_ID_PLACEHOLDER를 포맷된 값으로 교체
	return strings.ReplaceAll(yamlStr, "SERVICE_ID_PLACEHOLDER", replacement)
}

// ApplyBuildNumberToYAML - YAML 문자열에 빌드 번호 적용
// 템플릿 파일의 BUILD_NUMBER를 실제 빌드 번호로 교체
// buildNumber는 minor 버전이며, 전체 버전은 major.minor.patch 형식으로 생성
// major=4 (상수), patch=1 (상수)
// 예: buildNumber="10" → "4.10.1"
func ApplyBuildNumberToYAML(yamlStr string, buildNumber string) string {
	// 전체 버전 생성: 4.{minor}.1
	fullVersion := fmt.Sprintf("4.%s.1", buildNumber)
	return strings.ReplaceAll(yamlStr, "BUILD_NUMBER", fullVersion)
}

// ApplyArgumentsToYAML - YAML의 arguments 섹션을 사용자 제공 arguments로 교체
// arguments는 공백으로 구분된 문자열 (예: "111 222 333")
// arguments가 비어있거나 비어있는 문자열("")이면 template의 기본 arguments 유지
// template에 arguments 섹션이 없으면 새로 생성
func ApplyArgumentsToYAML(yamlStr string, arguments string) string {
	// arguments가 비어있으면 template의 기본값 유지
	if arguments == "" {
		return yamlStr
	}

	// 공백으로 구분하여 arguments 배열 생성
	argArray := strings.Fields(arguments)
	if len(argArray) == 0 {
		return yamlStr
	}

	// YAML에 arguments 섹션이 있는지 확인
	hasArgumentsSection := strings.Contains(yamlStr, "arguments:")

	if !hasArgumentsSection {
		// arguments 섹션이 없으면 새로 생성
		// "batchSchedulerOptions:" 라인 찾아서 그 앞에 삽입
		targetLine := "batchSchedulerOptions:"
		if idx := strings.Index(yamlStr, targetLine); idx >= 0 {
			// 들여쓰기 레벨 계산 (targetLine 앞의 빈 공백 확인)
			lineStart := strings.LastIndex(yamlStr[:idx], "\n") + 1
			prefix := yamlStr[lineStart:idx]
			// 빈 공백 확인 후 들여쓰기 계산 (최소 2칸 들여쓰기)
			indent := strings.TrimRight(prefix, " ")
			if indent == "" {
				indent = "  " // 최소 2칸
			}

			// arguments 섹션 생성
			argsBlock := fmt.Sprintf("%sarguments:\n", indent)
			for _, arg := range argArray {
				argsBlock += fmt.Sprintf("%s  - \"%s\"\n", indent, arg)
			}
			argsBlock += targetLine

			// YAML에 삽입
			result := yamlStr[:idx] + argsBlock + yamlStr[idx:]
			return result
		}
	}

	// arguments 섹션이 있으면 교체
	lines := strings.Split(yamlStr, "\n")
	resultLines := make([]string, 0)
	skipOldArgs := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// arguments: 섹션 찾기
		if strings.HasPrefix(trimmed, "arguments:") {
			resultLines = append(resultLines, line)
			indentLevel := len(line) - len(strings.TrimLeft(line, " "))
			// 사용자 arguments 추가
			indent := strings.Repeat(" ", indentLevel+2)
			for _, arg := range argArray {
				resultLines = append(resultLines, fmt.Sprintf("%s- \"%s\"", indent, arg))
			}
			skipOldArgs = true
			continue
		}

		// 기존 arguments 건너뛰기
		if skipOldArgs {
			if trimmed != "" && !strings.HasPrefix(trimmed, "-") {
				skipOldArgs = false
			}
			continue
		}

		resultLines = append(resultLines, line)
	}

	return strings.Join(resultLines, "\n")
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

// ApplySparkFileCountToYAML - YAML 문자열에 spark.file.count 추가
// 폴더로 인식한 경우 객체 개수를 sparkConf에 추가
func ApplySparkFileCountToYAML(yamlStr string, count int) string {
	lines := strings.Split(yamlStr, "\n")
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 이미 spark.file.count가 있는지 확인
		if strings.HasPrefix(trimmed, "spark.file.count:") {
			found = true
			break
		}

		// sparkConf 섹션에서 마지막 속성 다음에 추가
		// 들여쓰기가 8칸인 sparkConf: 다음 줄로 찾기
		if found && i > 0 {
			prevLine := strings.TrimSpace(lines[i-1])
			if strings.HasPrefix(prevLine, "sparkConf:") || strings.HasPrefix(prevLine, "spark.") {
				// 해당 줄의 인덴트 계산
				indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))
				// spark.file.count 삽입
				newLines := make([]string, len(lines)+1)
				copy(newLines, lines[:i])
				newLines[i] = fmt.Sprintf("%sspark.file.count: \"%d\"", indent, count)
				newLines = append(newLines, lines[i:]...)
				return strings.Join(newLines, "\n")
			}
		}

		// sparkConf 섹션 내에서 찾기
		if strings.HasPrefix(trimmed, "sparkConf:") {
			found = true
		}
	}

	// sparkConf 섹션을 못찾은 경우 (파일 객체일 때)
	// sparkConf 섹션 시작 부분에 spark.file.count 추가
	if !found {
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "sparkConf:") {
				indent := strings.Repeat(" ", 8) // sparkConf 들여쓰기
				newLines := make([]string, len(lines)+1)
				copy(newLines, lines[:i+1])
				newLines[i+1] = fmt.Sprintf("%sspark.file.count: \"%d\"", indent, count)
				newLines = append(newLines, lines[i+1:]...)
				return strings.Join(newLines, "\n")
			}
		}
	}

	return yamlStr
}

