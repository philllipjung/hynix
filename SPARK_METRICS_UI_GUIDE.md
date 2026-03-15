# Spark Metrics UI 사용 가이드

## 접속 방법

```
http://localhost:8082/spark-metrics-ui.html
```

## 사용 방법

### 1단계: Application Name 입력

UI 페이지 상단의 **"Application Name"** 입력 필드에 Spark 애플리케이션 이름을 입력합니다.

**예시:**
- `test-00020-fsa-123` (전체 이름)
- `test-00020` (부분 일치도 가능)

### 2단계: "Load Metrics" 버튼 클릭

입력 후 **"Load Metrics"** 버튼을 클릭합니다.

### 3단계: 메트릭 확인

- **Driver 메트릭**: Driver Pod의 모든 메트릭 표시
- **Executor 메트릭**: Executor Pod별 메트릭 필터링

---

## 예시

### 1. 현재 실행 중인 애플리케이션 확인

```bash
kubectl get sparkapplication
```

```
NAME                 STATUS
test-00020-fsa-123   RUNNING
```

### 2. UI에서 Application Name 입력

```
Application Name: test-00020-fsa-123
```

또는

```
Application Name: test-00020
```

### 3. Load Metrics 버튼 클릭

---

## 메트릭 카테고리

### Driver 메트릭
- **BlockManager**: 메모리, 디스크 사용량
- **DAGScheduler**: Job, Stage 상태
- **Executor**: 실행 중인 Executor 정보
- **HiveExternalCatalog**: 외부 카탈로그 작업

### Executor 메트릭
- **ExecutorMetrics**: JVM 메모리, CPU 사용량
- **ThreadPool**: 스레드 풀 상태

---

## 문제 해결

### "No pods found for this application" 오류

1. **Application Name 확인**
   ```bash
   kubectl get sparkapplication
   kubectl get pods -l spark-app=true
   ```

2. **Pod 상태 확인**
   ```bash
   kubectl get pods -l spark-app=true
   ```

3. **Pod 메트릭 엔드포인트 확인**
   ```bash
   kubectl exec <pod-name> -- curl -s http://localhost:4040/metrics/driver/prometheus/
   ```

### 메트릭이 표시되지 않음

1. **Proxy 서버 확인**
   ```bash
   ps aux | grep proxy-server
   ```

2. **API 테스트**
   ```bash
   # Pods 목록
   curl "http://localhost:8082/api/api/v1/namespaces/default/pods?labelSelector=spark-app%3Dtrue"

   # 메트릭
   curl "http://localhost:8082/api/api/v1/namespaces/default/pods/<pod-name>/proxy/metrics/driver/prometheus/"
   ```

---

## API 엔드포인트

### Pods 목록
```
GET /api/api/v1/namespaces/default/pods?labelSelector=spark-app=true
```

### Driver 메트릭
```
GET /api/api/v1/namespaces/default/pods/{pod-name}/proxy/metrics/driver/prometheus/
```

### Executor 메트릭
```
GET /api/api/v1/namespaces/default/pods/{pod-name}/proxy/metrics/executors/prometheus/
```

---

## 샘플 Application Names

| Application Name | 설명 |
|------------------|------|
| `test-00020-fsa-123` | 전체 이름 |
| `test-00020` | 부분 일치 |
| `test-00020-fsa` | Category 포함 |

---

## 주의 사항

1. **반드시 Application Name을 입력해야 합니다** (자동 로드 아님)
2. Spark 애플리케이션이 **RUNNING** 상태여야 합니다
3. Proxy 서버(8082)가 실행 중이어야 합니다
