# Prometheus 설정 및 사용 가이드

## 📋 목차

1. [개요](#개요)
2. [설정](#설정)
3. [사용 방법](#사용-방법)
4. [Spark 메트릭 수집](#spark-메트릭-수집)
5. [문제 해결](#문제-해결)
6. [PromQL 쿼리 예시](#promql-쿼리-예시)

---

## 개요

### Prometheus 아키텍처

```
┌─────────────────┐
│   Browser       │
│  :9090 Targets  │
└────────┬────────┘
         │
┌────────▼────────┐
│   Prometheus    │
│   (Scraping)    │
└────────┬────────┘
         │
┌────────▼────────┐
│  Spark Pods     │
│  Driver/Executor│
└─────────────────┘
```

### 현재 설정

| 컴포넌트 | 상태 | 포트 |
|----------|------|------|
| Prometheus | ✅ Running | 9090 |
| Grafana | ✅ Running | 3000 |
| Spark Operator | ✅ Running | 8080 |

---

## 설정

### ConfigMap 위치

```bash
kubectl get configmap prometheus-config -n default -o yaml
```

### 설정 파일 구조

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: default
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s

    scrape_configs:
      # Spark Operator Metrics
      - job_name: 'spark-operator'
        # ... 설정

      # Spark Application Pods
      - job_name: 'spark-applications'
        # ... 설정

      # Yunikorn Scheduler
      - job_name: 'yunikorn-scheduler'
        # ... 설정
```

### ConfigMap 업데이트 방법

```bash
# 1. 설정 파일 수정
kubectl edit configmap prometheus-config

# 2. Prometheus Pod 재시작
kubectl delete pod -l app=prometheus

# 3. 확인
kubectl get pod -l app=prometheus
```

---

## 사용 방법

### 1. Port-Forward 시작

```bash
kubectl port-forward svc/prometheus 9090:9090
```

### 2. Prometheus UI 접속

```
http://localhost:9090
```

### 3. 주요 페이지

| 페이지 | 경로 | 설명 |
|--------|------|------|
| **Targets** | /targets | 스크래핑 타겟 상태 |
| **Graph** | /graph | PromQL 쿼리 실행 |
| **Service Discovery** /config | 설정 확인 |

---

## Spark 메트릭 수집

### Spark 4.0.1 메트릭 구조

| 엔드포인트 | 위치 | 설명 |
|-----------|------|------|
| `/metrics/driver/prometheus/` | Driver Pod | Driver 전용 메트릭 |
| `/metrics/executors/prometheus/` | Driver Pod | **모든 Executor 메트릭** |
| (없음) | Executor Pod | ❌ 메트릭 노출 안 함 |

**중요:** Spark 4.0.1에서는 **Executor Pod가 메트릭을 제공하지 않습니다!** 모든 메트릭은 Driver Pod에서 제공됩니다.

### Pod Annotations

#### Driver Pod (정상)

```yaml
annotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "4040"
  prometheus.io/path: "/metrics/driver/prometheus/"
```

#### Executor Pod (Spark 4.0.1)

```yaml
annotations:
  # ❌ Prometheus scraping annotations 제거됨
  # Spark 4.0.1에서는 Executor Pod가 메트릭을 제공하지 않음
  yunikorn.apache.org/task-group-name: "spark-executor"
```

### Template 설정

**파일:** `/root/hynix/template/0002_wfbm.yaml`

```yaml
executor:
  instances: 2
  annotations:
    yunikorn.apache.org/task-group-name: "spark-executor"
    # Note: Spark 4.0.1 does NOT expose metrics on executor pods
    # All executor metrics are available on the driver pod at /metrics/executors/prometheus/
```

---

## 문제 해결

### 1. "too many colons in address" 에러

#### 증상
```
http://10.244.0.146:4040:/metrics/driver/prometheus/
```

#### 원인
Prometheus가 `__address__`와 `__metrics_path__`를 조합할 때 URL이 잘못 생성됨

#### 해결
ConfigMap의 relabel 설정 수정:

```yaml
# Set address: pod IP
- source_labels: [__meta_kubernetes_pod_ip]
  action: replace
  target_label: __address__
  replacement: '$1:8080'

# Override address with IP:PORT from annotation
- source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
  action: replace
  target_label: __address__
  regex: '([^:]+):8080;([0-9]+)'
  replacement: '$1:$2'
```

### 2. Executor Pod "connection refused"

#### 증상
```
Get "http://10.244.0.235:4040/metrics/executors/prometheus/":
dial tcp 10.244.0.235:4040: connect: connection refused
```

#### 원인
Spark 4.0.1에서 Executor Pod는 메트릭 엔드포인트를 제공하지 않음

#### 해결
Template 파일에서 executor Prometheus annotations 제거:

```yaml
executor:
  annotations:
    # ❌ 제거됨
    # prometheus.io/scrape: "true"
    # prometheus.io/port: "4040"
    # prometheus.io/path: "/metrics/executors/prometheus/"
```

### 3. Port-Forward가 계속 끊김

#### 해결: 백그라운드 실행

```bash
# 1. 백그라운드에서 실행
nohup kubectl port-forward svc/prometheus 9090:9090 > /dev/null 2>&1 &

# 2. 프로세스 확인
ps aux | grep "port-forward.*9090"

# 3. 종료할 때
pkill -f "port-forward.*9090"
```

---

## PromQL 쿼리 예시

### 기본 쿼리

```promql
# 모든 타겟 가용성
up

# Spark 애플리케이션
{job="spark-applications"}

# 드라이버 메트릭
spark_driver_memory

# DAGScheduler 메트릭
spark_driver_DAGScheduler_job_activeJobs
```

### 고급 쿼리

```promql
# Executor 메모리 사용량
sum(spark_executor_memoryUsed_bytes) by (app_name)

# Job 완료율
spark_driver_DAGScheduler_job_activeJobs / spark_driver_DAGScheduler_job_allJobs * 100

# 직접 GC 시간
rate(spark_driver_ExecutorMetrics_TotalGCTime_Value[5m])
```

---

## 타겟 상태 확인

### API를 통한 확인

```bash
# 전체 타겟
curl http://localhost:9090/api/v1/targets

# spark-applications만 필터
curl http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | select(.labels.job=="spark-applications")'
```

### UI에서 확인

```
http://localhost:9090/targets?search=spark
```

---

## Prometheus 재시작 방법

### ConfigMap 변경 후

```bash
# 1. Pod 삭제 (자동 재생성)
kubectl delete pod -l app=prometheus

# 2. 상태 확인
kubectl get pod -l app=prometheus -w

# 3. 로그 확인
kubectl logs -f prometheus-xxxxx
```

### Port-Forward 재시작

```bash
# 1. 기존 프로세스 종료
pkill -f "port-forward.*9090"

# 2. 다시 시작
kubectl port-forward svc/prometheus 9090:9090

# 3. 확인
curl http://localhost:9090/-/healthy
```

---

## 모니터링 대시보드

### 1. Spark Metrics UI

```
http://localhost:8082/spark-metrics-ui.html
```

- Application Name 입력: `test-00021`
- Load Metrics 클릭

### 2. Prometheus Graph

```
http://localhost:9090/graph
```

쿼리 입력:
```promql
spark_driver_memory
```

### 3. Grafana (선택사용)

```
http://localhost:3000
```

---

## 유용한 명령어

### Prometheus 관련

```bash
# ConfigMap 확인
kubectl get configmap prometheus-config -o yaml

# Pod 상태 확인
kubectl get pods -l app=prometheus

# Prometheus 로그 확인
kubectl logs -f prometheus-xxxxx

# Port-Forward 시작
kubectl port-forward svc/prometheus 9090:9090
```

### Spark 메트릭 확인

```bash
# Driver 메트릭 (Pod 내부)
kubectl exec <driver-pod> -- curl -s http://localhost:4040/metrics/driver/prometheus/

# Executor 메트릭 (Driver Pod에서 제공)
kubectl exec <driver-pod> -- curl -s http://localhost:4040/metrics/executors/prometheus/

# Pod annotations 확인
kubectl get pod <pod-name> -o jsonpath='{.metadata.annotations}'
```

---

## 참고 문서

- [Prometheus 공식 문서](https://prometheus.io/docs/)
- [Prometheus 운영 가이드](https://prometheus.io/docs/operating/)
- [Kubernetes SD Config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config)

---

## 변경 이력

| 날짜 | 변경 내용 |
|------|-----------|
| 2026-02-15 | Spark 4.0.1 Executor 메트릭 문제 해결 |
| 2026-02-15 | "too many colons" 오류 수정 |
| 2026-02-13 | Prometheus ConfigMap 최초 설정 |
