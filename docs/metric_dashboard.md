# Spark Metrics Dashboard & Prometheus Guide

Comprehensive guide to Spark metrics collection, Prometheus integration, and the Spark Metrics Dashboard for monitoring applications on Kubernetes.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Dashboard Features](#dashboard-features)
- [Spark Metrics Configuration](#spark-metrics-configuration)
- [Available Metrics](#available-metrics)
- [Metrics Explained](#metrics-explained)
- [Querying Metrics](#querying-metrics)
- [Dashboard Usage](#dashboard-usage)
- [Installation](#installation)
- [API Reference](#api-reference)
- [Integration with Prometheus](#integration-with-prometheus)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)
- [References](#references)

---

## Overview

Apache Spark exposes comprehensive metrics through the Prometheus metrics sink. This guide covers:

- **Spark Dashboard UI**: Web interface for visualizing Spark metrics
- **Prometheus Metrics**: Technical reference for metrics collection and querying
- **Spark Driver Metrics**: Application-level metrics, executor status, task progress
- **Spark Executor Metrics**: JVM memory, GC, CPU, shuffle metrics
- **Querying**: Direct access, proxy server, and PromQL examples

### Dashboard Location

```
http://localhost:8083/spark-metrics-ui.html
```

### Key Technologies

| Component | Purpose |
|-----------|---------|
| **Kubernetes API** | Pod and node metrics |
| **Prometheus Metrics** | Spark application metrics |
| **kubectl exec** | Remote command execution for metrics retrieval |
| **Proxy Server** | API passthrough via `/api/api/v1/` |

### Metric Types

| Type | Description | Example |
|------|-------------|---------|
| **Counter** | Monotonically increasing value | `spark.executor.tasks.total` |
| **Gauge** | Can go up or down | `spark.executor.memory.used` |
| **Histogram** | Distribution of values | `spark.task.duration.seconds` |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Web Browser                                │
│                  http://localhost:8083                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     Spark Metrics Dashboard                              │  │
│  │     /spark-metrics-ui.html                               │  │
│  │  - Pod Selection Dropdown                                │  │
│  │  - Memory Charts (Heap, GC)                              │  │
│  │  - Shuffle Metrics (Read/Write)                          │  │
│  │  - Executor Stats (Tasks, Active/Completed/Failed)       │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP API
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Proxy Server (Go)                             │
│                       :8083                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     Kubernetes API Proxy                                 │  │
│  │     /api/api/v1/namespaces/{ns}/pods/{pod}/proxy/...    │  │
│  │                                                          │  │
│  │     Metrics Endpoints:                                   │  │
│  │  - /metrics/driver/prometheus/                           │  │
│  │  - /metrics/executors/prometheus/                        │  │
│  │                                                          │  │
│  │     Implementation:                                      │  │
│  │  kubectl exec -n {ns} {pod} -- curl -s http://localhost│  │
│  │  4040{metrics_path}                                      │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ kubectl exec + curl
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Spark Application Pod                              │
│                   localhost:4040                                │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     Driver Pod                                           │  │
│  │  - Spark UI: http://localhost:4040                       │  │
│  │  - Driver Metrics: /metrics/driver/prometheus/           │  │
│  │  - Executor Metrics: /metrics/executors/prometheus/      │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     Executor Pods                                         │  │
│  │  - Each exposes metrics on localhost:4040                │  │
│  │  - Metrics scraped via driver pod                        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Spark Config:                                                  │
│  - spark.metrics.conf=*.sink.prometheus.class=...              │
│  - spark.metrics.app.status=true                               │
│  - spark.metrics.executors.status=true                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Dashboard Features

### 1. Pod Selection

**Location**: Top of the dashboard

**Functionality**:
- Dropdown listing all Spark driver pods in the cluster
- Pods are identified by the `spark-app-selector` label
- Format: `pod-name-namespace`

**Labels Used**:
```yaml
labels:
  spark-app-selector: "application-name"
  spark-role: "driver"
```

### 2. Memory Metrics

**Location**: Memory section of the dashboard

**Metrics Displayed**:

| Metric | Description | Source |
|--------|-------------|--------|
| **Heap Used** | Current JVM heap usage in bytes | `jvm.heap.used` |
| **Heap Max** | Maximum heap memory allocated | `jvm.heap.max` |
| **Heap %** | Percentage of heap used | Calculated |
| **GC Time** | Total garbage collection time | `jvm.gc.time.millis` |

**Visual Display**:
- Bar chart showing heap usage vs max
- Percentage display
- GC time in milliseconds

**Example Values**:
```
Heap Used:  134,217,728 bytes (128 MB)
Heap Max:   536,870,912 bytes (512 MB)
Heap %:     25.0%
GC Time:    1,234 ms
```

### 3. Shuffle Metrics

**Location**: Shuffle section

**Metrics Displayed**:

| Metric | Description | Source |
|--------|-------------|--------|
| **Shuffle Read** | Total shuffle data read | `spark.shuffle.read.bytes` |
| **Shuffle Write** | Total shuffle data written | `spark.shuffle.write.bytes` |
| **Fetch Wait Time** | Time waiting for shuffle data | `spark.shuffle.fetch.wait.time.millis` |

**Use Cases**:
- Identify data skew
- Monitor shuffle performance
- Optimize partitioning strategies

### 4. Executor Statistics

**Location**: Executors section

**Metrics Displayed**:

| Metric | Description | Source |
|--------|-------------|--------|
| **Active Tasks** | Currently running tasks | `spark.executor.active.tasks` |
| **Completed Tasks** | Total completed tasks | `spark.executor.completed.tasks` |
| **Failed Tasks** | Total failed tasks | `spark.executor.failed.tasks` |
| **Total Tasks** | Sum of all tasks | Calculated |

**Visual Display**:
- Stat cards for each metric type
- Task completion rate
- Failure percentage

### 5. Auto-Refresh

**Interval**: Every 5 seconds

**Functionality**:
- Automatically fetches latest metrics from selected pod
- Updates charts and statistics without manual refresh
- Graceful error handling on pod failures

---

## Spark Metrics Configuration

### Enable Prometheus Metrics

Add to your Spark application configuration:

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-app-example
spec:
  sparkVersion: "4.0.1"
  driver:
    sparkConf:
      # Enable Prometheus metrics sink
      "spark.metrics.conf.*.sink.prometheus.class": "org.apache.spark.metrics.sink.PrometheusSink"
      "spark.metrics.conf.*.sink.prometheus.period": "5s"
      "spark.metrics.conf.master.sink.prometheus.class": "org.apache.spark.metrics.sink.PrometheusSink"

      # Enable metrics for driver and executors
      "spark.metrics.app.status": "true"
      "spark.metrics.executors.status": "true"

      # Metrics port
      "spark.ui.port": "4040"
      "spark.metrics.app.status.source": "org.apache.spark.metrics.source.ApplicationSource"
```

### Full Metrics Configuration

```properties
# Enable Prometheus sink
spark.metrics.conf=*.sink.prometheus.class=org.apache.spark.metrics.sink.PrometheusSink
spark.metrics.conf=*.sink.prometheus.period=5s
spark.metrics.conf.master.sink.prometheus.class=org.apache.spark.metrics.sink.PrometheusSink

# Enable application status metrics
spark.metrics.app.status=true
spark.metrics.executors.status=true

# Enable executor metrics
spark.executor.metrics.sample=true
spark.executor.metrics.polling.period=1000

# Enable shuffle metrics
spark.shuffle.service.enabled=true
spark.shuffle.service.db.enabled=true

# Enable JVM source
spark.metrics.conf=*.source.jvm.class=org.apache.spark.metrics.source.JvmSource
```

### Spark Operator Configuration

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: pi-calculation
spec:
  type: Scala
  sparkVersion: "4.0.1"
  mainClass: org.apache.spark.examples.SparkPi
  mode: cluster
  image: "apache/spark:v4.0.1"
  imagePullPolicy: Always
  mainApplicationFile: "local:///opt/spark/examples/jars/spark-examples_2.12-4.0.1.jar"
  sparkConf:
    "spark.metrics.conf.*.sink.prometheus.class": "org.apache.spark.metrics.sink.PrometheusSink"
    "spark.metrics.conf.*.sink.prometheus.period": "5s"
    "spark.metrics.app.status": "true"
    "spark.metrics.executors.status": "true"
  driver:
    cores: 1
    coreLimit: "1200m"
    memory: "512m"
    serviceAccount: spark
  executor:
    cores: 1
    coreLimit: "1200m"
    instances: 2
    memory: "512m"
```

---

## Available Metrics

### Driver Metrics

Endpoint: `http://localhost:4040/metrics/driver/prometheus/`

#### Application Status

| Metric Name | Type | Description |
|-------------|------|-------------|
| `spark.app.status.uptime` | Gauge | Application uptime in milliseconds |
| `spark.app.status.running.executors` | Gauge | Number of running executors |
| `spark.app.status.waiting.executors` | Gauge | Number of executors waiting to start |
| `spark.app.status.completed.tasks` | Counter | Total completed tasks |
| `spark.app.status.failed.tasks` | Counter | Total failed tasks |

#### Memory Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `spark.driver.memory.used` | Gauge | Memory used by driver (MB) |
| `spark.driver.memory.remaining` | Gauge | Remaining memory (MB) |

#### Executor Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `spark.executor.metrics.count` | Gauge | Number of executors |
| `spark.executor.tasks.total` | Counter | Total tasks across executors |
| `spark.executor.failedTasks.total` | Counter | Total failed tasks |

### Executor Metrics

Endpoint: `http://localhost:4040/metrics/executors/prometheus/`

#### JVM Memory

| Metric Name | Type | Description |
|-------------|------|-------------|
| `jvm.heap.used` | Gauge | Heap memory used (bytes) |
| `jvm.heap.max` | Gauge | Max heap memory (bytes) |
| `jvm.heap.committed` | Gauge | Committed heap memory (bytes) |
| `jvm.non-heap.used` | Gauge | Non-heap memory used (bytes) |
| `jvm.non-heap.max` | Gauge | Max non-heap memory (bytes) |

#### Garbage Collection

| Metric Name | Type | Description |
|-------------|------|-------------|
| `jvm.gc.time.millis` | Counter | Total time spent in GC (ms) |
| `jvm.gc.count` | Counter | Total GC count |

#### Thread Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `jvm.thread.count` | Gauge | Current thread count |
| `jvm.thread.peak.count` | Gauge | Peak thread count |
| `jvm.thread.daemon.count` | Gauge | Daemon thread count |

#### Shuffle Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `spark.shuffle.read.bytes` | Counter | Total shuffle read bytes |
| `spark.shuffle.write.bytes` | Counter | Total shuffle write bytes |
| `spark.shuffle.fetch.wait.time.millis` | Counter | Time spent waiting for shuffle data (ms) |

#### Task Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `spark.executor.active.tasks` | Gauge | Currently active tasks |
| `spark.executor.failed.tasks` | Counter | Total failed tasks per executor |
| `spark.executor.completed.tasks` | Counter | Total completed tasks per executor |
| `spark.executor.task.duration.ms.sum` | Counter | Total task duration (ms) |
| `spark.executor.task.input.bytes` | Counter | Total input bytes |
| `spark.executor.task.output.bytes` | Counter | Total output bytes |

#### Disk Metrics

| Metric Name | Type | Description |
|-------------|------|-------------|
| `spark.executor.disk.used` | Gauge | Disk space used (bytes) |
| `spark.executor.disk.total` | Gauge | Total disk space (bytes) |

---

## Metrics Explained

### JVM Memory Metrics

#### Heap Memory

| Metric | Prometheus Name | Type | Description |
|--------|----------------|------|-------------|
| Heap Used | `jvm_heap_used_bytes` | Gauge | Current heap usage |
| Heap Max | `jvm_heap_max_bytes` | Gauge | Maximum heap allocated |
| Heap Committed | `jvm_heap_committed_bytes` | Gauge | Committed heap memory |

**Interpretation**:
- **< 60%**: Healthy
- **60-80%**: Monitor
- **> 80%**: Consider increasing heap or optimizing

#### Non-Heap Memory

| Metric | Prometheus Name | Type | Description |
|--------|----------------|------|-------------|
| Non-Heap Used | `jvm_non_heap_used_bytes` | Gauge | Metaspace, code cache, etc. |
| Non-Heap Max | `jvm_non_heap_max_bytes` | Gauge | Max non-heap memory |

### Garbage Collection Metrics

| Metric | Prometheus Name | Type | Description |
|--------|----------------|------|-------------|
| GC Time | `jvm_gc_time_millis_total` | Counter | Total GC time in ms |
| GC Count | `jvm_gc_count_total` | Counter | Number of GC runs |

**Interpretation**:
- **GC Time Rate**: `rate(jvm_gc_time_millis_total[5m])`
  - **< 5%**: Healthy
  - **5-10%**: Monitor
  - **> 10%**: High GC overhead

### Task Metrics

| Metric | Prometheus Name | Type | Description |
|--------|----------------|------|-------------|
| Active Tasks | `spark_executor_active_tasks` | Gauge | Currently running |
| Completed Tasks | `spark_executor_completed_tasks_total` | Counter | Total completed |
| Failed Tasks | `spark_executor_failed_tasks_total` | Counter | Total failed |
| Task Duration | `spark_executor_task_duration_ms_sum` | Counter | Total task time |

**Derived Metrics**:
- **Completion Rate**: `rate(completed_tasks[5m])`
- **Failure Rate**: `rate(failed_tasks[5m])`
- **Avg Duration**: `duration_sum / completed_tasks`

### Shuffle Metrics

| Metric | Prometheus Name | Type | Description |
|--------|----------------|------|-------------|
| Shuffle Read | `spark_shuffle_read_bytes_total` | Counter | Bytes read |
| Shuffle Write | `spark_shuffle_write_bytes_total` | Counter | Bytes written |
| Fetch Wait Time | `spark_shuffle_fetch_wait_time_millis_total` | Counter | Wait time in ms |

**Interpretation**:
- **High Shuffle Read**: Data skew or poor partitioning
- **High Fetch Wait**: Network bottleneck or stragglers
- **High Shuffle Write**: Large intermediate datasets

### Thread Metrics

| Metric | Prometheus Name | Type | Description |
|--------|----------------|------|-------------|
| Thread Count | `jvm_thread_count` | Gauge | Current thread count |
| Peak Thread Count | `jvm_thread_peak_count` | Gauge | Historical peak |
| Daemon Threads | `jvm_thread_daemon_count` | Gauge | Daemon thread count |

---

## Querying Metrics

### Direct Pod Query

```bash
# Get driver pod name
DRIVER_POD=$(kubectl get pods -n default -l spark-role=driver -o jsonpath='{.items[0].metadata.name}')

# Port-forward to driver pod
kubectl port-forward -n default $DRIVER_POD 4040:4040

# Query driver metrics
curl -s http://localhost:4040/metrics/driver/prometheus/ | grep spark.app.status

# Query executor metrics
curl -s http://localhost:4040/metrics/executors/prometheus/ | grep jvm.heap.used
```

### Via Proxy Server

```bash
# List Spark pods
curl -s http://localhost:8083/api/api/v1/namespaces/default/pods?labelSelector=spark-role%3Ddriver

# Get metrics from specific pod
POD_NAME="spark-pi-driver"
curl -s "http://localhost:8083/api/api/v1/namespaces/default/pods/${POD_NAME}/proxy/metrics/driver/prometheus/"
```

### Prometheus Query Examples

If using Prometheus server:

```promql
# Average heap usage across executors
avg(spark_executor_memory_used_bytes)

# Total shuffle read bytes
sum(rate(spark_shuffle_read_bytes_total[5m]))

# GC time per executor
rate(jvm_gc_time_millis_total[5m])

# Task completion rate
rate(spark_executor_completed_tasks_total[5m])

# Failed tasks
rate(spark_executor_failed_tasks_total[5m])

# Memory usage percentage
(spark_executor_memory_used_bytes / spark_executor_memory_max_bytes) * 100
```

---

## Dashboard Usage

### Access Dashboard

1. **Open browser** and navigate to:
   ```
   http://localhost:8083/spark-metrics-ui.html
   ```

2. **Select Spark Pod** from the dropdown

3. **View metrics**:
   - Memory usage and GC
   - Shuffle metrics
   - Executor statistics

### Dashboard Workflow

```
┌────────────────────────────────────────────────────────────┐
│  1. Select Pod from Dropdown                               │
│     └─ Lists all Spark driver pods in cluster              │
└────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌────────────────────────────────────────────────────────────┐
│  2. Dashboard Fetches Metrics                              │
│     ├─ Driver metrics: /metrics/driver/prometheus/         │
│     └─ Executor metrics: /metrics/executors/prometheus/    │
└────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌────────────────────────────────────────────────────────────┐
│  3. Display Visualizations                                 │
│     ├─ Memory chart (heap usage)                           │
│     ├─ Shuffle metrics (read/write)                        │
│     └─ Executor stats (tasks)                              │
└────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌────────────────────────────────────────────────────────────┐
│  4. Auto-Refresh (every 5 seconds)                         │
└────────────────────────────────────────────────────────────┘
```

### Understanding Dashboard Output

#### Memory Section

```
┌─────────────────────────────────────────────────────────┐
│ Memory Usage                                             │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Heap Used:  ████░░░░░░░░ 128 MB / 512 MB (25%)      │ │
│ │ GC Time:   1,234 ms                                 │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

**What to look for**:
- Heap % consistently > 80% → Increase memory
- GC time increasing rapidly → Memory leak or insufficient heap

#### Shuffle Section

```
┌─────────────────────────────────────────────────────────┐
│ Shuffle Metrics                                          │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Read:  100.5 MB                                      │ │
│ │ Write: 50.2 MB                                       │ │
│ └─────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

**What to look for**:
- Read >> Write → Data skew
- Write continuously growing → Spill to disk

#### Executors Section

```
┌─────────────────────────────────────────────────────────┐
│ Executor Statistics                                       │
│ ┌──────────┬──────────┬──────────┬──────────┐          │
│ │ Active   │ Complete │ Failed   │ Total    │          │
│ │ 4        │ 156      │ 0        │ 160      │          │
│ └──────────┴──────────┴──────────┴──────────┘          │
└─────────────────────────────────────────────────────────┘
```

**What to look for**:
- Failed > 0 → Check application logs
- Active = 0 but Complete < Total → Application stalled

---

## Installation

### Prerequisites

1. **Proxy server running** on port 8083:
   ```bash
   cd /root/hynix
   go run cmd/proxy/main.go > proxy.log 2>&1 &
   ```

2. **Spark applications deployed** with Prometheus metrics enabled:
   ```yaml
   sparkConf:
     "spark.metrics.conf.*.sink.prometheus.class": "org.apache.spark.metrics.sink.PrometheusSink"
     "spark.metrics.app.status": "true"
   ```

3. **kubectl configured** for cluster access

### Verify Setup

```bash
# Check proxy server
curl http://localhost:8083

# List Spark pods
kubectl get pods -A -l spark-role=driver

# Verify metrics endpoint
POD=$(kubectl get pods -A -l spark-role=driver -o jsonpath='{.items[0].metadata.name}')
NS=$(kubectl get pods -A -l spark-role=driver -o jsonpath='{.items[0].metadata.namespace}')
kubectl exec -n $NS $POD -- curl -s http://localhost:4040/metrics/driver/prometheus/ | head -20
```

---

## API Reference

### List Spark Driver Pods

**Endpoint:** `GET /api/api/v1/namespaces/{namespace}/pods?labelSelector=spark-role=driver`

**Description:** Lists all Spark driver pods in a namespace

**Example:**
```bash
curl -s "http://localhost:8083/api/api/v1/namespaces/default/pods?labelSelector=spark-role%3Ddriver" | jq '.items[] | {name: .metadata.name, app: .metadata.labels."spark-app-selector"}'
```

**Response:**
```json
{
  "items": [
    {
      "metadata": {
        "name": "spark-pi-driver",
        "labels": {
          "spark-app-selector": "spark-pi",
          "spark-role": "driver"
        }
      },
      "status": {
        "phase": "Running",
        "podIP": "10.244.1.5"
      }
    }
  ]
}
```

### Get Driver Metrics

**Endpoint:** `GET /api/api/v1/namespaces/{namespace}/pods/{pod-name}/proxy/metrics/driver/prometheus/`

**Description:** Fetches driver-level metrics in Prometheus format

**Example:**
```bash
curl -s "http://localhost:8083/api/api/v1/namespaces/default/pods/spark-pi-driver/proxy/metrics/driver/prometheus/"
```

**Response:**
```
# HELP spark_app_status_uptime Application uptime in milliseconds
# TYPE spark_app_status_uptime gauge
spark_app_status_uptime{app_id="spark-pi"} 123456789

# HELP spark_app_status_running_executors Number of running executors
# TYPE spark_app_status_running_executors gauge
spark_app_status_running_executors{app_id="spark-pi"} 2

# HELP jvm_heap_used_bytes JVM heap memory used
# TYPE jvm_heap_used_bytes gauge
jvm_heap_used_bytes{app_id="spark-pi"} 134217728
```

### Get Executor Metrics

**Endpoint:** `GET /api/api/v1/namespaces/{namespace}/pods/{pod-name}/proxy/metrics/executors/prometheus/`

**Description:** Fetches metrics for all executors in Prometheus format

**Example:**
```bash
curl -s "http://localhost:8083/api/api/v1/namespaces/default/pods/spark-pi-driver/proxy/metrics/executors/prometheus/"
```

**Response:**
```
# HELP jvm_heap_used_bytes JVM heap memory used
# TYPE jvm_heap_used_bytes gauge
jvm_heap_used_bytes{app_id="spark-pi",executor_id="1"} 134217728

# HELP spark_executor_completed_tasks_total Total completed tasks per executor
# TYPE spark_executor_completed_tasks_total counter
spark_executor_completed_tasks_total{app_id="spark-pi",executor_id="1"} 78

# HELP jvm_gc_time_millis_total Time spent in GC
# TYPE jvm_gc_time_millis_total counter
jvm_gc_time_millis_total{app_id="spark-pi",executor_id="1"} 1234
```

### Metrics via kubectl (Direct)

If not using the proxy:

```bash
# Get driver pod
POD="spark-pi-driver"
NS="default"

# Port-forward
kubectl port-forward -n $NS $POD 4040:4040 &

# Fetch metrics
curl -s http://localhost:4040/metrics/driver/prometheus/
curl -s http://localhost:4040/metrics/executors/prometheus/
```

---

## Integration with Prometheus

### Scrape Config

Add to `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'spark-driver'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - default
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_spark_role]
        action: keep
        regex: driver
      - source_labels: [__meta_kubernetes_pod_ip]
        action: replace
        target_label: __address__
        regex: (.+)
        replacement: ${1}:4040
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: pod
```

### Grafana Dashboard Queries

**Heap Usage:**
```promql
sum(jvm_heap_used_bytes{pod=~"spark-.*"}) by (pod) /
sum(jvm_heap_max_bytes{pod=~"spark-.*"}) by (pod) * 100
```

**Task Completion Rate:**
```promql
rate(spark_executor_completed_tasks_total[5m])
```

**Failed Tasks:**
```promql
rate(spark_executor_failed_tasks_total[5m])
```

**Shuffle Bytes:**
```promql
rate(spark_shuffle_read_bytes_total[5m])
rate(spark_shuffle_write_bytes_total[5m])
```

---

## Troubleshooting

### Issue: Dashboard Shows No Pods

**Symptoms:** Pod dropdown is empty

**Solutions:**

1. **Check for Spark applications:**
   ```bash
   kubectl get sparkapplication -A
   kubectl get pods -A -l spark-role=driver
   ```

2. **Verify labels:**
   ```bash
   kubectl get pods -A -l spark-role=driver --show-labels
   ```

3. **Check Spark Operator:**
   ```bash
   kubectl get pods -n spark-operator
   kubectl logs -n spark-operator -l app.kubernetes.io/name=spark-operator
   ```

### Issue: Metrics Not Loading

**Symptoms:** Charts show "No data available"

**Solutions:**

1. **Verify metrics endpoint:**
   ```bash
   POD=$(kubectl get pods -A -l spark-role=driver -o jsonpath='{.items[0].metadata.name}')
   NS=$(kubectl get pods -A -l spark-role=driver -o jsonpath='{.items[0].metadata.namespace}')
   kubectl exec -n $NS $POD -- curl -s http://localhost:4040/metrics/driver/prometheus/ | head -10
   ```

2. **Check Prometheus sink is configured:**
   ```bash
   kubectl get sparkapplication -A -o yaml | grep -A 5 "prometheus"
   ```

3. **Verify Spark configuration:**
   ```yaml
   sparkConf:
     "spark.metrics.conf.*.sink.prometheus.class": "org.apache.spark.metrics.sink.PrometheusSink"
     "spark.metrics.app.status": "true"
     "spark.metrics.executors.status": "true"
   ```

### Issue: "Failed to fetch metrics"

**Symptoms:** Error message in dashboard

**Solutions:**

1. **Check proxy server logs:**
   ```bash
   tail -f /root/hynix/proxy.log
   ```

2. **Verify pod is running:**
   ```bash
   kubectl get pods -A -l spark-role=driver
   ```

3. **Test direct access:**
   ```bash
   curl -s "http://localhost:8083/api/api/v1/namespaces/default/pods/spark-pi-driver/proxy/metrics/driver/prometheus/"
   ```

4. **Check kubectl permissions:**
   ```bash
   kubectl auth can-i get pods --all-namespaces
   kubectl auth can-i exec pods --all-namespaces
   ```

### Issue: Metrics Not Updating

**Symptoms:** Dashboard shows stale data

**Solutions:**

1. **Check auto-refresh is enabled:**
   - Open browser console (F12)
   - Look for errors in Console tab

2. **Manually refresh:**
   - Press F5 or Ctrl+R

3. **Verify metrics are being generated:**
   ```bash
   # Check if metrics timestamp is recent
   kubectl exec -n default spark-pi-driver -- curl -s http://localhost:4040/metrics/driver/prometheus/ | grep spark_app_status_uptime
   ```

### Issue: High Memory Usage but Low Task Count

**Symptoms:** Heap > 80% but Active Tasks = 0

**Possible Causes:**
1. **Memory leak** in application code
2. **Cached data** not being released
3. **Garbage collection** not freeing memory

**Solutions:**

1. **Check GC activity:**
   ```bash
   kubectl exec spark-pi-driver -- jcmd 1 GC.run
   kubectl exec spark-pi-driver -- jcmd 1 GC.run_finalization
   ```

2. **Increase heap size:**
   ```yaml
   driver:
     memory: "2048m"  # Increase from 512m
   ```

3. **Check for cached RDDs:**
   - Review application code for `persist()` or `cache()` calls
   - Ensure `unpersist()` is called when done

---

## Best Practices

### Monitoring Configuration

1. **Set appropriate metrics interval:**
   ```properties
   spark.metrics.conf.*.sink.prometheus.period=5s  # Balance between freshness and overhead
   ```

2. **Enable essential metrics only:**
   ```yaml
   sparkConf:
     "spark.metrics.app.status": "true"
     "spark.metrics.executors.status": "true"
     "spark.metrics.conf.master.source.jvm.class": "org.apache.spark.metrics.source.JvmSource"
   ```

3. **Monitor key ratios:**
   - **Heap %**: Keep < 80%
   - **GC Time Rate**: Keep < 5% of execution time
   - **Task Failure Rate**: Keep < 1%

### Dashboard Usage

1. **Regular checks:**
   - Monitor heap usage trends
   - Track shuffle growth
   - Watch for failed task spikes

2. **Alert thresholds:**
   - Heap > 85% → Investigate
   - GC rate > 10% → Consider tuning
   - Failed tasks > 0 → Check logs

3. **Historical analysis:**
   - Export metrics to Prometheus
   - Create Grafana dashboards
   - Track long-term trends

### Performance Optimization

1. **Memory tuning:**
   ```yaml
   sparkConf:
     "spark.memory.fraction": "0.6"
     "spark.memory.storageFraction": "0.5"
     "spark.executor.extraJavaOptions": "-XX:+UseG1GC"
   ```

2. **Shuffle optimization:**
   ```yaml
   sparkConf:
     "spark.sql.shuffle.partitions": "200"
     "spark.shuffle.compress": "true"
     "spark.shuffle.spill.compress": "true"
   ```

3. **Task tuning:**
   ```yaml
   sparkConf:
     "spark.task.maxFailures": "4"
     "spark.speculation": "true"
     "spark.speculation.multiplier": "1.5"
   ```

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: spark_alerts
    rules:
      - alert: HighGCOverhead
        expr: rate(jvm_gc_time_millis_total[5m]) > 100
        for: 10m
        annotations:
          summary: "High GC overhead for {{ $labels.app_id }}"

      - alert: FailedTasksSpike
        expr: rate(spark_executor_failed_tasks_total[5m]) > 10
        for: 5m
        annotations:
          summary: "Failed tasks spike in {{ $labels.app_id }}"

      - alert: ExecutorMemoryHigh
        expr: (spark_executor_memory_used_bytes / spark_executor_memory_max_bytes) > 0.9
        for: 10m
        annotations:
          summary: "Executor memory > 90% for {{ $labels.executor_id }}"
```

---

## References

- [Spark Monitoring](https://spark.apache.org/docs/latest/monitoring.html)
- [Spark Metrics System](https://spark.apache.org/docs/latest/configuration.html#metrics)
- [Prometheus Format](https://prometheus.io/docs/instrumenting/exposition_formats/)
- [Spark Operator Documentation](https://github.com/GoogleCloudPlatform/spark-on-k8s-operator)
- [Prometheus Metrics](https://prometheus.io/docs/practices/naming/)
- [JVM Metrics](https://docs.oracle.com/javase/8/docs/technotes/guides/management/jconsole.html)
