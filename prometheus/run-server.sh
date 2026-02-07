#!/bin/bash

# MinIO 환경변수 설정
export MINIO_ROOT_USER="your-access-key"
export MINIO_ROOT_PASSWORD="your-secret-key"

# 포트 설정
export PORT=8080

# 서버 실행
./main > /tmp/api-server-minio.log 2>&1
