# Yunikorn Scheduler Dashboard

Comprehensive guide to the Yunikorn Scheduler dashboard for monitoring resource allocation, queue management, and application scheduling on Kubernetes.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Dashboard Features](#dashboard-features)
- [Queue Management](#queue-management)
- [Application Monitoring](#application-monitoring)
- [Installation](#installation)
- [Usage](#usage)
- [API Reference](#api-reference)
- [Troubleshooting](#troubleshooting)

---

## Overview

The Yunikorn Scheduler Dashboard provides real-time visibility into:

- **Active Applications**: Running Spark applications with submission time and state
- **Queue Management**: Resource allocation by queue, pending applications
- **Resource Monitoring**: vcore and memory usage at partition and queue level
- **Application Details**: Per-application resource requests and allocations

### Dashboard Location

```
http://localhost:8083/yunikorn-ui.html
```

### Key Technologies

| Component | Purpose |
|-----------|---------|
| **Yunikorn Scheduler** | Gang scheduling for Kubernetes pods |
| **REST API** | Scheduler metrics and management |
| **Proxy Server** | Go proxy at :8083 forwarding to Yunikorn API |
| **Kubernetes** | Cluster resource management |

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Web Browser                                │
│                  http://localhost:8083                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     Yunikorn Scheduler Dashboard                         │  │
│  │     /yunikorn-ui.html                                   │  │
│  │  - Active Applications View                               │  │
│  │  - Queue Explorer                                        │  │
│  │  - Queue Applications                                    │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ HTTP API
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Proxy Server (Go)                             │
│                       :8083                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     Yunikorn API Proxy                                 │  │
│  │     /api/ws/* → http://localhost:9080/ws/v1/*         │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Yunikorn Scheduler                               │
│                      :9080                                     │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │     REST API Endpoints                                  │  │
│  │  - /ws/v1/partition/{partition}/applications/active    │  │
│  │  - /ws/v1/partition/{partition}/queue/{queueName}       │  │
│  │  - /ws/v1/partition/{partition}/queue/{queueName}/apps   │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Manages:                                                       │
│  - Gang scheduling (all-or-nothing pod placement)            │
│  - Queue hierarchy (root.default)                            │
│  - Resource allocation (vcores, memory)                        │
└─────────────────────────────────────────────────────────────────┘
```

---

## Dashboard Features

### 1. Active Applications View

**Location**: Main dashboard view (auto-loaded)

**Purpose**: Real-time list of all running applications

**Auto-refresh**: Every 5 seconds

**Columns Displayed**:

| Column | Description | Example |
|--------|-------------|---------|
| **Application ID** | Unique application identifier | `application_1723326868472_0001` |
| **Submission Time** | When app was submitted (formatted) | `2024-08-10 12:34:56` |
| **State** | Current application state | `RUNNING`, `COMPLETED`, `FAILED` |
| **Partition** | Yunikorn partition | `default` |

**Example Output**:
```
┌─────────────────────────────────────────────────────────────────┐
│  Active Applications (Auto-refresh: 5s)                           │
├─────────────────────────────────────────────────────────────────┤
│  Application ID                     │ State    │ Submission      │
│  application_1723326868472_0001   │ RUNNING  │ 2024-08-10 12:34│
│  application_1723326891234_0002   │ RUNNING  │ 2024-08-10 12:35│
│  application_1723326812345_0003   │ COMPLETED│ 2024-08-10 11:20│
└─────────────────────────────────────────────────────────────────┘
```

### 2. Queue Explorer

**Location**: Queue Explorer section

**Purpose**: View queue properties and resource allocation

**Input Fields**:
- **Partition Name**: Yunikorn partition (default: `default`)
- **Queue Name**: Queue path (e.g., `root.default`)

**Information Displayed**:

| Property | Description | Example |
|----------|-------------|---------|
| **Queue Name** | Full queue path | `root.default` |
| **Allocated Containers** | Number of pods allocated | `3` |
| **Allocated VCores** | CPU cores allocated | `3` |
| **Allocated Memory** | Memory allocated (MB) | `1536` |
| **Running Applications** | Active apps in queue | `2` |
| **Queue Properties** | Configuration settings | `application_sort_policy: fifo` |

### 3. Queue Applications

**Location**: Queue Applications section (after queue info)

**Purpose**: List all applications in a specific queue

**Filters**: Filter by application state (optional)

**Columns**:
- Application ID
- Submission time
- State
- Partition

---

## Queue Management

### Queue Hierarchy

Yunikorn uses hierarchical queues:

```
root (cluster root)
└── default (partition)
    └── root
        ├── default (default queue for Spark apps)
        └── production (production queue)
            ├── critical (high priority)
            └── batch (low priority)
```

### Queue Properties

| Property | Type | Description | Example |
|----------|------|-------------|---------|
| `application_sort_policy` | String | How apps are scheduled | `fifo`, `fair` |
| `priority_policy` | String | Priority policy | `fifo`, `priority` |
| `allocation_min` | Integer | Minimum resource allocation | `1024` (MB) |
| `allocation_max` | Integer | Maximum resource allocation | `8192` (MB) |
| `guaranteed_resource` | Object | Guaranteed resources | `{memory: 2048, vcore: 2}` |
| `max_resource` | Object | Maximum resources | `{memory: 8192, vcore: 8}` |

### Resource Allocation

**Vcores (CPU)**:
- Each executor typically uses 1 vcore
- Driver pod uses vcores based on configuration
- Total vcores = sum of all pod requests

**Memory**:
- Specified in MB
- Driver memory + executor memory × number of executors
- Overhead included for Spark framework

**Example Calculation**:
```
Driver: 1 vcore, 512 MB
Executors: 2 × (1 vcore, 1024 MB)
Total: 3 vcores, 2560 MB
```

---

## Application Monitoring

### Application States

| State | Description | Typical Duration |
|-------|-------------|------------------|
| **NEW** | Application submitted, not yet scheduled | Seconds |
| **RUNNING** | Resources allocated, tasks executing | Minutes to hours |
| **COMPLETED** | All tasks finished successfully | Terminal |
| **FAILED** | Application failed | Terminal |
| **KILLED** | Application terminated | Terminal |

### Application Lifecycle

```
┌─────────────┐
│   NEW       │ Submitted to Yunikorn
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  SCHEDULED  │ Gang scheduling: all or nothing
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  RUNNING    │ Tasks executing
└──────┬──────┘
       │
       ▼
┌─────────────────────────┐
│  COMPLETED or FAILED    │
└─────────────────────────┘
```

### Gang Scheduling

Yunikorn uses gang scheduling for Spark applications:

**Task Groups**:
1. **Driver Task Group**
   - `minMember`: 1
   - `minResource`: 100m CPU, 512Mi memory

2. **Executor Task Group**
   - `minMember`: Number of executors
   - `minResource`: 100m CPU, 512Mi memory per executor

**Behavior**:
- All-or-nothing scheduling
- Resources reserved for entire task group
- Placeholder pods used for reservation
- Prevents partial allocation (some executors but not all)

---

## Installation

### Prerequisites

1. **Yunikorn Scheduler installed** on Kubernetes
2. **Port-forward** for Yunikorn service:
   ```bash
   kubectl port-forward -n yunikorn svc/yunikorn-service 9080:8080
   ```
3. **Proxy server running** on port 8083:
   ```bash
   cd /root/hynix
   go run cmd/proxy/main.go > proxy.log 2>&1 &
   ```

### Access Dashboard

```
http://localhost:8083/yunikorn-ui.html
```

---

## Usage

### View Active Applications

1. **Open dashboard**: Auto-loads on page open
2. **Watch updates**: List refreshes every 5 seconds
3. **Interpret states**:
   - **RUNNING**: Application is actively processing
   - **COMPLETED**: Check logs for results
   - **FAILED**: Investigate failure reason

### Explore Queue Information

1. **Enter partition name** (default: `default`)
2. **Enter queue name** (e.g., `root.default`)
3. **Click "Get Queue Info"**
4. **Review**:
   - Allocated resources
   - Running applications
   - Queue properties

### View Queue Applications

1. **Enter partition name** (default: `default`)
2. **Enter queue name** (e.g., `root.default`)
3. **(Optional) Enter application state filter**
4. **Click "Get Queue Applications"**
5. **Review** application list for that queue

### Common Workflows

#### Check Resource Utilization

1. Go to Queue Explorer
2. Enter partition: `default`, queue: `root.default`
3. Check "Allocated VCores" and "Allocated Memory"
4. Compare with cluster capacity

#### Debug Scheduling Issues

1. Check Active Applications for stuck apps in `NEW` state
2. Use Queue Explorer to check available resources
3. Review Queue Applications for pending apps
4. Check Yunikorn scheduler logs

#### Monitor Specific Application

1. Note Application ID from Active Applications
2. Use Queue Applications with queue name
3. Filter by Application ID (if supported)
4. Check application state over time

---

## API Reference

### Base URL

```
http://localhost:9080/ws/v1
```

### Get Active Applications

**Endpoint**: `GET /ws/v1/partition/{partition}/applications/active`

**Example**:
```bash
curl http://localhost:9080/ws/v1/partition/default/applications/active
```

**Response**:
```json
{
  "applications": [
    {
      "applicationID": "application_1723326868472_0001",
      "submissionTime": 1723326868472,
      "state": "RUNNING",
      "partition": "default"
    }
  ]
}
```

### Get Queue Information

**Endpoint**: `GET /ws/v1/partition/{partition}/queue/{queueName}`

**Example**:
```bash
curl http://localhost:9080/ws/v1/partition/default/queue/root.default
```

**Response**:
```json
{
  "queueName": "root.default",
  "allocatedContainers": 3,
  "allocatedVCores": 3,
  "allocatedMemory": 1536,
  "properties": {
    "application_sort_policy": "fifo"
  },
  "state": "RUNNING"
}
```

### Get Queue Applications

**Endpoint**: `GET /ws/v1/partition/{partition}/queue/{queueName}/applications`

**Query Parameters**:
- `applicationStates` (optional): Filter by state (e.g., `RUNNING,COMPLETED`)

**Example**:
```bash
curl "http://localhost:9080/ws/v1/partition/default/queue/root.default/applications?applicationStates=RUNNING"
```

**Response**:
```json
{
  "applications": [
    {
      "applicationID": "application_1723326868472_0001",
      "submissionTime": 1723326868472,
      "state": "RUNNING",
      "partition": "default",
      "queueName": "root.default"
    }
  ]
}
```

---

## Troubleshooting

### Issue: Dashboard Shows "Connection Refused"

**Symptoms**: API calls fail, no applications displayed

**Solutions**:

1. **Check Yunikorn is running**:
   ```bash
   kubectl get pods -n yunikorn
   ```

2. **Start port-forward**:
   ```bash
   kubectl port-forward -n yunikorn svc/yunikorn-service 9080:8080
   ```

3. **Verify API accessible**:
   ```bash
   curl http://localhost:9080/ws/v1/partitions
   ```

### Issue: No Active Applications Shown

**Symptoms**: Active Applications list is empty

**Solutions**:

1. **Check partition name**:
   - Default partition is usually `default`
   - Verify with: `kubectl get cm -n yunikorn yunikorn-configs -o yaml`

2. **Check if apps are running**:
   ```bash
   kubectl get sparkapplication -A
   kubectl get pods -A -l spark-app-selector
   ```

3. **Check Yunikorn API directly**:
   ```bash
   curl http://localhost:9080/ws/v1/partition/default/applications/active
   ```

### Issue: Queue Not Found

**Symptoms**: "Queue not found" error

**Solutions**:

1. **Verify queue exists**:
   - Check Yunikorn configuration: `kubectl get cm -n yunikorn yunikorn-configs -o yaml`
   - Look for queue in `queues.yaml` section

2. **Use full queue path**:
   - Include full hierarchy: `root.default`
   - Not just: `default`

3. **Check partition name**:
   - Partition is typically `default`
   - Verify with: `curl http://localhost:9080/ws/v1/partitions`

### Issue: Applications Stuck in NEW State

**Symptoms**: Applications show `NEW` but never start

**Possible Causes**:
1. Insufficient cluster resources
2. Gang scheduling can't satisfy all task groups
3. Queue resource limits reached

**Solutions**:

1. **Check cluster resources**:
   ```bash
   kubectl top nodes
   kubectl describe node | grep -A 5 "Allocated resources"
   ```

2. **Check Yunikorn logs**:
   ```bash
   kubectl logs -n yunikorn deployment/yunikorn-scheduler
   ```

3. **Review application resource requests**:
   ```bash
   kubectl get sparkapplication <app-name> -o yaml | grep -A 20 "gang scheduling"
   ```

4. **Check queue capacity**:
   - Use Queue Explorer to see allocated vs max resources
   - Consider increasing queue `max_resource` limits

### Issue: Proxy Server Errors

**Symptoms**: Browser console shows 404 or connection errors

**Solutions**:

1. **Check proxy server is running**:
   ```bash
   ps aux | grep "go run cmd/proxy/main.go"
   tail -f /root/hynix/proxy.log
   ```

2. **Restart proxy server**:
   ```bash
   pkill -f "go run cmd/proxy/main.go"
   cd /root/hynix && go run cmd/proxy/main.go > proxy.log 2>&1 &
   ```

3. **Test Yunikorn API directly**:
   ```bash
   curl http://localhost:8083/api/ws/v1/partitions
   ```

---

## Best Practices

### Queue Configuration

1. **Set appropriate resource limits**:
   - `max_resource` should be less than cluster capacity
   - Leave headroom for system pods and other workloads

2. **Use priority policies**:
   - `fifo`: First-in-first-out (fair, but may starve large apps)
   - `priority`: Higher priority apps scheduled first

3. **Enable application sorting**:
   - `fifo`: Simple, predictable
   - `fair`: Balanced across users/queues

### Gang Scheduling

1. **Set correct minMember values**:
   - Driver: always `1`
   - Executors: match your executor count
   - Must match actual pod count for scheduling to succeed

2. **Reasonable minResource values**:
   - Don't set too high or apps may never schedule
   - Consider minimum viable resources for your workload

3. **Monitor placeholder pods**:
   - Yunikorn creates placeholder pods for reservation
   - These occupy resources while waiting for full task group
   - Too many placeholders = inefficient cluster usage

### Monitoring

1. **Track resource utilization**:
   - Monitor allocated vs available resources
   - Scale queue limits as needed
   - Remove completed applications from view

2. **Alert on scheduling failures**:
   - Apps stuck in NEW state > 5 minutes
   - Queue reaching max capacity
   - High number of placeholder pods

3. **Regular queue cleanup**:
   - Remove old queue configurations
   - Update resource limits based on usage patterns
   - Archive historical metrics data

---

## Integration with Spark Operator

### Gang Scheduling Annotations

Add to SparkApplication metadata:

```yaml
apiVersion: sparkoperator.k8s.io/v1beta2
kind: SparkApplication
metadata:
  name: spark-app
  annotations:
    yunikorn.apache.org/task-groups: |-
      [
        {
          "name": "spark-driver",
          "minMember": 1,
          "minResource": {
            "cpu": "100m",
            "memory": "512Mi"
          }
        },
        {
          "name": "spark-executor",
          "minMember": 2,
          "minResource": {
            "cpu": "100m",
            "memory": "512Mi"
          }
        }
      ]
spec:
  batchScheduler: yunikorn
  batchSchedulerOptions:
    queue: root.default
  driver:
    cores: 1
    coreLimit: "1200m"
    memory: "512m"
  executor:
    cores: 1
    coreLimit: "1200m"
    instances: 2
    memory: "512m"
```

### Key Points

- **batchScheduler**: Must be `yunikorn`
- **queue**: Target queue (must exist)
- **minMember**: Must match `executor.instances`
- **minResource**: Should match pod resource requests

---

## References

- [Yunikorn Scheduler Documentation](https://yunikorn.apache.org/docs/)
- [Yunikorn REST API](https://yunikorn.apache.org/docs/design/rest_api/)
- [Spark Operator Integration](https://yunikorn.apache.org/docs/design/spark_integration/)
- [Gang Scheduling](https://yunikorn.apache.org/docs/design/gang_scheduling/)
