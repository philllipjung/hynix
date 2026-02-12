package services

import (
	"os"
)

// ReadFile - 파일 읽기 헬퍼 함수
func ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}
