# Proxy Server

Spark Operator와 Yunikorn Scheduler를 위한 웹 프록시 서버입니다.

## 🎯 핵심 기능

1. **정적 파일 제공**: Yunikorn UI, Spark Metrics UI, OpenSearch Discovery UI
2. **API 프록시**: Yunikorn REST API, Kubernetes API (kubectl 통해), OpenSearch API
3. **자동 기능**: OpenSearch 서비스 자동 발견 및 port-forward 관리
4. **CORS 지원**: 모든 endpoint에서 크로스-오리진 요청 허용

## 📡 아키텍처

### 포트 의존성 다이어그램

```
┌─────────────────────────────────────────────────────────────────┐
│                     User Browser                              │
│  http://localhost:8082/yunikorn-ui.html                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Proxy Server (:8082)                        │
│  - Serves UI HTML files                                       │
│  - Proxies API calls to backend services                       │
└─────────────────────────────────────────────────────────────────┘
         │               │               │               │
         ▼               ▼               ▼               ▼
   Port 9080       Port 9090       Port 9200       Port 8080
   Yunikorn        Prometheus      OpenSearch       API Server
```

### 상세 아키텍처

```
┌─────────────────────────────────────────────────────────────────────┐
│                   Web Browser                                │
│  ┌───────────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Yunikorn UI     │  │Spark Metrics │  │ OpenSearch UI   │  │
│  │ /yunikorn-ui.html │  │ /spark-      │  │ /opensearch-    │  │
│  │                   │  │ metrics-ui.html │  │ discovery-ui.html │  │
│  └───────────────────┘  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
                    ┌───────────────────────────────────────────────────────────┐
                    │
                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                   Proxy Server (:8082)                     │
│  ┌───────────────────────────────────────────────────────────────┤│
│ │  Serves static HTML files                                │
│ │ Proxies Kubernetes API requests (via kubectl)            │
│ │ Proxies Yunikorn API requests (:9080)                  │
│ │ Proxies OpenSearch API (:9200, auto port-forward)       │
│ │ Auto-discovers OpenSearch service                        │
└─────────────────────────────────────────────────────────────────────────┘
          │                │               │               │
          ▼                ▼               ▼               ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│ Kubernetes   │  │ Yunikorn     │  │ Spark Pods   │  │ OpenSearch   │  │
│ API (kubectl)│  │ API :9080   │  │(metrics)    │  │ :9200 (pf) │  │
└──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘
```

## 🚀 빠른 시작

### 모든 포트-포워드 및 서비스 시작

```bash
#!/bin/bash
# start-all-services.sh

echo "Starting all services..."

# 1. API Server (Port 8080)
cd /root/hynix
pkill -f "go run main.go" 2>/dev/null
go run main.go > /tmp/hynix-api.log 2>&1 &
echo "✓ API Server started on :8080"

# 2. Proxy Server (Port 8082)
cd /root/hynix/cmd/proxy
pkill -f "cmd/proxy/main.go" 2>/dev/null
go run main.go > /tmp/proxy-server.log 2>&1 &
echo "✓ Proxy Server started on :8082"

# 3. Yunikorn Port-Forward (Port 9080)
pkill -f "port-forward.*9080" 2>/dev/null
kubectl port-forward -n default svc/yunikorn-service 9080:9080 > /dev/null 2>&1 &
echo "✓ Yunikorn port-forward started on :9080"

# 4. OpenSearch Port-Forward (Port 9200) - 자동 시작됨
pkill -f "port-forward.*9200" 2>/dev/null
kubectl port-forward -n opensearch svc/opensearch 9200:9200 > /dev/null 2>&1 &
echo "✓ OpenSearch port-forward started on :9200"

# 5. Prometheus Port-Forward (Port 9090)
pkill -f "port-forward.*9090" 2>/dev/null
kubectl port-forward -n monitoring svc/prometheus-k8s 9090:9090 > /dev/null 2>&1 &
echo "✓ Prometheus port-forward started on :9090"

echo ""
echo "All services started!"
echo "  - Yunikorn UI: http://localhost:8082/yunikorn-ui.html"
echo "  - Spark Metrics: http://localhost:8082/spark-metrics-ui.html"
echo "  - OpenSearch: http://localhost:8082/opensearch-discovery-ui.html"
```

### 포트-포워드 요약

| Purpose | Local Port | K8s Service | Namespace | Command |
|---------|-----------|-------------|-----------|---------|
| **API Server** | 8080 | - | - | `cd /root/hynix && go run main.go` |
| **Proxy Server** | 8082 | - | - | `cd /root/hynix/cmd/proxy && go run main.go` |
| **Yunikorn API** | 9080 | svc/yunikorn-service | default | `kubectl port-forward -n default svc/yunikorn-service 9080:9080` |
| **OpenSearch API** | 9200 | svc/opensearch | opensearch | `kubectl port-forward -n opensearch svc/opensearch 9200:9200` |
| **Prometheus API** | 9090 | svc/prometheus-k8s | monitoring | `kubectl port-forward -n monitoring svc/prometheus-k8s 9090:9090` |

### 빌드
```bash
cd /root/hynix/cmd/proxy
go build -o proxy-server main.go
./proxy-server
```

### 확인
```bash
# 서버 시작 확인
curl http://localhost:8082/

# 로그 확인
tail -f /tmp/proxy-server.log

# 포트 확인
lsof -i :8082
```

## ⚙️ 설정

### 포트 설정
```go
const (
    yunikornAPI = "http://localhost:9080"  // Yunikorn REST API
    uiPort      = 8082                      // Proxy server port (8080에서 8082로 변경)
)
```

## 📡 엔드포인트

### 1. 정적 파일 제공
- **Yunikorn UI** - `GET /yunikorn-ui.html`
  - 리다이렉트: `/root/hynix/docs/yunikorn-ui.html`
- **Spark Metrics UI** - `GET /spark-metrics-ui.html`
  - 리다이렉트: `/root/hynix/docs/spark-metrics-ui.html`
- **OpenSearch Discovery UI** - `GET /opensearch-discovery-ui.html`
  - 리다이렉트: `/root/hynix/docs/opensearch-discovery-ui.html`

### 2. Kubernetes API 프록시 (kubectl 통해)
- **Pod 목록 조회** - `GET /api/api/v1/namespaces/{ns}/pods`
  - 라벨 선택자 지원 (labelSelector)
  - 메트릭 조회 (kubectl exec)

---

## 🔍 Pod 라벨링으로 애플케이션 이름 조회

### 문제점
기존 구현에서는 모든 파드(pod)을 조회한 후 라벨로 필터링했습니다. 이로 인해 동일 애플케이션 이름(ex: test-00020-fsa-123)으로 메트릭을 조회하려면 **정확한 라벨**이 파드에 미리 추가되어야 합니다.

### 해결방법: Pod 생성 시 라벨 자동 추가

SparkApplication CR을 생성할 때 `spark-app: "true"` 라벨이 자동으로 추가되도록 설정하세요. 그러면 별도의 쿼리 작업 없이 **애플케이션 이름으로 메트릭을 조회**할 수 있습니다.

#### 방법 1: 템플릿 수정 (권장)

**template/0002_wfbm.yaml 수정:**
```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: SERVICE_ID_PLACEHOLDER
  labels:
    spark-app: "true"           # 중요: 애플케이션 이름으로 그룹핑을 위한 라벨
    yunikorn.apache.org/app-id: "SERVICE_ID_PLACEHOLDER"
spec:
  driver:
    metadata:
      labels:
        spark-app: "true"                    # 드라이버 파드에도 라벨 복사
        yunikorn.apache.org/app-id: "SERVICE_ID_PLACEHOLDER"
  executor:
    replicas: 3
    replicas:
      metadata:
        labels:
          spark-app: "true"                  # 익스큐터 파드에도 라벨 복사
          yunikorn.apache.org/app-id: "SERVICE_ID_PLACEHOLDER"
```

**장점:**
- ✅ 장점: 템플릿만 수정하면 됨
- ✅ 자동화: Spark Operator가 CR을 생성하면 라벨이 자동으로 추가됨
- ⚠️ 단점: Spark Operator가 이미 실행 중인 경우 기존 파드에는 라벨이 없을 수 있음

#### 방법 2: 라벨 수동 추가 (kubectl label 명령)

**생성 후 라벨 추가:**
```bash
# 1. SparkApplication CR 생성 후 기다림
crd_poll_interval=5
while true; do
  sleep 5
  # 'driver-' 또는 '-exec-'로 시작하는 파드 찾기
  DRIVER_POD=$(kubectl get pods -n default -l spark-app=true -o name | grep -E 'driver-.{5}' | head -1)
  EXEC_PODS=$(kubectl get pods -n default -l spark-app=true -o name | grep -E 'exec-.{3,5}' | tr '\n' ' ')

  # 드라이버 파드에 라벨 추가
  if [ -n "$DRIVER_POD" ]; then
    echo "Adding labels to driver pod: $DRIVER_POD"
    kubectl label pod "$DRIVER_POD" \
      spark-app="true" \
      yunikorn.apache.org/app-id="test-00020-fsa-123"
  fi

  # 익스큐터 파드에 라벨 추가
  for POD in $EXEC_PODS; do
    echo "Adding labels to executor pod: $POD"
    kubectl label pod "$POD" \
      spark-app="true" \
      yunikorn.apache.org/app-id="test-00020-fsa-123"
  done

  # 모든 파드에 라벨 추가 확인
  LABELED_PODS=$(kubectl get pods -n default -l spark-app=true \
      -L yunikorn.apache.org/app-id --no-headers 2>/dev/null)

  if echo "$LABELED_PODS" | grep -q "test-00020-fsa-123"; then
    echo "Labels successfully added!"
    break
  fi
done
```

**간소 스크립트:**
```bash
# 5초마다 라벨 확인 (실제로는 생성 즉시 라벨이 추가됨)
watch -n 5 'kubectl get pods -n default -l spark-app=true -L yunikorn.apache.org/app-id --no-headers 2>/dev/null'
```

#### 방법 3: Proxy 서버에 app-name 필터링 추가 (추천)

**기존 k8sProxyHandler 개선:**
```go
// From cmd/proxy/main.go
func k8sProxyHandler(w http.ResponseWriter, r *http.Request, path string) {
    // ... existing code ...

    // Parse query parameters
    r.ParseForm()

    // Get query parameters
    appFilter := r.FormValue("app-name")  // 새로운 app-name 필터
    labelFilter := r.FormValue("label")  // 새로운 label 필터

    // Build kubectl command with app-name filter
    args := []string{"get", "pods", "-n", namespace}

    // app-name 라벨 필터링 추가
    if appFilter != "" {
        args = append(args, "-l", fmt.Sprintf("spark-app=%s", appFilter))
        log.Printf("Querying pods with app-name filter: %s", appFilter)
    } else if labelFilter != "" {
        args = append(args, "-l", labelFilter)
        log.Printf("Querying pods with label filter: %s", labelFilter)
    }

    // Execute kubectl
    cmd := exec.Command("kubectl", args...)
    output, err := cmd.CombinedOutput()

    // Return results
    w.Header().Set("Content-Type", "application/json")
    w.Write(output)
}
```

**Proxy 서버 재시작:**
```bash
# 프로세스 종료
pkill -f proxy-server

# 재빌드
go run main.go

# 테스트
# app-name으로 조회 (labelSelector 대신 app-name 필터 사용)
curl "http://localhost:8082/api/v1/namespaces/default/pods?app-name=test-00020-fsa-123"

# label로 조회 (기존 labelSelector 사용)
curl "http://localhost:8082/api/v1/namespaces/default/pods?labelSelector=spark-app%3Dtest-00020-fsa-123"
```

**장점:**
- ✅ 기존 labelSelector와 호환
- ✅ app-name 필터로 더 직관적인 쿼리 가능
- ✅ UI에서 app-name 입력 없이 기존 labelSelector 파싱 그대로 사용 가능

---

## 📊 사용 예시

### 1. 템플릿 자동 라벨링 (권장 있을 때)

```bash
# SparkApplication CR 생성
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002_wfbm",
    "service_id": "test-00020",
    "category": "fsa",
    "region": "ic",
    "uid": "123"
  }'

# 자동으로 라벨 추가됨 확인
kubectl get pods -n default -l spark-app=true -L yunikorn.apache.org/app-id
```

**출력 예시:**
```
NAME                                READY   STATUS    RESTARTS   AGE   LABELS
test-00020-fsa-123-driver       1/1     Running   0          23s   spark-app=true,yunikorn.apache.org/app-id=test-00020-fsa-123
test-00020-fsa-123-executor-1     1/1     Running   0          23s   spark-app=true,yunikorn.apache.org/app-id=test-00020-fsa-123
test-00020-fsa-123-executor-2     1/1     Running   0          23s   spark-app=true,yunikorn.apache.org/app-id=test-00020-fsa-123
```

### 2. Proxy 서버 app-name 필터 사용

```bash
# 1. app-name으로 정확한 조회
curl "http://localhost:8082/api/v1/namespaces/default/pods?app-name=test-00020-fsa-123"

# 2. 라벨 조회
curl "http://localhost:8082/api/v1/namespaces/default/pods?labelSelector=spark-app%3Dtest-00020-fsa-123"

# 3. 여러 앱 동시 조회
# 여러 앱 이름을 쉼표로 구분하여 조회
curl "http://localhost:8082/api/v1/namespaces/default/pods" \
  -G -d 'apps@{"test-00020-fsa-123,test-00020-cp-456"}'
```

---

## 🎯 권장 사항

### 방법 1: 템플릿 수정 (권장 필요)
- ⚠️ 단점: Spark Operator 수정 권한 필요
- ✅ 장점: 템플릿만으로 설정 가능

### 방법 2: 수동 라벨링 (kubectl label)
- ✅ 장점: 추가 작업 없음
- ✅ 장점: 이미 생성된 파드에도 라벨 추가 가능
- ⚠️ 단점: Spark Operator가 재시작하면 라벨이 초기화될 수 있음

### 방법 3: Proxy 서버 app-name 필터 (추천)
- ✅ 장점: UI에서 직관적인 앱 이름 조회 가능
- ✅ 장점: app-name으로 더 정확한 필터링
- ✅ 장점: 이미 있는 labelSelector와 호환
- ⚠️ 단점: Proxy 서버 코드 수정 필요

---

## 🔧 트러븅슈팅

### 포트 및 포트-포워드 관련

| 이슈 | 해결 방법 | 설명 |
|------|----------|--------|
| **HTTP 502 Bad Gateway** | 포트-포워드 확인 | `lsof -i :9080`로 Yunikorn 포트-포워드 동작 확인 |
| **Yunikorn UI 로딩 실패** | Yunikorn 포트-포워드 시작 | `kubectl port-forward -n default svc/yunikorn-service 9080:9080` |
| **OpenSearch 연결 실패** | OpenSearch 포트-포워드 시작 | `kubectl port-forward -n opensearch svc/opensearch 9200:9200` |
| **Prometheus 메트릭 없음** | Prometheus 포트-포워드 시작 | `kubectl port-forward -n monitoring svc/prometheus-k8s 9090:9090` |
| **포트 충돌 (8080/8082)** | API Server와 Proxy Server 포트 확인 | API Server: 8080, Proxy Server: 8082 |

### 포트 확인 명령어

```bash
# 모든 필요 포트 확인
echo "=== Port Status ==="
echo "Port 8080 (API Server):"
lsof -i :8080 2>/dev/null || echo "  NOT RUNNING"

echo "Port 8082 (Proxy Server):"
lsof -i :8082 2>/dev/null || echo "  NOT RUNNING"

echo "Port 9080 (Yunikorn):"
lsof -i :9080 2>/dev/null || echo "  NOT RUNNING"

echo "Port 9200 (OpenSearch):"
lsof -i :9200 2>/dev/null || echo "  NOT RUNNING"

echo "Port 9090 (Prometheus):"
lsof -i :9090 2>/dev/null || echo "  NOT RUNNING"
```

### Pod 라벨링 관련

| 이슈 | 해결 방법 | 설명 |
|------|----------|--------|
| **파드가 라벨이 없음** | SparkApplication CR 생성 시 자동 라벨 추가 | template의 `spark-app: "true"` 설정 확인 |
| **kubectl label 명령 복잡** | 라벨 추가할 때마다 `kubectl label pod` 실행하면 모든 파드에 중복 실행 | 너무 안전하고 빠름 |
| **app-name 필터 미지원** | UI에서 app-name 입력 받을 수 있도록 Proxy 서버에 `/api/v1/namespaces/default/pods?app-name=xxx` 엔드포인트 추가 필요 |
| **프로세스 권한** | kubectl 실행 권한과 pod 수정 권한 확인 | ServiceAccount에 adequate RBAC 설정 되어 있는지 확인 |

---

## 📌 빠른 시작

### 추천 순서
1. ⭐ **방법 1 (테플릿 수정)**: 가장 깔끔 방법
2. ⭐ **방법 3 (Proxy 서버 개선)**: app-name 필터를 UI에 추가하여 사용자 친화성 개선

### 파일 수정 순서
1. **template/0002_wfbm.yaml**에 위 내용 추가
2. **cmd/proxy/main.go**를 수정하여 app-name 필터링 로직 추가

---

**버전:** 1.0 (2026-02-12)

**작성자:** Data Engineering Team
