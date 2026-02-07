# 서비스공통 (Service Common)

Spark Operator에서 사용되는 SparkApplication CR을 생성하는 Go 마이크로서비스입니다.

## 🎯 핵심 기능

### 1. MinIO 기반 리소스 계산
- **파일 크기 기반 큐 선택**: MinIO 메타데이터를 활용한 Yunikorn 큐 자동 선택
  - 파일 < threshold → `min_queue` 사용
  - 파일 ≥ threshold → `max_queue` 사용
- **StatObject 사용**: 파일 다운로드 없이 메타데이터만 조회
- **동적 경로 구성**: `{minio_base_path}/{service_id}`

### 2. 템플릿 처리
- **BUILD_NUMBER 치환**: config.json의 build_number.number를 템플릿에 적용
- **Executor 설정**: gang_scheduling.executor를 instances와 minMember에 적용
- **서비스 ID 라벨**: SERVICE_ID_PLACEHOLDER를 실제 서비스 ID로 치환

### 3. 구조화된 JSON 로깅
- 5가지 JSON 로그 형식으로 완전한 추적 가능
- OpenTelemetry Collector와 호환

### 4. 동적 프로비저닝 관리
- **enabled 필드**: 프로비저닝별 활성화/비활성화 제어
  - `enabled: "true"`: 리소스 계산 및 갱스케줄러 설정 적용
  - `enabled: "false"`: 템플릿 원본 유지, BUILD_NUMBER만 적용

## 📡 엔드포인트

### 1. Reference (GET) - YAML 조회 및 미리보기

SparkApplication CR을 조회하고 리소스 계산을 수행하여 YAML로 반환합니다. **Kubernetes에는 제출하지 않습니다.**

**URL:** `GET /api/v1/spark/reference`

**Query Parameters:**
- `provision_id`: 프로비저닝 ID (예: `0001_wfbm`, `0002_wfbm`)
- `service_id`: 서비스 ID (예: `test-00001`)
- `category`: 카테고리 (예: `test`)

**Response:**
- **Content-Type**: `application/x-yaml`
- **Body**: 전체 SparkApplication YAML

**예시:**
```bash
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002_wfbm&service_id=test-00001&category=test&region=default"
```

**서버 로그:**
```json
{
  "level": "info",
  "msg": "생성된 YAML (비활성화 모드)",
  "endpoint": "reference",
  "provision_id": "0001_wfbm",
  "service_id": "test-0001",
  "content": "apiVersion: sparkoperator.k8s.io/v1beta2\nkind: SparkApplication\n..."
}
```

### 2. Create (POST) - Kubernetes 제출

SparkApplication CR을 Kubernetes 클러스터에 생성합니다.

**URL:** `POST /api/v1/spark/create`

**Request Body:**
```json
{
  "provision_id": "0002_wfbm",
  "service_id": "test-00001",
  "category": "test",
  "region": "default"
}
```

**Response (성공):**
```json
{
  "message": "SparkApplication CR 생성 성공",
  "provision_id": "0002_wfbm",
  "service_id": "test-00001",
  "category": "test",
  "region": "default",
  "result": {
    "name": "test-00001",
    "namespace": "default"
  }
}
```

**예시:**
```bash
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002_wfbm",
    "service_id": "test-00001",
    "category": "test",
    "region": "default"
  }'
```

## ⚙️ Configuration

### config.json Structure

```json
{
  "config_specs": [
    {
      "provision_id": "0002_wfbm",
      "enabled": "true",
      "resource_calculation": {
        "minio": "1234/5678",
        "threshold": 10000000,
        "min_queue": "min",
        "max_queue": "max"
      },
      "gang_scheduling": {
        "cpu": "5",
        "memory": "10",
        "executor": "1"
      },
      "build_number": {
        "number": "4.0.1"
      }
    }
  ]
}
```

### Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `provision_id` | string | 고유 프로비저닝 식별자 |
| `enabled` | string | 활성화/비활성화 ("true"/"false") |
| `resource_calculation.minio` | string | MinIO 베이스 경로 (bucket/object_prefix) |
| `resource_calculation.threshold` | integer | 파일 크기 기준값 (bytes) |
| `resource_calculation.min_queue` | string | 작은 파일용 큐 이름 |
| `resource_calculation.max_queue` | string | 큰 파일용 큐 이름 |
| `gang_scheduling.cpu` | string | CPU 코어 수 |
| `gang_scheduling.memory` | string | 메모리 크기 |
| `gang_scheduling.executor` | string | Executor 인스턴스 수 |
| `build_number.number` | string | 빌드 버전 |

## 🔄 Template Processing

### Template Files

Templates are stored in `template/` directory: `{provision_id}.yaml`

```
template/
├── 0001_wfbm.yaml
└── 0002_wfbm.yaml
```

### Template Placeholders

| Placeholder | Description | Source |
|-------------|-------------|--------|
| `SERVICE_ID_PLACEHOLDER` | Service ID | Request parameter |
| `BUILD_NUMBER` | Build version | config.json `build_number.number` |
| `instances:` | Executor instances | config.json `gang_scheduling.executor` |
| `minMember:` | Task group min member | config.json `gang_scheduling.executor` |

### Processing Steps

1. **Read template** based on `provision_id`
2. **Apply build number** - Replace `BUILD_NUMBER` placeholder
3. **Calculate queue** - Based on MinIO file size vs threshold
4. **Apply executor settings** - Update `instances` and `minMember`
5. **Apply service ID labels** - Replace `SERVICE_ID_PLACEHOLDER`
6. **Return final YAML**

## 🗄️ MinIO Integration

### Resource Calculation

MinIO의 `StatObject` API를 사용하여 파일 다운로드 없이 메타데이터만 조회:

```go
MinIO Path: {resource_calculation.minio}/{service_id}
Example: 1234/5678/test-00001
```

**Logic:**
```
if file_size < threshold:
    selected_queue = min_queue
else:
    selected_queue = max_queue
```

### Environment Variables

- `MINIO_ROOT_USER`: MinIO access key
- `MINIO_ROOT_PASSWORD`: MinIO secret key
- `MINIO_ENDPOINT`: MinIO server (default: localhost:9000)

### Retrieved Metadata

```json
{
  "minio_path": "1234/5678/test-00001",
  "size_bytes": 14081741,
  "size_formatted": "13.4 MiB",
  "etag": "4442d5294978a87a54bb74fa8e734a0c",
  "last_modified": "2026-02-06T00:31:34Z",
  "content_type": "application/octet-stream"
}
```

## 📝 Structured Logging

Each `/create` request generates 5 structured JSON logs:

### 1. Client Input Log
```json
{
  "log_type": "client_input",
  "endpoint": "create",
  "provision_id": "0002_wfbm",
  "service_id": "test-00001",
  "category": "test",
  "region": "default",
  "received_at": "2026-02-06T13:50:22+09:00"
}
```

### 2. Config Values Log
```json
{
  "log_type": "config_values",
  "provision_id": "0002_wfbm",
  "enabled": "true",
  "resource_calculation": {
    "minio": "1234/5678",
    "threshold": 10000000,
    "min_queue": "min",
    "max_queue": "max"
  },
  "gang_scheduling": {
    "cpu": "5",
    "memory": "10",
    "executor": "1"
  },
  "build_number": {
    "number": "4.0.1"
  }
}
```

### 3. MinIO Resource Calculation Log
```json
{
  "log_type": "minio_resource_calculation",
  "endpoint": "create",
  "provision_id": "0002_wfbm",
  "service_id": "test-00001",
  "minio_path": "1234/5678/test-00001",
  "file_size": 14081741,
  "threshold": 10000000,
  "selected_queue": "max",
  "calculated_at": "2026-02-06T13:50:22+09:00"
}
```

### 4. Final YAML Result Log
```json
{
  "log_type": "final_yaml_result",
  "content": "apiVersion: sparkoperator.k8s.io/v1beta2\nkind: SparkApplication\n...",
  "generated_at": "2026-02-06T13:50:22+09:00"
}
```

### 5. MinIO Metadata Log
```json
{
  "log_type": "minio_metadata",
  "endpoint": "create",
  "provision_id": "0002_wfbm",
  "service_id": "test-00001",
  "minio_path": "1234/5678/test-00001",
  "size_bytes": 14081741,
  "size_formatted": "13.4 MiB",
  "etag": "4442d5294978a87a54bb74fa8e734a0c",
  "last_modified": "2026-02-06T00:31:34Z",
  "content_type": "application/octet-stream",
  "storage_class": "",
  "user_metadata": {},
  "fetched_at": "2026-02-06T13:50:22+09:00"
}
```

## 📊 Project Structure

```
/root/hynix/
├── main.go                      # Application entry point
├── config/
│   └── config.json              # Provision configurations
├── template/
│   ├── 0001_wfbm.yaml           # Template for 0001_wfbm
│   └── 0002_wfbm.yaml           # Template for 0002_wfbm
├── handlers/
│   ├── create.go                # /create endpoint handler
│   └── reference.go             # /reference endpoint handler
├── services/
│   ├── config.go                # Configuration management
│   └── template.go              # Template processing
├── logger/
│   └── logger.go                # Structured logging
├── metrics/
│   └── metrics.go               # Prometheus metrics
├── run-server.sh                # Startup script
└── README.md                    # Documentation
```

## 🚀 Quick Start

### Build
```bash
cd /root/hynix
go build -o main .
```

### Set Environment Variables
```bash
export MINIO_ROOT_USER="your-access-key"
export MINIO_ROOT_PASSWORD="your-secret-key"
export PORT=8080
```

### Start Server
```bash
./main
```

Or use the startup script:
```bash
chmod +x run-server.sh
./run-server.sh
```

## 🔧 Troubleshooting

### MinIO Connection Failed
```bash
# Check environment variables
echo $MINIO_ROOT_USER
echo $MINIO_ROOT_PASSWORD

# Test MinIO connection
curl http://localhost:9000
```

### Template Not Found
```bash
# Verify template file exists
ls -la template/0002_wfbm.yaml

# Check config.json for correct provision_id
cat config/config.json | grep provision_id
```

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>
```

## 📊 Metrics

Prometheus metrics are exposed at `/metrics`:

- `hynix_requests_total`: Total request count
- `hynix_request_duration_seconds`: Request latency
- `hynix_provision_mode`: Provision mode (enabled/disabled)
- `hynix_queue_selection`: Queue selection count

## 🔍 Health Check

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "2.0"
}
```

## 📌 Notes

### 필수 구성 요건
1. **Spark Operator**: 클러스터에 Spark Operator가 사전 설치되어야 함
2. **Yunikorn Scheduler**: Gang Scheduling을 위해 Yunikorn이 구성되어 있어야 함
3. **MinIO**: 리소스 계산을 위해 MinIO 서버 필요
4. **ServiceAccount**: `spark-operator-spark` ServiceAccount와 RBAC 권한 필요

### enabled 모드 비교

| 특징 | enabled: "true" (활성화) | enabled: "false" (비활성화) |
|------|---------------------|----------------------|
| 리소스 계산 | ✅ 수행 (MinIO) | ❌ 건너뜀 |
| 큐 계산 | ✅ 파일 크기에 따라 min/max 선택 | ❌ 템플릿 그대로 |
| BUILD_NUMBER 적용 | ✅ 수행 | ✅ 수행 |
| 서비스 ID 치환 | ✅ 수행 | ✅ 수행 |

## 🔄 Version History

### 2.0 (2026-02-06)
- ✅ MinIO 통합으로 리소스 계산 개선
- ✅ 5가지 구조화된 JSON 로그 추가
- ✅ BUILD_NUMBER 템플릿 처리
- ✅ log 필드명 "yaml_content" → "content" 변경
- ✅ 비활성화 모드에서도 BUILD_NUMBER 적용

---

**Version**: 2.0
**Last Updated**: 2026-02-06
**Maintained By**: Data Engineering Team
