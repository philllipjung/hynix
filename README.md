# 서비스공통 (Service Common)

Spark Operator에서 사용되는 SparkApplication CR을 생성하는 Go 마이크로서비스입니다.

## 🎯 핵심 기능

### 1. 동적 프로비저닝 관리
- **enabled 필드**: 프로비저닝별 활성화/비활성화 제어
  - `enabled: "true"`: 리소스 계산 및 갱스케줄러 설정 적용
  - `enabled: "false"`: 템플릿 원본 그대로 제출 (리소스 계산 건너뜀)

### 2. 리소스 최적화
- **파일 크기 기반 큐 선택**: 데이터 크기에 따라 Yunikorn 큐 자동 선택
  - 파일 < threshold → `min_queue` 사용
  - 파일 ≥ threshold → `max_queue` 사용
- **Gang Scheduling**: Executor 개수 동적 설정

### 3. 자동화된 라벨링
- **SERVICE_ID_PLACEHOLDER**: 템플릿 플레이스홀더를 실제 서비스 ID로 자동 치환
- 13개 위치에 서비스 ID 라벨 자동 적용

## 📡 엔드포인트

### 1. Reference (GET) - YAML 조회 및 미리보기

SparkApplication CR을 조회하고 리소스 계산을 수행하여 YAML로 반환합니다. **Kubernetes에는 제출하지 않습니다.**

**URL:** `GET /api/v1/spark/reference`

**Query Parameters:**
- `provision_id`: 프로비저닝 ID (예: `0001-wfbm`, `0002-wfbm`)
- `service_id`: 서비스 ID (예: `test-service-001`)
- **category**: 카테고리 (예: `tttm`)

**Response:**
- **Content-Type**: `application/x-yaml`
- **Body**: 전체 SparkApplication YAML

**예시:**
```bash
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=test-service-001&category=tttm"
```

**서버 로그:**
```json
{
  "level": "info",
  "msg": "생성된 YAML (활성화 모드)",
  "endpoint": "reference",
  "provision_id": "0002-wfbm",
  "service_id": "test-service-001",
  "yaml_content": "apiVersion: sparkoperator.k8s.io/v1beta2\nkind: SparkApplication\n..."
}
```

**로그 확인 방법:**
```bash
# 최근 YAML 로그 확인
tail -50 server.log | jq 'select(.msg == "생성된 YAML (활성화 모드)")'

# YAML 내용만 추출
tail -200 server.log | jq -r 'select(.msg == "생성된 YAML (활성화 모드)") | .yaml_content'

# 실시간 모니터링
tail -f server.log | jq 'select(.msg == "생성된 YAML (활성화 모드)")'
```

### 2. Create (POST) - Kubernetes 제출

SparkApplication CR을 Kubernetes 클러스터에 생성합니다.

**URL:** `POST /api/v1/spark/create`

**Request Body:**
```json
{
  "provision_id": "0002-wfbm",
  "service_id": "test-service-001",
  "category": "tttm",
  "region": "ic"
}
```

**Response (성공 - 활성화 모드):**
```json
{
  "message": "SparkApplication CR 생성 성공",
  "provision_id": "0002-wfbm",
  "service_id": "test-service-001",
  "category": "tttm",
  "region": "ic",
  "result": {
    "name": "spark-pi-yunikorn-0002",
    "namespace": "default"
  }
}
```

**Response (성공 - 비활성화 모드):**
```json
{
  "message": "SparkApplication CR 생성 성공 (비활성화 모드)",
  "provision_id": "0001-wfbm",
  "service_id": "test-service-001",
  "category": "tttm",
  "region": "ic",
  "result": {
    "name": "spark-pi-yunikorn",
    "namespace": "default"
  }
}
```

**예시:**
```bash
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002-wfbm",
    "service_id": "test-service-001",
    "category": "tttm",
    "region": "ic"
  }'
```

## 🔄 엔드포인트 비교: /reference vs /create

| 구분 | /reference (GET) | /create (POST) |
|------|------------------|----------------|
| **목적** | YAML 미리보기 및 검증 | Kubernetes에 실제 제출 |
| **HTTP Method** | GET | POST |
| **Kubernetes 제출** | ✗ 하지 않음 | ✓ 함 |
| **리소스 실행** | ✗ 안 함 | ✓ Driver/Executor Pods 생성 |
| **응답 형식** | `application/x-yaml` | `application/json` |
| **결과** | YAML 텍스트 반환 | 제출 결과 JSON 반환 |
| **부작용** | 없음 | Pod 생성, 리소스 사용 |
| **용도** | 설정 검증, 미리보기 | 실제 작업 실행 |

### 사용 시나리오

**/reference 사용 경우:**
- 설정이 올바른지 미리 확인하고 싶을 때
- 생성될 YAML 내용을 검토하고 싶을 때
- 리소스 계산 결과를 확인하고 싶을 때
- 테스트 전 미리보기

**/create 사용 경우:**
- 실제로 Spark 작업을 실행하고 싶을 때
- Kubernetes에 제출하여 처리하고 싶을 때
- 프로덕션 환경에서 배포할 때

### 로그 차이점

**/reference 로그:**
```
Reference 요청 수신 → 리소스 계산 완료 → YAML 반환 완료 → 생성된 YAML (활성화 모드)
```
- 전체 YAML이 `yaml_content` 필드에 기록됨
- Kubernetes 관련 로그 없음

**/create 로그:**
```
Create 요청 수신 → 리소스 계산 완료 → SparkApplication 생성됨 → SparkApplication CR 생성 성공
```
- 생성된 리소스 정보가 기록됨
- Kubernetes API 호출 로그 포함

## ⚙️ 동작 방식

### 처리 흐름도

```
클라이언트 요청
    ↓
1. 프로비저닝 ID 식별
    ↓
2. 템플릿 YAML 로드 (template/{provision_id}.yaml)
    ↓
3. config.json 로드 및 설정 조회
    ↓
4. enabled 확인
    ├─ true → 활성화 모드
    │   ├─ 파일 크기 확인
    │   ├─ 큐 계산 (min 또는 max)
    │   ├─ Gang Scheduling 설정 (executor minMember)
    │   └─ 서비스 ID 치환
    │
    └─ false → 비활성화 모드
        └─ 서비스 ID 치환만 수행
    ↓
5. Kubernetes에 제출 (Create) 또는 클라이언트에게 반환 (Reference)
```

### Reference 엔드포인트 상세 동작

1. **템플릿 로드**: `template/` 폴더에서 `{provision_id}.yaml` 템플릿 로드
2. **설정 로드**: `config/config.json`에서 프로비저닝 설정 조회
3. **enabled 확인**:
   - `false`: 템플릿 원본 유지, 서비스 ID만 치환 후 반환
   - `true`: 다음 단계 진행
4. **큐 계산**: 파일 크기에 따라 큐 선택
   ```bash
   # config.json 설정
   {
     "resource_calculation": {
       "minio": "/root/hynix/kubernetes.zip",  # 확인할 파일
       "threshold": 10,                          # 10MB 기준
       "min_queue": "min",
       "max_queue": "max"
     }
   }

   # 계산 로직
   파일 크기 < 10MB  →  queue: "min"
   파일 크기 ≥ 10MB  →  queue: "max"
   ```
5. **Gang Scheduling**: `executor` 값으로 task-groups의 minMember 설정
6. **서비스 ID 치환**: `SERVICE_ID_PLACEHOLDER` → 실제 서비스 ID
7. **YAML 반환**: 최종 YAML 응답

### Create 엔드포인트 상세 동작

Reference와 동일한 템플릿 처리 후, Kubernetes API 서버로 SparkApplication CR 생성

## 📝 구조화된 로깅

zap 라이브러리를 사용하여 JSON 형식의 구조화된 로그를 출력합니다.

### 로그 형식

```json
{
  "level": "info",
  "timestamp": "2026-01-27T15:10:32.132+0900",
  "caller": "handlers/create.go:53",
  "msg": "Create 요청 수신",
  "endpoint": "create",
  "provision_id": "0001-wfbm",
  "service_id": "test-app-001",
  "category": "tttm",
  "region": "ic"
}
```

### 로그 레벨별 출력 항목

#### INFO 레벨

**요청 수신:**
```json
{
  "level": "info",
  "msg": "Create 요청 수신",
  "endpoint": "create",
  "provision_id": "0001-wfbm",
  "service_id": "test-app-001",
  "category": "tttm",
  "region": "ic"
}
```

**프로비저닝 모드:**
```json
{
  "level": "info",
  "msg": "프로비저닝 활성화 모드",
  "endpoint": "create",
  "provision_id": "0001-wfbm",
  "service_id": "test-app-001",
  "category": "tttm",
  "enabled": "true"
}
```

**리소스 계산 완료:**
```json
{
  "level": "info",
  "msg": "리소스 계산 완료",
  "endpoint": "create",
  "provision_id": "0001-wfbm",
  "service_id": "test-app-001",
  "category": "tttm",
  "file_path": "/root/hynix/kubernetes.zip",
  "file_size_mb": 0.052,
  "threshold_mb": 10,
  "selected_queue": "min"
}
```

**Gang Scheduling 구성:**
```json
{
  "level": "info",
  "msg": "Gang Scheduling 구성",
  "endpoint": "create",
  "provision_id": "0001-wfbm",
  "service_id": "test-app-001",
  "category": "tttm",
  "executor_min_member": 1,
  "cpu": "1",
  "memory": "5"
}
```

**SparkApplication CR 생성 성공:**
```json
{
  "level": "info",
  "msg": "SparkApplication CR 생성 성공",
  "endpoint": "create",
  "provision_id": "0001-wfbm",
  "service_id": "test-app-001",
  "category": "tttm",
  "region": "ic",
  "namespace": "default",
  "resource_name": "spark-pi-yunikorn",
  "duration_ms": 127
}
```

#### ERROR 레벨

```json
{
  "level": "error",
  "msg": "템플릿 로드 실패",
  "endpoint": "create",
  "provision_id": "9999-wfbm",
  "service_id": "test-app-001",
  "category": "tttm",
  "error": "template file not found"
}
```

### 로그 확인

```bash
# 전체 로그 확인
tail -f server.log

# JSON 로그만 필터링
tail -f server.log | grep "^{"

# jq로 예쁘게 출력
tail -f server.log | grep "^{" | jq '.'

# 특정 필드만 출력
tail -f server.log | grep "^{" | jq '{timestamp, level, msg, endpoint, provision_id, service_id, category}'
```

### 로그 분석 도구

구조화된 JSON 로그는 다음 도구들과 쉽게 연동할 수 있습니다:

- **OpenSearch + OpenSearch Dashboards**: 로그 수집 및 시각화 (현재 사용 중)
- **Grafana Loki**: 효율적인 로그 집계 시스템
- **Splunk**: 엔터프라이즈 로그 분석
- **jq**: 커맨드라인 JSON 파서

## 🔍 로그 수집 및 모니터링

### OpenSearch & Fluent Bit 구성

시스템 로그를 중앙 집중식으로 수집하고 분석하기 위해 OpenSearch와 Fluent Bit를 사용합니다.

### 아키텍처

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Host Logs      │────▶│  Fluent Bit     │────▶│   OpenSearch    │
│  - /var/log     │     │  (Docker)       │     │   Port: 9200    │
│  - /root/hynix  │     │                 │     │                 │
│  - minikube     │     │  1. Host Logs   │     │  - logs        │
│                 │     │  2. K8s Logs    │     │  - k8s-logs    │
├─────────────────┤     │  3. kubelet     │     │  - hynix-*     │
│  K8s Logs       │────▶│                 │     │                 │
│  - Containers   │     │  (DaemonSet)    │     │  OpenSearch     │
│  - Pods         │     │  1. Containers  │────▶│  Dashboards     │
│                 │     │  2. Systemd     │     │  Port: 5601    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

### 1. OpenSearch 및 Dashboards

#### 설치
```bash
# Docker Compose로 실행
cd /home/philip/opensearch
docker-compose up -d
```

#### 접속 정보
- **OpenSearch API**: http://192.168.201.152:9200
- **OpenSearch Dashboards**: http://192.168.201.152:5601

#### 인덱스 현황
| 인덱스 | 문서 수 | 설명 |
|--------|---------|------|
| logs | 60,995 | 호스트 + kubelet + **CRI/CNI** 로그 |
| k8s-logs | 15,327 | Kubernetes 컨테이너 로그 |
| hynix-2026.01.28 | 11,531 | 날짜 기반 호스트 로그 (Logstash_Format) |

### 2. Fluent Bit (호스트 로그 수집)

#### 설정 파일: `/home/philip/opensearch/fluent-bit.conf`
```conf
[SERVICE]
    Flush         5
    Daemon        off
    Log_Level     info

# 호스트 시스템 로그
[INPUT]
    Name              tail
    Path              /host/logs/syslog
    Tag               host.syslog
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Read_from_Head    True

# 커널 로그
[INPUT]
    Name              tail
    Path              /host/logs/kern.log
    Tag               host.kernel
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On

# Hynix 마이크로서비스 로그
[INPUT]
    Name              tail
    Path              /root/hynix/server.log
    Tag               hynix.service
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Read_from_Head    True

# minikube/kubelet 로그
[INPUT]
    Name              tail
    Path              /host/minikube/b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b-json.log
    Tag               minikube.kubelet
    Refresh_Interval  5
    Mem_Buf_Limit     50MB
    Skip_Long_Lines   On
    Read_from_Head    True

# VMware 네트워크 로그 (CNI)
[INPUT]
    Name              tail
    Path              /host/logs/vmware-network.log
    Tag               host.vmware-network
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Read_from_Head    True

# Docker daemon 로그 (CRI)
[INPUT]
    Name              systemd
    Tag               host.docker
    Systemd_Filter   _SYSTEMD_UNIT=docker.service
    Read_from_Tail    True
    Path              /host/journal

# 필터: 마이크로서비스 태그 추가
[FILTER]
    Name                modify
    Match               hynix.service
    Add                 source microservice
    Add                 service_name hynix-spark-common

# OpenSearch 출력
[OUTPUT]
    Name            opensearch
    Match           *
    Host            opensearch
    Port            9200
    Index           logs
    Suppress_Type_Name On
```

#### 시작 스크립트: `/tmp/start-fluent-bit.sh`
```bash
#!/bin/bash
docker run -d \
  --name fluent-bit \
  --network opensearch-net \
  -v /var/log:/host/logs:ro \
  -v /root/hynix:/root/hynix:ro \
  -v /var/lib/docker/containers/b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b:/host/minikube:ro \
  -v /var/log/journal:/host/journal:ro \
  -v /home/philip/opensearch/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf:ro \
  --restart unless-stopped \
  fluent/fluent-bit:2.2
```

### 3. Fluent Bit DaemonSet (K8s 로그 수집)

#### 설정 파일: `/home/philip/opensearch/fluent-bit-k8s-v2.yaml`
```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluent-bit
  namespace: kube-system
spec:
  template:
    spec:
      serviceAccountName: fluent-bit
      containers:
      - name: fluent-bit
        image: fluent/fluent-bit:2.2
        securityContext:
          privileged: true
        env:
        - name: FLUENT_OPENSEARCH_HOST
          value: "192.168.201.152"
        - name: FLUENT_OPENSEARCH_PORT
          value: "9200"
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: false
        - name: etcfluentbit-main
          mountPath: /fluent-bit/etc/
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
      - name: etcfluentbit-main
        configMap:
          name: fluent-bit-config
```

#### 배포
```bash
kubectl apply -f /home/philip/opensearch/fluent-bit-k8s-v2.yaml
```

### 4. kubelet 로그 수집

minikube 환경에서 kubelet은 systemd 서비스가 아닌 바이너리로 직접 실행되므로, Docker 컨테이너 로그에서 수집합니다.

#### kubelet이란?
- **Kubernetes Node Agent**: 각 노드에서 실행되는 K8s의 핵심 에이전트
- **역할**:
  - 파드 생성/관리
  - 컨테이너 런타임(containerd/Docker)과 통신
  - 노드 리소스 모니터링
  - API 서버와 통신

#### kubelet 로그 위치
```bash
# minikube 컨테이너 ID 확인
docker ps | grep minikube
# b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b

# kubelet 로그 파일
/var/lib/docker/containers/b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b/b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b-json.log
```

#### Fluent Bit 설정 (kubelet)
```conf
# 입력: minikube/kubelet 로그
[INPUT]
    Name              tail
    Path              /host/minikube/b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b-json.log
    Path_Key          filename
    Tag               minikube.kubelet
    Refresh_Interval  5
    Mem_Buf_Limit     50MB
    Skip_Long_Lines   On
    Read_from_Head    True
```

#### 수집된 로그 예시
```json
{
  "log": "[OK] Unmounted /var/lib/kubelet…/admission-controller-secrets."
}
```

#### kubelet 로그 분석
```bash
# Pod 마운트/언마운트 로그
curl "http://localhost:9200/logs/_search?q=kubelet*AND*Unmounted&pretty=true"

# kubelet 서비스 상태 로그
curl "http://localhost:9200/logs/_search?q=kubelet*AND*Stopping&pretty=true"

# API 접근 관련 로그
curl "http://localhost:9200/logs/_search?q=kube-api-access&pretty=true"
```

### 5. CRI (Container Runtime Interface) 로그 수집

CRI는 Kubernetes가 컨테이너 런타임(Docker, containerd 등)과 통신하는 표준 인터페이스입니다.

#### CRI 로그 소스

| 런타임 | 로그 위치 | Tag | 수집 건수 |
|---------|-----------|-----|-----------|
| **containerd** | /var/log/syslog | host.syslog | 1,077+ |
| Docker | minikube 컨테이너 로그 | minikube.kubelet | 721+ |

#### containerd 로그
containerd는 Docker의 후속 프로젝트로, 컨테이너 라이프사이클을 관리하는 CRI 런타임입니다.

**로그 예시:**
```json
{
  "log": "time=\"2026-01-27T13:24:36.531205625+09:00\" level=info msg=\"Connect containerd service\""
}
```

**Fluent Bit 설정:**
```conf
# containerd 로그는 syslog에서 자동 수집
# syslog에 containerd 프로세스 로그가 포함됨
[INPUT]
    Name              tail
    Path              /host/logs/syslog
    Tag               host.syslog
    Refresh_Interval  5
    Read_from_Head    True
```

#### CRI 로그 검색
```bash
# containerd 연결 로그
curl "http://localhost:9200/logs/_search?q=containerd*AND*Connect&pretty=true"

# CRI 관련 에러
curl "http://localhost:9200/logs/_search?q=containerd*AND*error&pretty=true"

# 컨테이너 시작/중지 로그
curl "http://localhost:9200/logs/_search?q=containerd*AND*start&pretty=true"
```

#### Docker daemon 로그 (systemd)
```conf
# 입력: Docker daemon 로그 (systemd/journald)
[INPUT]
    Name              systemd
    Tag               host.docker
    Systemd_Filter   _SYSTEMD_UNIT=docker.service
    Read_from_Tail    True
    Path              /host/journal
```

**journald 마운트:**
```bash
# /tmp/start-fluent-bit.sh에 추가
-v /var/log/journal:/host/journal:ro
```

### 6. CNI (Container Network Interface) 로그 수집

CNI는 Kubernetes 파드의 네트워킹을 담당하는 플러그인입니다. 현재 환경에서는 별도의 CNI 파드(Calico, Flannel 등)가 없고 Docker 네트워크를 사용합니다.

#### 현재 환경 CNI 현황
```bash
# CNI 파드 확인 (결과 없음)
kubectl get pods -A | grep -E "calico|flannel|weave|cilium"

# 대신 Docker 네트워크 사용
docker network ls
# bridge - 기본 브릿지 네트워크
# minikube - minikube 전용 네트워크
```

#### VMware 네트워크 로그
VMware 가상머신 환경에서의 네트워크 설정 로그입니다.

**로그 위치:**
```bash
/var/log/vmware-network.log
```

**로그 내용:**
```
2026. 01. 28. (수) 09:21:37 KST : Executing '/etc/vmware-tools/scripts/vmware/network resume-vm'
2026. 01. 28. (수) 09:21:37 KST : [rescue_nic] ens33 is already active.
2026. 01. 28. (수) 09:21:37 KST : [rescue_nic] br-e5c74f5ea9ca is already active.
2026. 01. 28. (수) 09:21:37 KST : [rescue_nic] docker0 is already active.
2026. 01. 28. (수) 09:21:37 KST : [rescue_nic] veth33801cf is already active.
```

**Fluent Bit 설정:**
```conf
# 입력: VMware 네트워크 로그 (CNI 관련)
[INPUT]
    Name              tail
    Path              /host/logs/vmware-network.log
    Path_Key          filename
    Tag               host.vmware-network
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Read_from_Head    True
```

#### CNI 로그 검색
```bash
# 네트워크 인터페이스 로그
curl "http://localhost:9200/logs/_search?q=ens33&pretty=true"

# Docker 브릿지 로그
curl "http://localhost:9200/logs/_search?q=docker0&pretty=true"

# 가상 네트워크 로그
curl "http://localhost:9200/logs/_search?q=veth&pretty=true"

# 네트워크 활성화 로그
curl "http://localhost:9200/logs/_search?q=rescue_nic&pretty=true"
```

#### K8s 네트워크 관련 컴포넌트 로그

| 컴포넌트 | 로그 위치 | 설명 |
|----------|-----------|------|
| **kube-proxy** | k8s-logs 인덱스 | 서비스 로드밸런싱, iptables 규칙 관리 |
| **coredns** | k8s-logs 인덱스 | DNS 쿼리 처리 |

```bash
# kube-proxy 로그
curl "http://localhost:9200/k8s-logs/_search?q=kubernetes.container_name:kube-proxy&pretty=true"

# coredns 로그
curl "http://localhost:9200/k8s-logs/_search?q=kubernetes.container_name:coredns&pretty=true"
```

### 7. 수집되는 로그 종류

#### 호스트 로그 (logs 인덱스)
| 로그 소스 | 경로 | Tag | 설명 | 수집 건수 |
|-----------|------|-----|------|-----------|
| 시스템 로그 | /var/log/syslog | host.syslog | 호스트 시스템 로그 | 수집됨 |
| 커널 로그 | /var/log/kern.log | host.kernel | 커널 메시지 | 수집됨 |
| 마이크로서비스 | /root/hynix/server.log | hynix.service | Go 애플리케이션 로그 | 수집됨 |
| **kubelet** | minikube 컨테이너 로그 | **minikube.kubelet** | **K8s 노드 에이전트** | 1,092+ |
| **containerd** | /var/log/syslog | host.syslog | **CRI 런타임** | 1,077+ |
| **vmware-network** | /var/log/vmware-network.log | host.vmware-network | **CNI/네트워크** | 10,000+ |

#### K8s 컨테이너 로그 (k8s-logs 인덱스)
| 컴포넌트 | 로그 수 | 설명 |
|----------|---------|------|
| kube-apiserver | 76+ | K8s API 서버 |
| kube-scheduler | 수집됨 | 스케줄러 |
| kube-controller-manager | 수집됨 | 컨트롤러 매니저 |
| kube-proxy | 수집됨 | 네트워크 프록시, 서비스 로드밸런싱 |
| coredns | 수집됨 | DNS 서버 |
| yunikorn-scheduler | 43+ | Yunikorn 스케줄러 |
| spark-operator | 수집됨 | Spark Operator |
| etcd | 수집됨 | K8s etcd |

### 8. OpenSearch 쿼리 예시

#### 인덱스 목록 확인
```bash
curl "http://localhost:9200/_cat/indices?v"
```

#### 마이크로서비스 로그 검색
```bash
curl "http://localhost:9200/logs/_search?q=service_name:hynix-spark-common&pretty=true&size=10"
```

#### kubelet 로그 검색
```bash
curl "http://localhost:9200/logs/_search?q=kubelet&pretty=true&size=10"
```

#### K8s 로그 검색 (yunikorn-scheduler)
```bash
curl "http://localhost:9200/k8s-logs/_search?q=kubernetes.container_name:yunikorn-scheduler-k8s&pretty=true&size=10"
```

#### 날짜 범위 검색
```bash
curl "http://localhost:9200/logs/_search" -H 'Content-Type: application/json' -d '{
  "query": {
    "range": {
      "@timestamp": {
        "gte": "now-1h"
      }
    }
  }
}'
```

#### CRI 로그 검색
```bash
# containerd 연결 로그
curl "http://localhost:9200/logs/_search?q=containerd&pretty=true"

# CRI 관련 에러
curl "http://localhost:9200/logs/_search?q=containerd*AND*error&pretty=true"

# Docker daemon 로그
curl "http://localhost:9200/logs/_search?q=docker.service&pretty=true"
```

#### CNI 로그 검색
```bash
# 네트워크 인터페이스 로그
curl "http://localhost:9200/logs/_search?q=ens33&pretty=true"

# Docker 브릿지 로그
curl "http://localhost:9200/logs/_search?q=docker0&pretty=true"

# 가상 이더넷 로그
curl "http://localhost:9200/logs/_search?q=veth&pretty=true"

# 네트워크 활성화 로그
curl "http://localhost:9200/logs/_search?q=rescue_nic&pretty=true"
```

#### kubelet 심층 분석
```bash
# Pod 마운트/언마운트 로그
curl "http://localhost:9200/logs/_search?q=kubelet*AND*Unmounted&pretty=true"

# kubelet 서비스 상태 로그
curl "http://localhost:9200/logs/_search?q=kubelet*AND*Stopping&pretty=true"

# API 접근 관련 로그
curl "http://localhost:9200/logs/_search?q=kube-api-access&pretty=true"
```

### 9. OpenSearch Dashboards 사용

#### 인덱스 패턴 생성
1. OpenSearch Dashboards 접속: http://192.168.201.152:5601
2. Stack Management → Index Patterns
3. 인덱스 패턴 생성:
   - `logs*` - 호스트 및 kubelet 로그
   - `k8s-logs*` - K8s 컨테이너 로그
   - `hynix-*` - 날짜 기반 마이크로서비스 로그

#### 로그 탐색
```bash
# Discover 탭에서 로그 실시간 확인
- 필터: service_name: "hynix-spark-common"
- 필터: kubernetes.container_name: "yunikorn-scheduler-k8s"
- 필터: log: "kubelet"
```

### 10. 모니터링 및 관리

#### Fluent Bit 상태 확인
```bash
# Docker 컨테이너 상태
docker ps | grep fluent-bit

# 로그 확인
docker logs fluent-bit -f

# DaemonSet 상태
kubectl get pods -n kube-system -l app=fluent-bit
```

#### OpenSearch 상태 확인
```bash
# 클러스터 상태
curl "http://localhost:9200/_cluster/health?pretty=true"

# 인덱스 통계
curl "http://localhost:9200/_cat/indices?v"

# 문서 수 확인
curl "http://localhost:9200/_count" -H 'Content-Type: application/json' -d '{
  "query": {"match_all": {}}
}'
```

### 11. 로그 분석 예시

#### 마이크로서비스 요청 추적
```bash
# 특정 provision_id의 모든 로그
curl "http://localhost:9200/logs/_search" -H 'Content-Type: application/json' -d '{
  "query": {
    "match": {
      "provision_id": "0002-wfbm"
    }
  }
}'
```

#### 에러 로그만 추출
```bash
curl "http://localhost:9200/logs/_search?q=level:error&pretty=true"
```

#### Spark Pod 로그 확인
```bash
# Spark Operator 로그
curl "http://localhost:9200/k8s-logs/_search?q=spark-operator&pretty=true"

# Yunikorn Scheduler 로그
curl "http://localhost:9200/k8s-logs/_search?q=yunikorn&pretty=true"
```

---

## 📊 Prometheus 메트릭

### 메트릭 엔드포인트

**URL:** `GET /metrics`

모든 메트릭은 Prometheus 형식으로 노출됩니다.

**예시:**
```bash
curl http://localhost:8080/metrics
```

### 사용 가능한 메트릭

#### 1. 요청 관련 메트릭

```promql
# 총 요청 수 (프로비저닝 ID, 엔드포인트, 상태별)
spark_service_requests_total{provision_id="0001-wfbm", endpoint="create", status="success"}

# 요청 처리 시간 (프로비저닝 ID, 엔드포인트별)
spark_service_request_duration_seconds{provision_id="0001-wfbm", endpoint="create"}
```

**목적:**
- 각 프로비저닝별 사용량 추적
- 엔드포인트별 부하 분석
- 응답 시간 모니터링 (p50, p95, p99)

---

#### 2. 리소스 계산 관련 메트릭

```promql
# 큐 선택 수 (min/max)
spark_service_queue_selection_total{provision_id="0001-wfbm", queue="min"}
```

**목적:**
- 데이터 크기 변화 추적
- 큐 선택 패턴 분석
- capacity planning

---

#### 3. 프로비저닝 모드 관련 메트릭

```promql
# enabled 모드 사용 현황
spark_service_provision_mode_total{provision_id="0001-wfbm", enabled="true"}

# 리소스 계산 스킵 횟수
spark_service_resource_calculation_skipped_total{provision_id="0001-wfbm", reason="disabled"}
```

**목적:**
- 활성화/비활성화 모드 사용 패턴
- 최적화 기능 활용도 측정

---

#### 4. Kubernetes 생성 관련 메트릭

```promql
# SparkApplication 생성 성공/실패
spark_service_k8s_creation_total{provision_id="0001-wfbm", namespace="default", status="success"}

# 기존 리소스 삭제 횟수
spark_service_k8s_deletion_total{provision_id="0001-wfbm", namespace="default"}
```

**목적:**
- Kubernetes 생성 성공률 모니터링
- 리소스 충돌 추적

---

#### 5. Gang Scheduling 관련 메트릭

```promql
# Executor minMember 설정값
spark_service_executor_min_member{provision_id="0001-wfbm"}

# CPU/Memory 설정
spark_service_gang_scheduling_resources{provision_id="0001-wfbm", resource_type="cpu"}    # 1
spark_service_gang_scheduling_resources{provision_id="0001-wfbm", resource_type="memory"} # 5
```

**목적:**
- Gang Scheduling 설정 추적
- 리소스 할당 패턴 분석

---

### Grafana 대시보드 예시

**PromQL 쿼리 예시:**

```promql
# 총 요청 수 (성공/실패)
sum(rate(spark_service_requests_total[5m])) by (provision_id, status)

# 요청 처리 시간 (p95)
histogram_quantile(0.95,
  sum(rate(spark_service_request_duration_seconds_bucket[5m])) by (provision_id, endpoint, le)
)

# 큐 선택 비율
sum(spark_service_queue_selection_total) by (queue) /
  sum(spark_service_queue_selection_total)

# 프로비저닝 모드 사용 현황
sum(spark_service_provision_mode_total) by (provision_id, enabled)

# Kubernetes 생성 성공률
sum(spark_service_k8s_creation_total{status="success"}) by (provision_id) /
  sum(spark_service_k8s_creation_total) by (provision_id)

# Gang Scheduling 리소스 사용량
spark_service_gang_scheduling_resources
```

---

### 메트릭 수집 예시

```bash
# 1. Create 요청 전송
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0001-wfbm",
    "service_id": "test-app-001",
    "category": "tttm",
    "region": "ic"
  }'

# 2. 메트릭 확인
curl http://localhost:8080/metrics | grep spark_service

# 출력 예시:
# spark_service_requests_total{endpoint="create",provision_id="0001-wfbm",status="success"} 1
# spark_service_request_duration_seconds_sum{endpoint="create",provision_id="0001-wfbm"} 0.311576768
# spark_service_queue_selection_total{provision_id="0001-wfbm",queue="min"} 1
# spark_service_provision_mode_total{enabled="true",provision_id="0001-wfbm"} 1
# spark_service_k8s_creation_total{namespace="default",provision_id="0001-wfbm",status="success"} 1
# spark_service_executor_min_member{provision_id="0001-wfbm"} 1
# spark_service_gang_scheduling_resources{provision_id="0001-wfbm",resource_type="cpu"} 1
# spark_service_gang_scheduling_resources{provision_id="0001-wfbm",resource_type="memory"} 5
```

---

### Prometheus 구성 예시

**prometheus.yml:**
```yaml
scrape_configs:
  - job_name: 'spark-service-common'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080']
```

---

## 🔧 리소스 계산 로직

### 현재 설정 (config.json)

```json
{
  "config_specs": [
    {
      "provision_id": "0001-wfbm",
      "enabled": "true",
      "resource_calculation": {
        "minio": "/root/hynix/kubernetes.zip",
        "threshold": 10,
        "min_queue": "min",
        "max_queue": "max"
      },
      "gang_scheduling": {
        "cpu": "1",
        "memory": "5",
        "executor": "1"
      }
    },
    {
      "provision_id": "0002-wfbm",
      "enabled": "true",
      "resource_calculation": {
        "minio": "/root/hynix/kubernetes.zip",
        "threshold": 10,
        "min_queue": "min",
        "max_queue": "max"
      },
      "gang_scheduling": {
        "cpu": "5",
        "memory": "10",
        "executor": "1"
      }
    }
  ]
}
```

### 큐 (Queue) 계산 예시

```
현재 파일 크기: 52KB (0.052MB)
Threshold: 10MB

결과: 0.052MB < 10MB → queue = "min" ✓
```

**만약 파일이 10MB 이상이면:**
```
파일 크기: 15MB
Threshold: 10MB

결과: 15MB ≥ 10MB → queue = "max" ✓
```

## 🏷️ 서비스 ID 라벨 적용

서비스 ID는 템플릿의 `SERVICE_ID_PLACEHOLDER`를 교체하여 다음 위치에 자동 적용됩니다:

### Kubernetes Labels (6개 위치)
- `metadata.labels[yunikorn.apache.org/app-id]`
- `metadata.labels[build-number]`
- `spec.driver.labels[yunikorn.apache.org/app-id]`
- `spec.driver.labels[build-number]`
- `spec.executor.labels[yunikorn.apache.org/app-id]`
- `spec.executor.labels[build-number]`

### Spark Configuration (3개 위치)
- `spec.sparkConf[spark.app.name]`
- `spec.sparkConf[spark.kubernetes.executor.podNamePrefix]`
- `spec.sparkConf[spark.kubernetes.driver.pod.name]`

### 기타 (1개 위치)
- `spec.driver.podName`

**총 10개 위치**에 서비스 ID가 자동 적용됩니다!

**참고**: `spark-app-name` 라벨은 Spark Operator 예약 라벨이므로 적용하지 않습니다.

## 🐳 컨테이너 설정 (template 필드)

`template` 필드를 사용하여 Spark Driver와 Executor 컨테이너의 이름과 이미지를 명시적으로 지정할 수 있습니다. Spark Operator v2.4.0에서 컨테이너 설정을 제어하는 권장 방법입니다.

### template 필드란?

Spark Operator는 기본적으로 Spark 컨테이너를 자동 생성하지만, `template` 필드를 사용하여 다음을 제어할 수 있습니다:
- **컨테이너 이름**: 서비스 ID를 컨테이너 이름으로 사용 가능
- **컨테이너 이미지**: 기본 이미지를 오버라이드
- **리소스 설정**: 메모리, CPU 요청/제한

### 템플릿 구조

```yaml
spec:
  driver:
    cores: 1
    memory: 512m
    podName: "SERVICE_ID_PLACEHOLDER"
    template:
      spec:
        containers:
          - name: SERVICE_ID_PLACEHOLDER  # 서비스 ID로 치환됨
            image: docker.io/library/spark:4.0.1
            resources:
              limits:
                memory: 512m
                cpu: "1"
              requests:
                memory: 512m
                cpu: "500m"

  executor:
    instances: 1
    cores: 1
    memory: 512m
    template:
      spec:
        containers:
          - name: SERVICE_ID_PLACEHOLDER  # 서비스 ID로 치환됨
            image: docker.io/library/spark:4.0.1
            resources:
              limits:
                memory: 512m
                cpu: "1"
              requests:
                memory: 512m
                cpu: "500m"
```

### 서비스 ID를 컨테이너 이름으로 사용

`SERVICE_ID_PLACEHOLDER`를 컨테이너 이름으로 지정하면, Go 마이크로서비스가 실제 서비스 ID로 자동 치환합니다.

**치환 예시:**
```
SERVICE_ID_PLACEHOLDER → final-test
```

**실제 생성되는 컨테이너:**
- Driver Pod 컨테이너: `final-test`
- Executor Pod 컨테이너: `final-test`

### Driver 컨테이너 설정

| 필드 | 값 | 설명 |
|------|-----|------|
| `spec.driver.template.spec.containers[0].name` | `SERVICE_ID_PLACEHOLDER` | Driver 컨테이너 이름 (서비스 ID로 치환) |
| `spec.driver.template.spec.containers[0].image` | `docker.io/library/spark:4.0.1` | Driver 컨테이너 이미지 |
| `spec.driver.template.spec.containers[0].resources` | memory: 512m, cpu: "1" | 리소스 요청/제한 |

### Executor 컨테이너 설정

| 필드 | 값 | 설명 |
|------|-----|------|
| `spec.executor.template.spec.containers[0].name` | `SERVICE_ID_PLACEHOLDER` | Executor 컨테이너 이름 (서비스 ID로 치환) |
| `spec.executor.template.spec.containers[0].image` | `docker.io/library/spark:4.0.1` | Executor 컨테이너 이미지 |
| `spec.executor.template.spec.containers[0].resources` | memory: 512m, cpu: "1" | 리소스 요청/제한 |

### 컨테이너 이름 확인 명령어

#### 특정 파드의 컨테이너 이름 확인
```bash
# Driver Pod 컨테이너 이름 확인
su - philip -c "minikube kubectl -- get pod final-test -o jsonpath='{.spec.containers[*].name}'"

# 출력: final-test pause
```

#### 모든 파드의 컨테이너 이름 목록
```bash
# 모든 파드와 컨테이너 이름 확인
su - philip -c "minikube kubectl -- get pods -o jsonpath='{range .items[*]}{.metadata.name}{\" \"}{.spec.containers[*].name}{\"\\n\"}{end}'"

# 출력 예시:
# final-test final-test pause
# tg-final-test-spark-executor-xxx final-test pause
```

#### 특정 서비스 ID 관련 파드 확인
```bash
# 서비스 ID로 필터링
su - philip -c "minikube kubectl -- get pods -o jsonpath='{range .items[?(@.metadata.name==\"final-test\")]}{.metadata.name}: {.spec.containers[*].name}{\"\\n\"}{end}'"
```

#### describe로 상세 정보 확인
```bash
# 파드 상세 정보 (컨테이너 이름 포함)
su - philip -c "minikube kubectl -- describe pod final-test"
```

### 사용 시나리오

**1. 커스텀 컨테이너 이름 (서비스 ID 사용)**
```yaml
driver:
  template:
    spec:
      containers:
        - name: SERVICE_ID_PLACEHOLDER  # → "my-spark-app"으로 치환
          image: docker.io/library/spark:4.0.1
```

**2. 커스텀 이미지 사용**
```yaml
driver:
  template:
    spec:
      containers:
        - name: SERVICE_ID_PLACEHOLDER
          image: my-registry/spark:4.0.1-custom  # 커스텀 이미지
```

**3. 리소스 오버라이드**
```yaml
driver:
  template:
    spec:
      containers:
        - name: SERVICE_ID_PLACEHOLDER
          image: docker.io/library/spark:4.0.1
          resources:
            limits:
              memory: 1Gi        # 증가
              cpu: "2"            # CPU 추가
            requests:
              memory: 512Mi
              cpu: "500m"
```

**4. 환경 변수 추가**
```yaml
driver:
  template:
    spec:
      containers:
        - name: SERVICE_ID_PLACEHOLDER
          image: docker.io/library/spark:4.0.1
          env:
            - name: MY_VAR
              value: "my-value"
```

### 참고사항

- **Spark Operator v2.4.0**: `coreSpec` 대신 `template` 필드를 사용해야 함
- **SERVICE_ID_PLACEHOLDER 치환**: Go 마이크로서비스가 템플릿의 모든 `SERVICE_ID_PLACEHOLDER`를 실제 서비스 ID로 치환
- **기본값**: `template`를 지정하지 않으면 Spark Operator가 기본값 사용 (`spark-kubernetes-driver`, `spark-kubernetes-executor`)
- **첫 번째 컨테이너**: `containers[0]`는 Spark 메인 컨테이너여야만 함
- **pause 컨테이너**: Kubernetes 인프라 컨테이너로 자동 추가됨

## 🎯 Pod 이름 규칙

### Driver Pod
```
[서비스 ID]
```
- `spec.driver.podName` 설정에 의해 결정됩니다
- Spark Application CR 이름과 동일하게 설정됩니다
- 예: `my-spark-app`

### Executor Pod

Yunikorn Gang Scheduling 사용 시 두 가지 형태의 Pod가 생성됩니다:

**1. Yunikorn Placeholder Pod (tg 접두사)**
```
tg-[서비스 ID]-spark-executor-[고유ID]
```
- Yunikorn Scheduler가 Gang Scheduling을 위해 생성하는 placeholder
- `tg` = **T**ask **G**roup (Yunikorn Task Group)
- 예: `tg-my-spark-app-spark-executor-abc123`

**2. 실제 Spark Executor Pod**
```
[서비스 ID]-exec-[숫자]
```
- 실제 Spark 작업을 실행하는 Executor
- Spark Operator가 생성
- 예: `my-spark-app-exec-1`, `my-spark-app-exec-2`

### Pod 이름 설정 흐름

```
1. 템플릿 설정:
   spec.driver.podName: SERVICE_ID_PLACEHOLDER
   sparkConf:
     spark.kubernetes.executor.podNamePrefix: SERVICE_ID_PLACEHOLDER

2. 서비스 ID 치환:
   spec.driver.podName: my-spark-app
   sparkConf:
     spark.kubernetes.executor.podNamePrefix: my-spark-app

3. 실제 생성되는 Pod:
   Driver:   my-spark-app
   Executor: tg-my-spark-app-spark-executor-xyz (Yunikorn placeholder)
   Executor: my-spark-app-exec-1 (실제 Spark executor)
```

### 중요한 점

- **Driver Pod**: 1개만 존재, 이름이 그대로 사용됨
- **Executor Pod**: N개 존재, Yunikorn placeholder + 실제 executor 두 가지 형태로 생성됨
- **tg 접두사**: Yunikorn Gang Scheduling이 활성화된 경우에만 붙습니다
- **서비스 ID**: 모든 Pod 이름에 포함되어 애플리케이션 식별이 가능합니다

## 📁 프로젝트 구조

```
/root/hynix/
├── config/
│   └── config.json          # 프로비저닝 설정 (enabled, 리소스 계산, Gang Scheduling)
├── template/
│   ├── 0001_wfbm.yaml       # SparkApplication 템플릿 (0001-wfbm)
│   └── 0002_wfbm.yaml       # SparkApplication 템플릿 (0002-wfbm)
├── handlers/
│   ├── reference.go         # Reference 엔드포인트 핸들러
│   ├── create.go            # Create 엔드포인트 핸들러
│   └── types.go             # 요청/응답 타입 정의
├── services/
│   ├── config.go            # 설정 로드, enabled 체크, 리소스 계산
│   ├── template.go          # 템플릿 로드, 서비스 ID 치환, executor minMember 업데이트
│   ├── k8s.go               # Kubernetes API 클라이언트, CR 생성/삭제
│   └── utils.go             # 파일 읽기 유틸리티
├── metrics/
│   └── metrics.go           # Prometheus 메트릭 정의
├── middleware/
│   └── logging.go           # 로깅 미들웨어 (요청 로그, Correlation ID)
├── main.go                  # 애플리케이션 진입점
├── go.mod                   # Go 모듈 정의
├── go.sum                   # 의존성 잠금 파일
├── Dockerfile               # Docker 이미지 빌드
└── README.md                # 이 파일
```

## 🚀 실행 방법

### 로컬 개발

```bash
# 의존성 다운로드
go mod download

# 빌드
go build -o service-common .

# 서버 실행 (kubeconfig 필요)
KUBECONFIG=/home/philip/.kube/config ./service-common
```

### Docker 빌드 및 실행

```bash
# Docker 이미지 빌드
docker build -t service-common:latest .

# 컨테이너 실행
docker run -p 8080:8080 \
  -v /root/hynix/config:/app/config \
  -v /root/hynix/template:/app/template \
  -v /home/philip/.kube/config:/root/.kube/config \
  service-common:latest
```

## 🔑 환경 변수

- `KUBECONFIG`: Kubernetes kubeconfig 파일 경로 (필수)
  - 예: `KUBECONFIG=/home/philip/.kube/config`

## 📦 의존성

- [Gin](https://github.com/gin-gonic/gin) v1.10.0 - HTTP 웹 프레임워크
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) v0.17.2 - Kubernetes 클라이언트
- [k8s.io/apimachinery](https://github.com/kubernetes/apimachinery) v0.29.0 - Kubernetes API 타입
- [prometheus/client_golang](https://github.com/prometheus/client_golang) v1.23.2 - Prometheus 메트릭 수집
- [zap](https://github.com/uber-go/zap) v1.27.1 - 구조화된 로깅

## 🧪 테스트 예시

### 1. Reference 엔드포인트 테스트

#### 기본 테스트

```bash
# 0001-wfbm 프로비저닝 테스트
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0001-wfbm&service_id=test-app-001&category=wfbm"

# 0002-wfbm 프로비저닝 테스트
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=test-app-002&category=wfbm"
```

#### YAML 파일로 저장

```bash
# YAML을 파일로 저장하여 검토
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=yaml-test&category=wfbm" > /tmp/test-spark.yaml

# 저장된 YAML 확인
cat /tmp/test-spark.yaml

# Kubernetes에 직접 제출 테스트 (선택사항)
su - philip -c "minikube kubectl -- apply -f /tmp/test-spark.yaml"
```

#### YAML 내용 검증

```bash
# 서비스 ID 치환 확인
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=my-app&category=wfbm" | grep "my-app"

# 컨테이너 이름 확인
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=container-test&category=wfbm" | grep -A 5 "template:"

# Yunikorn 설정 확인
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=queue-test&category=wfbm" | grep -A 10 "batchScheduler:"
```

#### 서버 로그 확인

```bash
# Reference 요청 로그 확인
tail -50 /root/hynix/server.log | grep "Reference 요청 수신"

# 생성된 YAML 로그 확인
tail -100 /root/hynix/server.log | jq 'select(.msg == "생성된 YAML (활성화 모드)")'

# 실시간 로그 모니터링
tail -f /root/hynix/server.log | jq 'select(.endpoint == "reference")'
```

### 2. Create 엔드포인트 테스트

#### 기본 테스트

```bash
# 0001-wfbm 프로비저닝으로 Spark Application 생성
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0001-wfbm",
    "service_id": "spark-test-001",
    "category": "wfbm",
    "region": "default"
  }'

# 0002-wfbm 프로비저닝으로 Spark Application 생성
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002-wfbm",
    "service_id": "spark-test-002",
    "category": "wfbm",
    "region": "default"
  }'
```

#### JSON 응답 예쁘게 출력

```bash
# jq로 JSON 응답 포맷팅
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002-wfbm",
    "service_id": "pretty-test",
    "category": "wfbm",
    "region": "default"
  }' | jq '.'
```

#### 에러 처리 테스트

```bash
# 잘못된 프로비저닝 ID (404 에러)
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "9999-wfbm",
    "service_id": "error-test",
    "category": "wfbm",
    "region": "default"
  }'

# 필수 파라미터 누락 (400 에러)
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002-wfbm",
    "service_id": "missing-region"
  }'
```

#### 서버 로그 확인

```bash
# Create 요청 로그 확인
tail -50 /root/hynix/server.log | grep "Create 요청 수신"

# 리소스 계산 로그 확인
tail -50 /root/hynix/server.log | grep "리소스 계산 완료"

# Kubernetes 생성 로그 확인
tail -50 /root/hynix/server.log | grep "SparkApplication CR 생성 성공"

# 실시간 로그 모니터링
tail -f /root/hynix/server.log | jq 'select(.endpoint == "create")'
```

### 3. 스크립트로 자동화된 테스트

#### 테스트 스크립트 작성

```bash
#!/bin/bash
# /root/hynix/test-endpoints.sh

echo "=== 1. Reference 엔드포인트 테스트 ==="
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=script-test&category=wfbm" > /tmp/reference-test.yaml
echo "YAML 저장됨: /tmp/reference-test.yaml"
echo ""

echo "=== 2. Create 엔드포인트 테스트 ==="
RESPONSE=$(curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002-wfbm",
    "service_id": "script-create-test",
    "category": "wfbm",
    "region": "default"
  }')
echo "$RESPONSE" | jq '.'
echo ""

echo "=== 3. Kubernetes 리소스 확인 ==="
su - philip -c "minikube kubectl -- get sparkapplication script-create-test"
su - philip -c "minikube kubectl -- get pods | grep script-create-test"
echo ""

echo "=== 테스트 완료 ==="
```

#### 테스트 실행

```bash
chmod +x /root/hynix/test-endpoints.sh
/root/hynix/test-endpoints.sh
```

### 4. 통합 테스트 시나리오

#### 시나리오 1: YAML 미리보기 후 제출

```bash
# Step 1: Reference로 YAML 미리보기
curl "http://localhost:8080/api/v1/spark/reference?provision_id=0002-wfbm&service_id=integration-test&category=wfbm" > /tmp/integration-test.yaml

# Step 2: YAML 검토
cat /tmp/integration-test.yaml | grep -A 5 "template:"
cat /tmp/integration-test.yaml | grep "container-test"

# Step 3: 만족스러우면 Create로 제출
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002-wfbm",
    "service_id": "integration-test",
    "category": "wfbm",
    "region": "default"
  }'

# Step 4: Kubernetes에서 확인
su - philip -c "minikube kubectl -- get sparkapplication integration-test"
su - philip -c "minikube kubectl -- get pods -l spark-app-name=integration-test"
```

#### 시나리오 2: 컨테이너 이름 확인 테스트

```bash
# Step 1: Spark Application 생성
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002-wfbm",
    "service_id": "container-name-test",
    "category": "wfbm",
    "region": "default"
  }'

# Step 2: Pod가 생성될 때까지 대기
sleep 5

# Step 3: Driver Pod 컨테이너 이름 확인
su - philip -c "minikube kubectl -- get pod container-name-test -o jsonpath='{.spec.containers[*].name}'"
echo ""
echo "예상 출력: container-name-test pause"

# Step 4: Executor Pod 컨테이너 이름 확인
su - philip -c "minikube kubectl -- get pods -o jsonpath='{range .items[?(@.metadata.name==\"container-name-test\")]}{.metadata.name}: {.spec.containers[*].name}{\"\\n\"}{end}'"
```

#### 시나리오 3: 여러 프로비저닝 비교 테스트

```bash
# Step 1: 0001-wfbm으로 생성
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{"provision_id":"0001-wfbm","service_id":"compare-0001","category":"wfbm","region":"default"}'

# Step 2: 0002-wfbm으로 생성
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{"provision_id":"0002-wfbm","service_id":"compare-0002","category":"wfbm","region":"default"}'

# Step 3: 두 SparkApplication 비교
su - philip -c "minikube kubectl -- get sparkapplication compare-0001,compare-0002"

# Step 4: Gang Scheduling 설정 비교
su - philip -c "minikube kubectl -- get sparkapplication compare-0001 -o jsonpath='{.spec.driver.annotations.yunikorn\.apache\.org/task-groups}'" | jq .
su - philip -c "minikube kubectl -- get sparkapplication compare-0002 -o jsonpath='{.spec.driver.annotations.yunikorn\.apache\.org/task-groups}'" | jq .

# Step 5: 정리
su - philip -c "minikube kubectl -- delete sparkapplication compare-0001 compare-0002"
```

### 5. Kubernetes에서 생성된 리소스 확인

#### SparkApplication 목록 확인

```bash
# 모든 SparkApplication 목록
su - philip -c "minikube kubectl -- get sparkapplications"

# 네임스페이스 지정
su - philip -c "minikube kubectl -- get sparkapplications -n default"

# 상세 정보 포함
su - philip -c "minikube kubectl -- get sparkapplications -o wide"

# 라벨 포함
su - philip -c "minikube kubectl -- get sparkapplications --show-labels"
```

#### 특정 SparkApplication 상세 정보

```bash
# describe로 상세 정보 확인
su - philip -c "minikube kubectl -- describe sparkapplication final-test"

# YAML 전체 확인
su - philip -c "minikube kubectl -- get sparkapplication final-test -o yaml"

# 특정 필드만 추출
su - philip -c "minikube kubectl -- get sparkapplication final-test -o jsonpath='{.spec.driver.template.spec.containers[*].name}'"

# Gang Scheduling 설정 확인
su - philip -c "minikube kubectl -- get sparkapplication final-test -o jsonpath='{.spec.driver.annotations.yunikorn\.apache\.org/task-groups}'" | jq .
```

#### Pod 목록 및 상태 확인

```bash
# 모든 Pod 목록
su - philip -c "minikube kubectl -- get pods"

# 서비스 ID로 필터링
su - philip -c "minikube kubectl -- get pods | grep final-test"

# 라벨로 필터링 (Driver Pod)
su - philip -c "minikube kubectl -- get pods -l spark-role=driver"

# 라벨로 필터링 (Executor Pod)
su - philip -c "minikube kubectl -- get pods -l spark-role=executor"

# 넓은 출력 형식
su - philip -c "minikube kubectl -- get pods -o wide"

# 모든 라벨 표시
su - philip -c "minikube kubectl -- get pods --show-labels"
```

#### Pod 상세 정보 및 로그

```bash
# Pod 상세 정보
su - philip -c "minikube kubectl -- describe pod final-test"

# Driver Pod 로그 확인
su - philip -c "minikube kubectl -- logs final-test"

# Executor Pod 로그 확인
su - philip -c "minikube kubectl -- logs final-test-exec-1"

# 로그 실시간 추적
su - philip -c "minikube kubectl -- logs -f final-test"

# 컨테이너 지정 로그 확인 (여러 컨테이너가 있는 경우)
su - philip -c "minikube kubectl -- logs final-test -c final-test"
```

#### 컨테이너 이름 확인

```bash
# Driver Pod의 컨테이너 이름
su - philip -c "minikube kubectl -- get pod final-test -o jsonpath='{.spec.containers[*].name}' && echo"

# Executor Pod의 컨테이너 이름
su - philip -c "minikube kubectl -- get pod final-test-exec-1 -o jsonpath='{.spec.containers[*].name}' && echo"

# 모든 파드의 컨테이너 이름 목록
su - philip -c "minikube kubectl -- get pods -o jsonpath='{range .items[*]}{.metadata.name}{\" \":}{.spec.containers[*].name}{\"\\n\"}{end}'"

# 테이블 형식으로 정리
su - philip -c "minikube kubectl -- get pods -o custom-columns=NAME:.metadata.name,CONTAINERS:.spec.containers[*].name"
```

#### SparkApplication 상태 모니터링

```bash
# 실시간 상태 모니터링
watch -n 2 'su - philip -c "minikube kubectl -- get sparkapplications"'

# Pod 실시간 모니터링
watch -n 2 'su - philip -c "minikube kubectl -- get pods"'

# SparkApplication 이벤트 확인
su - philip -c "minikube kubectl -- get events --field-selector involvedObject.kind=SparkApplication"

# 특정 애플리케이션의 이벤트
su - philip -c "minikube kubectl -- describe sparkapplication final-test | grep -A 20 Events:"
```

#### 템플릿 설정 확인

```bash
# template 필드 확인
su - philip -c "minikube kubectl -- get sparkapplication final-test -o yaml | grep -A 10 'template:'"

# 컨테이너 이름이 서비스 ID로 치환되었는지 확인
su - philip -c "minikube kubectl -- get sparkapplication final-test -o yaml | grep 'name: final-test'"

# Pod의 컨테이너 이름이 서비스 ID인지 확인
su - philip -c "minikube kubectl -- get pod final-test -o jsonpath='{.spec.containers[0].name}' && echo"
# 예상 출력: final-test
```

#### 리소스 정리

```bash
# SparkApplication 삭제 (Pod도 자동 삭제됨)
su - philip -c "minikube kubectl -- delete sparkapplication final-test"

# 여러 SparkApplication 한번에 삭제
su - philip -c "minikube kubectl -- delete sparkapplication test-001 test-002 test-003"

# 라벨로 필터링하여 삭제
su - philip -c "minikube kubectl -- delete sparkapplication -l app=test"

# 강제 삭제
su - philip -c "minikube kubectl -- delete sparkapplication final-test --force --grace-period=0"
```

### 4. Pod 상태 확인
```bash
# Driver Pod 확인 (spark-role=driver)
kubectl get pods -n default -l spark-role=driver

# 로그 확인
kubectl logs <pod-name> -n default
```

## 📊 현재 시스템 정보

### 파일 크기
```
/root/hynix/kubernetes.zip: 52KB (0.052MB)
```

### 리소스 계산 결과
```
현재: 0.052MB < 10MB (threshold)
→ queue = "min" 선택됨
```

### 프로비저닝 설정
| 프로비저닝 ID | Enabled | Executor | CPU  | Memory |
|--------------|---------|----------|------|--------|
| 0001-wfbm    | true    | 1        | 1    | 5      |
| 0002-wfbm    | true    | 1        | 5    | 10     |

## 📌 참고사항

### 필수 구성 요건
1. **Spark Operator**: 클러스터에 Spark Operator가 사전 설치되어 있어야 합니다
2. **Yunikorn Scheduler**: Gang Scheduling을 위해 Yunikorn이 구성되어 있어야 합니다
3. **ServiceAccount**: `spark-operator-spark` ServiceAccount와 RBAC 권한 필요
4. **이미지**: 템플릿에 지정된 Spark 이미지 (`docker.io/library/spark:4.0.1`)가 접근 가능해야 합니다

### 템플릿 관리
- 템플릿 파일명: `{provision_id}.yaml` → `{provision_id}.yaml` (예: `0001-wfbm.yaml`)
- 템플릿에는 `SERVICE_ID_PLACEHOLDER` 플레이스홀더를 포함해야 함
- 템플릿의 `spec.batchSchedulerOptions.queue`는 `root.default`로 설정

### enabled 모드 비교

| 특징 | enabled: true (활성화) | enabled: false (비활성화) |
|------|---------------------|----------------------|
| 리소스 계산 | ✅ 수행 | ❌ 건너뜀 |
| 큐 계산 | ✅ 파일 크기에 따라 min/max 선택 | ❌ 템플릿 그대로 |
| 갱스케줄러 | ✅ executor minMember 설정 | ❌ 템릿릿 그대로 |
| 서비스 ID 치환 | ✅ 수행 | ✅ 수행 |
| 용도 | 최적화된 리소스 사용 | 원본 템플릿 그대로 사용 |

## 🔍 헬스체크

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy"
}
```

## 📊 로그 확인

### 서버 로그
```bash
tail -f server.log
```

### 로그 예시
```
[Create] 요청 수신 - provision_id: 0001-wfbm, service_id: test-app, category: tttm, region: ic
[Create] 프로비저닝 활성화됨 - 리소스 계산 및 갱스케줄러 설정 적용
[Create] Kubernetes 클라이언트 초기화 완료
[Create] SparkApplication 생성됨: default/spark-pi-yunikorn
[Create] SparkApplication CR 생성 성공: spark-pi-yunikorn
```

## 🐛 트러블슈팅

### 1. Pod가 Pending 상태인 경우
```bash
# Pod 이벤트 확인
su - philip -c "minikube kubectl -- describe pod <pod-name>"

# Yunikorn 큐 리소스 확인
su - philip -c "minikube kubectl -- get queues -n yunikorn"

# Pod 상세 이벤트 확인
su - philip -c "minikube kubectl -- get events --field-selector involvedObject.kind=Pod"
```

### 2. SparkApplication 실패 시
```bash
# 상세 정보 확인
su - philip -c "minikube kubectl -- describe sparkapplication <app-name>"

# Driver Pod 로그 확인
su - philip -c "minikube kubectl -- logs <driver-pod-name>"

# SparkApplication 이벤트 확인
su - philip -c "minikube kubectl -- get events --field-selector involvedObject.kind=SparkApplication"
```

### 3. 에러: "프로비저닝 ID를 찾을 수 없음"
- `template/` 폴더에 해당 `{provision_id}.yaml` 파일이 있는지 확인
- 파일명이 올바른지 확인 (예: `0001_wfbm.yaml`)
```bash
# 템플릿 파일 목록 확인
ls -la /root/hynix/template/
```

### 4. 에러: "템플릿 로드 실패"
- 템플릿 파일 경로를 확인
- 파일 내용이 올바른 YAML 형식인지 확인
```bash
# YAML 문법 검증
python3 -c "import yaml; yaml.safe_load(open('/root/hynix/template/0002_wfbm.yaml'))"

# 또는 yamllint 사용
yamllint /root/hynix/template/0002_wfbm.yaml
```

### 5. 컨테이너 이름이 서비스 ID로 치환되지 않는 경우
```bash
# template 필드가 제대로 설정되었는지 확인
su - philip -c "minikube kubectl -- get sparkapplication <app-name> -o yaml | grep -A 10 'template:'"

# 컨테이너 이름 확인
su - philip -c "minikube kubectl -- get pod <app-name> -o jsonpath='{.spec.containers[*].name}'"
```

### 6. Yunikorn Gang Scheduling 관련 이슈
```bash
# Yunikorn Scheduler 로그 확인
su - philip -c "minikube kubectl -- logs -n yunikorn deployment/yunikorn-scheduler-k8s"

# Task Group 설정 확인
su - philip -c "minikube kubectl -- get sparkapplication <app-name> -o jsonpath='{.spec.driver.annotations}'" | jq .

# Queue 상태 확인
su - philip -c "minikube kubectl -- get queues -n yunikorn -o yaml"
```

## 🔄 시나리오별 동작 예시

### 시나리오 1: 작은 파일 (현재 상태)
```
파일: 52KB
Threshold: 10MB
결과: queue = "min"
```

### 시나리오 2: 큰 파일
```
파일: 15MB
Threshold: 10MB
결과: queue = "max"
```

### 시나리오 3: 활성화 모드 (enabled=true)
```
→ 리소스 계산 수행
→ queue: "min" 또는 "max"
→ executor minMember: config.json 값 (예: 1)
→ 최적화된 YAML 제출
```

### 시나리오 4: 비활성화 모드 (enabled=false)
```
→ 리소스 계산 건너뜀
→ queue: 템플릿 원본 (root.default)
→ executor minMember: 템플릿 원본 (예: 2)
→ 템플릿 원본대로 제출
```
# Pipeline Test
