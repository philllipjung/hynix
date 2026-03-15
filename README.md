# Hynix Spark Operator Service

Spark Operator에서 사용되는 SparkApplication CR을 미리보기하는 Go 마이크로서비스입니다.

## 🎯 핵심 기능

### 1. MinIO 기반 리소스 계산
- **파일 크기 기반 큐 선택**: MinIO 메타데이터를 활용한 Yunikorn 큐 자동 선택
  - 파일 < threshold → `min_queue` 사용
  - 파일 ≥ threshold → `max_queue` 사용
- **StatObject 사용**: 파일 다운로드 없이 메타데이터만 조회
- **동적 경로 구성**: `{minio_base_path}/{service_id}`
- **폴더인 경우 spark.file.count 추가**: 폴더(여러 오브젝트)인 경우 오브젝트 수를 YAML에 추가
- **SERVICE_ID_PLACEHOLDER 치환**: `<<service_id>>` 플레이스홀더를 실제 서비스 ID로 치환

### 2. 템플릿 처리
- **BUILD_NUMBER 치환**: config.json의 build_number.number를 템플릿에 적용
- **Executor 설정**: gang_scheduling.executor를 instances와 minMember에 적용
- **서비스 ID 라벨**: SERVICE_ID_PLACEHOLDER를 실제 서비스 ID로 치환
- **UID 파라미터 지원**: Create/Reference 엔드포인트에서 `uid` 파라미터로 카테고리 구분

### 3. 구조화된 JSON 로깅
- 5가지 JSON 로그 형식으로 완전한 추적 가능
- OpenTelemetry Collector와 통합

### 4. 동적 프로비저닝 관리
- **enabled 필드**: 프로비저닝별 활성화/비활성화 제어
  - `enabled: "true"`: 리소스 계산 및 갱스케줄러 설정 적용
  - `enabled: "false"`: 템플릿 원본 유지, BUILD_NUMBER만 적용

## 📡 엔드포인트

### Reference (GET) - YAML 조회 및 미리보기
SparkApplication CR을 조회하고 리소스 계산을 수행하여 YAML로 반환합니다. **Kubernetes에는 제출하지 않습니다.**

**URL:** `GET /api/v1/spark/reference`

**Query Parameters:**
| 파라미터 | 타입 | 필수 여부 | 설명 |
|---------|------|----------|--------|
| `provision_id` | string | ✅ 필수 | 프로비저닝 ID (예: `0001_wfbm`, `0002_wfbm`) |
| `service_id` | string | ✅ 필수 | 서비스 ID (예: `test-00001`, `test-00020`) |
| `category` | string | ✅ 필수 | 카테고리 (예: `test`, `tttm`, `fsa`, `cpa`) |
| `uid` | string | ✅ 필수 | 고유 ID (예: `123`) |
| `arguments` | string | ❌ 선택 | Arguments (공백으로 구분된 문자열) |

**Response:**
- **Content-Type**: `application/x-yaml`
- **Body**: 전체 SparkApplication YAML

#### 요청 예시 1: enabled=true (UID 포함)
```bash
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002_wfbm&service_id=test-00020&category=fsa&uid=123"
```

**서버 로그:**
```json
{
  "level": "info",
  "msg": "생성된 YAML (활성화 모드)",
  "endpoint": "reference",
  "provision_id": "0002_wfbm",
  "service_id": "test-00020",
  "content": "apiVersion: sparkoperator.k8s.io/v1beta2\nkind: SparkApplication\nmetadata:\n  name: test-00020-fsa-123\n..."
}
```

#### 요청 예시 2: enabled=false
```bash
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0001_wfbm&service_id=test-00001&category=test&uid=123"
```

**응답 예시 (service_id-label만 적용됨):**
```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: test-00001-test
  namespace: default
  labels:
    yunikorn.apache.org/app-id: "test-00001-test"
    build-number: "4.0.1"
    spark-app: "true"
spec:
  # ... (테플릿만 적용됨)
```

**처리 로직 흐름도:**
```
1. Request 수신 (parseReferenceRequest)
2. 필수 파라미터 검증 (validateReferenceRequest)
3. 템플릿 로드 (services.LoadTemplateRaw)
4. config.json 로드 (services.LoadConfig)
5. 프로비저닝 설정 찾기 (services.FindProvisionConfig)
6. enabled 확인:
   - false: handleReferenceDisabled() → 템플릿만 적용, 서비스 ID 라벨 적용
   - true: handleReferenceEnabled() → 리소스 계산, 갱스케줄러 설정 적용
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
        "minio": "1234/5678/<<service_id>>/input/",
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

### 3. Template Files
Templates are stored in `template/` directory: `{provision_id}.yaml`

**Template 목록:**
```
template/
├── 0001_wfbm.yaml
├── 0002_wfbm.yaml     # enabled: true (리소스 계산 적용)
└── 0003_wfbm.yaml     # enabled: false (텍플릿 원본 유지)
```

### Template Placeholders

| Placeholder | 설명 | 출처 | 치환되는 값 |
|-------------|---------|--------|---------------|
| `SERVICE_ID_PLACEHOLDER` | 서비스 ID 플레이스홀더 | Request의 `service_id` 파라미터 또는 config.json의 `resource_calculation.minio` 경로의 `<<service_id>>` |
| `<<service_id>>` | 서비스 ID 플레이스홀더 (MinIO 경로용) | config.json의 `resource_calculation.minio` 값에서 실제 `service_id`로 치환 (`services.BuildMinioPath()`) |
| `BUILD_NUMBER` | 빌드 번호 플레이스홀더 | config.json의 `build_number.number` 값 (`services.ApplyBuildNumberToYAML()`) |
| `instances:` | Executor 인스턴스 | config.json의 `gang_scheduling.executor` 값 (`services.UpdateExecutorInstances()`) |
| `minMember:` | Task group 최소 멤버 | config.json의 `gang_scheduling.executor` 값 (task-groups annotation) |

### Processing Steps

1. **Read template** based on `provision_id`
2. **Apply build number** - Replace `BUILD_NUMBER` placeholder
3. **Calculate queue** - Based on MinIO file size vs threshold
4. **Apply executor settings** - Update `instances` and `minMember`
5. **Apply service ID labels** - Replace `SERVICE_ID_PLACEHOLDER` (with category and uid)
   - Format: `{service_id}-{category}-{uid}` or `{service_id}-{category}`
6. **Return final YAML**

## 🗄️ MinIO Integration

### Resource Calculation
MinIO의 `StatObject` API를 사용하여 파일 다운로드 없이 메타데이터만 조회:

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

The service uses structured logging with 5 distinct log types for each request:

### 1. Client Input Log
요청이 보낸 클라이언트 입력 값들을 기록합니다.

```json
{
  "log_type": "client_input",
  "endpoint": "create",
  "provision_id": "0002_wfbm",
  "service_id": "test-00020",
  "category": "fsa",
  "region": "ic",
  "uid": "123",
  "received_at": "2026-02-06T13:50:22+09:00"
}
```

### 2. Config Values Log
config.json에서 로드한 프로비저닝 설정 값을 기록합니다.

```json
{
  "log_type": "config_values",
  "provision_id": "0002_wfbm",
  "enabled": "true",
  "resource_calculation": {
    "minio": "1234/5678/<<service_id>>/input/",
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
MinIO StatObject API로 조회한 파일/폴더 정보를 기록합니다.

```json
{
  "log_type": "minio_resource_calculation",
  "endpoint": "create",
  "provision_id": "0002_wfbm",
  "service_id": "test-00020",
  "minio_path": "1234/5678/test-00020",
  "file_size": 14081741,
  "threshold": 10000000,
  "selected_queue": "max",
  "calculated_at": "2026-02-06T13:50:22+09:00"
}
```

### 4. MinIO Metadata Log
MinIO 객체의 상세 메타데이터를 기록합니다.

```json
{
  "log_type": "minio_metadata",
  "endpoint": "create",
  "provision_id": "0002_wfbm",
  "service_id": "test-00020",
  "minio_path": "1234/5678/test-00020",
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

### 5. Final YAML Result Log
최종적으로 생성된 전체 SparkApplication YAML을 기록합니다.

```json
{
  "log_type": "final_yaml_result",
  "content": "apiVersion: sparkoperator.k8s.io/v1beta2\nkind: SparkApplication\n..."
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
│   ├── 0002_wfbm.yaml           # Template for 0002_wfbm (enabled)
│   └── 0003_wfbm.yaml           # Template for 0003_wfbm
├── handlers/
│   ├── reference.go             # /reference endpoint handler
│   ├── types.go                 # Common types
│   ├── health.go                # Health check handler
│   └── doc.go                   # Package documentation
├── services/
│   ├── config.go                # Configuration management
│   ├── template.go              # Template processing
│   ├── k8s.go                   # Kubernetes client utilities
│   └── utils.go                 # Utility functions
├── logger/
│   └── logger.go                # Structured logging
├── metrics/
│   └── metrics.go               # Prometheus metrics
├── middleware/
│   └── logging.go               # Logging middleware
├── cmd/
│   └── proxy/
│       └── main.go            # Proxy server
├── docs/
│   ├── yunikorn-ui.html         # Yunikorn dashboard
│   ├── spark-metrics-ui.html     # Spark metrics dashboard
│   └── opensearch-discovery-ui.html # OpenSearch dashboard
└── README.md                    # This file
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

### Start API Server
```bash
./main
```

**Output:**
```
2026/02/12 11:13:35 Starting Hynix microservice
2026/02/12 11:13:35 Server listening addr: :8080
```

## 🔄 Code Flow


### Reference Endpoint 처리 흐름
```
┌─────────────────────────────────────────────────────────────────────────┐
│                   GET /api/v1/spark/reference?provision_id=...&service_id=...&category=...&uid=...   │
│                                                               │
└─────────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
                    ┌─────────────────────────────────────────────────────────────────┐
                    │
                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                                                               │
│  GetSparkReference()                                           │
│  ┌─────────────────────────────────────────────────────────────────────┤│
│ │                                                               │
│ │ validateReferenceRequest()                                  │
│ └── parseReferenceRequest()                                   │
│     └── loadProvisionConfig()                            │
│           └── services.LoadConfig()                      │
│                                      └── services.FindProvisionConfig()      │
│                                               │
│  ┌─────────────────────────────────────────────────────────┤       │
│ │ handleReferenceRequest()                                  │
│ │ ├─────────────────────────────────────────────────────────────┤│
│ │ └─ enabled? ─── NO ──┐    YES ──┐       │
│ │                              │          │         │
│ │   │   handleReferenceDisabled()      │   handleReferenceEnabled()  │
│ │   │                            │         │         │
│ │   │                            │   │   services.LoadTemplateRaw()  │         │
│ │   │                            │   │         │   services.ApplyBuildNumberToYAML()  │
│ │   │                            │   │         │   services.ApplyServiceIDLabelsWithUIDToYAML()  │
│ │   │                            │   │         │   └─────────────────────────────────────┐│
│ │   │                            │   │         │   │   CalculateQueueWithMetadata()   │    │
│ │   │                            │   │         │   └── folder? ── count>0 ──┐ │
│ │   │                            │   │         │       │       │       services.ApplySparkFileCountToYAML()│
│ │   │                            │   │         │       │       │   └────────────────────────────────────┘│
│ │   │                            │   │         │       │   sendYAMLResponse(c, yamlOutput) │
│ └────────────────────────────────────────────────────────────────────────────┘│
│                                                               │
│                                                               ▼
└─────────────────────────────────────────────────────────────────────────┘
```

## 🔧 Troubleshooting

### 템플릿 파일 없음
**증상:**
```
Failed to load template: no such file
```

**해결 방법:**
```bash
# 1. provision_id 확인
echo "provision_id: 0002_wfbm"

# 2. 템플릿 파일 존재 여부 확인
ls -la template/ | grep "0002_wfbm.yaml"

# 3. config.json 설정 확인
cat config/config.json | grep -A 5 "0002_wfbm"
```

### 포트 충돌 (Port Already in Use)
**증상:**
```
listen tcp :8080: bind: address already in use
listen tcp :8082: bind: address already in use
```

**확인:**
```bash
# 포트 사용 중인 프로세스 확인
lsof -i :8080 -i :8082

# 특정 포트를 사용하는 프로세스 찾기
ps aux | grep -E "(main|proxy|hynix)" | grep -v ":8080|:8082"

# 필요없는 프로세스 종료
kill -9 <PID>
```

**해결 방법:**
```bash
# 1. 모든 관련 프로세스 종료
pkill -f "main|proxy" 2>/dev/null

# 2. 재시작
cd /root/hynix && ./main

# 또는 백그라운드로 실행
nohup ./main
```

### API 요청 실패 (404/500)
**증상:**
```
{"error": "Unsupported Kubernetes API path"}
{"error": "YAML 파싱 실패: error converting YAML to JSON"}
```

**원인 분석:**

| 에러 타입 | 원인 | 해결 방법 |
|-----------|------|----------|
| **404 Not Found** | 경로가 잘못됨 | 1. URL 경로 확인 (/api/v1/spark/reference) | 2. 메서드 확인 (GET) | 3. config.json에 provision_id 존재 확인 | 4. 템플릿 파일 존재 확인 |
| **500 Server Error** | 서버 내부 오류 | 로그 파일 확인 (/tmp/hynix-api.log) | 1. MinIO 연결 확인 (MINIO_ROOT_USER, MINIO_ROOT_PASSWORD 설정) | 2. Kubernetes 연결 확인 (kubectl cluster-info) |

### MinIO 연결 실패
**증상:**
```
Failed to reach MinIO: dial tcp 127.0.0.1:9000: connect: connection refused
MinIO 파일 크기 확인 실패: MinIO 환경 변수 설정 안됨 (MINIO_ROOT_USER, MINIO_ROOT_PASSWORD) (기본값: min 사용)
```

**해결 방법:**
```bash
# 1. MinIO 서비스 동작 확인
docker ps | grep minio
kubectl get pods -n minio

# 2. 환경 변수 설정
export MINIO_ROOT_USER="your-access-key"
export MINIO_ROOT_PASSWORD="your-secret-key"

# 3. 재시도
curl http://localhost:8080/api/v1/spark/reference?provision_id=0002_wfbm&service_id=test-00020&category=fsa&uid=123"
```

### YAML 파싱 실패
**증상:**
```
error converting YAML to JSON: yaml: line 5: mapping values are not allowed in this context
```

**원인:**
- 템플릿 파일 구문 오류 (indent, 탭/스페이스 혼합)

**해결 방법:**
```bash
# 1. 템플릿 파일 구문 검사
yamllint template/0002_wfbm.yaml

# 2. YAML 내용 확인
cat template/0002_wfbm.yaml | less
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

## 🌐 Proxy Server & Dashboards

### Proxy Server 개요 (`cmd/proxy/main.go`)

웹 대시 서버로서 UI와 API 프록시, OpenSearch 자동 발견 기능 제공.

**포트:** 8082

**주요 기능:**
1. **정적 파일 제공**
   - `/yunikorn-ui.html` - Yunikorn 스케줄러 대시보드
   - `/spark-metrics-ui.html` - Spark 메트릭 대시보드
   - `/opensearch-discovery-ui.html` - OpenSearch 로그 분석 대시보드

2. **API 프록시**
   - `/api/ws/*` → Yunikorn REST API (:9080)
   - `/api/opensearch/*` → OpenSearch API (:9200, port-forward 자동)
   - `/api/api/v1/*` → Kubernetes API (kubectl 통해)

3. **자동 기능**
   - OpenSearch 서비스 자동 발견 및 port-forward 관리
   - CORS 지원 (모든 endpoint)
   - Kubernetes API 프록시 (pod 메트릭)

### 아키텍처

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Web Browser                                │
│  ┌───────────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Yunikorn UI     │  │Spark Metrics │  │ OpenSearch UI   │  │
│  │ /yunikorn-ui.html │  │ /spark-      │  │ /opensearch-    │  │
│  │                   │  │ metrics-ui   │  │ discovery-ui    │  │
│  └───────────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                   Proxy Server (:8082)                     │
│  - Serves static HTML files                                │
│  - Proxies Kubernetes API requests (via kubectl)            │
│  - Proxies Yunikorn API requests                          │
│  - Proxies Spark metrics (via kubectl)                     │
│  - Proxies OpenSearch API (auto port-forward)               │
└─────────────────────────────────────────────────────────────────────────┘
          │                │               │               │
          ▼                ▼               ▼               ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ Kubernetes   │  │ Yunikorn     │  │ Spark Pods   │  │ OpenSearch   │
│ API (kubectl)│  │ API :9080   │  │(metrics)    │  │ :9200 (pf) │
└──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘
```

### 접속 방법

**Yunikorn UI:**
```bash
curl http://localhost:8082/yunikorn-ui.html
```

**Spark Metrics UI:**
```bash
curl http://localhost:8082/spark-metrics-ui.html
```

**API 프록시 (Yunikorn):**
```bash
# Active applications 조회
curl http://localhost:8082/api/ws/v1/partition/default/applications/active

# Queue 정보 조회
curl http://localhost:8082/api/ws/v1/partition/default/queue/root.max
```

**Kubernetes API 프록시 (Pod 메트릭):**
```bash
# Pod 목록 조회 (label selector 지원)
curl "http://localhost:8082/api/api/v1/namespaces/default/pods?labelSelector=spark-app%3Dtest-00020"

# 특정 Pod 메트릭 조회
curl "http://localhost:8082/api/api/v1/namespaces/default/pods/test-00020-fsa-123/proxy/metrics/driver/prometheus/"
curl "http://localhost:8082/api/api/v1/namespaces/default/pods/test-00020-fsa-123/proxy/metrics/executors/prometheus/"
```

**OpenSearch API 프록시:**
```bash
# 전체 로그 검색
curl -X POST http://localhost:8082/api/opensearch/ss4o_logs-*/_search \
  -H "Content-Type: application/json" \
  -d '{"query": "service_id:test-00020"}'

# 시간대별 검색
curl -X POST http://localhost:8082/api/opensearch/ss4o_logs-*/_search \
  -H "Content-Type: application/json" \
  -d '{
    "query": "service_id:test-00020",
    "timeRange": "1h"
  }'
```

### Yunikorn Scheduler UI

**주요 기능:**
- **활성화된 애플리케이션 목록** - 제출된 Spark 애플리케이션 표시
- **큐 할당 현황** - root.default 큐의 리소스 사용량 확인
- **파티션별 리소스** - 각 파티션의 vcore, memory 사용량 모니터링

**화면 구성:**
- 상단: 애플리케이션 목록 (자동 갱신 5초)
- 중단: 파티션별 상태 및 리소스
- 하단: 큐 정보 및 파티션 탐색

### Spark Metrics UI

**주요 기능:**
- **Driver 메트릭** - JVM Memory, GC, CPU, Memory, Shuffle
- **Executor 메트릭** - Per-executor 리소스 사용량 (Active Tasks, Memory, Shuffle)
- **실시간 그래프** - Prometheus endpoint에서 직접 수집

**사용 방법:**
1. 애플리케이션 이름 클릭
2. "Load Metrics" 버튼 클릭
3. 드라이버/executor 선택
4. 시간대별 필터링 (1분/5분/15분)

### OpenSearch Discovery UI

**주요 기능:**
- **전체 텍스트 검색** - 로그 본문에서 키워드 검색
- **시간대별 필터링** - Last 1 hour, 7 days, 30 days, Custom
- **포드 식별** - 각 로그 엔트리에 pod 이름, namespace 표시
- **최대 10,000건** - 한 번에 최대 1만건 표시
- **타임스탬프 정렬** - 최신 로그 먼저 표시

**사용 방법:**
1. 검색어 입력 후 엔터
2. Service ID 필터링
3. 시간 범위 선택
4. 로그 엔트리 클릭하여 상세 보기

## 📌 Service Ports

| Service | Port | Purpose | Access URL |
|---------|-------|----------|------------|
| **API Server** | 8080 | Spark reference endpoint | http://localhost:8080 |
| **Proxy Server** | 8082 | UI + API proxy (Yunikorn/OpenSearch/K8s) | http://localhost:8082 |
| **Yunikorn API** | 9080 | Yunikorn REST API (via proxy) | http://localhost:9080 |
| **OpenSearch** | 9200 | Port-forward (auto) | localhost:9200 |

## 🔄 Version History

### 3.0 (2026-03-15)
- ✅ **Create 엔드포인트 제거**: SparkApplication CR 생성 기능 제거
- ✅ **Reference 전용 서비스**: YAML 미리보기 기능만 제공
- ✅ **Arguments 파라미터 추가**: reference 엔드포인트에 arguments 쿼리 파라미터 추가
- ✅ **서비스 간소화**: Kubernetes API 제출 기능 제거

### 2.0 (2026-02-12)
- ✅ MinIO 통합으로 리소스 계산 개선
- ✅ 5가지 구조화된 JSON 로그 추가
- ✅ BUILD_NUMBER 템플릿 처리
- ✅ **서비스 ID 라벨에 category 포함**: category 파라미터를 사용하여 `{service_id}-{category}-{uid}` 형식 지원
- ✅ **폴더 시 spark.file.count 추가**: 오브젝트 수를 YAML에 추가
- ✅ log 필드명 변경: `yaml_content` → `content`
- ✅ Proxy Server 포트 변경: 8080 → 8082 (충돌 방지)

---

**Version**: 3.0
**Last Updated**: 2026-03-15
**Maintained By**: Data Engineering Team
