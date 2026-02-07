# Fluent Bit 완전 가이드

## 목차

1. [개요](#1-개요)
2. [아키텍처](#2-아키텍처)
3. [설치](#3-설치)
4. [설정](#4-설정)
5. [로그 수집 현황](#5-로그-수집-현황)
6. [검색 쿼리](#6-검색-쿼리)
7. [관리](#7-관리)
8. [문제 해결](#8-문제-해결)

---

## 1. 개요

### 시스템 구성

본 시스템은 **이중 Fluent Bit 아키텍처**를 사용하여 모든 로그를 **unified-logs** 인덱스에 통합 저장합니다.

| 구분 | 실행 위치 | 역할 |
|------|-----------|------|
| **K8s Fluent Bit** | Kubernetes Pod (DaemonSet) | 컨테이너/파드 로그, K8s Events |
| **Host Fluent Bit** | Host Linux (systemd) | 마이크로서비스, Kubelet, CRI, Syslog, systemd/journald |

### Unified Logs 인덱스

- **인덱스명**: `unified-logs`
- **OpenSearch**: http://192.168.201.152:9200
- **총 문서**: 35,805건+
- **목적**: 단일 인덱스에서 모든 로그 통합 검색

---

## 2. 아키텍처

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                          │
│                                                               │
│  ┌───────────────────────────────────────────────────────┐      │
│  │  Fluent Bit DaemonSet (logging 네임스페이스)        │      │
│  │  - 컨테이너 로그 (/var/log/containers/*.log)       │      │
│  │  - Kubernetes Events                             │      │
│  │  - 네임스페이스: default, yunikorn, kube-system   │      │
│  └──────────────────────┬────────────────────────────────┘      │
│                         │                                  │
└─────────────────────────┼──────────────────────────┘       │
                          │                                   │
┌─────────────────────────┴───────────────────────────┐     │
│              unified-logs (OpenSearch Port:9200)        │     │
│                                                           │     │
└───────────────────────────────────────────────────────┘     │
                          │                                   │
┌─────────────────────────────────────────────────────────────┐
│                      Host Linux                            │
│  ┌───────────────────────────────────────────────────────┐ │
│  │  Fluent Bit Host (systemd: fluent-bit-host)          │ │
│  │  - 마이크로서비스 (/root/hynix/server.log)          │ │
│  │  - Kubelet (minikube 컨테이너)                       │ │
│  │  - Syslog (/var/log/syslog)                          │ │
│  │  - systemd/journald (docker, containerd, crio)       │ │
│  │  - Kernel (/var/log/kern.log)                         │ │
│  └──────────────────────┬────────────────────────────────┘ │
└─────────────────────────┴───────────────────────────────┘
```

---

## 3. 설치

### 3.1 K8s Fluent Bit DaemonSet

**설치 위치**: `/root/hynix/fluent-bit/fluent-bit-k8s.yaml`

```bash
# 배포
su - philip -c "minikube kubectl -- apply -f /root/hynix/fluent-bit/fluent-bit-k8s.yaml"

# 상태 확인
su - philip -c "minikube kubectl -- -n logging get pods"
```

**리소스**:
- ConfigMap: `fluent-bit-config`
- DaemonSet: `fluent-bit`
- ServiceAccount: `fluent-bit`

### 3.2 Host Fluent Bit (systemd)

**바이너리**: `/usr/local/bin/fluent-bit` (v2.2.0)
**서비스**: `fluent-bit-host`

```bash
# 설치된 바이너리 확인
fluent-bit --version

# 서비스 상태
sudo systemctl status fluent-bit-host.service
```

**의존성 패키지**:
- `libssl1.1`
- `libpq5`

---

## 4. 설정

### 4.1 K8s Fluent Bit ConfigMap

**파일**: `/root/hynix/fluent-bit/fluent-bit-k8s.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: logging
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush         1
        Daemon        off
        Log_Level     info
        Parsers_File  parsers.conf

    # INPUT: 컨테이너 로그 수집
    [INPUT]
        Name              tail
        Path              /var/log/containers/*.log
        Parser            docker
        Tag               kube.*
        Refresh_Interval  1
        Mem_Buf_Limit     50MB
        Skip_Long_Lines   On
        DB                /fluent-bit/tmp/flb_containers.db
        DB.Sync           Normal
        DB.Locking        True

    # INPUT: Kubernetes Events
    [INPUT]
        Name              kubernetes_events
        Tag               k8s.events

    # FILTER: Kubernetes 메타데이터
    [FILTER]
        Name                kubernetes
        Match               kube.*
        Kube_URL            https://kubernetes.default.svc:443
        Kube_CA_File        /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        Kube_Token_File      /var/run/secrets/kubernetes.io/serviceaccount/token
        Merge_Log           On
        Keep_Log            Off
        K8s-Logging.Parser  On
        K8s-Logging.Exclude Off
        Labels              On
        Annotations        On

    # FILTER: 로그 타입 태그 추가
    [FILTER]
        Name                modify
        Match               kube.*
        Add                 log_type     container

    [FILTER]
        Name                modify
        Match               k8s.events
        Add                 log_type     kubernetes_event

    # OUTPUT: OpenSearch 통합
    [OUTPUT]
        Name                opensearch
        Match               *
        Host                192.168.201.152
        Port                9200
        Index               unified-logs
        Suppress_Type_Name  On
        Retry_Limit         5
```

### 4.2 Host Fluent Bit Config

**파일**: `/etc/fluent-bit/fluent-bit.conf`

```ini
[SERVICE]
    Flush         5
    Daemon        off
    Log_Level     info
    Parsers_File  parsers.conf

# INPUT: 마이크로서비스 로그
[INPUT]
    Name              tail
    Path              /root/hynix/server.log
    Tag               microservice.hynix
    Refresh_Interval  5
    Mem_Buf_Limit     50MB
    Skip_Long_Lines   On
    DB                /var/lib/fluent-bit/flb_microservice.db
    DB.Sync           Normal
    DB.Locking        True
    Read_from_Head    False

# INPUT: kubelet 로그
[INPUT]
    Name              tail
    Path              /var/lib/docker/containers/b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b/b8ef17dd229aedc4050b62ddc4d11d5c636ce4e22a199a251a333d9072ba2d7b-json.log
    Tag               host.kubelet
    Refresh_Interval  5
    Mem_Buf_Limit     50MB
    Skip_Long_Lines   On
    DB                /var/lib/fluent-bit/flb_kubelet.db
    DB.Sync           Normal
    DB.Locking        True
    Read_from_Head    False

# INPUT: Syslog (CRI 데몬, 시스템 로그)
[INPUT]
    Name              tail
    Path              /var/log/syslog
    Tag               host.syslog
    Parser            syslog-rfc5424
    Refresh_Interval  5
    Mem_Buf_Limit     50MB
    Skip_Long_Lines   On
    DB                /var/lib/fluent-bit/flb_syslog.db
    DB.Sync           Normal
    DB.Locking        True
    Read_from_Head    False

# INPUT: Journald (systemd 서비스 로그)
[INPUT]
    Name              systemd
    Tag               journal.containerd
    Systemd_Filter    _SYSTEMD_UNIT=containerd.service
    Read_From_Tail    True
    Path              /run/log/journal

[INPUT]
    Name              systemd
    Tag               journal.docker
    Systemd_Filter    _SYSTEMD_UNIT=docker.service
    Read_From_Tail    True
    Path              /run/log/journal

[INPUT]
    Name              systemd
    Tag               journal.crio
    Systemd_Filter    _SYSTEMD_UNIT=crio.service
    Read_From_Tail    True
    Path              /run/log/journal

# INPUT: Kernel 로그
[INPUT]
    Name              tail
    Path              /var/log/kern.log
    Tag               host.kernel
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On

# FILTER: 태그 추가
[FILTER]
    Name                modify
    Match               microservice.*
    Add                 log_type     microservice

[FILTER]
    Name                modify
    Match               host.kubelet
    Add                 log_type     kubelet

[FILTER]
    Name                modify
    Match               host.syslog
    Add                 log_type     cri

[FILTER]
    Name                modify
    Match               host.kernel
    Add                 log_type     kernel

[FILTER]
    Name                modify
    Match               journal.*
    Add                 log_type     systemd

[FILTER]
    Name                modify
    Match               *
    Add                 log_source   host

# OUTPUT: OpenSearch
[OUTPUT]
    Name                opensearch
    Match               *
    Host                192.168.201.152
    Port                9200
    Index               unified-logs
    Suppress_Type_Name  On
    Retry_Limit         5
```

### 4.3 Parser 설정

**K8s Fluent Bit Parsers**:
- `docker` - 컨테이너 JSON 로그
- `cri` - CRI 로그 파싱
- `syslog` - syslog RFC3164
- `json` - 일반 JSON

**Host Fluent Bit Parsers**:
- `syslog-rfc5424` - syslog RFC5424
- `json` - JSON 파싱
- `kubelet` - Kubelet 로그
- `containerd` - Containerd 로그
- `cri` - CRI 로그

---

## 5. 로그 수집 현황

### unified-logs 인덱스

| log_type | 건수 | 출처 |
|----------|------|------|
| **container** | 19,732건 | K8s 컨테이너 (Pod) |
| **host** | 14,630건 | Host 시스템 (systemd/journald) |
| **kubelet** | 1,088건 | Kubelet |
| **cri** | 290건 | Syslog (CRI 데몬) |
| **microservice** | 173건 | 마이크로서비스 |
| **kubernetes_event** | 92건 | K8s Events |
| **kernel** | 포함됨 | 커널 로그 |
| **systemd** | 포함됨 | systemd 서비스 |
| **total** | **35,805건** | unified-logs |

### 로그 소스별 상세

| 로그 소스 | 경로 | Fluent Bit | log_type |
|----------|------|-----------|----------|
| **마이크로서비스** | /root/hynix/server.log | Host | microservice |
| **Kubelet** | /var/lib/docker/containers/.../...-json.log | Host | kubelet |
| **Syslog** | /var/log/syslog | Host | cri |
| **systemd** | /run/log/journal | Host | systemd |
| **Kernel** | /var/log/kern.log | Host | kernel |
| **컨테이너** | /var/log/containers/*.log | K8s | container |
| **K8s Events** | K8s API | K8s | kubernetes_event |

---

## 6. 검색 쿼리

### 6.1 전체 로그 검색

```bash
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {"match_all": {}},
    "size": 10,
    "sort": [{"@timestamp": {"order": "desc"}}]
  }'
```

### 6.2 log_type별 검색

```bash
# 마이크로서비스
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "microservice"}, "size": 10}'

# Kubelet
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "kubelet"}, "size": 10}'

# CRI
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "cri"}, "size": 10}'

# systemd/journald
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "systemd"}, "size": 10}'

# 컨테이너
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{"query": {"term": {"log_type.keyword": "container"}, "size": 10}'
```

### 6.3 서비스 ID별 검색

```bash
SERVICE_ID="test-11111"

# 전체 검색
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "should": [
          {"match": {"log": "'"$SERVICE_ID"'"}},
          {"wildcard": {"kubernetes.pod_name.keyword": "*'"$SERVICE_ID"'*"}},
          {"term": {"kubernetes.labels.applicationId.keyword": "'"$SERVICE_ID"'"}}
        ]
      }
    },
    "size": 10
  }'
```

### 6.4 Executor 로그 식별

```bash
# Executor만 검색
curl -X POST "http://192.168.201.152:9200/unified-logs/_search?pretty=true" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "term": {"kubernetes.labels.spark-role.keyword": "executor"}
    },
    "size": 10
  }'
```

### 6.5 Spark Operator & Yunikorn Scheduler

```bash
# Spark Operator
curl -s "http://192.168.201.152:9200/unified-logs/_count?q=kubernetes.namespace_name:spark-operator&pretty=true"

# Yunikorn Scheduler
curl -s "http://192.168.201.152:9200/unified-logs/_count?q=kubernetes.namespace_name:yunikorn&pretty=true"
```

---

## 7. 관리

### 7.1 K8s Fluent Bit 관리

```bash
# 상태 확인
su - philip -c "minikube kubectl -- -n logging get pods"

# 로그 확인
su - philip -c "minikube kubectl -- -n logging logs -l app=fluent-bit --tail=20"

# 재시작
su - philip -c "minikube kubectl -- -n logging delete pod -l app=fluent-bit"

# 설정 업데이트
su - philip -c "minikube kubectl -- -n logging apply -f /root/hynix/fluent-bit/fluent-bit-k8s.yaml"
```

### 7.2 Host Fluent Bit 관리

```bash
# 상태 확인
sudo systemctl status fluent-bit-host.service

# 로그 확인
sudo journalctl -u fluent-bit-host -f

# 재시작
sudo systemctl restart fluent-bit-host.service

# 설정 다시 로드
sudo systemctl daemon-reload
sudo systemctl restart fluent-bit-host.service

# 설정 파일 위치
cat /etc/fluent-bit/fluent-bit.conf
```

### 7.3 OpenSearch 관리

```bash
# 인덱스 목록
curl -s "http://192.168.201.152:9200/_cat/indices?v"

# unified-logs 인덱스 정보
curl -s "http://192.168.201.152:9200/unified-logs/_search?pretty=true" -H 'Content-Type: application/json' -d '{"query": {"match_all": {}}, "size": 0}'

# 인덱스 삭제 (주의!)
curl -X DELETE "http://192.168.201.152:9200/old-index-name"
```

---

## 8. 문제 해결

### 8.1 /var/log/pods 권한 문제

**문제**: spark-operator, yunikorn 네임스페이스 로그 수집 안 됨

**해결**:
```bash
# 권한 변경 (755)
sudo chmod 755 /var/log/pods

# 확인
ls -la /var/log/ | grep pods
```

### 8.2 Fluent Bit 전송 실패

**에러**:
```
[warn] [engine] failed to flush chunk '1-1769666068.xxx.flb'
```

**원인**: OpenSearch 일시적 부하 or 네트워크 문제

**해결**:
```bash
# OpenSearch 상태 확인
curl "http://192.168.201.152:9200/_cluster/health?pretty=true"

# Fluent Bit 로그 확인
sudo journalctl -u fluent-bit-host
```

### 8.3 중복 로그 수집 확인

**문제**: K8s와 Host에서 같은 로그를 수집하지 않는지 확인 필요

**해결**: log_type으로 분리됨
- K8s: `container` (컨테이너 로그)
- Host: `microservice`, `kubelet`, `cri`, `systemd`, `kernel`

### 8.4 필드 충돌

**에러**:
```
mapper_parsing_exception: object mapping for [source] tried to parse field [source] as object, but found a concrete value
```

**해결**: `log_source` 필드 사용 (K8s는 `kubernetes.source`, Host는 `log_source`)

---

## 9. 파일 위치

### K8s Fluent Bit
- **설치**: `/root/hynix/fluent-bit/fluent-bit-k8s.yaml`
- **ConfigMap**: `logging/fluent-bit-config`

### Host Fluent Bit
- **바이너리**: `/usr/local/bin/fluent-bit`
- **설정**: `/etc/fluent-bit/fluent-bit.conf`
- **Parser**: `/etc/fluent-bit/parsers.conf`
- **서비스**: `/etc/systemd/system/fluent-bit-host.service`

### 문서
- **README**: `/root/hynix/FLUENT_BIT_README.md` (이 파일)
- **분석**: `/root/hynix/FLUENT_BIT_ANALYSIS.md`
- **호스트 완료**: `/root/hynix/FLUENT_BIT_HOST_COMPLETE.md`
- **검색**: `/root/hynix/HOST_LOG_QUERIES.md`

---

## 10. 참고

### OpenSearch
- **URL**: http://192.168.201.152:9200
- **Dashboards**: http://192.168.201.152:5601

### Fluent Bit 공식 문서
- https://docs.fluentbit.io/
- https://github.com/fluent/fluent-bit

### 관련 문서
- `/root/hynix/EXECUTOR_LOG_IDENTIFICATION.md`
- `/root/hynix/IDENTIFICATION_SPARK_YUNIKORN.md`

---

**버전**: Fluent Bit v2.2.0
**마지막 최종 업데이트**: 2026-01-30
