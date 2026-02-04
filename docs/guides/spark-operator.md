# Spark Operator - Complete Guide

**Version**: 1.0
**Last Updated**: 2026-02-02
**Kubernetes Namespace**: default
**Operator Version**: sparkoperator-k8s.io

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Deployment](#deployment)
4. [Log Collection](#log-collection)
5. [Usage Examples](#usage-examples)
6. [Monitoring & Debugging](#monitoring--debugging)
7. [Troubleshooting](#troubleshooting)
8. [API Reference](#api-reference)

---

## Overview

### What is Spark Operator?

Spark Operator is a Kubernetes operator that manages **Spark applications** on Kubernetes. It extends Kubernetes with the `SparkApplication` custom resource, allowing you to define and manage Spark jobs natively.

### Key Features

- **Native Kubernetes Integration**: Manage Spark jobs as Kubernetes custom resources
- **Automatic Submission**: Operator handles driver/executor pod submission
- **Lifecycle Management**: Automatic restart, monitoring, and cleanup
- **Yunikorn Integration**: Gang scheduling support for Spark applications
- **Event Broadcasting**: Kubernetes events for state changes

### Components

| Component | Description |
|-----------|-------------|
| **Spark Operator Controller** | Main operator pod, watches for SparkApplication resources |
| **Spark Driver Pod** | Submits job to Kubernetes, coordinates executors |
| **Spark Executor Pods** | Execute Spark tasks |
| **Yunikorn Scheduler** | Gang scheduling for resource guarantees |
| **Yunikorn Admission Controller** | Pod admission validation |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    Spark Operator Controller                  │  │
│  │  - Watches SparkApplication CRs                              │  │
│  │  - Submits Spark jobs to Kubernetes                           │  │
│  │  - Manages driver/executor pods                               │  │
│  │  - Broadcasts Kubernetes events                               │  │
│  └───────────────────────────┬──────────────────────────────────┘  │
│                              │                                       │
│                              │ Creates                               │
│                              ▼                                       │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     SparkApplication                          │  │
│  │  (Custom Resource - Your Job Definition)                      │  │
│  └───────────────────────────┬──────────────────────────────────┘  │
│                              │                                       │
│                  ┌───────────┴───────────┐                          │
│                  ▼                       ▼                          │
│  ┌───────────────────────┐   ┌───────────────────────┐            │
│  │   Spark Driver Pod    │   │  Spark Executor Pods  │            │
│  │  - job submission     │   │  - task execution     │            │
│  │  - coordination       │   │  - data processing    │            │
│  └───────────────────────┘   └───────────────────────┘            │
│             │                             │                         │
│             └──────────┬──────────────────┘                         │
│                        ▼                                            │
│           ┌─────────────────────┐                                   │
│           │  Yunikorn Scheduler │◀── Gang Scheduling               │
│           └─────────────────────┘                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Deployment

### Current Status

```bash
# Check Spark Operator pod
kubectl get pods -n default | grep spark-operator

# Output:
# spark-operator-controller-cbfc96fd6-ztwrb   1/1     Running   0          5h32m
```

### CRD (Custom Resource Definition)

```bash
# Check SparkApplication CRD
kubectl get crd | grep sparkapplication

# Output:
# sparkapplications.sparkoperator.k8s.io        2025-11-14T00:56:53Z
```

### Service Account & RBAC

The Spark Operator uses service account `spark-operator` with permissions to:
- Create/manage pods
- Create/configmaps/services
- Watch SparkApplication resources
- Broadcast Kubernetes events

---

## Log Collection

### Overview

Spark Operator logs are now **successfully collected** in OpenSearch with full Kubernetes metadata enrichment.

### Collection Status

| Component | Log Count | Status | Collection Rate |
|-----------|-----------|--------|-----------------|
| **Spark Operator Controller** | 55+ logs/job | ✅ Active | 100% |
| **Spark Driver Pod** | ~25 logs/job | ⚠️ Partial | ~25% |
| **Spark Executor Pods** | ~12 logs/job | ⚠️ Partial | ~24% |

### Log Sources

#### 1. Spark Operator Controller Logs
- **Location**: `/var/log/containers/spark-operator-controller-*.log`
- **Stream**: stderr (all logs written to stderr)
- **Format**: Tab-separated with JSON metadata

**Sample Log:**
```
2026-02-02T07:11:49.684Z	INFO	sparkapplication/controller.go:194	Reconciling SparkApplication	{"controller": "spark-application-controller", "namespace": "default", "name": "test-clean", "reconcileID": "a5310f17-de46-43ae-95f2-f925b4c5aa3c", "state": "COMPLETED"}
```

**Fields:**
- Timestamp
- Log Level (INFO, ERROR, WARN)
- Source File (e.g., `sparkapplication/controller.go:194`)
- Message
- JSON Metadata (controller, namespace, name, reconcileID, state)

#### 2. Spark Driver/Executor Logs
- **Location**: `/var/log/containers/spark-*-driver-*.log`
- **Location**: `/var/log/containers/spark-*-exec-*.log`
- **Streams**: stdout and stderr

### Fluent Bit Configuration

**ConfigMap**: `fluent-bit-config` in `logging` namespace

```ini
[INPUT]
    Name              tail
    Path              /var/log/containers/*.log
    Exclude_Path      /var/log/containers/*fluent-bit*.log
    Parser            docker
    Tag               kube.*
    Refresh_Interval  1
    Mem_Buf_Limit     50MB
    DB                /fluent-bit/tmp/flb_containers.db

[FILTER]
    Name                kubernetes
    Match               kube.*
    Merge_Log           On
    K8s-Logging.Parser  On
    Labels              On
    Annotations        On

[OUTPUT]
    Name                opensearch
    Match               *
    Host                192.168.201.152
    Port                9200
    Index               unified-logs
    generate_id         On
```

### Querying Spark Operator Logs in OpenSearch

#### All Spark Operator Logs
```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "wildcard": {
        "kubernetes.pod_name.keyword": "*spark-operator*"
      }
    },
    "size": 20,
    "sort": [{"@timestamp": "desc"}]
  }' | jq '.hits.hits[]'
```

#### Spark Operator Logs for Specific Job
```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"wildcard": {"kubernetes.pod_name.keyword": "*spark-operator*"}},
          {"wildcard": {"log": "*test-clean*"}}
        ]
      }
    }
  }' | jq '.hits.total'
```

#### Spark Operator Reconcile Events
```bash
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"term": {"kubernetes.container_name.keyword": "spark-operator-controller"}},
          {"wildcard": {"log": "*Reconciling SparkApplication*"}}
        ]
      }
    },
    "size": 0,
    "aggs": {
      "jobs": {
        "terms": {
          "field": "log.keyword",
          "size": 100
        }
      }
    }
  }'
```

### Log Types

| Log Pattern | Description |
|-------------|-------------|
| `Reconciling SparkApplication` | Operator processing SparkApplication CR |
| `Finished reconciling SparkApplication` | Reconciliation cycle completed |
| `SparkApplication updated` | State transition occurred |
| `Spark pod updated` | Driver/executor pod status changed |
| `Submission failed` | Job submission failed (error) |

### Common Log Messages

#### Successful Job Execution
```
Reconciling SparkApplication	{"name": "test-clean", "state": "SUBMITTED"}
SparkApplication updated	{"oldState": "SUBMITTED", "newState": "RUNNING"}
Spark pod updated	{"oldPhase": "Pending", "newPhase": "Running"}
Reconciling SparkApplication	{"name": "test-clean", "state": "COMPLETED"}
```

#### Failed Job Execution
```
Submission failed	{"error": "driver pod failed"}
SparkApplication updated	{"oldState": "RUNNING", "newState": "FAILING"}
Reconciling SparkApplication	{"name": "test-clean", "state": "FAILED"}
```

---

## Usage Examples

### Creating a Spark Application

#### Via API (Current Setup)
```bash
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{
    "provision_id": "0002_wfbm",
    "service_id": "test-55555",
    "category": "tttm",
    "region": "ic"
  }'
```

#### Direct YAML
```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: test-55555
  namespace: default
spec:
  type: Scala
  mode: cluster
  image: spark:3.5.0
  imagePullPolicy: Always
  mainClass: org.apache.spark.examples.SparkPi
  mainApplicationFile: local:///opt/spark/examples/jars/spark-examples_2.12-3.5.0.jar
  sparkVersion: 3.5.0
  restartPolicy: OnFailure
  driver:
    cores: 1
    coreLimit: 1200m
    memory: 512m
    serviceAccount: spark
  executor:
    cores: 1
    instances: 2
    memory: 512m
    memoryOverhead: 100m
  deps: {}
```

Apply with:
```bash
kubectl apply -f spark-application.yaml
```

### Monitoring Job Status

#### Watch SparkApplication Status
```bash
kubectl get sparkapplication test-55555 -w
```

#### Get Detailed Status
```bash
kubectl describe sparkapplication test-55555
```

#### Check Driver Pod
```bash
kubectl get pods -l spark-role=driver,spark-app-name=test-55555
```

#### Check Executor Pods
```bash
kubectl get pods -l spark-role=executor,spark-app-name=test-55555
```

### Viewing Logs

#### Spark Operator Controller Logs
```bash
# From pod
kubectl logs -n default spark-operator-controller-cbfc96fd6-ztwrb --tail=50 -f

# From OpenSearch
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "wildcard": {
        "kubernetes.pod_name.keyword": "*spark-operator*"
      }
    },
    "size": 10,
    "sort": [{"@timestamp": "desc"}]
  }'
```

#### Spark Driver Logs
```bash
# From pod
kubectl logs test-55555-driver -f

# From OpenSearch
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "wildcard": {
        "kubernetes.pod_name.keyword": "*55555-driver*"
      }
    }
  }'
```

#### Spark Executor Logs
```bash
# From pod
kubectl logs test-55555-exec-1 -f

# From OpenSearch
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{
    "query": {
      "bool": {
        "must": [
          {"wildcard": {"kubernetes.pod_name.keyword": "*55555-exec*"}}
        ]
      }
    }
  }'
```

---

## Monitoring & Debugging

### Spark Application States

```
SUBMITTED → RUNNING → SUCCEEDING → COMPLETED
              ↓
           FAILING → FAILED
              ↓
           PENDING_RESubmission → SUBMITTED
```

### Kubernetes Events

Spark Operator broadcasts Kubernetes events for state changes:

```bash
# Get events for a SparkApplication
kubectl get events --field-selector involvedObject.kind=SparkApplication,involvedObject.name=test-55555

# Sample events:
# 10s         Normal   SparkApplicationUpdated  sparkapplication/test-55555
# SparkApplication test-55555 updated: SUBMITTED -> RUNNING

# 5s ago     Normal   SparkApplicationUpdated  sparkapplication/test-55555
# SparkApplication test-55555 updated: RUNNING -> COMPLETED
```

### Key Metrics to Monitor

| Metric | Description | Query |
|--------|-------------|-------|
| Spark Operator Log Count | Number of operator logs per job | OpenSearch aggregation |
| Job Success Rate | Percentage of successful jobs | kubectl get sparkapplication |
| Average Job Duration | Time from SUBMITTED to COMPLETED | Event timestamps |
| Driver Pod Start Time | Time from submission to driver running | Pod creation timestamp |
| Executor Count | Number of executor pods per job | kubectl get pods -l spark-role=executor |

### Common Issues

#### 1. Stuck in SUBMITTED State
**Symptoms**: SparkApplication stays in SUBMITTED state

**Diagnosis**:
```bash
kubectl describe sparkapplication <name>
kubectl get events --field-selector involvedObject.kind=SparkApplication,involvedObject.name=<name>
```

**Causes**:
- Yunikorn scheduler not available
- Insufficient cluster resources
- Driver pod image pull failure

#### 2. Frequent Restarts
**Symptoms**: Driver or executor pods restarting

**Diagnosis**:
```bash
kubectl describe pod <driver-or-executor-pod>
kubectl logs <driver-or-executor-pod> --previous
```

**Causes**:
- OOMKilled (insufficient memory)
- Image pull errors
- Application errors

#### 3. Missing Spark Operator Logs
**Symptoms**: No spark-operator logs in OpenSearch

**Diagnosis**:
```bash
# Check Fluent Bit is watching
kubectl -n logging logs fluent-bit-<pod> | grep spark-operator

# Check log file exists
kubectl -n default exec spark-operator-controller-xxx -- ls -la /var/log/
```

**Solution**:
- Restart Fluent Bit DaemonSet
- Check Fluent Bit configuration

---

## Troubleshooting

### Spark Operator Not Responding

```bash
# Check operator pod status
kubectl get pods -n default | grep spark-operator

# Check operator logs
kubectl logs -n default spark-operator-controller-xxx --tail=100

# Restart operator
kubectl delete pod spark-operator-controller-xxx
# It will be automatically recreated
```

### Log Collection Issues

#### No Spark Operator Logs in OpenSearch

**Step 1: Verify Log File Exists**
```bash
# On minikube node
minikube ssh "ls -la /var/log/containers/*spark-operator*"
```

**Step 2: Check Fluent Bit Watching File**
```bash
kubectl -n logging logs fluent-bit-xxx | grep spark-operator | grep inotify
```

**Step 3: Check Fluent Bit Errors**
```bash
kubectl -n logging logs fluent-bit-xxx | grep -i error
```

**Step 4: Verify OpenSearch Connection**
```bash
curl -s "http://192.168.201.152:9200/_cluster/health"
```

### Spark Application Not Starting

**Step 1: Check SparkApplication Status**
```bash
kubectl get sparkapplication
kubectl describe sparkapplication <name>
```

**Step 2: Check Kubernetes Events**
```bash
kubectl get events --sort-by='.lastTimestamp' | tail -20
```

**Step 3: Check Yunikorn Scheduler**
```bash
kubectl get pods -n yunikorn
kubectl logs -n yunikorn yunikorn-scheduler-xxx
```

**Step 4: Check Resource Availability**
```bash
kubectl top nodes
kubectl describe nodes
```

---

## API Reference

### SparkApplication Resource

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: string          # Application name
  namespace: string     # Target namespace
spec:
  type: string          # Scala, Python, Java, R
  mode: string          # cluster, client
  image: string         # Docker image
  sparkVersion: string  # Spark version (e.g., 3.5.0)

  # Job specification
  mainClass: string     # Main class (for Scala/Java)
  mainApplicationFile: string  # Job file path

  # Driver configuration
  driver:
    cores: int
    coreLimit: string   # e.g., "1200m"
    memory: string      # e.g., "512m"
    serviceAccount: string
    env: []             # Environment variables

  # Executor configuration
  executor:
    cores: int
    instances: int
    memory: string
    memoryOverhead: string

  # Restart policy
  restartPolicy: string # OnFailure, Never, Always

  # Dependencies
  deps:
    jars: []
    pyFiles: []
    files: []

  # Monitoring
  monitoring:
    exposeDriverMetrics: bool
    exposeExecutorMetrics: bool

  # Timeouts
  timeToLiveSeconds: int
  pendingRetryIntervalSeconds: int
```

### Application States

| State | Description |
|-------|-------------|
| `NEW` | Application created, not yet submitted |
| `SUBMITTED` | Application submitted to Kubernetes |
| `RUNNING` | Driver pod is running |
| `COMPLETING` | Application completing |
| `SUCCEEDING` | Application succeeded |
| `COMPLETED` | Application fully completed |
| `FAILING` | Application is failing |
| `FAILED` | Application failed |
| `PENDING_RERUN` | Pending rerun after failure |
| `UNKNOWN` | State unknown |

---

## Configuration Files

### Fluent Bit K8s Config
**Location**: `/root/hynix/fluent-bit/fluent-bit-k8s.yaml`

### Spark Operator Deployment
**Namespace**: default
**Pod Name**: spark-operator-controller-cbfc96fd6-ztwrb

### OpenSearch Index
**Name**: unified-logs
**Host**: 192.168.201.152:9200

---

## Related Documentation

- [Fluent Bit Status Report](/root/hynix/FLUENT_BIT_STATUS.md)
- [Spark Operator Log Status](/root/hynix/SPARK_OPERATOR_LOG_STATUS.md)
- [Final Resolution Report](/tmp/FINAL_RESOLUTION_REPORT.md)

---

## Quick Reference Commands

```bash
# Create Spark job via API
curl -X POST http://localhost:8080/api/v1/spark/create \
  -H "Content-Type: application/json" \
  -d '{"provision_id":"0002_wfbm","service_id":"test-55555","category":"tttm","region":"ic"}'

# Check Spark applications
kubectl get sparkapplication

# Check Spark operator logs
kubectl logs -n default spark-operator-controller-xxx --tail=50

# Query spark-operator logs in OpenSearch
curl -s "http://192.168.201.152:9200/unified-logs/_search" \
  -H 'Content-Type: application/json' \
  -d '{"query":{"wildcard":{"kubernetes.pod_name.keyword":"*spark-operator*"}}}' | jq '.hits.total'

# Get Spark job events
kubectl get events --field-selector involvedObject.kind=SparkApplication

# Check driver pod
kubectl get pods -l spark-role=driver

# Check executor pods
kubectl get pods -l spark-role=executor
```

---

## Summary

✅ Spark Operator is **fully operational**
✅ Logs are **successfully collected** in OpenSearch (55+ logs/job)
✅ Kubernetes metadata enrichment **working**
✅ Gang scheduling via Yunikorn **enabled**
⚠️ Driver/executor log collection at **25-40%** (needs investigation)

---

**Document Version**: 1.0
**Last Updated**: 2026-02-02
**Maintained By**: Data Engineering Team
