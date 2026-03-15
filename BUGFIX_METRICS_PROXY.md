# Spark Metrics Proxy 버그 수정 보고서

## 문제 개요

**날짜:** 2026-02-13
**컴포넌트:** Proxy Server (`/root/hynix/cmd/proxy/main.go`)
**영향:** Spark Metrics UI에서 메트릭이 표시되지 않음

---

## 문제 현상

### 증상
- `http://localhost:8082/spark-metrics-ui.html` 접속 시 메트릭 데이터가 비어있음
- Kubernetes API를 통한 메트릭 조회 시 0 bytes 반환
- Prometheus에서는 메트릭이 정상 수집됨

### 환경
```bash
# 실행 중인 Spark 애플리케이션
test-00020-fsa-123          (Driver Pod)
test-00020-fsa-123-exec-1   (Executor Pod)

# 메트릭 엔드포인트
http://localhost:4040/metrics/driver/prometheus/
http://localhost:4040/metrics/executors/prometheus/
```

---

## 원인 분석

### 1. 메트릭 직접 확인 (정상)

```bash
# kubectl exec로 직접 확인
kubectl exec test-00020-fsa-123 -- curl -s http://localhost:4040/metrics/driver/prometheus/
# 결과: 34054 bytes ✅

kubectl exec test-00020-fsa-123 -- curl -s http://localhost:4040/metrics/driver/prometheus
# 결과: 0 bytes ❌
```

**발견:** Spark 메트릭 엔드포인트는 **trailing slash `/`가 필수**

### 2. Proxy 서버 로그 분석

```log
2026/02/13 01:45:25 Proxied metrics from pod test-00020-fsa-123: /metrics/driver/prometheus/ (0 bytes)
```

→ 메트릭 요청은 전달되었으나 0 bytes 반환

### 3. 버그 코드 분석

**수정 전 코드 (`main.go:206-215`):**
```go
metricsPath := "/" + proxyParts[1]
namespace := "default"

// Use kubectl exec to get metrics from pod
// Support both /metrics/driver/prometheus/ and /metrics/executors/prometheus/ endpoints
proxyPath := metricsPath
if strings.HasSuffix(metricsPath, "/") {
    proxyPath = strings.TrimSuffix(metricsPath, "/")  // ❌ 버그!
}
cmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--", "curl", "-s", "http://localhost:4040"+proxyPath)
```

**문제점:**
- `metricsPath = "/metrics/driver/prometheus/"` (입력)
- `proxyPath = "/metrics/driver/prometheus"` (trim suffix 적용)
- 최종 URL: `http://localhost:4040/metrics/driver/prometheus` ❌

---

## 해결 방법

### 코드 수정

**파일:** `/root/hynix/cmd/proxy/main.go`

**수정 내용:**
```go
// 수정 후
metricsPath := "/" + proxyParts[1]
namespace := "default"

// Use kubectl exec to get metrics from pod
// Keep trailing slash as required by Spark metrics endpoint
cmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--", "curl", "-s", "http://localhost:4040"+metricsPath)
```

**변경 사항:**
1. 불필요한 `proxyPath` 변수 및 trim suffix 로직 제거
2. `metricsPath`를 직접 사용 (trailing slash 유지)
3. 로그에 bytes 수 추가: `log.Printf("Proxied metrics from pod %s: %s (%d bytes)", ...)`

### 비교

| 항목 | 수정 전 | 수정 후 |
|------|----------|----------|
| 처리 경로 | `/metrics/driver/prometheus` | `/metrics/driver/prometheus/` |
| 반환 바이트 | 0 bytes | 34054 bytes |
| 메트릭 표시 | ❌ 안 됨 | ✅ 정상 |

---

## 테스트 결과

### 1. API 테스트

```bash
curl -s "http://localhost:8082/api/api/v1/namespaces/default/pods/test-00020-fsa-123/proxy/metrics/driver/prometheus/"
```

**결과 (수정 전):**
```
(빈 응답)
```

**결과 (수정 후):**
```
metrics_spark_3d79bb47e7b44d21a2bd842f2442cfa8_driver_BlockManager_memory_maxMem_MB_Value{type="gauges"} 233
metrics_spark_3d79bb47e7b44d21a2bd842f2442cfa8_driver_DAGScheduler_job_activeJobs_Value{type="gauges"} 1
...
```

### 2. UI 테스트

**접속:** `http://localhost:8082/spark-metrics-ui.html`

- ✅ Spark Pods 목록 표시
- ✅ Driver/Executor 메트릭 표시
- ✅ 메트릭 실시간 갱신

---

## 메트릭 예시

### Driver 메트릭
```
metrics_spark_*_driver_BlockManager_memory_maxMem_MB_Value 233
metrics_spark_*_driver_BlockManager_memUsed_MB_Value 0
metrics_spark_*_driver_DAGScheduler_job_activeJobs_Value 1
metrics_spark_*_driver_DAGScheduler_stage_runningStages_Value 1
```

### Executor 메트릭
```
metrics_spark_*_executor_*_memoryUsed_Value
metrics_spark_*_executor_*_threadPoolActiveTasks_Value
```

---

## 배포 방법

```bash
# 1. 기존 프로세스 종료
lsof -ti:8082 | xargs kill -9

# 2. 빌드
cd /root/hynix/cmd/proxy
go build -o proxy-server main.go

# 3. 실행
nohup ./proxy-server > /tmp/proxy.log 2>&1 &

# 4. 확인
tail -f /tmp/proxy.log
```

---

## 참고 사항

### Spark 메트릭 엔드포인트 구조

| 경로 | 설명 |
|------|------|
| `/metrics/driver/prometheus/` | Driver 메트릭 (trailing `/` 필수) |
| `/metrics/executors/prometheus/` | Executor 메트릭 (trailing `/` 필수) |

### Prometheus 설정

```yaml
scrape_configs:
  - job_name: 'spark-applications'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        target_label: __metrics_path__
        regex: (.+)
```

---

## 결론

**버그:** Trailing slash 제거 로직으로 인해 Spark 메트릭 엔드포인트 호출 실패
**해결:** Spark 메트릭 요구사항에 맞춰 trailing slash 유지
**결과:** 메트릭 정상 수집 및 UI 표시 ✅

---

## 관련 파일

- `/root/hynix/cmd/proxy/main.go` - Proxy 서버 메인 코드
- `/root/hynix/docs/spark-metrics-ui.html` - 메트릭 대시보드 UI
- `/root/hynix/PROMETHEUS_METRICS_README.md` - Prometheus 설정 가이드
